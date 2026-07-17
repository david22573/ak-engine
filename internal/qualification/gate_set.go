package qualification

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
)

const (
	PR4B0GateSetSchemaVersion = "ak.engine.qualification_gate_set.v1"
	PR4B0GateSetID            = "ak.engine.qualification-gates.pr4b0.v1"
)

type GateComparison struct {
	GateID    string `json:"gate_id"`
	Version   string `json:"version"`
	Metric    string `json:"metric"`
	Operator  string `json:"comparison_operator"`
	Threshold string `json:"threshold"`
	Failure   string `json:"failure_behavior"`
}

type PR4B0GateSet struct {
	SchemaVersion     string           `json:"schema_version"`
	GateSetID         string           `json:"gate_set_id"`
	PolicyStatus      string           `json:"policy_status"`
	DataIntegrity     map[string]any   `json:"data_integrity"`
	Sample            SampleGates      `json:"sample_sufficiency"`
	Performance       PerformanceGates `json:"performance"`
	Robustness        RobustnessGates  `json:"robustness"`
	Cost              CostGates        `json:"cost_stress"`
	Comparisons       []GateComparison `json:"comparison_semantics"`
	LeakageRules      []string         `json:"leakage_rules"`
	SimplicityRule    string           `json:"simplicity_rule"`
	RejectionBehavior string           `json:"rejection_behavior"`
}

func AcceptedPR4B0GateSet() PR4B0GateSet {
	return PR4B0GateSet{
		SchemaVersion: PR4B0GateSetSchemaVersion,
		GateSetID:     PR4B0GateSetID,
		PolicyStatus:  "DECLARED_FOR_EXISTING_EVIDENCE_ASSESSMENT; NO_NEW_SEARCH_EXECUTED",
		DataIntegrity: map[string]any{
			"dataset_id":               "required exact immutable identity; none established for an Engine candidate",
			"dataset_version":          "required immutable version; mutable aliases prohibited",
			"manifest_id_and_hash":     "required and must match accepted Historian PIT evidence",
			"windows":                  "non-overlapping development, validation, and one-time final holdout",
			"required_symbols":         []string{"ADAUSDT", "AVAXUSDT", "BNBUSDT", "DOGEUSDT", "ETHUSDT", "LINKUSDT", "SOLUSDT", "XRPUSDT"},
			"required_context_symbols": []string{"BTCUSDT", "ETHUSDT where candidate capability requires them"},
			"gap_policy":               "reject undeclared internal gaps",
			"pit_policy":               "source available_at must be no later than decision cutoff",
		},
		Sample:      SampleGates{MinimumEvents: 300, MinimumIndependentClusters: 300, MinimumTradesOrDecisions: 300, MinimumSymbols: 4, MinimumMonths: 12, MinimumPositiveRegimes: 1, MinimumNegativeRegimes: 1},
		Performance: PerformanceGates{MinimumNetExpectancyBPS: 0.01, MinimumProfitFactor: 1.10, MaximumDrawdownBPS: 2500, MinimumConfidenceLowerBoundBPS: 0.0, DownsideTailPolicy: "lower confidence bound must be positive and worst-decile/worst-symbol-month loss must fit declared drawdown budget"},
		Robustness:  RobustnessGates{RequireOutOfSample: true, RequireWalkForward: true, MinimumWorstPeriodProfitFactor: 0.95, MaximumSymbolContributionPercent: 50, MaximumTemporalContributionPercent: 50, MaximumRegimeContributionPercent: 60, MinimumStableNeighbors: 2, RequireClusterDeduplication: true, RequireMissingContextSensitivity: true},
		Cost:        CostGates{FeeBPS: 5, SpreadBPS: 1, SlippageBPS: 1, FundingBPS: 1, AdverseSelectionBPS: 2, StressTotalBPS: 10, MinimumStressProfitFactor: 1.01, MinimumStressExpectancyBPS: 0.01},
		Comparisons: []GateComparison{
			{"minimum_events", "v1", "event_count", ">=", "300", "REJECT"},
			{"minimum_independent_clusters", "v1", "independent_cluster_count", ">=", "300", "REJECT"},
			{"minimum_trades_or_decisions", "v1", "trades_or_decisions", ">=", "300", "REJECT"},
			{"minimum_symbols", "v1", "symbols_represented", ">=", "4", "REJECT"},
			{"minimum_months", "v1", "months_represented", ">=", "12", "REJECT"},
			{"minimum_net_expectancy", "v1", "net_expectancy_bps", ">=", "0.01", "REJECT"},
			{"minimum_profit_factor", "v1", "profit_factor", ">=", "1.10", "REJECT"},
			{"maximum_drawdown", "v1", "drawdown_bps", "<=", "2500", "REJECT"},
			{"uncertainty_lower_bound", "v2", "cluster_bootstrap_lower_bound_bps", ">", "0", "REJECT"},
			{"minimum_worst_period_pf", "v1", "worst_period_profit_factor", ">=", "0.95", "REJECT"},
			{"maximum_symbol_concentration", "v3", "structural_symbol_share", "<=", "1/2", "REJECT"},
			{"maximum_temporal_concentration", "v3", "structural_temporal_share", "<=", "1/2", "REJECT"},
			{"maximum_largest_cluster", "v3", "largest_cluster_member_share", "<=", "1/2", "REJECT"},
			{"maximum_top_five_clusters", "v3", "top_five_cluster_member_share", "<=", "7/10", "REJECT"},
			{"minimum_stable_neighbors", "v1", "stable_parameter_neighbors", ">=", "2", "REJECT"},
			{"minimum_stress_profit_factor", "v1", "stress_profit_factor", ">=", "1.01", "REJECT"},
			{"minimum_stress_expectancy", "v1", "stress_expectancy_bps", ">=", "0.01", "REJECT"},
		},
		LeakageRules:      []string{"reject future candles", "reject revised data unavailable at cutoff", "reject final outcomes in features", "reject holdout-derived feature selection", "require PIT-compatible source timing", "reject manifest/dataset mismatch"},
		SimplicityRule:    "select the simplest candidate only after all mandatory gates pass; additional degrees of freedom require material validation improvement",
		RejectionBehavior: "every missing or failed mandatory gate rejects; integrity or identity failures block rather than become performance failures; no fallback gate set",
	}
}

