package preconditions

import (
	"encoding/json"
	"fmt"
	"math/big"
	"sort"
	"testing"
	"time"
)

func TestV3GovernanceDecisionAndContractAreImmutable(t *testing.T) {
	decision := DefaultConcentrationGovernanceDecisionV3()
	if decision.Decision != "ACCEPT_ALTERNATIVE" || decision.SelectedAlternative != "STRUCTURAL_COUNT_BASED_CONCENTRATION" || decision.DecisionScope != "FUTURE_PR4B0_R1_RESEARCH_ONLY" || decision.HistoricalAuthorityClaimed {
		t.Fatalf("human decision was not recorded exactly: %+v", decision)
	}
	if decision.ProspectiveAuthorityStatement != ProspectiveGovernanceAuthorityText1 || decision.HistoricalAuthorityDisclaimer != ProspectiveGovernanceAuthorityText2 {
		t.Fatal("prospective-authority disclaimer is incomplete")
	}
	if err := ValidateConcentrationGovernanceDecisionV3(decision); err != nil {
		t.Fatal(err)
	}
	mutatedDecision := decision
	mutatedDecision.SelectedAlternative = "OUTCOME_CONTRIBUTION_BASED_CONCENTRATION"
	if err := ValidateConcentrationGovernanceDecisionV3(mutatedDecision); err == nil {
		t.Fatal("decision mutation passed")
	}

	policy := AcceptedIndependencePolicyV3Default()
	if policy.Version != AcceptedIndependencePolicyVersionV3 || policy.Status != PolicyStatusAccepted || policy.Clustering.ExposureHorizonMinutes != 240 || len(policy.ConcentrationAuthorities) != 4 {
		t.Fatalf("accepted V3 contract is incomplete: %+v", policy)
	}
	if err := ValidateAcceptedIndependencePolicyV3(policy); err != nil {
		t.Fatal(err)
	}
	mutatedPolicy := policy
	mutatedPolicy.RoundingRule = "round before comparison"
	if err := ValidateAcceptedIndependencePolicyV3(mutatedPolicy); err == nil {
		t.Fatal("same-version policy mutation passed")
	}
	v2 := RevisedIndependencePolicyV2()
	if v2.Status != PolicyStatusRevisedPendingConcentration || len(v2.UnresolvedItems) == 0 {
		t.Fatal("pending V2 was mutated or relabeled")
	}
}

func TestV3ClusteringPreservesV2SemanticsAndDeduplicatesEvents(t *testing.T) {
	base := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	policy := AcceptedIndependencePolicyV3Default()
	a := syntheticEvent(t, "AAAUSDT", base)
	b := syntheticEvent(t, "AAAUSDT", base.Add(3*time.Hour))
	c := syntheticEvent(t, "AAAUSDT", base.Add(6*time.Hour))
	clusters := mustClusterV3(t, []RetainedEvent{a, b, c, a}, policy)
	if len(clusters) != 1 || len(clusters[0].MemberEventIDs) != 3 {
		t.Fatalf("transitive/dedup clustering=%+v", clusters)
	}
	atBoundary := mustClusterV3(t, []RetainedEvent{a, syntheticEvent(t, "AAAUSDT", base.Add(4*time.Hour))}, policy)
	if len(atBoundary) != 2 {
		t.Fatal("half-open exact boundary did not split")
	}
	cross := withSyntheticEpisodeV2(t, syntheticEvent(t, "BBBUSDT", base.Add(time.Hour)), a)
	if got := mustClusterV3(t, []RetainedEvent{a, cross}, policy); len(got) != 1 || len(got[0].MemberSymbols) != 2 {
		t.Fatal("common-market cross-symbol semantics changed")
	}
}

