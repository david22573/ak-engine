package preconditions

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

const (
	IndependencePolicyVersion       = "ak.engine.independence.downtrend-midvol-relief.v1"
	PolicyStatusProposedNotAccepted = "PROPOSED_NOT_ACCEPTED"
)

type IndependencePolicy struct {
	Version            string `json:"version"`
	Status             string `json:"status"`
	MinimumSpacingMS   int64  `json:"minimum_spacing_ms"`
	OverlapHorizonMS   int64  `json:"overlap_horizon_ms"`
	SameSymbolOverlap  string `json:"same_symbol_overlap"`
	CrossSymbolOverlap string `json:"cross_symbol_common_market_overlap"`
	BoundaryRule       string `json:"boundary_rule"`
	OrderingRule       string `json:"ordering_rule"`
	DuplicateRule      string `json:"duplicate_rule"`
	ClusterIDRule      string `json:"cluster_id_rule"`
}

type IndependentCluster struct {
	PolicyVersion               string            `json:"policy_version"`
	ClusterID                   string            `json:"cluster_id"`
	Start                       time.Time         `json:"start"`
	End                         time.Time         `json:"end"`
	MemberEventIDs              []string          `json:"member_event_ids"`
	MemberSymbols               []string          `json:"member_symbols"`
	ClusterTimestamps           []time.Time       `json:"cluster_timestamps"`
	SameSymbolOverlapIdentities map[string]string `json:"same_symbol_overlap_identities"`
	CommonMarketClusterIdentity string            `json:"common_market_cluster_identity"`
}

func DefaultIndependencePolicy() IndependencePolicy {
	horizon := int64(4 * time.Hour / time.Millisecond)
	return IndependencePolicy{
		Version: IndependencePolicyVersion, Status: PolicyStatusProposedNotAccepted,
		MinimumSpacingMS: horizon, OverlapHorizonMS: horizon,
		SameSymbolOverlap:  "cluster when half-open decision horizons overlap",
		CrossSymbolOverlap: "cluster overlapping events across symbols sharing the futures-um BTC/ETH market context",
		BoundaryRule:       "an event exactly at the active cluster end starts a new cluster",
		OrderingRule:       "decision_timestamp ascending, then event_id ascending",
		DuplicateRule:      "byte-equivalent event IDs are ignored; conflicting duplicate IDs fail",
		ClusterIDRule:      "sha256(policy_hash, sorted member event IDs)",
	}
}

func IndependencePolicyHash(policy IndependencePolicy) (string, error) {
	if err := validateIndependencePolicy(policy); err != nil {
		return "", err
	}
	return canonicalDigest(policy)
}

func ClusterEvents(events []RetainedEvent, policy IndependencePolicy) ([]IndependentCluster, error) {
	if err := validateIndependencePolicy(policy); err != nil {
		return nil, err
	}
	unique, err := DeduplicateRetainedEvents(events)
	if err != nil {
		return nil, err
	}
	if len(unique) == 0 {
		return []IndependentCluster{}, nil
	}
	policyHash, _ := IndependencePolicyHash(policy)
	type building struct {
		events     []RetainedEvent
		start, end time.Time
	}
	groups := []building{}
	for _, event := range unique {
		if event.DecisionTimestamp.IsZero() {
			return nil, errors.New("missing decision timestamp")
		}
		end := event.DecisionTimestamp.Add(time.Duration(max64(policy.MinimumSpacingMS, policy.OverlapHorizonMS)) * time.Millisecond)
		if len(groups) == 0 || !event.DecisionTimestamp.Before(groups[len(groups)-1].end) {
			groups = append(groups, building{events: []RetainedEvent{event}, start: event.DecisionTimestamp.UTC(), end: end.UTC()})
			continue
		}
		last := &groups[len(groups)-1]
		last.events = append(last.events, event)
		if end.After(last.end) {
			last.end = end.UTC()
		}
	}
	clusters := make([]IndependentCluster, 0, len(groups))
	for _, group := range groups {
		ids, symbols, timestamps := []string{}, []string{}, []time.Time{}
		for _, event := range group.events {
			ids = append(ids, event.EventID)
			symbols = append(symbols, event.PrimarySymbol)
			timestamps = append(timestamps, event.DecisionTimestamp.UTC())
		}
		sort.Strings(ids)
		symbols = uniqueStrings(symbols)
		digest, err := canonicalDigest(struct {
			PolicyHash string   `json:"policy_hash"`
			EventIDs   []string `json:"event_ids"`
		}{policyHash, ids})
		if err != nil {
			return nil, err
		}
		clusterID := "cluster:" + strings.TrimPrefix(digest, "sha256:")
		same := map[string]string{}
		for _, symbol := range symbols {
			symbolDigest, err := canonicalDigest(struct{ ClusterID, Symbol string }{clusterID, symbol})
			if err != nil {
				return nil, err
			}
			same[symbol] = "same-symbol:" + strings.TrimPrefix(symbolDigest, "sha256:")
		}
		clusters = append(clusters, IndependentCluster{
			PolicyVersion: policy.Version, ClusterID: clusterID, Start: group.start, End: group.end,
			MemberEventIDs: ids, MemberSymbols: symbols, ClusterTimestamps: timestamps,
			SameSymbolOverlapIdentities: same, CommonMarketClusterIdentity: clusterID,
		})
	}
	return clusters, nil
}

func ClusterConcentration(clusters []IndependentCluster) (float64, error) {
	total, largest := 0, 0
	seen := map[string]struct{}{}
	for _, cluster := range clusters {
		if cluster.ClusterID == "" {
			return 0, errors.New("cluster ID is required")
		}
		if _, exists := seen[cluster.ClusterID]; exists {
			return 0, errors.New("duplicate cluster ID")
		}
		seen[cluster.ClusterID] = struct{}{}
		total += len(cluster.MemberEventIDs)
		if len(cluster.MemberEventIDs) > largest {
			largest = len(cluster.MemberEventIDs)
		}
	}
	if total == 0 {
		return 0, nil
	}
	return float64(largest) / float64(total), nil
}

func validateIndependencePolicy(policy IndependencePolicy) error {
	if policy.Version != IndependencePolicyVersion || policy.Status != PolicyStatusProposedNotAccepted {
		return errors.New("unsupported or incorrectly governed independence policy")
	}
	if policy.MinimumSpacingMS <= 0 || policy.OverlapHorizonMS <= 0 {
		return errors.New("spacing and horizon must be positive")
	}
	for name, value := range map[string]string{"same-symbol rule": policy.SameSymbolOverlap, "cross-symbol rule": policy.CrossSymbolOverlap, "boundary rule": policy.BoundaryRule, "ordering rule": policy.OrderingRule, "duplicate rule": policy.DuplicateRule, "cluster ID rule": policy.ClusterIDRule} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s is required", name)
		}
	}
	return nil
}

func uniqueStrings(values []string) []string {
	seen := map[string]struct{}{}
	for _, value := range values {
		seen[value] = struct{}{}
	}
	result := make([]string, 0, len(seen))
	for value := range seen {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}
func max64(left, right int64) int64 {
	if left > right {
		return left
	}
	return right
}
