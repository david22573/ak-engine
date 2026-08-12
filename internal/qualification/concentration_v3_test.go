package qualification

import (
	"math/big"
	"strings"
	"testing"

	"github.com/david22573/ak-engine/internal/preconditions"
)

func TestQualificationV3EveryConcentrationGatePassing(t *testing.T) {
	evidence := passingConcentrationEvidenceV3("DEVELOPMENT", "VALIDATION", "WALK_FORWARD_EVALUATION_SLICE:001", "FINAL_HOLDOUT")
	decision := EvaluateQualificationConcentrationV3(evidence)
	if !decision.Passed || len(decision.ReasonCodes) != 0 {
		t.Fatalf("passing evidence=%+v", decision)
	}
	candidate := validCandidateRecord()
	candidate.EligibilityClassification = ClassificationQualificationCandidate
	candidate.ImplementationReproducible = true
	gates := allGateEvidence()
	gates.ConcentrationV3 = evidence
	if got := QualificationStatus(candidate, gates, false); got != StatusQualified {
		t.Fatalf("qualification=%s", got)
	}
}

func TestQualificationV3EachGateFailsIndependently(t *testing.T) {
	tests := []struct {
		name string
		code string
		set  func(*preconditions.PartitionConcentrationResultV3, preconditions.ConcentrationMetricResultV3)
	}{
		{"symbol", "CONCENTRATION_SYMBOL_EXCEEDED", func(p *preconditions.PartitionConcentrationResultV3, m preconditions.ConcentrationMetricResultV3) {
			p.Symbol = m
		}},
		{"temporal", "CONCENTRATION_TEMPORAL_EXCEEDED", func(p *preconditions.PartitionConcentrationResultV3, m preconditions.ConcentrationMetricResultV3) {
			p.Temporal = m
		}},
		{"largest", "CONCENTRATION_LARGEST_CLUSTER_EXCEEDED", func(p *preconditions.PartitionConcentrationResultV3, m preconditions.ConcentrationMetricResultV3) {
			p.LargestCluster = m
		}},
		{"top_five", "CONCENTRATION_TOP_FIVE_CLUSTER_EXCEEDED", func(p *preconditions.PartitionConcentrationResultV3, m preconditions.ConcentrationMetricResultV3) {
			p.TopFiveCluster = m
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			evidence := passingConcentrationEvidenceV3("DEVELOPMENT")
			threshold := exact("1", "2")
			if test.name == "top_five" {
				threshold = exact("7", "10")
			}
			metricID := map[string]string{"symbol": "symbol_concentration", "temporal": "temporal_concentration", "largest": "largest_cluster_concentration", "top_five": "top_five_cluster_concentration"}[test.name]
			failed := metricEvidenceV3(metricID, "51", "100", threshold, false, test.code)
			if test.name == "top_five" {
				failed = metricEvidenceV3(metricID, "71", "100", threshold, false, test.code)
			}
			test.set(&evidence.Evaluation.Partitions[0], failed)
			evidence.Evaluation.Partitions[0].Passed = false
			evidence.Evaluation.Passed = false
			decision := EvaluateQualificationConcentrationV3(evidence)
			if decision.Passed || !containsCode(decision.ReasonCodes, test.code) {
				t.Fatalf("decision=%+v", decision)
			}
		})
	}
}

func TestQualificationV3MissingZeroAndPolicyFailures(t *testing.T) {
	tests := []struct {
		name, code string
		mutate     func(*QualificationConcentrationEvidenceV3)
	}{
		{"missing_metric", "CONCENTRATION_METRIC_MISSING", func(e *QualificationConcentrationEvidenceV3) {
			e.Evaluation.Partitions[0].Symbol = preconditions.ConcentrationMetricResultV3{}
		}},
		{"zero_denominator", "CONCENTRATION_DENOMINATOR_INVALID", func(e *QualificationConcentrationEvidenceV3) { e.Evaluation.Partitions[0].Symbol.Denominator = "0" }},
		{"wrong_version", "CONCENTRATION_POLICY_VERSION_UNKNOWN", func(e *QualificationConcentrationEvidenceV3) {
			e.Evaluation.PolicyVersion = "ak.engine.independence.downtrend-midvol-relief.v4"
		}},
		{"wrong_hash", "CONCENTRATION_POLICY_HASH_MISMATCH", func(e *QualificationConcentrationEvidenceV3) { e.Evaluation.PolicyHash = sha('a') }},
		{"pending_v2", "CONCENTRATION_POLICY_V2_PENDING", func(e *QualificationConcentrationEvidenceV3) {
			e.Evaluation.PolicyVersion = preconditions.RevisedIndependencePolicyVersion
			e.Evaluation.PolicyHash = pendingR1P3IndependencePolicyHash
		}},
		{"unknown_policy", "CONCENTRATION_POLICY_VERSION_UNKNOWN", func(e *QualificationConcentrationEvidenceV3) { e.Evaluation.PolicyVersion = "unknown" }},
		{"wrong_governance", "CONCENTRATION_GOVERNANCE_HASH_MISMATCH", func(e *QualificationConcentrationEvidenceV3) { e.Evaluation.GovernanceDecisionHash = sha('b') }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			evidence := passingConcentrationEvidenceV3("DEVELOPMENT")
			test.mutate(evidence)
			decision := EvaluateQualificationConcentrationV3(evidence)
			if decision.Passed || !containsCode(decision.ReasonCodes, test.code) {
				t.Fatalf("decision=%+v", decision)
			}
		})
	}
}

