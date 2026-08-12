package qualification

import (
	"math/big"
	"sort"
	"strings"

	"github.com/david22573/ak-engine/internal/preconditions"
)

type QualificationConcentrationEvidenceV3 struct {
	ExpectedPartitions []string                                      `json:"expected_partitions"`
	Evaluation         *preconditions.ConcentrationEvaluationV3      `json:"evaluation"`
	CombinedDiagnostic *preconditions.PartitionConcentrationResultV3 `json:"combined_development_validation_diagnostic,omitempty"`
	ReportBooleans     map[string]bool                               `json:"report_booleans,omitempty"`
}

type QualificationConcentrationDecisionV3 struct {
	Passed      bool     `json:"passed"`
	ReasonCodes []string `json:"reason_codes"`
}

func EvaluateQualificationConcentrationV3(evidence *QualificationConcentrationEvidenceV3) QualificationConcentrationDecisionV3 {
	reasons := map[string]struct{}{}
	add := func(code string) { reasons[code] = struct{}{} }
	if evidence == nil || evidence.Evaluation == nil {
		add("CONCENTRATION_METRIC_MISSING")
		return concentrationDecision(reasons)
	}
	evaluation := evidence.Evaluation
	policy := preconditions.AcceptedIndependencePolicyV3Default()
	wantHash, err := preconditions.AcceptedIndependencePolicyHashV3(policy)
	if err != nil {
		add("CONCENTRATION_POLICY_HASH_MISMATCH")
		return concentrationDecision(reasons)
	}
	switch evaluation.PolicyVersion {
	case "":
		add("CONCENTRATION_POLICY_VERSION_MISSING")
	case preconditions.RevisedIndependencePolicyVersion:
		add("CONCENTRATION_POLICY_V2_PENDING")
	case preconditions.AcceptedIndependencePolicyVersionV3:
	default:
		add("CONCENTRATION_POLICY_VERSION_UNKNOWN")
	}
	if evaluation.SchemaVersion != preconditions.ConcentrationEvaluationVersionV3 {
		add("CONCENTRATION_POLICY_VERSION_UNKNOWN")
	}
	if evaluation.PolicyHash != wantHash {
		add("CONCENTRATION_POLICY_HASH_MISMATCH")
	}
	if evaluation.GovernanceDecisionHash != policy.GovernanceDecisionHash {
		add("CONCENTRATION_GOVERNANCE_HASH_MISMATCH")
	}
	expected, valid := normalizedQualificationPartitions(evidence.ExpectedPartitions)
	if !valid {
		add("CONCENTRATION_PARTITION_MISSING")
	}
	expectedSet := make(map[string]struct{}, len(expected))
	for _, partition := range expected {
		expectedSet[partition] = struct{}{}
	}
	seen := map[string]struct{}{}
	allNumericPassed := true
	for _, partition := range evaluation.Partitions {
		name := strings.TrimSpace(partition.Partition)
		if _, duplicate := seen[name]; duplicate {
			add("CONCENTRATION_PARTITION_DUPLICATE")
		}
		seen[name] = struct{}{}
		if _, ok := expectedSet[name]; !ok || name == "" {
			add("CONCENTRATION_PARTITION_UNEXPECTED")
		}
		metrics := []struct {
			result    preconditions.ConcentrationMetricResultV3
			id        string
			threshold *big.Rat
			failure   string
		}{
			{partition.Symbol, "symbol_concentration", big.NewRat(1, 2), "CONCENTRATION_SYMBOL_EXCEEDED"},
			{partition.Temporal, "temporal_concentration", big.NewRat(1, 2), "CONCENTRATION_TEMPORAL_EXCEEDED"},
			{partition.LargestCluster, "largest_cluster_concentration", big.NewRat(1, 2), "CONCENTRATION_LARGEST_CLUSTER_EXCEEDED"},
			{partition.TopFiveCluster, "top_five_cluster_concentration", big.NewRat(7, 10), "CONCENTRATION_TOP_FIVE_CLUSTER_EXCEEDED"},
		}
		partitionPassed := true
		for _, metric := range metrics {
			passed, code := validateQualificationMetricV3(metric.result, metric.id, metric.threshold, metric.failure)
			if code != "" {
				add(code)
			}
			partitionPassed = partitionPassed && passed
		}
		if !partitionPassed {
			allNumericPassed = false
		}
		if partition.Passed != partitionPassed {
			add("CONCENTRATION_NUMERIC_INVALID")
		}
	}
	for _, partition := range expected {
		if _, ok := seen[partition]; !ok {
			add("CONCENTRATION_PARTITION_MISSING")
		}
	}
	if len(evaluation.Partitions) == 0 {
		add("CONCENTRATION_PARTITION_MISSING")
	}
	if evaluation.Passed != allNumericPassed {
		add("CONCENTRATION_NUMERIC_INVALID")
	}
	// CombinedDiagnostic and ReportBooleans are deliberately non-authoritative.
	// Their values cannot remove any partition or numeric reason above.
	return concentrationDecision(reasons)
}

