package preconditions

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
)

const (
	ConcentrationAlternativeCountV1  = "ak.engine.concentration.alternative.cluster-count.v1"
	ConcentrationAlternativeReturnV1 = "ak.engine.concentration.alternative.positive-cluster-return.v1"
	ConcentrationAlternativeStatus   = "UNACCEPTED_GOVERNANCE_ALTERNATIVE"
)

type ConcentrationBasis string

const (
	ConcentrationBasisClusterCount          ConcentrationBasis = "INDEPENDENT_CLUSTER_COUNT"
	ConcentrationBasisPositiveClusterReturn ConcentrationBasis = "POSITIVE_CLUSTER_NET_RETURN"
)

type ConcentrationAlternativePolicy struct {
	MetricVersion                  string             `json:"metric_version"`
	Status                         string             `json:"status"`
	Basis                          ConcentrationBasis `json:"basis"`
	SymbolNumerator                string             `json:"symbol_numerator_definition"`
	SymbolDenominator              string             `json:"symbol_denominator_definition"`
	SymbolAttribution              string             `json:"symbol_attribution"`
	TemporalNumerator              string             `json:"temporal_numerator_definition"`
	TemporalDenominator            string             `json:"temporal_denominator_definition"`
	TimeBucket                     string             `json:"time_bucket"`
	EmptyBucketRule                string             `json:"empty_bucket_rule"`
	LargestClusterNumerator        string             `json:"largest_cluster_numerator_definition"`
	LargestClusterDenominator      string             `json:"largest_cluster_denominator_definition"`
	AggregateClusterNumerator      string             `json:"aggregate_cluster_numerator_definition"`
	AggregateClusterDenominator    string             `json:"aggregate_cluster_denominator_definition"`
	AggregateTopN                  int                `json:"aggregate_top_n"`
	SymbolThresholdPercent         float64            `json:"symbol_threshold_percent"`
	TemporalThresholdPercent       float64            `json:"temporal_threshold_percent"`
	LargestClusterThresholdPercent float64            `json:"largest_cluster_threshold_percent"`
	AggregateThresholdPercent      float64            `json:"aggregate_threshold_percent"`
	ComparisonOperator             string             `json:"comparison_operator"`
	PartitionScope                 string             `json:"partition_scope"`
	CombinedScope                  string             `json:"combined_scope"`
	DeduplicationStage             string             `json:"deduplication_stage"`
	ClusteringStage                string             `json:"clustering_stage"`
	RoundingRule                   string             `json:"rounding_rule"`
	ZeroDenominatorRule            string             `json:"zero_denominator_rule"`
}

type ConcentrationObservation struct {
	Partition        string               `json:"partition"`
	Cluster          IndependentClusterV2 `json:"cluster"`
	ClusterNetReturn *float64             `json:"cluster_net_return,omitempty"`
}

type ConcentrationMetricResult struct {
	MetricID           string  `json:"metric_id"`
	Numerator          float64 `json:"numerator"`
	Denominator        float64 `json:"denominator"`
	Percent            float64 `json:"percent"`
	ThresholdPercent   float64 `json:"threshold_percent"`
	ComparisonOperator string  `json:"comparison_operator"`
	Passed             bool    `json:"passed"`
	FailureCode        string  `json:"failure_code,omitempty"`
}

type PartitionConcentrationResult struct {
	Partition        string                    `json:"partition"`
	Symbol           ConcentrationMetricResult `json:"symbol_concentration"`
	Temporal         ConcentrationMetricResult `json:"temporal_concentration"`
	LargestCluster   ConcentrationMetricResult `json:"largest_cluster_concentration"`
	AggregateCluster ConcentrationMetricResult `json:"aggregate_cluster_concentration"`
	Passed           bool                      `json:"passed"`
}

type ConcentrationEvaluation struct {
	PolicyVersion string                         `json:"policy_version"`
	PolicyHash    string                         `json:"policy_hash"`
	Partitions    []PartitionConcentrationResult `json:"partitions"`
	Passed        bool                           `json:"passed"`
}

