package preconditions

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestRevisedIndependenceV2SyntheticClustering(t *testing.T) {
	base := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	policy := RevisedIndependencePolicyV2()
	t.Run("isolated same symbol", func(t *testing.T) {
		clusters := mustClusterV2(t, []RetainedEvent{syntheticEvent(t, "AAAUSDT", base), syntheticEvent(t, "AAAUSDT", base.Add(5*time.Hour))}, policy)
		if len(clusters) != 2 {
			t.Fatalf("clusters=%d, want 2", len(clusters))
		}
	})
	t.Run("overlapping and transitive same symbol", func(t *testing.T) {
		events := []RetainedEvent{syntheticEvent(t, "AAAUSDT", base), syntheticEvent(t, "AAAUSDT", base.Add(3*time.Hour)), syntheticEvent(t, "AAAUSDT", base.Add(6*time.Hour))}
		clusters := mustClusterV2(t, events, policy)
		if len(clusters) != 1 || len(clusters[0].MemberEventIDs) != 3 {
			t.Fatalf("transitive clustering=%v", clusters)
		}
	})
	t.Run("exact boundary and one nanosecond before", func(t *testing.T) {
		atBoundary := mustClusterV2(t, []RetainedEvent{syntheticEvent(t, "AAAUSDT", base), syntheticEvent(t, "AAAUSDT", base.Add(4*time.Hour))}, policy)
		before := mustClusterV2(t, []RetainedEvent{syntheticEvent(t, "AAAUSDT", base), syntheticEvent(t, "AAAUSDT", base.Add(4*time.Hour-time.Nanosecond))}, policy)
		if len(atBoundary) != 2 || len(before) != 1 {
			t.Fatalf("boundary clusters exact=%d before=%d", len(atBoundary), len(before))
		}
	})
	t.Run("same instant encodings deduplicate", func(t *testing.T) {
		local := time.FixedZone("synthetic-offset", 9*60*60)
		first := syntheticEvent(t, "AAAUSDT", base)
		second := syntheticEvent(t, "AAAUSDT", base.In(local))
		clusters := mustClusterV2(t, []RetainedEvent{first, second}, policy)
		if first.EventID != second.EventID || len(clusters) != 1 || len(clusters[0].MemberEventIDs) != 1 {
			t.Fatal("equivalent UTC instants did not normalize and deduplicate")
		}
	})
}

func TestRevisedIndependenceV2CrossSymbolEpisodeRules(t *testing.T) {
	base := time.Date(2030, 2, 1, 0, 0, 0, 0, time.UTC)
	policy := RevisedIndependencePolicyV2()
	left := syntheticEvent(t, "AAAUSDT", base)
	rightSame := withSyntheticEpisodeV2(t, syntheticEvent(t, "BBBUSDT", base.Add(time.Hour)), left)
	rightDifferent := syntheticEvent(t, "BBBUSDT", base.Add(time.Hour))
	nonoverlap := withSyntheticEpisodeV2(t, syntheticEvent(t, "BBBUSDT", base.Add(4*time.Hour)), left)

	same := mustClusterV2(t, []RetainedEvent{left, rightSame}, policy)
	different := mustClusterV2(t, []RetainedEvent{left, rightDifferent}, policy)
	separate := mustClusterV2(t, []RetainedEvent{left, nonoverlap}, policy)
	if len(same) != 1 || len(same[0].CommonMarketEpisodeIdentities) != 1 {
		t.Fatal("overlapping equal episode did not cross-symbol cluster")
	}
	if len(different) != 2 || len(separate) != 2 {
		t.Fatal("different or nonoverlapping cross-symbol events were clustered")
	}
	if same[0].CommonMarketEpisodeIdentities[0] == "" {
		t.Fatal("cross-symbol cluster omitted common-market identity")
	}
}

