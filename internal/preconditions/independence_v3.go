package preconditions

import (
	"errors"
	"fmt"
	"math/big"
	"reflect"
	"sort"
	"strings"
	"time"
)

const (
	AcceptedIndependencePolicyVersionV3 = "ak.engine.independence.downtrend-midvol-relief.v3"
	IndependentClusterSchemaVersionV3   = "ak.engine.independent-cluster.downtrend-midvol-relief.v3"
	ConcentrationEvaluationVersionV3    = "ak.engine.concentration-evaluation.downtrend-midvol-relief.v3"
	GovernanceDecisionSchemaVersionV3   = "ak.engine.governance.concentration-decision.v1"
	GovernanceDecisionIDV3              = "pr4b0-r1p3b-structural-count-based-concentration"
	PolicyStatusAccepted                = "ACCEPTED"
	GovernedCandidateFamily             = "phase12/DowntrendMidVolReliefLong240m"
	GovernanceDecisionTimestampV3       = "2026-07-14T00:00:00Z"
	GovernanceDecisionAuthorityV3       = "HUMAN_GOVERNANCE_AUTHORITY"
	PriorConcentrationPacketHash        = "sha256:5fa980666b8d11bc405b1d31948041b2ff07f371178d88f2f485e957e356f992"
	PendingIndependencePolicyHashV2     = "sha256:006f19c3f89650f6905931164d6c98ead20800a2346369dadda708cfadf36528"
	AcceptedUncertaintyMethodDigestV2   = "sha256:1a91541c94378cc6f34e62a39ae504d3d013b5dab63a2b622641cdd1088148fb"
	ProspectiveGovernanceAuthorityText1 = "This decision creates prospective governance authority."
	ProspectiveGovernanceAuthorityText2 = "It does not claim recovery of historical authority."
)

type ConcentrationGovernanceDecisionV3 struct {
	SchemaVersion                 string `json:"schema_version"`
	DecisionID                    string `json:"decision_id"`
	PriorPacketHash               string `json:"prior_packet_hash"`
	Decision                      string `json:"decision"`
	SelectedAlternative           string `json:"selected_alternative"`
	RejectedAlternative           string `json:"rejected_alternative"`
	RejectionReason               string `json:"rejection_reason"`
	DecisionTimestamp             string `json:"decision_timestamp"`
	DecisionScope                 string `json:"decision_scope"`
	HistoricalAuthorityClaimed    bool   `json:"historical_authority_claimed"`
	AuthorityResponsible          string `json:"authority_responsible"`
	ProspectiveAuthorityStatement string `json:"prospective_authority_statement"`
	HistoricalAuthorityDisclaimer string `json:"historical_authority_disclaimer"`
	RetroactiveClassificationRule string `json:"retroactive_classification_rule"`
	CanonicalDecisionHash         string `json:"canonical_decision_hash"`
}

func DefaultConcentrationGovernanceDecisionV3() ConcentrationGovernanceDecisionV3 {
	record := ConcentrationGovernanceDecisionV3{
		SchemaVersion:                 GovernanceDecisionSchemaVersionV3,
		DecisionID:                    GovernanceDecisionIDV3,
		PriorPacketHash:               PriorConcentrationPacketHash,
		Decision:                      "ACCEPT_ALTERNATIVE",
		SelectedAlternative:           "STRUCTURAL_COUNT_BASED_CONCENTRATION",
		RejectedAlternative:           "OUTCOME_CONTRIBUTION_BASED_CONCENTRATION",
		RejectionReason:               "outcome-contribution denominators are endogenous to realized returns, can change with outcome magnitude or sign, and do not measure structural dependence in the independent-cluster result",
		DecisionTimestamp:             GovernanceDecisionTimestampV3,
		DecisionScope:                 "FUTURE_PR4B0_R1_RESEARCH_ONLY",
		HistoricalAuthorityClaimed:    false,
		AuthorityResponsible:          GovernanceDecisionAuthorityV3,
		ProspectiveAuthorityStatement: ProspectiveGovernanceAuthorityText1,
		HistoricalAuthorityDisclaimer: ProspectiveGovernanceAuthorityText2,
		RetroactiveClassificationRule: "prior candidate classifications remain unchanged; this authority applies prospectively only",
	}
	record.CanonicalDecisionHash, _ = ConcentrationGovernanceDecisionHashV3(record)
	return record
}