func TestV3SymbolConcentrationMandatoryCases(t *testing.T) {
	policy := AcceptedIndependencePolicyV3Default()
	for _, test := range []struct {
		dominant int
		passed   bool
	}{{49, true}, {50, true}, {51, false}} {
		t.Run(fmt.Sprintf("dominant_%d", test.dominant), func(t *testing.T) {
			got := evaluateV3(t, policy, []string{"DEVELOPMENT"}, symbolFixtureV3(t, test.dominant))
			metric := got.Partitions[0].Symbol
			want := big.NewRat(int64(test.dominant), 100)
			assertExactShare(t, metric, want, test.passed)
		})
	}

	base := time.Date(2031, 1, 1, 0, 0, 0, 0, time.UTC)
	twoA := syntheticEvent(t, "AAAUSDT", base)
	twoB := withSyntheticEpisodeV2(t, syntheticEvent(t, "BBBUSDT", base.Add(time.Minute)), twoA)
	twoCluster := mustClusterV3(t, []RetainedEvent{twoA, twoB}, policy)[0]
	threeA := syntheticEvent(t, "AAAUSDT", base.Add(10*time.Hour))
	threeB := withSyntheticEpisodeV2(t, syntheticEvent(t, "BBBUSDT", base.Add(10*time.Hour+time.Minute)), threeA)
	threeC := withSyntheticEpisodeV2(t, syntheticEvent(t, "CCCUSDT", base.Add(10*time.Hour+2*time.Minute)), threeA)
	threeCluster := mustClusterV3(t, []RetainedEvent{threeA, threeB, threeC}, policy)[0]
	rows := []ConcentrationObservationV3{{"DEVELOPMENT", singletonClusterV3(t, "DDDUSDT", base.Add(20*time.Hour))}, {"DEVELOPMENT", twoCluster}, {"DEVELOPMENT", threeCluster}}
	got := evaluateV3(t, policy, []string{"DEVELOPMENT"}, rows)
	assertExactShare(t, got.Partitions[0].Symbol, big.NewRat(1, 3), true)

	duplicatedInput := mustClusterV3(t, []RetainedEvent{twoA, twoA}, policy)
	if len(duplicatedInput) != 1 || len(duplicatedInput[0].MemberEventIDs) != 1 {
		t.Fatal("duplicate input event changed symbol mass")
	}
	forward, _ := json.Marshal(got)
	for i, j := 0, len(rows)-1; i < j; i, j = i+1, j-1 {
		rows[i], rows[j] = rows[j], rows[i]
	}
	reverse, _ := json.Marshal(evaluateV3(t, policy, []string{"DEVELOPMENT"}, rows))
	if string(forward) != string(reverse) {
		t.Fatal("reordered clusters changed exact symbol result")
	}

	bad := singletonClusterV3(t, "ZZZUSDT", base.Add(20*time.Hour))
	bad.MemberSymbols = []string{""}
	if _, err := EvaluateConcentrationV3(policy, []string{"DEVELOPMENT"}, []ConcentrationObservationV3{{"DEVELOPMENT", bad}}); err == nil {
		t.Fatal("missing symbol passed")
	}
	duplicate := singletonClusterV3(t, "YYYUSDT", base.Add(30*time.Hour))
	if _, err := EvaluateConcentrationV3(policy, []string{"DEVELOPMENT"}, []ConcentrationObservationV3{{"DEVELOPMENT", duplicate}, {"DEVELOPMENT", duplicate}}); err == nil {
		t.Fatal("duplicate cluster identity passed")
	}
}

