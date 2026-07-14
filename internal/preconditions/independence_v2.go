package preconditions

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"sort"
	"strings"
	"time"
)

const (
	RevisedIndependencePolicyVersion        = "ak.engine.independence.downtrend-midvol-relief.v2"
	IndependentClusterSchemaVersionV2       = "ak.engine.independent-cluster.downtrend-midvol-relief.v2"
	PolicyStatusRevisedPendingConcentration = "REVISED_PENDING_CONCENTRATION_AUTHORITY"
)

type ConcentrationAuthority struct {
	Name             string `json:"name"`
	Definition       string `json:"definition"`
	Threshold        string `json:"threshold"`
	Operator         string `json:"comparison_operator"`
	Denominator      string `json:"denominator"`
	FailureSemantics string `json:"failure_semantics"`
	SourceReport     string `json:"source_report"`
	SourceCommit     string `json:"source_commit"`
	AuthorityStatus  string `json:"authority_status"`
}

type RevisedIndependencePolicy struct {
	Version                    string                   `json:"version"`
	Status                     string                   `json:"status"`
	ExposureHorizonMinutes     int                      `json:"exposure_horizon_minutes"`
	ExposureIntervalRule       string                   `json:"exposure_interval_rule"`
	TimestampNormalizationRule string                   `json:"timestamp_normalization_rule"`
	SameSymbolRule             string                   `json:"same_symbol_rule"`
	CrossSymbolRule            string                   `json:"cross_symbol_rule"`
	CommonMarketEpisodeRule    string                   `json:"common_market_episode_rule"`
	MissingEpisodeRule         string                   `json:"missing_episode_rule"`
	TransitivityRule           string                   `json:"transitivity_rule"`
	OrderingRule               string                   `json:"ordering_rule"`
	DuplicateRule              string                   `json:"duplicate_rule"`
	ClusterIDRule              string                   `json:"cluster_id_rule"`
	IndependentSampleRule      string                   `json:"independent_sample_rule"`
	ClusterNetReturnRule       string                   `json:"cluster_net_return_rule"`
	ConcentrationAuthorities   []ConcentrationAuthority `json:"concentration_authorities"`
	UnresolvedItems            []string                 `json:"unresolved_items"`
	KnownLimitations           []string                 `json:"known_limitations"`
}

type IndependentClusterV2 struct {
	SchemaVersion                 string      `json:"schema_version"`
	PolicyVersion                 string      `json:"policy_version"`
	PolicyHash                    string      `json:"policy_hash"`
	ClusterID                     string      `json:"cluster_id"`
	EarliestEventTime             time.Time   `json:"earliest_event_time"`
	LatestExposureEndpoint        time.Time   `json:"latest_exposure_endpoint"`
	MemberEventIDs                []string    `json:"member_event_ids"`
	MemberSymbols                 []string    `json:"member_symbols"`
	MemberEventTimes              []time.Time `json:"member_event_times"`
	CommonMarketEpisodeIdentities []string    `json:"common_market_episode_identities"`
}

type MemberNetReturn struct {
	EventID  string  `json:"event_id"`
	NetValue float64 `json:"net_value"`
}

type ClusterNetReturn struct {
	ClusterID string  `json:"cluster_id"`
	NetValue  float64 `json:"net_value"`
}