func PR4B0GateSetHash(gates PR4B0GateSet) (string, error) {
	if err := ValidatePR4B0GateSet(gates); err != nil {
		return "", err
	}
	data, err := json.Marshal(gates)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(digest[:]), nil
}

func ValidatePR4B0GateSet(gates PR4B0GateSet) error {
	if !reflect.DeepEqual(gates, AcceptedPR4B0GateSet()) {
		return errors.New("PR4B0 gate-set mutation or substitution requires a new authority")
	}
	if gates.SchemaVersion != PR4B0GateSetSchemaVersion || gates.GateSetID != PR4B0GateSetID {
		return errors.New("unsupported PR4B0 gate-set identity")
	}
	seen := map[string]struct{}{}
	for _, gate := range gates.Comparisons {
		if gate.GateID == "" || gate.Version == "" || gate.Metric == "" || gate.Threshold == "" || gate.Failure != "REJECT" || (gate.Operator != ">=" && gate.Operator != ">" && gate.Operator != "<=") {
			return fmt.Errorf("invalid gate comparison %q", gate.GateID)
		}
		if _, duplicate := seen[gate.GateID]; duplicate {
			return fmt.Errorf("duplicate gate comparison %q", gate.GateID)
		}
		seen[gate.GateID] = struct{}{}
	}
	return nil
}

func PR4B0GateIdentities(gates PR4B0GateSet) ([]EvidenceReference, error) {
	if err := ValidatePR4B0GateSet(gates); err != nil {
		return nil, err
	}
	identities := make([]EvidenceReference, 0, len(gates.Comparisons))
	for _, gate := range gates.Comparisons {
		data, err := json.Marshal(gate)
		if err != nil {
			return nil, err
		}
		digest := sha256.Sum256(data)
		identities = append(identities, EvidenceReference{ArtifactID: gate.GateID + ":" + gate.Version, SourceRef: PR4B0GateSetID, SHA256: "sha256:" + hex.EncodeToString(digest[:])})
	}
	return identities, nil
}