func GovernanceConcentrationAlternatives() []ConcentrationAlternativePolicy {
	common := ConcentrationAlternativePolicy{
		Status:                         ConcentrationAlternativeStatus,
		SymbolAttribution:              "each independent cluster contributes 1/N to each of its N distinct primary symbols; attribution sums to one cluster",
		TemporalNumerator:              "maximum value attributed to one UTC calendar month by cluster earliest_event_time",
		TimeBucket:                     "UTC calendar month of earliest_event_time; [month start,next month start)",
		EmptyBucketRule:                "empty calendar months do not enter the maximum and do not change the denominator",
		AggregateTopN:                  5,
		SymbolThresholdPercent:         50,
		TemporalThresholdPercent:       50,
		LargestClusterThresholdPercent: 50,
		AggregateThresholdPercent:      70,
		ComparisonOperator:             "<=",
		PartitionScope:                 "evaluate every required partition independently; any partition failure fails the evidence",
		CombinedScope:                  "no combined-partition substitution; a combined scope requires its own explicitly named partition",
		DeduplicationStage:             "canonical retained-event deduplication before accepted V2 clustering",
		ClusteringStage:                "after accepted V2 connected-component clustering; raw events never substitute for symbol or temporal cluster units",
		RoundingRule:                   "compare unrounded numerators and denominators; report percentages rounded to six decimal places",
		ZeroDenominatorRule:            "fail closed; never default missing or zero-denominator concentration to zero",
	}
	count := common
	count.MetricVersion = ConcentrationAlternativeCountV1
	count.Basis = ConcentrationBasisClusterCount
	count.SymbolNumerator = "fractional count of independent clusters attributed to the most represented symbol"
	count.SymbolDenominator = "count of independent clusters in the partition"
	count.TemporalDenominator = "count of independent clusters in the partition"
	count.LargestClusterNumerator = "deduplicated member-event count in the largest independent cluster"
	count.LargestClusterDenominator = "deduplicated member-event count across all independent clusters in the partition"
	count.AggregateClusterNumerator = "deduplicated member-event count in the five largest independent clusters; include all selected values after deterministic descending size then cluster-ID ordering"
	count.AggregateClusterDenominator = count.LargestClusterDenominator

	returns := common
	returns.MetricVersion = ConcentrationAlternativeReturnV1
	returns.Basis = ConcentrationBasisPositiveClusterReturn
	returns.SymbolNumerator = "positive mandatory-cost cluster net return fractionally attributed to the highest-contributing symbol"
	returns.SymbolDenominator = "sum of positive mandatory-cost cluster net returns in the partition"
	returns.TemporalDenominator = returns.SymbolDenominator
	returns.LargestClusterNumerator = "largest positive mandatory-cost net return of one independent cluster"
	returns.LargestClusterDenominator = returns.SymbolDenominator
	returns.AggregateClusterNumerator = "sum of the five largest positive mandatory-cost independent-cluster net returns; descending return then cluster-ID ordering"
	returns.AggregateClusterDenominator = returns.SymbolDenominator
	return []ConcentrationAlternativePolicy{count, returns}
}

func ConcentrationAlternativeHash(policy ConcentrationAlternativePolicy) (string, error) {
	if err := validateConcentrationAlternative(policy); err != nil {
		return "", err
	}
	return canonicalDigest(policy)
}