func RevisedIndependencePolicyV2() RevisedIndependencePolicy {
	const qualificationReport = "runs/reports/pr4b0_candidate_qualification.json"
	const qualificationCommit = "205cf59555006ce23fc58bc2c73262660a894850"
	return RevisedIndependencePolicy{
		Version:                    RevisedIndependencePolicyVersion,
		Status:                     PolicyStatusRevisedPendingConcentration,
		ExposureHorizonMinutes:     240,
		ExposureIntervalRule:       "each event occupies the half-open UTC interval [event_timestamp,event_timestamp+240m); a timestamp strictly before the endpoint overlaps and a timestamp exactly at the endpoint does not",
		TimestampNormalizationRule: "compare instants after UTC normalization and serialize with RFC3339Nano so equivalent offsets and subsecond representations normalize identically",
		SameSymbolRule:             "overlapping exposure intervals for the same primary symbol form an edge; connected components apply transitive overlap",
		CrossSymbolRule:            "different primary symbols form an edge only when exposure intervals overlap and their deterministic common-market episode identities are equal",
		CommonMarketEpisodeRule:    "sha256 canonical JSON of BTCUSDT and ETHUSDT context symbol, snapshot_id, source_input_hash, and UTC available_at; decision-time provenance only; primary-symbol state and all outcomes are excluded",
		MissingEpisodeRule:         "missing or invalid decision-time common-context provenance fails closed; never substitute a blanket market identity",
		TransitivityRule:           "an independent cluster is one connected component of the permitted same-symbol and cross-symbol overlap edges",
		OrderingRule:               "deduplicate by canonical event identity, sort components by earliest event time then cluster ID, and sort all member and identity lists lexically",
		DuplicateRule:              "byte-equivalent duplicate event identities are one member; conflicting duplicates fail closed",
		ClusterIDRule:              "sha256 canonical JSON binding policy version/hash, sorted member event IDs, earliest event time, latest exposure endpoint, sorted primary symbols, and sorted applicable common-market episode identities",
		IndependentSampleRule:      "one independent cluster is exactly one qualification decision; raw events never substitute for clusters",
		ClusterNetReturnRule:       "later, sort member event IDs and sum exactly one finite mandatory-cost net return for every member in that order; missing, extra, or duplicate members fail closed; this phase uses synthetic values only",
		ConcentrationAuthorities: []ConcentrationAuthority{
			{Name: "largest-cluster concentration", AuthorityStatus: "UNRESOLVED_NO_ACCEPTED_SOURCE_REPORT", Definition: "", Threshold: "", Operator: "", Denominator: "", FailureSemantics: "fail closed; independence policy cannot be accepted", SourceReport: "", SourceCommit: ""},
			{Name: "cluster concentration", AuthorityStatus: "UNRESOLVED_NO_ACCEPTED_SOURCE_REPORT", Definition: "", Threshold: "", Operator: "", Denominator: "", FailureSemantics: "fail closed; independence policy cannot be accepted", SourceReport: "", SourceCommit: ""},
			{Name: "symbol concentration", AuthorityStatus: "THRESHOLD_RECOVERED_DEFINITION_INCOMPLETE", Definition: "maximum symbol contribution in a legal evaluated scope", Threshold: "50%", Operator: "<=", Denominator: "not normatively specified by the accepted source report", FailureSemantics: "values greater than 50% fail; missing denominator authority fails closed", SourceReport: qualificationReport, SourceCommit: qualificationCommit},
			{Name: "temporal concentration", AuthorityStatus: "THRESHOLD_RECOVERED_DEFINITION_INCOMPLETE", Definition: "maximum temporal contribution in a legal evaluated scope", Threshold: "50%", Operator: "<=", Denominator: "not normatively specified by the accepted source report", FailureSemantics: "values greater than 50% fail; missing denominator authority fails closed", SourceReport: qualificationReport, SourceCommit: qualificationCommit},
		},
		UnresolvedItems: []string{
			"accepted source report, threshold, operator, denominator, and failure semantics for largest-cluster concentration",
			"accepted source report, threshold, operator, denominator, and failure semantics for aggregate cluster concentration",
			"accepted denominator definitions for symbol and temporal contribution percentages",
		},
		KnownLimitations: []string{
			"exact context-snapshot identity deliberately under-clusters common episodes when authoritative collectors issue distinct provenance identities for the same market move",
			"same-symbol transitive bridges can place more than one common-market episode identity in one connected component",
			"the rule captures declared BTCUSDT/ETHUSDT context provenance only and does not infer latent dependence from future volatility, returns, or retrospective behavior",
		},
	}
}

func RevisedIndependencePolicyHashV2(policy RevisedIndependencePolicy) (string, error) {
	if err := validateRevisedIndependencePolicyV2(policy); err != nil {
		return "", err
	}
	return canonicalDigest(policy)
}

func IndependentClusterSchemaDescriptorV2() SchemaDescriptor {
	return SchemaDescriptor{IndependentClusterSchemaVersionV2, []string{
		"schema_version", "policy_version", "policy_hash", "cluster_id", "earliest_event_time", "latest_exposure_endpoint",
		"member_event_ids", "member_symbols", "member_event_times", "common_market_episode_identities",
	}}
}

func IndependentClusterSchemaHashV2() (string, error) {
	return canonicalDigest(IndependentClusterSchemaDescriptorV2())
}