func TestQualificationV3ReportBooleansCannotBypassNumericFailure(t *testing.T) {
	evidence := passingConcentrationEvidenceV3("DEVELOPMENT")
	evidence.ReportBooleans = map[string]bool{"symbol_concentration": true, "temporal_concentration": true, "largest_cluster_concentration": true, "top_five_cluster_concentration": true}
	evidence.Evaluation.Partitions[0].Symbol = metricEvidenceV3("symbol_concentration", "51", "100", exact("1", "2"), false, "CONCENTRATION_SYMBOL_EXCEEDED")
	evidence.Evaluation.Partitions[0].Passed = false
	evidence.Evaluation.Passed = false
	decision := EvaluateQualificationConcentrationV3(evidence)
	if decision.Passed || !containsCode(decision.ReasonCodes, "CONCENTRATION_SYMBOL_EXCEEDED") {
		t.Fatalf("true booleans bypassed numeric evidence: %+v", decision)
	}
}

func TestQualificationV3PartitionFailureCannotBeRescued(t *testing.T) {
	for _, failingPartition := range []string{"VALIDATION", "FINAL_HOLDOUT"} {
		t.Run(strings.ToLower(failingPartition), func(t *testing.T) {
			evidence := passingConcentrationEvidenceV3("DEVELOPMENT", "VALIDATION", "FINAL_HOLDOUT")
			for i := range evidence.Evaluation.Partitions {
				if evidence.Evaluation.Partitions[i].Partition == failingPartition {
					evidence.Evaluation.Partitions[i].Temporal = metricEvidenceV3("temporal_concentration", "51", "100", exact("1", "2"), false, "CONCENTRATION_TEMPORAL_EXCEEDED")
					evidence.Evaluation.Partitions[i].Passed = false
				}
			}
			evidence.Evaluation.Passed = false
			combined := passingPartitionV3("COMBINED_DEVELOPMENT_VALIDATION")
			evidence.CombinedDiagnostic = &combined
			decision := EvaluateQualificationConcentrationV3(evidence)
			if decision.Passed || !containsCode(decision.ReasonCodes, "CONCENTRATION_TEMPORAL_EXCEEDED") {
				t.Fatalf("partition rescued: %+v", decision)
			}
		})
	}
}

func TestQualificationV3PriorRejectedAndNearMissCannotUpgrade(t *testing.T) {
	gates := allGateEvidence()
	gates.ConcentrationV3 = passingConcentrationEvidenceV3("DEVELOPMENT")
	for _, test := range []struct {
		class EligibilityClassification
		want  FinalStatus
	}{{ClassificationRejected, StatusRejected}, {ClassificationNearMiss, StatusNearMiss}} {
		candidate := validCandidateRecord()
		candidate.EligibilityClassification = test.class
		candidate.ImplementationReproducible = true
		if got := QualificationStatus(candidate, gates, true); got != test.want {
			t.Fatalf("class=%s got=%s", test.class, got)
		}
	}
}

func passingConcentrationEvidenceV3(partitions ...string) *QualificationConcentrationEvidenceV3 {
	policy := preconditions.AcceptedIndependencePolicyV3Default()
	hash, _ := preconditions.AcceptedIndependencePolicyHashV3(policy)
	evaluation := &preconditions.ConcentrationEvaluationV3{
		SchemaVersion: preconditions.ConcentrationEvaluationVersionV3, PolicyVersion: policy.Version,
		PolicyHash: hash, GovernanceDecisionHash: policy.GovernanceDecisionHash, Passed: true,
	}
	for _, partition := range partitions {
		evaluation.Partitions = append(evaluation.Partitions, passingPartitionV3(partition))
	}
	return &QualificationConcentrationEvidenceV3{ExpectedPartitions: append([]string(nil), partitions...), Evaluation: evaluation}
}

func passingPartitionV3(partition string) preconditions.PartitionConcentrationResultV3 {
	return preconditions.PartitionConcentrationResultV3{
		Partition: partition, ClusterCount: "10", MemberEventCount: "10",
		Symbol:         metricEvidenceV3("symbol_concentration", "5", "10", exact("1", "2"), true, ""),
		Temporal:       metricEvidenceV3("temporal_concentration", "5", "10", exact("1", "2"), true, ""),
		LargestCluster: metricEvidenceV3("largest_cluster_concentration", "5", "10", exact("1", "2"), true, ""),
		TopFiveCluster: metricEvidenceV3("top_five_cluster_concentration", "7", "10", exact("7", "10"), true, ""),
		Passed:         true,
	}
}

func metricEvidenceV3(id, numerator, denominator string, threshold preconditions.ExactRational, passed bool, failure string) preconditions.ConcentrationMetricResultV3 {
	share := exact(numerator, denominator)
	return preconditions.ConcentrationMetricResultV3{
		MetricID: id, Numerator: preconditions.ExactRational{Numerator: numerator, Denominator: "1"}, Denominator: denominator,
		Share: share, DiagnosticPercent: "NON_AUTHORITATIVE", Threshold: threshold, ComparisonOperator: "<=", Passed: passed, FailureCode: failure,
	}
}

func exact(numerator, denominator string) preconditions.ExactRational {
	n, _ := new(big.Int).SetString(numerator, 10)
	d, _ := new(big.Int).SetString(denominator, 10)
	ratio := new(big.Rat).SetFrac(n, d)
	return preconditions.ExactRational{Numerator: ratio.Num().String(), Denominator: ratio.Denom().String()}
}

func containsCode(codes []string, want string) bool {
	for _, code := range codes {
		if code == want {
			return true
		}
	}
	return false
}