func EvaluateConcentrationAlternative(policy ConcentrationAlternativePolicy, independence RevisedIndependencePolicy, expectedPartitions []string, observations []ConcentrationObservation) (ConcentrationEvaluation, error) {
	if err := validateConcentrationAlternative(policy); err != nil {
		return ConcentrationEvaluation{}, err
	}
	policyHash, err := ConcentrationAlternativeHash(policy)
	if err != nil {
		return ConcentrationEvaluation{}, err
	}
	partitions, err := normalizedRequiredPartitions(expectedPartitions)
	if err != nil {
		return ConcentrationEvaluation{}, err
	}
	byPartition := make(map[string][]ConcentrationObservation, len(partitions))
	allowed := make(map[string]struct{}, len(partitions))
	for _, partition := range partitions {
		allowed[partition] = struct{}{}
	}
	seenClusters := map[string]string{}
	seenMembers := map[string]string{}
	for _, observation := range observations {
		partition := strings.TrimSpace(observation.Partition)
		if _, ok := allowed[partition]; !ok {
			return ConcentrationEvaluation{}, fmt.Errorf("unexpected or missing partition %q", observation.Partition)
		}
		if err := validateConcentrationCluster(observation.Cluster, independence); err != nil {
			return ConcentrationEvaluation{}, fmt.Errorf("partition %q: %w", partition, err)
		}
		identity, err := canonicalDigest(observation.Cluster)
		if err != nil {
			return ConcentrationEvaluation{}, err
		}
		if prior, duplicate := seenClusters[observation.Cluster.ClusterID]; duplicate {
			if prior != identity {
				return ConcentrationEvaluation{}, fmt.Errorf("conflicting duplicate cluster %q", observation.Cluster.ClusterID)
			}
			return ConcentrationEvaluation{}, fmt.Errorf("duplicate cluster evidence %q", observation.Cluster.ClusterID)
		}
		seenClusters[observation.Cluster.ClusterID] = identity
		for _, member := range observation.Cluster.MemberEventIDs {
			if prior, duplicate := seenMembers[member]; duplicate {
				return ConcentrationEvaluation{}, fmt.Errorf("member event %q appears in clusters %q and %q", member, prior, observation.Cluster.ClusterID)
			}
			seenMembers[member] = observation.Cluster.ClusterID
		}
		if policy.Basis == ConcentrationBasisPositiveClusterReturn {
			if observation.ClusterNetReturn == nil || !finite(*observation.ClusterNetReturn) {
				return ConcentrationEvaluation{}, errors.New("positive-return alternative requires one finite cluster net return per cluster")
			}
		} else if observation.ClusterNetReturn != nil && !finite(*observation.ClusterNetReturn) {
			return ConcentrationEvaluation{}, errors.New("optional cluster net return must be finite")
		}
		byPartition[partition] = append(byPartition[partition], observation)
	}

	result := ConcentrationEvaluation{PolicyVersion: policy.MetricVersion, PolicyHash: policyHash, Passed: true}
	for _, partition := range partitions {
		rows := byPartition[partition]
		if len(rows) == 0 {
			return ConcentrationEvaluation{}, fmt.Errorf("partition %q has zero concentration denominator", partition)
		}
		sort.Slice(rows, func(i, j int) bool { return rows[i].Cluster.ClusterID < rows[j].Cluster.ClusterID })
		partitionResult, err := evaluateConcentrationPartition(policy, partition, rows)
		if err != nil {
			return ConcentrationEvaluation{}, err
		}
		result.Partitions = append(result.Partitions, partitionResult)
		result.Passed = result.Passed && partitionResult.Passed
	}
	return result, nil
}