func CommonMarketEpisodeIdentityV2(event RetainedEvent) (string, error) {
	if err := ValidateRetainedEvent(event); err != nil {
		return "", err
	}
	type contextIdentity struct {
		Symbol          string `json:"symbol"`
		SnapshotID      string `json:"snapshot_id"`
		SourceInputHash string `json:"source_input_hash"`
		AvailableAt     string `json:"available_at"`
	}
	payload := struct {
		Schema string          `json:"schema"`
		BTC    contextIdentity `json:"btc_context"`
		ETH    contextIdentity `json:"eth_context"`
	}{
		Schema: "ak.engine.common-market-episode.context-provenance.v1",
		BTC:    contextIdentity{event.BTCContext.Symbol, event.BTCContext.SnapshotID, event.BTCContext.SourceInputHash, canonicalTime(event.BTCContext.AvailableAt)},
		ETH:    contextIdentity{event.ETHContext.Symbol, event.ETHContext.SnapshotID, event.ETHContext.SourceInputHash, canonicalTime(event.ETHContext.AvailableAt)},
	}
	digest, err := canonicalDigest(payload)
	if err != nil {
		return "", err
	}
	return "episode:" + strings.TrimPrefix(digest, "sha256:"), nil
}

func ClusterEventsV2(events []RetainedEvent, policy RevisedIndependencePolicy) ([]IndependentClusterV2, error) {
	if err := validateRevisedIndependencePolicyV2(policy); err != nil {
		return nil, err
	}
	normalized := make([]RetainedEvent, len(events))
	for i, event := range events {
		var err error
		normalized[i], err = normalizeRetainedEventUTCV2(event)
		if err != nil {
			return nil, err
		}
	}
	unique, err := DeduplicateRetainedEvents(normalized)
	if err != nil {
		return nil, err
	}
	if len(unique) == 0 {
		return []IndependentClusterV2{}, nil
	}
	policyHash, err := RevisedIndependencePolicyHashV2(policy)
	if err != nil {
		return nil, err
	}
	episodes := make([]string, len(unique))
	for i, event := range unique {
		episodes[i], err = CommonMarketEpisodeIdentityV2(event)
		if err != nil {
			return nil, fmt.Errorf("event %q common-market identity: %w", event.EventID, err)
		}
	}
	parent := make([]int, len(unique))
	for i := range parent {
		parent[i] = i
	}
	var find func(int) int
	find = func(i int) int {
		if parent[i] != i {
			parent[i] = find(parent[i])
		}
		return parent[i]
	}
	union := func(left, right int) {
		left, right = find(left), find(right)
		if left != right {
			if left > right {
				left, right = right, left
			}
			parent[right] = left
		}
	}
	crossEpisodes := make(map[[2]int]string)
	horizon := time.Duration(policy.ExposureHorizonMinutes) * time.Minute
	for i := 0; i < len(unique); i++ {
		for j := i + 1; j < len(unique); j++ {
			if !halfOpenIntervalsOverlap(unique[i].EventTimestamp, unique[i].EventTimestamp.Add(horizon), unique[j].EventTimestamp, unique[j].EventTimestamp.Add(horizon)) {
				continue
			}
			if unique[i].PrimarySymbol == unique[j].PrimarySymbol {
				union(i, j)
				continue
			}
			if episodes[i] == "" || episodes[j] == "" {
				return nil, errors.New("cross-symbol grouping requires deterministic common-market episode identities")
			}
			if episodes[i] == episodes[j] {
				union(i, j)
				crossEpisodes[[2]int{i, j}] = episodes[i]
			}
		}
	}
	components := map[int][]int{}
	for i := range unique {
		root := find(i)
		components[root] = append(components[root], i)
	}
	clusters := make([]IndependentClusterV2, 0, len(components))
	for _, indices := range components {
		ids := make([]string, 0, len(indices))
		symbols := make([]string, 0, len(indices))
		times := make([]time.Time, 0, len(indices))
		earliest := unique[indices[0]].EventTimestamp.UTC()
		latest := earliest.Add(horizon)
		memberSet := map[int]struct{}{}
		for _, index := range indices {
			memberSet[index] = struct{}{}
			event := unique[index]
			at := event.EventTimestamp.UTC()
			endpoint := at.Add(horizon)
			if at.Before(earliest) {
				earliest = at
			}
			if endpoint.After(latest) {
				latest = endpoint
			}
			ids = append(ids, event.EventID)
			symbols = append(symbols, event.PrimarySymbol)
			times = append(times, at)
		}
		episodeSet := map[string]struct{}{}
		for pair, episode := range crossEpisodes {
			if _, left := memberSet[pair[0]]; left {
				if _, right := memberSet[pair[1]]; right {
					episodeSet[episode] = struct{}{}
				}
			}
		}
		sort.Strings(ids)
		symbols = uniqueStrings(symbols)
		sort.Slice(times, func(i, j int) bool { return times[i].Before(times[j]) })
		episodeIDs := make([]string, 0, len(episodeSet))
		for episode := range episodeSet {
			episodeIDs = append(episodeIDs, episode)
		}
		sort.Strings(episodeIDs)
		payload := struct {
			PolicyVersion          string   `json:"policy_version"`
			PolicyHash             string   `json:"policy_hash"`
			MemberEventIDs         []string `json:"member_event_ids"`
			EarliestEventTime      string   `json:"earliest_event_time"`
			LatestExposureEndpoint string   `json:"latest_exposure_endpoint"`
			PrimarySymbols         []string `json:"primary_symbols"`
			CommonMarketEpisodes   []string `json:"common_market_episode_identities"`
		}{policy.Version, policyHash, ids, canonicalTime(earliest), canonicalTime(latest), symbols, episodeIDs}
		digest, err := canonicalDigest(payload)
		if err != nil {
			return nil, err
		}
		clusters = append(clusters, IndependentClusterV2{
			SchemaVersion: IndependentClusterSchemaVersionV2, PolicyVersion: policy.Version, PolicyHash: policyHash,
			ClusterID: "cluster:" + strings.TrimPrefix(digest, "sha256:"), EarliestEventTime: earliest, LatestExposureEndpoint: latest,
			MemberEventIDs: ids, MemberSymbols: symbols, MemberEventTimes: times, CommonMarketEpisodeIdentities: episodeIDs,
		})
	}
	sort.Slice(clusters, func(i, j int) bool {
		if !clusters[i].EarliestEventTime.Equal(clusters[j].EarliestEventTime) {
			return clusters[i].EarliestEventTime.Before(clusters[j].EarliestEventTime)
		}
		return clusters[i].ClusterID < clusters[j].ClusterID
	})
	return clusters, nil
}