func ConcentrationGovernanceDecisionHashV3(record ConcentrationGovernanceDecisionV3) (string, error) {
	record.CanonicalDecisionHash = ""
	return canonicalDigest(record)
}

func ValidateConcentrationGovernanceDecisionV3(record ConcentrationGovernanceDecisionV3) error {
	want := DefaultConcentrationGovernanceDecisionV3()
	if !reflect.DeepEqual(record, want) {
		return errors.New("governance decision mutation requires a new immutable decision identity")
	}
	hash, err := ConcentrationGovernanceDecisionHashV3(record)
	if err != nil {
		return err
	}
	if record.CanonicalDecisionHash != hash {
		return errors.New("canonical governance decision hash mismatch")
	}
	return nil
}

type ClusteringSemanticsV3 struct {
	SourcePolicyVersion        string `json:"source_policy_version"`
	SourcePolicyHash           string `json:"source_policy_hash"`
	ExposureHorizonMinutes     int    `json:"exposure_horizon_minutes"`
	ExposureIntervalRule       string `json:"exposure_interval_rule"`
	TimestampNormalizationRule string `json:"timestamp_normalization_rule"`
	SameSymbolRule             string `json:"same_symbol_rule"`
	CrossSymbolRule            string `json:"cross_symbol_rule"`
	CommonMarketEpisodeRule    string `json:"common_market_episode_rule"`
	MissingEpisodeRule         string `json:"missing_episode_rule"`
	TransitivityRule           string `json:"transitivity_rule"`
	OrderingRule               string `json:"ordering_rule"`
	DuplicateRule              string `json:"duplicate_rule"`
	ClusterIDRule              string `json:"cluster_id_rule"`
	IndependentSampleRule      string `json:"independent_sample_rule"`
}

type ConcentrationAuthorityV3 struct {
	MetricID             string `json:"metric_id"`
	Definition           string `json:"definition"`
	Numerator            string `json:"numerator"`
	Denominator          string `json:"denominator"`
	ThresholdNumerator   string `json:"threshold_numerator"`
	ThresholdDenominator string `json:"threshold_denominator"`
	ComparisonOperator   string `json:"comparison_operator"`
	FailureCode          string `json:"failure_code"`
}

type QualificationGateScopeV3 struct {
	RequiredPartitionClasses []string `json:"required_partition_classes"`
	WalkForwardRule          string   `json:"walk_forward_rule"`
	PartitionIsolationRule   string   `json:"partition_isolation_rule"`
	CombinedScopeRule        string   `json:"combined_scope_rule"`
}

type AcceptedIndependencePolicyV3 struct {
	Version                     string                     `json:"version"`
	Status                      string                     `json:"status"`
	CandidateFamily             string                     `json:"candidate_family"`
	GovernanceDecisionID        string                     `json:"governance_decision_id"`
	GovernanceDecisionHash      string                     `json:"governance_decision_hash"`
	Clustering                  ClusteringSemanticsV3      `json:"clustering"`
	ConcentrationAuthorities    []ConcentrationAuthorityV3 `json:"concentration_authorities"`
	GateScope                   QualificationGateScopeV3   `json:"qualification_gate_scope"`
	ExactArithmeticRule         string                     `json:"exact_arithmetic_rule"`
	RoundingRule                string                     `json:"rounding_rule"`
	ZeroDenominatorRule         string                     `json:"zero_denominator_rule"`
	MissingEvidenceRule         string                     `json:"missing_evidence_rule"`
	FailureReasonCodes          []string                   `json:"failure_reason_codes"`
	Supersedes                  []string                   `json:"supersedes"`
	RejectedOutcomeContribution string                     `json:"rejected_outcome_contribution"`
	KnownLimitations            []string                   `json:"known_limitations"`
}