func validateQualificationMetricV3(result preconditions.ConcentrationMetricResultV3, id string, threshold *big.Rat, failure string) (bool, string) {
	if result.MetricID == "" {
		return false, "CONCENTRATION_METRIC_MISSING"
	}
	if result.MetricID != id || result.ComparisonOperator != "<=" {
		return false, "CONCENTRATION_NUMERIC_INVALID"
	}
	numerator, err := preconditions.ParseExactRational(result.Numerator)
	if err != nil || numerator.Sign() < 0 {
		return false, "CONCENTRATION_NUMERIC_INVALID"
	}
	denominator, ok := new(big.Int).SetString(result.Denominator, 10)
	if !ok || denominator.Sign() <= 0 {
		return false, "CONCENTRATION_DENOMINATOR_INVALID"
	}
	share, err := preconditions.ParseExactRational(result.Share)
	if err != nil {
		return false, "CONCENTRATION_NUMERIC_INVALID"
	}
	wantShare := new(big.Rat).Quo(numerator, new(big.Rat).SetInt(denominator))
	if share.Cmp(wantShare) != 0 {
		return false, "CONCENTRATION_NUMERIC_INVALID"
	}
	recordedThreshold, err := preconditions.ParseExactRational(result.Threshold)
	if err != nil || recordedThreshold.Cmp(threshold) != 0 {
		return false, "CONCENTRATION_NUMERIC_INVALID"
	}
	passed := share.Cmp(threshold) <= 0
	if result.Passed != passed {
		return false, "CONCENTRATION_NUMERIC_INVALID"
	}
	if passed {
		if result.FailureCode != "" {
			return false, "CONCENTRATION_NUMERIC_INVALID"
		}
		return true, ""
	}
	if result.FailureCode != failure {
		return false, "CONCENTRATION_NUMERIC_INVALID"
	}
	return false, failure
}

func normalizedQualificationPartitions(values []string) ([]string, bool) {
	if len(values) == 0 {
		return nil, false
	}
	seen := map[string]struct{}{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			return nil, false
		}
		if _, duplicate := seen[value]; duplicate {
			return nil, false
		}
		seen[value] = struct{}{}
	}
	out := make([]string, 0, len(seen))
	for value := range seen {
		out = append(out, value)
	}
	sort.Strings(out)
	return out, true
}

func concentrationDecision(reasons map[string]struct{}) QualificationConcentrationDecisionV3 {
	codes := make([]string, 0, len(reasons))
	for code := range reasons {
		codes = append(codes, code)
	}
	sort.Strings(codes)
	return QualificationConcentrationDecisionV3{Passed: len(codes) == 0, ReasonCodes: codes}
}

func concentrationAuthorityReason(decision QualificationConcentrationDecisionV3) bool {
	for _, code := range decision.ReasonCodes {
		switch code {
		case "CONCENTRATION_POLICY_VERSION_MISSING", "CONCENTRATION_POLICY_VERSION_UNKNOWN", "CONCENTRATION_POLICY_V2_PENDING", "CONCENTRATION_POLICY_HASH_MISMATCH", "CONCENTRATION_GOVERNANCE_HASH_MISMATCH", "CONCENTRATION_METRIC_MISSING":
			return true
		}
	}
	return false
}