func AggregateClusterNetReturnV2(cluster IndependentClusterV2, returns []MemberNetReturn) (ClusterNetReturn, error) {
	if cluster.SchemaVersion != IndependentClusterSchemaVersionV2 || strings.TrimSpace(cluster.ClusterID) == "" || len(cluster.MemberEventIDs) == 0 {
		return ClusterNetReturn{}, errors.New("valid V2 cluster identity and members are required")
	}
	if len(returns) != len(cluster.MemberEventIDs) {
		return ClusterNetReturn{}, errors.New("exactly one net return per cluster member is required")
	}
	byID := make(map[string]float64, len(returns))
	for _, observation := range returns {
		if strings.TrimSpace(observation.EventID) == "" || !finite(observation.NetValue) {
			return ClusterNetReturn{}, errors.New("member net return identity and finite value are required")
		}
		if _, duplicate := byID[observation.EventID]; duplicate {
			return ClusterNetReturn{}, fmt.Errorf("duplicate member net return %q", observation.EventID)
		}
		byID[observation.EventID] = observation.NetValue
	}
	ids := append([]string(nil), cluster.MemberEventIDs...)
	sort.Strings(ids)
	total := 0.0
	compensation := 0.0
	for _, id := range ids {
		value, exists := byID[id]
		if !exists {
			return ClusterNetReturn{}, fmt.Errorf("missing member net return %q", id)
		}
		y := value - compensation
		next := total + y
		compensation = (next - total) - y
		total = next
	}
	if len(byID) != len(ids) || math.IsNaN(total) || math.IsInf(total, 0) {
		return ClusterNetReturn{}, errors.New("extra member or non-finite aggregate")
	}
	return ClusterNetReturn{ClusterID: cluster.ClusterID, NetValue: total}, nil
}