func evaluateConcentrationPartition(policy ConcentrationAlternativePolicy, partition string, rows []ConcentrationObservation) (PartitionConcentrationResult, error) {
	symbols := map[string]float64{}
	months := map[string]float64{}
	clusterValues := make([]float64, 0, len(rows))
	denominator := 0.0
	clusterDenominator := 0.0
	for _, row := range rows {
		weight := 1.0
		if policy.Basis == ConcentrationBasisPositiveClusterReturn {
			weight = *row.ClusterNetReturn
			if weight <= 0 {
				continue
			}
		}
		denominator = deterministicAdd(denominator, weight)
		share := weight / float64(len(row.Cluster.MemberSymbols))
		for _, symbol := range row.Cluster.MemberSymbols {
			symbols[symbol] = deterministicAdd(symbols[symbol], share)
		}
		month := row.Cluster.EarliestEventTime.UTC().Format("2006-01")
		months[month] = deterministicAdd(months[month], weight)
		if policy.Basis == ConcentrationBasisClusterCount {
			members := float64(len(row.Cluster.MemberEventIDs))
			clusterValues = append(clusterValues, members)
			clusterDenominator = deterministicAdd(clusterDenominator, members)
		} else {
			clusterValues = append(clusterValues, weight)
		}
	}
	if denominator <= 0 {
		return PartitionConcentrationResult{}, fmt.Errorf("partition %q has zero concentration denominator", partition)
	}
	if policy.Basis == ConcentrationBasisPositiveClusterReturn {
		clusterDenominator = denominator
	}
	if clusterDenominator <= 0 || len(clusterValues) == 0 {
		return PartitionConcentrationResult{}, fmt.Errorf("partition %q has zero cluster concentration denominator", partition)
	}
	sort.Slice(clusterValues, func(i, j int) bool { return clusterValues[i] > clusterValues[j] })
	largest := clusterValues[0]
	topN := policy.AggregateTopN
	if topN > len(clusterValues) {
		topN = len(clusterValues)
	}
	aggregate := 0.0
	for i := 0; i < topN; i++ {
		aggregate = deterministicAdd(aggregate, clusterValues[i])
	}
	result := PartitionConcentrationResult{
		Partition:        partition,
		Symbol:           metricResult("symbol_concentration", maxMapValue(symbols), denominator, policy.SymbolThresholdPercent, "CONCENTRATION_SYMBOL_EXCEEDED"),
		Temporal:         metricResult("temporal_concentration", maxMapValue(months), denominator, policy.TemporalThresholdPercent, "CONCENTRATION_TEMPORAL_EXCEEDED"),
		LargestCluster:   metricResult("largest_cluster_concentration", largest, clusterDenominator, policy.LargestClusterThresholdPercent, "CONCENTRATION_LARGEST_CLUSTER_EXCEEDED"),
		AggregateCluster: metricResult("aggregate_cluster_concentration", aggregate, clusterDenominator, policy.AggregateThresholdPercent, "CONCENTRATION_AGGREGATE_CLUSTER_EXCEEDED"),
	}
	result.Passed = result.Symbol.Passed && result.Temporal.Passed && result.LargestCluster.Passed && result.AggregateCluster.Passed
	return result, nil
}

func metricResult(id string, numerator, denominator, threshold float64, failure string) ConcentrationMetricResult {
	percent := numerator / denominator * 100
	passed := numerator*100 <= threshold*denominator
	result := ConcentrationMetricResult{
		MetricID: id, Numerator: numerator, Denominator: denominator, Percent: roundSix(percent),
		ThresholdPercent: threshold, ComparisonOperator: "<=", Passed: passed,
	}
	if !passed {
		result.FailureCode = failure
	}
	return result
}

func validateConcentrationAlternative(policy ConcentrationAlternativePolicy) error {
	for _, known := range GovernanceConcentrationAlternatives() {
		if policy == known {
			return nil
		}
	}
	return errors.New("unregistered concentration alternative or same-version policy mutation")
}

func validateConcentrationCluster(cluster IndependentClusterV2, independence RevisedIndependencePolicy) error {
	if err := ValidateIndependentClusterV2(cluster, independence); err != nil {
		return err
	}
	for _, symbol := range cluster.MemberSymbols {
		if strings.TrimSpace(symbol) == "" {
			return errors.New("cluster member symbol is missing")
		}
	}
	for _, at := range cluster.MemberEventTimes {
		if at.IsZero() {
			return errors.New("cluster member timestamp is missing")
		}
	}
	return nil
}

func normalizedRequiredPartitions(values []string) ([]string, error) {
	if len(values) == 0 {
		return nil, errors.New("at least one required partition is required")
	}
	seen := map[string]struct{}{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			return nil, errors.New("partition identity is missing")
		}
		if _, duplicate := seen[value]; duplicate {
			return nil, fmt.Errorf("duplicate partition %q", value)
		}
		seen[value] = struct{}{}
	}
	out := make([]string, 0, len(seen))
	for value := range seen {
		out = append(out, value)
	}
	sort.Strings(out)
	return out, nil
}

func maxMapValue(values map[string]float64) float64 {
	maximum := 0.0
	for _, value := range values {
		if value > maximum {
			maximum = value
		}
	}
	return maximum
}

func deterministicAdd(total, value float64) float64 {
	// Inputs are first sorted by canonical cluster identity, making this stable.
	return total + value
}

func roundSix(value float64) float64 {
	return math.Round(value*1_000_000) / 1_000_000
}