func AcceptedIndependencePolicyV3Default() AcceptedIndependencePolicyV3 {
	v2 := RevisedIndependencePolicyV2()
	v2Hash, _ := RevisedIndependencePolicyHashV2(v2)
	decision := DefaultConcentrationGovernanceDecisionV3()
	return AcceptedIndependencePolicyV3{
		Version:                AcceptedIndependencePolicyVersionV3,
		Status:                 PolicyStatusAccepted,
		CandidateFamily:        GovernedCandidateFamily,
		GovernanceDecisionID:   decision.DecisionID,
		GovernanceDecisionHash: decision.CanonicalDecisionHash,
		Clustering: ClusteringSemanticsV3{
			SourcePolicyVersion: v2.Version, SourcePolicyHash: v2Hash,
			ExposureHorizonMinutes: v2.ExposureHorizonMinutes, ExposureIntervalRule: v2.ExposureIntervalRule,
			TimestampNormalizationRule: v2.TimestampNormalizationRule, SameSymbolRule: v2.SameSymbolRule,
			CrossSymbolRule: v2.CrossSymbolRule, CommonMarketEpisodeRule: v2.CommonMarketEpisodeRule,
			MissingEpisodeRule: v2.MissingEpisodeRule, TransitivityRule: v2.TransitivityRule,
			OrderingRule: v2.OrderingRule, DuplicateRule: v2.DuplicateRule,
			ClusterIDRule:         "sha256 canonical JSON binding accepted V3 policy version/hash, sorted member event IDs, earliest normalized event timestamp, latest exposure endpoint, sorted unique primary symbols, and sorted applicable common-market episode identities",
			IndependentSampleRule: v2.IndependentSampleRule,
		},
		ConcentrationAuthorities: []ConcentrationAuthorityV3{
			{MetricID: "symbol_concentration", Definition: "maximum exact symbol share after each independent cluster contributes total mass one split equally across its sorted unique primary symbols", Numerator: "sum of exact fractional independent-cluster mass attributed to the symbol", Denominator: "number of valid independent clusters in the partition", ThresholdNumerator: "1", ThresholdDenominator: "2", ComparisonOperator: "<=", FailureCode: "CONCENTRATION_SYMBOL_EXCEEDED"},
			{MetricID: "temporal_concentration", Definition: "maximum share of independent clusters assigned to one UTC calendar month by earliest normalized member event timestamp using half-open UTC month intervals", Numerator: "number of independent clusters assigned to the UTC calendar month", Denominator: "number of valid independent clusters in the partition; empty months omitted", ThresholdNumerator: "1", ThresholdDenominator: "2", ComparisonOperator: "<=", FailureCode: "CONCENTRATION_TEMPORAL_EXCEEDED"},
			{MetricID: "largest_cluster_concentration", Definition: "largest unique member-event count in one independent cluster divided by all unique member events represented by valid clusters", Numerator: "maximum unique deduplicated member event ID count in one cluster", Denominator: "sum of unique member event ID counts across all valid independent clusters", ThresholdNumerator: "1", ThresholdDenominator: "2", ComparisonOperator: "<=", FailureCode: "CONCENTRATION_LARGEST_CLUSTER_EXCEEDED"},
			{MetricID: "top_five_cluster_concentration", Definition: "exact share of unique member events in up to five largest clusters ordered by descending member count then ascending canonical cluster ID", Numerator: "sum of unique member-event counts in the first five deterministically ordered clusters, or all clusters when fewer than five exist", Denominator: "sum of unique member event ID counts across all valid independent clusters", ThresholdNumerator: "7", ThresholdDenominator: "10", ComparisonOperator: "<=", FailureCode: "CONCENTRATION_TOP_FIVE_CLUSTER_EXCEEDED"},
		},
		GateScope: QualificationGateScopeV3{
			RequiredPartitionClasses: []string{"DEVELOPMENT", "VALIDATION", "MANDATORY_WALK_FORWARD_EVALUATION_SLICE", "FINAL_HOLDOUT"},
			WalkForwardRule:          "evaluate every mandatory walk-forward evaluation slice for which the accepted gate matrix requires concentration checks",
			PartitionIsolationRule:   "calculate all four metrics separately for every qualification partition; any partition failure fails qualification and FINAL_HOLDOUT cannot be averaged with earlier partitions",
			CombinedScopeRule:        "combined development-and-validation diagnostics may be reported but can never rescue or override a partition-level failure",
		},
		ExactArithmeticRule:         "reduce integer and fractional cluster mass with exact rational arithmetic; compare by exact cross multiplication only",
		RoundingRule:                "never round for qualification; decimal strings are non-authoritative diagnostics only and no value above a threshold can round into a pass",
		ZeroDenominatorRule:         "missing, empty, malformed, or nonpositive denominators fail closed",
		MissingEvidenceRule:         "all four numeric metrics and the exact accepted V3 version, policy hash, and governance decision hash are mandatory for every applicable partition",
		FailureReasonCodes:          []string{"CONCENTRATION_POLICY_VERSION_MISSING", "CONCENTRATION_POLICY_VERSION_UNKNOWN", "CONCENTRATION_POLICY_V2_PENDING", "CONCENTRATION_POLICY_HASH_MISMATCH", "CONCENTRATION_GOVERNANCE_HASH_MISMATCH", "CONCENTRATION_PARTITION_MISSING", "CONCENTRATION_PARTITION_DUPLICATE", "CONCENTRATION_PARTITION_UNEXPECTED", "CONCENTRATION_METRIC_MISSING", "CONCENTRATION_DENOMINATOR_INVALID", "CONCENTRATION_NUMERIC_INVALID", "CONCENTRATION_SYMBOL_EXCEEDED", "CONCENTRATION_TEMPORAL_EXCEEDED", "CONCENTRATION_LARGEST_CLUSTER_EXCEEDED", "CONCENTRATION_TOP_FIVE_CLUSTER_EXCEEDED"},
		Supersedes:                  []string{"ak.engine.independence.downtrend-midvol-relief.v1:REJECTED_FOR_FUTURE_PR4B0_R1", RevisedIndependencePolicyVersion + ":REVISED_PENDING_CONCENTRATION_AUTHORITY_REJECTED_FOR_QUALIFICATION"},
		RejectedOutcomeContribution: "raw-event, winning-event, positive-return, signed-return, and gross-profit contribution are excluded from mandatory concentration qualification",
		KnownLimitations:            append([]string(nil), v2.KnownLimitations...),
	}
}