func ValidateIndependentClusterV2(cluster IndependentClusterV2, policy RevisedIndependencePolicy) error {
	policyHash, err := RevisedIndependencePolicyHashV2(policy)
	if err != nil {
		return err
	}
	if cluster.SchemaVersion != IndependentClusterSchemaVersionV2 || cluster.PolicyVersion != policy.Version || cluster.PolicyHash != policyHash {
		return errors.New("cluster schema or policy identity mismatch")
	}
	if len(cluster.MemberEventIDs) == 0 || len(cluster.MemberSymbols) == 0 || len(cluster.MemberEventTimes) != len(cluster.MemberEventIDs) || cluster.EarliestEventTime.IsZero() || !cluster.EarliestEventTime.Before(cluster.LatestExposureEndpoint) {
		return errors.New("cluster membership and exposure identity are incomplete")
	}
	ids := append([]string(nil), cluster.MemberEventIDs...)
	sort.Strings(ids)
	for i, id := range ids {
		if strings.TrimSpace(id) == "" || (i > 0 && id == ids[i-1]) {
			return errors.New("cluster member identities must be nonempty and unique")
		}
	}
	symbols := uniqueStrings(cluster.MemberSymbols)
	if len(symbols) != len(cluster.MemberSymbols) {
		return errors.New("cluster symbols must be unique")
	}
	episodes := append([]string{}, cluster.CommonMarketEpisodeIdentities...)
	sort.Strings(episodes)
	for i, episode := range episodes {
		if !strings.HasPrefix(episode, "episode:") || (i > 0 && episode == episodes[i-1]) {
			return errors.New("common-market episode identities must be canonical and unique")
		}
	}
	payload := struct {
		PolicyVersion          string   `json:"policy_version"`
		PolicyHash             string   `json:"policy_hash"`
		MemberEventIDs         []string `json:"member_event_ids"`
		EarliestEventTime      string   `json:"earliest_event_time"`
		LatestExposureEndpoint string   `json:"latest_exposure_endpoint"`
		PrimarySymbols         []string `json:"primary_symbols"`
		CommonMarketEpisodes   []string `json:"common_market_episode_identities"`
	}{policy.Version, policyHash, ids, canonicalTime(cluster.EarliestEventTime), canonicalTime(cluster.LatestExposureEndpoint), symbols, episodes}
	digest, err := canonicalDigest(payload)
	if err != nil {
		return err
	}
	wantID := "cluster:" + strings.TrimPrefix(digest, "sha256:")
	if cluster.ClusterID != wantID {
		return errors.New("cluster ID does not match canonical membership identity")
	}
	return nil
}

func validateRevisedIndependencePolicyV2(policy RevisedIndependencePolicy) error {
	if policy.Version != RevisedIndependencePolicyVersion || policy.Status != PolicyStatusRevisedPendingConcentration || policy.ExposureHorizonMinutes != 240 {
		return errors.New("unsupported or incorrectly governed revised independence policy")
	}
	for name, value := range map[string]string{
		"interval": policy.ExposureIntervalRule, "timestamp normalization": policy.TimestampNormalizationRule, "same-symbol": policy.SameSymbolRule,
		"cross-symbol": policy.CrossSymbolRule, "episode": policy.CommonMarketEpisodeRule, "missing episode": policy.MissingEpisodeRule,
		"transitivity": policy.TransitivityRule, "ordering": policy.OrderingRule, "duplicate": policy.DuplicateRule,
		"cluster ID": policy.ClusterIDRule, "sample": policy.IndependentSampleRule, "cluster return": policy.ClusterNetReturnRule,
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s rule is required", name)
		}
	}
	if len(policy.ConcentrationAuthorities) != 4 || len(policy.UnresolvedItems) == 0 || len(policy.KnownLimitations) == 0 {
		return errors.New("revised policy must preserve unresolved concentration authority")
	}
	want := RevisedIndependencePolicyV2()
	if !reflect.DeepEqual(policy, want) {
		return errors.New("V2 policy mutation requires a new policy version and governance decision")
	}
	return nil
}

func halfOpenIntervalsOverlap(leftStart, leftEnd, rightStart, rightEnd time.Time) bool {
	return leftStart.Before(rightEnd) && rightStart.Before(leftEnd)
}

func normalizeRetainedEventUTCV2(event RetainedEvent) (RetainedEvent, error) {
	if err := ValidateRetainedEvent(event); err != nil {
		return RetainedEvent{}, err
	}
	event.EventTimestamp = event.EventTimestamp.UTC()
	event.DecisionTimestamp = event.DecisionTimestamp.UTC()
	event.BTCContext.AvailableAt = event.BTCContext.AvailableAt.UTC()
	event.ETHContext.AvailableAt = event.ETHContext.AvailableAt.UTC()
	return SealRetainedEvent(event)
}