func TestRevisedIndependenceV2DeterminismDuplicatesAndHashMutation(t *testing.T) {
	base := time.Date(2030, 3, 1, 0, 0, 0, 0, time.UTC)
	policy := RevisedIndependencePolicyV2()
	a := syntheticEvent(t, "AAAUSDT", base)
	b := syntheticEvent(t, "AAAUSDT", base.Add(time.Hour))
	forward := mustClusterV2(t, []RetainedEvent{a, b, a}, policy)
	reverse := mustClusterV2(t, []RetainedEvent{b, a}, policy)
	left, _ := json.Marshal(forward)
	right, _ := json.Marshal(reverse)
	if string(left) != string(right) || len(forward[0].MemberEventIDs) != 2 {
		t.Fatal("duplicates or input order changed canonical clusters")
	}
	if err := ValidateIndependentClusterV2(forward[0], policy); err != nil {
		t.Fatal(err)
	}
	mutatedCluster := forward[0]
	mutatedCluster.MemberEventIDs = append([]string(nil), mutatedCluster.MemberEventIDs...)
	mutatedCluster.MemberEventIDs[0] += "-mutated"
	if err := ValidateIndependentClusterV2(mutatedCluster, policy); err == nil {
		t.Fatal("cluster mutation did not invalidate its hash identity")
	}
	originalHash, err := RevisedIndependencePolicyHashV2(policy)
	if err != nil {
		t.Fatal(err)
	}
	mutatedPolicy := policy
	mutatedPolicy.SameSymbolRule += "; synthetic mutation"
	mutatedHash, err := canonicalDigest(mutatedPolicy)
	if err != nil {
		t.Fatal(err)
	}
	if originalHash == mutatedHash {
		t.Fatal("contract mutation did not change hash")
	}
	if _, err := RevisedIndependencePolicyHashV2(mutatedPolicy); err == nil {
		t.Fatal("same-version policy mutation was accepted")
	}
}

func TestRevisedIndependenceV2MissingIdentitiesFailClosed(t *testing.T) {
	base := time.Date(2030, 4, 1, 0, 0, 0, 0, time.UTC)
	policy := RevisedIndependencePolicyV2()
	tests := []struct {
		name   string
		mutate func(*RetainedEvent)
	}{
		{"event identity", func(event *RetainedEvent) { event.EventID = "" }},
		{"timestamp", func(event *RetainedEvent) { event.EventTimestamp = time.Time{} }},
		{"common market identity", func(event *RetainedEvent) { event.BTCContext.SnapshotID = "" }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			event := syntheticEvent(t, "AAAUSDT", base)
			test.mutate(&event)
			if _, err := ClusterEventsV2([]RetainedEvent{event}, policy); err == nil {
				t.Fatal("invalid event passed closed-boundary validation")
			}
		})
	}
}

func TestRevisedIndependenceV2SyntheticClusterReturn(t *testing.T) {
	base := time.Date(2030, 5, 1, 0, 0, 0, 0, time.UTC)
	cluster := mustClusterV2(t, []RetainedEvent{syntheticEvent(t, "AAAUSDT", base), syntheticEvent(t, "AAAUSDT", base.Add(time.Hour))}, RevisedIndependencePolicyV2())[0]
	returns := []MemberNetReturn{{cluster.MemberEventIDs[1], 2.5}, {cluster.MemberEventIDs[0], -1}}
	result, err := AggregateClusterNetReturnV2(cluster, returns)
	if err != nil || result.NetValue != 1.5 {
		t.Fatalf("synthetic cluster return=%+v err=%v", result, err)
	}
	if _, err := AggregateClusterNetReturnV2(cluster, append(returns, returns[0])); err == nil {
		t.Fatal("duplicate synthetic member return passed")
	}
}

func TestRevisedIndependenceV2RemainsUnaccepted(t *testing.T) {
	policy := RevisedIndependencePolicyV2()
	if policy.Status != PolicyStatusRevisedPendingConcentration || len(policy.UnresolvedItems) == 0 {
		t.Fatal("missing concentration authority was hidden")
	}
	for _, authority := range policy.ConcentrationAuthorities[:2] {
		if !strings.HasPrefix(authority.AuthorityStatus, "UNRESOLVED") || authority.Threshold != "" {
			t.Fatal("largest-cluster authority was invented")
		}
	}
}

func mustClusterV2(t *testing.T, events []RetainedEvent, policy RevisedIndependencePolicy) []IndependentClusterV2 {
	t.Helper()
	clusters, err := ClusterEventsV2(events, policy)
	if err != nil {
		t.Fatal(err)
	}
	for _, cluster := range clusters {
		if err := ValidateIndependentClusterV2(cluster, policy); err != nil {
			t.Fatal(err)
		}
	}
	return clusters
}

func withSyntheticEpisodeV2(t *testing.T, event, episodeSource RetainedEvent) RetainedEvent {
	t.Helper()
	event.BTCContext = episodeSource.BTCContext
	event.ETHContext = episodeSource.ETHContext
	sealed, err := SealRetainedEvent(event)
	if err != nil {
		t.Fatal(err)
	}
	return sealed
}