func AcceptedIndependencePolicyHashV3(policy AcceptedIndependencePolicyV3) (string, error) {
	if err := ValidateAcceptedIndependencePolicyV3(policy); err != nil {
		return "", err
	}
	return canonicalDigest(policy)
}

func ValidateAcceptedIndependencePolicyV3(policy AcceptedIndependencePolicyV3) error {
	if !reflect.DeepEqual(policy, AcceptedIndependencePolicyV3Default()) {
		return errors.New("accepted V3 policy mutation requires a new version and governance decision")
	}
	if err := ValidateConcentrationGovernanceDecisionV3(DefaultConcentrationGovernanceDecisionV3()); err != nil {
		return err
	}
	return nil
}

type IndependentClusterV3 struct {
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

func ClusterEventsV3(events []RetainedEvent, policy AcceptedIndependencePolicyV3) ([]IndependentClusterV3, error) {
	if err := ValidateAcceptedIndependencePolicyV3(policy); err != nil {
		return nil, err
	}
	v2Clusters, err := ClusterEventsV2(events, RevisedIndependencePolicyV2())
	if err != nil {
		return nil, err
	}
	policyHash, err := AcceptedIndependencePolicyHashV3(policy)
	if err != nil {
		return nil, err
	}
	clusters := make([]IndependentClusterV3, 0, len(v2Clusters))
	for _, source := range v2Clusters {
		cluster := IndependentClusterV3{
			SchemaVersion: IndependentClusterSchemaVersionV3, PolicyVersion: policy.Version, PolicyHash: policyHash,
			EarliestEventTime: source.EarliestEventTime.UTC(), LatestExposureEndpoint: source.LatestExposureEndpoint.UTC(),
			MemberEventIDs: append([]string(nil), source.MemberEventIDs...), MemberSymbols: append([]string(nil), source.MemberSymbols...),
			MemberEventTimes: append([]time.Time(nil), source.MemberEventTimes...), CommonMarketEpisodeIdentities: append([]string(nil), source.CommonMarketEpisodeIdentities...),
		}
		cluster.ClusterID, err = independentClusterIDV3(cluster)
		if err != nil {
			return nil, err
		}
		if err := ValidateIndependentClusterV3(cluster, policy); err != nil {
			return nil, err
		}
		clusters = append(clusters, cluster)
	}
	sort.Slice(clusters, func(i, j int) bool {
		if !clusters[i].EarliestEventTime.Equal(clusters[j].EarliestEventTime) {
			return clusters[i].EarliestEventTime.Before(clusters[j].EarliestEventTime)
		}
		return clusters[i].ClusterID < clusters[j].ClusterID
	})
	return clusters, nil
}

func independentClusterIDV3(cluster IndependentClusterV3) (string, error) {
	payload := struct {
		PolicyVersion          string   `json:"policy_version"`
		PolicyHash             string   `json:"policy_hash"`
		MemberEventIDs         []string `json:"member_event_ids"`
		EarliestEventTime      string   `json:"earliest_event_time"`
		LatestExposureEndpoint string   `json:"latest_exposure_endpoint"`
		PrimarySymbols         []string `json:"primary_symbols"`
		CommonMarketEpisodes   []string `json:"common_market_episode_identities"`
	}{cluster.PolicyVersion, cluster.PolicyHash, cluster.MemberEventIDs, canonicalTime(cluster.EarliestEventTime), canonicalTime(cluster.LatestExposureEndpoint), cluster.MemberSymbols, cluster.CommonMarketEpisodeIdentities}
	hash, err := canonicalDigest(payload)
	if err != nil {
		return "", err
	}
	return "cluster:" + strings.TrimPrefix(hash, "sha256:"), nil
}

func ValidateIndependentClusterV3(cluster IndependentClusterV3, policy AcceptedIndependencePolicyV3) error {
	policyHash, err := AcceptedIndependencePolicyHashV3(policy)
	if err != nil {
		return err
	}
	if cluster.SchemaVersion != IndependentClusterSchemaVersionV3 || cluster.PolicyVersion != policy.Version || cluster.PolicyHash != policyHash {
		return errors.New("cluster schema or accepted V3 policy identity mismatch")
	}
	if len(cluster.MemberEventIDs) == 0 || len(cluster.MemberSymbols) == 0 || len(cluster.MemberEventTimes) != len(cluster.MemberEventIDs) || cluster.EarliestEventTime.IsZero() || !cluster.EarliestEventTime.Before(cluster.LatestExposureEndpoint) {
		return errors.New("cluster membership and exposure identity are incomplete")
	}
	if !sort.StringsAreSorted(cluster.MemberEventIDs) || !sort.StringsAreSorted(cluster.MemberSymbols) || !sort.StringsAreSorted(cluster.CommonMarketEpisodeIdentities) {
		return errors.New("cluster identities must use canonical ordering")
	}
	seenIDs := map[string]struct{}{}
	for _, id := range cluster.MemberEventIDs {
		if strings.TrimSpace(id) == "" {
			return errors.New("missing member event identity")
		}
		if _, duplicate := seenIDs[id]; duplicate {
			return errors.New("duplicate member event identity")
		}
		seenIDs[id] = struct{}{}
	}
	seenSymbols := map[string]struct{}{}
	for _, symbol := range cluster.MemberSymbols {
		if strings.TrimSpace(symbol) == "" {
			return errors.New("missing primary symbol")
		}
		if _, duplicate := seenSymbols[symbol]; duplicate {
			return errors.New("duplicate primary symbol")
		}
		seenSymbols[symbol] = struct{}{}
	}
	earliest := cluster.MemberEventTimes[0]
	for _, at := range cluster.MemberEventTimes {
		if at.IsZero() {
			return errors.New("missing member event timestamp")
		}
		if at.Before(earliest) {
			earliest = at
		}
	}
	if !earliest.UTC().Equal(cluster.EarliestEventTime.UTC()) {
		return errors.New("earliest event timestamp does not match normalized member timestamps")
	}
	wantID, err := independentClusterIDV3(cluster)
	if err != nil {
		return err
	}
	if cluster.ClusterID != wantID {
		return errors.New("cluster ID does not match canonical V3 membership identity")
	}
	return nil
}

type ExactRational struct {
	Numerator   string `json:"numerator"`
	Denominator string `json:"denominator"`
}

type ConcentrationMetricResultV3 struct {
	MetricID           string        `json:"metric_id"`
	Numerator          ExactRational `json:"numerator"`
	Denominator        string        `json:"denominator"`
	Share              ExactRational `json:"share"`
	DiagnosticPercent  string        `json:"diagnostic_percent"`
	Threshold          ExactRational `json:"threshold"`
	ComparisonOperator string        `json:"comparison_operator"`
	Passed             bool          `json:"passed"`
	FailureCode        string        `json:"failure_code,omitempty"`
}

type PartitionConcentrationResultV3 struct {
	Partition         string                      `json:"partition"`
	ClusterCount      string                      `json:"cluster_count"`
	MemberEventCount  string                      `json:"unique_member_event_count"`
	Symbol            ConcentrationMetricResultV3 `json:"symbol_concentration"`
	Temporal          ConcentrationMetricResultV3 `json:"temporal_concentration"`
	LargestCluster    ConcentrationMetricResultV3 `json:"largest_cluster_concentration"`
	TopFiveCluster    ConcentrationMetricResultV3 `json:"top_five_cluster_concentration"`
	TopFiveClusterIDs []string                    `json:"top_five_cluster_ids"`
	Passed            bool                        `json:"passed"`
}

type ConcentrationObservationV3 struct {
	Partition string               `json:"partition"`
	Cluster   IndependentClusterV3 `json:"cluster"`
}

type ConcentrationEvaluationV3 struct {
	SchemaVersion          string                           `json:"schema_version"`
	PolicyVersion          string                           `json:"policy_version"`
	PolicyHash             string                           `json:"policy_hash"`
	GovernanceDecisionHash string                           `json:"governance_decision_hash"`
	Partitions             []PartitionConcentrationResultV3 `json:"partitions"`
	Passed                 bool                             `json:"passed"`
}

func EvaluateConcentrationV3(policy AcceptedIndependencePolicyV3, expectedPartitions []string, observations []ConcentrationObservationV3) (ConcentrationEvaluationV3, error) {
	if err := ValidateAcceptedIndependencePolicyV3(policy); err != nil {
		return ConcentrationEvaluationV3{}, err
	}
	partitions, err := normalizedRequiredPartitions(expectedPartitions)
	if err != nil {
		return ConcentrationEvaluationV3{}, err
	}
	allowed := make(map[string]struct{}, len(partitions))
	byPartition := make(map[string][]IndependentClusterV3, len(partitions))
	for _, partition := range partitions {
		allowed[partition] = struct{}{}
	}
	seenClusters := map[string]struct{}{}
	seenMembers := map[string]string{}
	for _, observation := range observations {
		partition := strings.TrimSpace(observation.Partition)
		if _, ok := allowed[partition]; !ok {
			return ConcentrationEvaluationV3{}, fmt.Errorf("unexpected or missing partition %q", observation.Partition)
		}
		if err := ValidateIndependentClusterV3(observation.Cluster, policy); err != nil {
			return ConcentrationEvaluationV3{}, fmt.Errorf("partition %q: %w", partition, err)
		}
		if _, duplicate := seenClusters[observation.Cluster.ClusterID]; duplicate {
			return ConcentrationEvaluationV3{}, fmt.Errorf("duplicate cluster identity %q", observation.Cluster.ClusterID)
		}
		seenClusters[observation.Cluster.ClusterID] = struct{}{}
		for _, member := range observation.Cluster.MemberEventIDs {
			if prior, duplicate := seenMembers[member]; duplicate {
				return ConcentrationEvaluationV3{}, fmt.Errorf("member event %q appears in clusters %q and %q", member, prior, observation.Cluster.ClusterID)
			}
			seenMembers[member] = observation.Cluster.ClusterID
		}
		byPartition[partition] = append(byPartition[partition], observation.Cluster)
	}
	policyHash, err := AcceptedIndependencePolicyHashV3(policy)
	if err != nil {
		return ConcentrationEvaluationV3{}, err
	}
	result := ConcentrationEvaluationV3{SchemaVersion: ConcentrationEvaluationVersionV3, PolicyVersion: policy.Version, PolicyHash: policyHash, GovernanceDecisionHash: policy.GovernanceDecisionHash, Passed: true}
	for _, partition := range partitions {
		clusters := byPartition[partition]
		if len(clusters) == 0 {
			return ConcentrationEvaluationV3{}, fmt.Errorf("partition %q has zero concentration denominator", partition)
		}
		partitionResult, err := evaluateConcentrationPartitionV3(partition, clusters)
		if err != nil {
			return ConcentrationEvaluationV3{}, err
		}
		result.Partitions = append(result.Partitions, partitionResult)
		result.Passed = result.Passed && partitionResult.Passed
	}
	return result, nil
}

func evaluateConcentrationPartitionV3(partition string, clusters []IndependentClusterV3) (PartitionConcentrationResultV3, error) {
	symbolMass := map[string]*big.Rat{}
	monthCounts := map[string]*big.Int{}
	type sizedCluster struct {
		id    string
		count int
	}
	sizes := make([]sizedCluster, 0, len(clusters))
	totalMembers := big.NewInt(0)
	for _, cluster := range clusters {
		fraction := new(big.Rat).SetFrac64(1, int64(len(cluster.MemberSymbols)))
		for _, symbol := range cluster.MemberSymbols {
			if symbolMass[symbol] == nil {
				symbolMass[symbol] = new(big.Rat)
			}
			symbolMass[symbol].Add(symbolMass[symbol], fraction)
		}
		month := cluster.EarliestEventTime.UTC().Format("2006-01")
		if monthCounts[month] == nil {
			monthCounts[month] = big.NewInt(0)
		}
		monthCounts[month].Add(monthCounts[month], big.NewInt(1))
		count := len(cluster.MemberEventIDs)
		sizes = append(sizes, sizedCluster{cluster.ClusterID, count})
		totalMembers.Add(totalMembers, big.NewInt(int64(count)))
	}
	clusterDenominator := big.NewInt(int64(len(clusters)))
	if clusterDenominator.Sign() <= 0 || totalMembers.Sign() <= 0 {
		return PartitionConcentrationResultV3{}, fmt.Errorf("partition %q has nonpositive concentration denominator", partition)
	}
	maxSymbol := new(big.Rat)
	for _, value := range symbolMass {
		if value.Cmp(maxSymbol) > 0 {
			maxSymbol.Set(value)
		}
	}
	maxMonth := big.NewInt(0)
	for _, value := range monthCounts {
		if value.Cmp(maxMonth) > 0 {
			maxMonth.Set(value)
		}
	}
	sort.Slice(sizes, func(i, j int) bool {
		if sizes[i].count != sizes[j].count {
			return sizes[i].count > sizes[j].count
		}
		return sizes[i].id < sizes[j].id
	})
	largest := big.NewInt(int64(sizes[0].count))
	topFive := big.NewInt(0)
	selected := make([]string, 0, 5)
	for i := 0; i < len(sizes) && i < 5; i++ {
		topFive.Add(topFive, big.NewInt(int64(sizes[i].count)))
		selected = append(selected, sizes[i].id)
	}
	symbol := newMetricResultV3("symbol_concentration", maxSymbol, clusterDenominator, big.NewRat(1, 2), "CONCENTRATION_SYMBOL_EXCEEDED")
	temporal := newMetricResultV3("temporal_concentration", new(big.Rat).SetInt(maxMonth), clusterDenominator, big.NewRat(1, 2), "CONCENTRATION_TEMPORAL_EXCEEDED")
	largestMetric := newMetricResultV3("largest_cluster_concentration", new(big.Rat).SetInt(largest), totalMembers, big.NewRat(1, 2), "CONCENTRATION_LARGEST_CLUSTER_EXCEEDED")
	topFiveMetric := newMetricResultV3("top_five_cluster_concentration", new(big.Rat).SetInt(topFive), totalMembers, big.NewRat(7, 10), "CONCENTRATION_TOP_FIVE_CLUSTER_EXCEEDED")
	return PartitionConcentrationResultV3{
		Partition: partition, ClusterCount: clusterDenominator.String(), MemberEventCount: totalMembers.String(),
		Symbol: symbol, Temporal: temporal, LargestCluster: largestMetric, TopFiveCluster: topFiveMetric, TopFiveClusterIDs: selected,
		Passed: symbol.Passed && temporal.Passed && largestMetric.Passed && topFiveMetric.Passed,
	}, nil
}

func newMetricResultV3(id string, numerator *big.Rat, denominator *big.Int, threshold *big.Rat, failure string) ConcentrationMetricResultV3 {
	share := new(big.Rat).Quo(new(big.Rat).Set(numerator), new(big.Rat).SetInt(denominator))
	passed := share.Cmp(threshold) <= 0
	result := ConcentrationMetricResultV3{
		MetricID: id, Numerator: exactRational(numerator), Denominator: denominator.String(), Share: exactRational(share),
		DiagnosticPercent: new(big.Rat).Mul(new(big.Rat).Set(share), big.NewRat(100, 1)).FloatString(12),
		Threshold:         exactRational(threshold), ComparisonOperator: "<=", Passed: passed,
	}
	if !passed {
		result.FailureCode = failure
	}
	return result
}

func exactRational(value *big.Rat) ExactRational {
	return ExactRational{Numerator: value.Num().String(), Denominator: value.Denom().String()}
}

func ParseExactRational(value ExactRational) (*big.Rat, error) {
	numerator, ok := new(big.Int).SetString(value.Numerator, 10)
	if !ok {
		return nil, errors.New("invalid exact rational numerator")
	}
	denominator, ok := new(big.Int).SetString(value.Denominator, 10)
	if !ok || denominator.Sign() <= 0 {
		return nil, errors.New("invalid exact rational denominator")
	}
	ratio := new(big.Rat).SetFrac(numerator, denominator)
	if ratio.Num().String() != value.Numerator || ratio.Denom().String() != value.Denominator {
		return nil, errors.New("exact rational is not reduced and canonical")
	}
	return ratio, nil
}