func TestV3TemporalConcentrationMandatoryCases(t *testing.T) {
	policy := AcceptedIndependencePolicyV3Default()
	for _, test := range []struct {
		dominant int
		passed   bool
	}{{49, true}, {50, true}, {51, false}} {
		got := evaluateV3(t, policy, []string{"VALIDATION"}, monthFixtureV3(t, test.dominant))
		assertExactShare(t, got.Partitions[0].Temporal, big.NewRat(int64(test.dominant), 100), test.passed)
	}
	janEnd := time.Date(2032, 1, 31, 23, 59, 59, 999999999, time.UTC)
	febStart := time.Date(2032, 2, 1, 0, 0, 0, 0, time.UTC)
	boundary := []ConcentrationObservationV3{{"VALIDATION", singletonClusterV3(t, "AAAUSDT", janEnd)}, {"VALIDATION", singletonClusterV3(t, "BBBUSDT", febStart)}}
	boundaryForward := evaluateV3(t, policy, []string{"VALIDATION"}, boundary)
	assertExactShare(t, boundaryForward.Partitions[0].Temporal, big.NewRat(1, 2), true)
	boundary[0], boundary[1] = boundary[1], boundary[0]
	forwardJSON, _ := json.Marshal(boundaryForward)
	reverseJSON, _ := json.Marshal(evaluateV3(t, policy, []string{"VALIDATION"}, boundary))
	if string(forwardJSON) != string(reverseJSON) {
		t.Fatal("reordered clusters changed temporal result")
	}

	offset := time.FixedZone("synthetic-minus-8", -8*60*60)
	sameUTCMonth := []ConcentrationObservationV3{{"VALIDATION", singletonClusterV3(t, "CCCUSDT", time.Date(2032, 2, 28, 23, 30, 0, 0, time.UTC))}, {"VALIDATION", singletonClusterV3(t, "DDDUSDT", time.Date(2032, 2, 28, 15, 45, 0, 0, offset))}}
	assertExactShare(t, evaluateV3(t, policy, []string{"VALIDATION"}, sameUTCMonth).Partitions[0].Temporal, big.NewRat(1, 1), false)

	emptyMonth := []ConcentrationObservationV3{{"VALIDATION", singletonClusterV3(t, "EEEUSDT", time.Date(2032, 3, 1, 0, 0, 0, 0, time.UTC))}, {"VALIDATION", singletonClusterV3(t, "FFFUSDT", time.Date(2032, 5, 1, 0, 0, 0, 0, time.UTC))}}
	assertExactShare(t, evaluateV3(t, policy, []string{"VALIDATION"}, emptyMonth).Partitions[0].Temporal, big.NewRat(1, 2), true)

	crossMonthCluster := sizedClusterV3(t, time.Date(2032, 6, 30, 23, 59, 0, 0, time.UTC), 2, "GGGUSDT")
	if crossMonthCluster.EarliestEventTime.UTC().Format("2006-01") != "2032-06" || !crossMonthCluster.LatestExposureEndpoint.After(time.Date(2032, 7, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatal("synthetic earliest/latest fixture invalid")
	}
	missing := singletonClusterV3(t, "HHHUSDT", time.Date(2032, 7, 1, 0, 0, 0, 0, time.UTC))
	missing.MemberEventTimes[0] = time.Time{}
	if _, err := EvaluateConcentrationV3(policy, []string{"VALIDATION"}, []ConcentrationObservationV3{{"VALIDATION", missing}}); err == nil {
		t.Fatal("missing timestamp passed")
	}
}

func TestV3LargestClusterConcentrationMandatoryCases(t *testing.T) {
	policy := AcceptedIndependencePolicyV3Default()
	for _, test := range []struct {
		sizes  []int
		want   *big.Rat
		passed bool
	}{
		{[]int{49, 26, 25}, big.NewRat(49, 100), true}, {[]int{50, 25, 25}, big.NewRat(1, 2), true}, {[]int{51, 25, 24}, big.NewRat(51, 100), false}, {[]int{2, 2, 1, 1}, big.NewRat(1, 3), true},
	} {
		metric := evaluateV3(t, policy, []string{"DEVELOPMENT"}, sizeFixtureV3(t, "DEVELOPMENT", test.sizes)).Partitions[0].LargestCluster
		assertExactShare(t, metric, test.want, test.passed)
	}
	duplicateMember := sizedClusterV3(t, time.Date(2033, 1, 1, 0, 0, 0, 0, time.UTC), 2, "AAAUSDT")
	duplicateMember.MemberEventIDs[1] = duplicateMember.MemberEventIDs[0]
	if _, err := EvaluateConcentrationV3(policy, []string{"DEVELOPMENT"}, []ConcentrationObservationV3{{"DEVELOPMENT", duplicateMember}}); err == nil {
		t.Fatal("duplicate member ID passed")
	}
	if _, err := EvaluateConcentrationV3(policy, []string{"DEVELOPMENT"}, nil); err == nil {
		t.Fatal("zero denominator passed")
	}
	shared := singletonClusterV3(t, "SHAREDUSDT", time.Date(2033, 2, 1, 0, 0, 0, 0, time.UTC))
	conflict := singletonClusterV3(t, "OTHERUSDT", time.Date(2033, 3, 1, 0, 0, 0, 0, time.UTC))
	conflict.MemberEventIDs[0] = shared.MemberEventIDs[0]
	conflict.ClusterID, _ = independentClusterIDV3(conflict)
	if err := ValidateIndependentClusterV3(conflict, policy); err != nil {
		t.Fatalf("conflicting-membership fixture invalid: %v", err)
	}
	if _, err := EvaluateConcentrationV3(policy, []string{"DEVELOPMENT"}, []ConcentrationObservationV3{{"DEVELOPMENT", shared}, {"DEVELOPMENT", conflict}}); err == nil {
		t.Fatal("conflicting membership passed")
	}
	reordered := sizeFixtureV3(t, "DEVELOPMENT", []int{49, 26, 25})
	forward := evaluateV3(t, policy, []string{"DEVELOPMENT"}, reordered)
	for i, j := 0, len(reordered)-1; i < j; i, j = i+1, j-1 {
		reordered[i], reordered[j] = reordered[j], reordered[i]
	}
	left, _ := json.Marshal(forward)
	right, _ := json.Marshal(evaluateV3(t, policy, []string{"DEVELOPMENT"}, reordered))
	if string(left) != string(right) {
		t.Fatal("reordered clusters changed largest-cluster result")
	}
}

func TestV3TopFiveConcentrationMandatoryCases(t *testing.T) {
	policy := AcceptedIndependencePolicyV3Default()
	for _, test := range []struct {
		name   string
		sizes  []int
		want   *big.Rat
		passed bool
	}{
		{"below", []int{14, 14, 14, 14, 13, 9, 9, 9, 2, 2}, big.NewRat(69, 100), true},
		{"exact", []int{14, 14, 14, 14, 14, 6, 6, 6, 6, 6}, big.NewRat(7, 10), true},
		{"above", []int{15, 14, 14, 14, 14, 9, 8, 6, 3, 3}, big.NewRat(71, 100), false},
		{"fewer_than_five", []int{1, 1, 1, 1}, big.NewRat(1, 1), false},
		{"exactly_five", []int{1, 1, 1, 1, 1}, big.NewRat(1, 1), false},
		{"more_than_five", []int{1, 1, 1, 1, 1, 1, 1, 1, 1, 1}, big.NewRat(1, 2), true},
	} {
		t.Run(test.name, func(t *testing.T) {
			metric := evaluateV3(t, policy, []string{"FINAL_HOLDOUT"}, sizeFixtureV3(t, "FINAL_HOLDOUT", test.sizes)).Partitions[0].TopFiveCluster
			assertExactShare(t, metric, test.want, test.passed)
		})
	}
	ties := evaluateV3(t, policy, []string{"FINAL_HOLDOUT"}, sizeFixtureV3(t, "FINAL_HOLDOUT", []int{1, 1, 1, 1, 1, 1})).Partitions[0]
	allIDs := append([]string(nil), ties.TopFiveClusterIDs...)
	rows := sizeFixtureV3(t, "FINAL_HOLDOUT", []int{1, 1, 1, 1, 1, 1})
	wantIDs := make([]string, len(rows))
	for i := range rows {
		wantIDs[i] = rows[i].Cluster.ClusterID
	}
	sort.Strings(wantIDs)
	if fmt.Sprint(allIDs) != fmt.Sprint(wantIDs[:5]) {
		t.Fatalf("tie ordering=%v want=%v", allIDs, wantIDs[:5])
	}
	for i, j := 0, len(rows)-1; i < j; i, j = i+1, j-1 {
		rows[i], rows[j] = rows[j], rows[i]
	}
	reordered := evaluateV3(t, policy, []string{"FINAL_HOLDOUT"}, rows).Partitions[0]
	if fmt.Sprint(reordered.TopFiveClusterIDs) != fmt.Sprint(wantIDs[:5]) {
		t.Fatal("reordered top-five selection changed")
	}
}

func TestV3PartitionIsolation(t *testing.T) {
	policy := AcceptedIndependencePolicyV3Default()
	rows := append(balancedPartitionFixtureV3(t, "DEVELOPMENT", time.Date(2044, 1, 1, 0, 0, 0, 0, time.UTC)), sizeFixtureV3At(t, "VALIDATION", []int{10}, time.Date(2045, 1, 1, 0, 0, 0, 0, time.UTC))...)
	got := evaluateV3(t, policy, []string{"DEVELOPMENT", "VALIDATION"}, rows)
	if got.Passed || !got.Partitions[0].Passed || got.Partitions[1].Passed {
		t.Fatalf("partition failure was averaged away: %+v", got)
	}
}

func evaluateV3(t *testing.T, policy AcceptedIndependencePolicyV3, partitions []string, rows []ConcentrationObservationV3) ConcentrationEvaluationV3 {
	t.Helper()
	got, err := EvaluateConcentrationV3(policy, partitions, rows)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func assertExactShare(t *testing.T, metric ConcentrationMetricResultV3, want *big.Rat, passed bool) {
	t.Helper()
	got, err := ParseExactRational(metric.Share)
	if err != nil || got.Cmp(want) != 0 || metric.Passed != passed {
		t.Fatalf("metric=%+v want=%s passed=%t err=%v", metric, want.RatString(), passed, err)
	}
}

func mustClusterV3(t *testing.T, events []RetainedEvent, policy AcceptedIndependencePolicyV3) []IndependentClusterV3 {
	t.Helper()
	clusters, err := ClusterEventsV3(events, policy)
	if err != nil {
		t.Fatal(err)
	}
	return clusters
}

func singletonClusterV3(t *testing.T, symbol string, at time.Time) IndependentClusterV3 {
	t.Helper()
	return mustClusterV3(t, []RetainedEvent{syntheticEvent(t, symbol, at)}, AcceptedIndependencePolicyV3Default())[0]
}

func sizedClusterV3(t *testing.T, base time.Time, size int, symbol string) IndependentClusterV3 {
	t.Helper()
	events := make([]RetainedEvent, size)
	for i := range events {
		events[i] = syntheticEvent(t, symbol, base.Add(time.Duration(i)*time.Minute))
	}
	clusters := mustClusterV3(t, events, AcceptedIndependencePolicyV3Default())
	if len(clusters) != 1 || len(clusters[0].MemberEventIDs) != size {
		t.Fatalf("sized cluster=%d/%d", len(clusters), size)
	}
	return clusters[0]
}

func symbolFixtureV3(t *testing.T, dominant int) []ConcentrationObservationV3 {
	t.Helper()
	base := time.Date(2034, 1, 1, 0, 0, 0, 0, time.UTC)
	rows := make([]ConcentrationObservationV3, 0, 100)
	second := (100 - dominant) / 2
	for i := 0; i < 100; i++ {
		symbol := "CCCUSDT"
		if i < dominant {
			symbol = "AAAUSDT"
		} else if i < dominant+second {
			symbol = "BBBUSDT"
		}
		rows = append(rows, ConcentrationObservationV3{"DEVELOPMENT", singletonClusterV3(t, symbol, base.Add(time.Duration(i)*5*time.Hour))})
	}
	return rows
}

func monthFixtureV3(t *testing.T, dominant int) []ConcentrationObservationV3 {
	t.Helper()
	counts := []int{dominant, (100 - dominant) / 2, 100 - dominant - (100-dominant)/2}
	rows := make([]ConcentrationObservationV3, 0, 100)
	for month, count := range counts {
		base := time.Date(2035, time.Month(month+1), 1, 0, 0, 0, 0, time.UTC)
		for i := 0; i < count; i++ {
			rows = append(rows, ConcentrationObservationV3{"VALIDATION", singletonClusterV3(t, fmt.Sprintf("M%02dUSDT", month), base.Add(time.Duration(i)*5*time.Hour))})
		}
	}
	return rows
}

func sizeFixtureV3(t *testing.T, partition string, sizes []int) []ConcentrationObservationV3 {
	return sizeFixtureV3At(t, partition, sizes, time.Date(2036, 1, 1, 0, 0, 0, 0, time.UTC))
}

func sizeFixtureV3At(t *testing.T, partition string, sizes []int, base time.Time) []ConcentrationObservationV3 {
	t.Helper()
	rows := make([]ConcentrationObservationV3, len(sizes))
	for i, size := range sizes {
		rows[i] = ConcentrationObservationV3{partition, sizedClusterV3(t, base.Add(time.Duration(i)*48*time.Hour), size, fmt.Sprintf("C%02dUSDT", i))}
	}
	return rows
}

func balancedPartitionFixtureV3(t *testing.T, partition string, base time.Time) []ConcentrationObservationV3 {
	t.Helper()
	rows := make([]ConcentrationObservationV3, 0, 10)
	for i := 0; i < 10; i++ {
		at := base.Add(time.Duration(i%5) * 5 * time.Hour)
		if i >= 5 {
			at = at.AddDate(0, 1, 0)
		}
		rows = append(rows, ConcentrationObservationV3{partition, singletonClusterV3(t, fmt.Sprintf("B%02dUSDT", i), at)})
	}
	return rows
}
