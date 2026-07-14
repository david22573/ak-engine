package preconditions

import (
	"encoding/json"
	"math"
	"strings"
	"testing"
	"time"
)

func TestRetainedEventRoundTripAndIntegrity(t *testing.T) {
	event := syntheticEvent(t, "AAAUSDT", time.Date(2025, 1, 15, 1, 0, 0, 0, time.UTC))
	first, err := EncodeRetainedEvent(event)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeRetainedEvent(first)
	if err != nil {
		t.Fatal(err)
	}
	second, err := EncodeRetainedEvent(decoded)
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) {
		t.Fatal("canonical retained-event bytes changed after round trip")
	}
	mutated := decoded
	mutated.Features.RealizedVol60 += 0.0001
	if _, err := EncodeRetainedEvent(mutated); err == nil {
		t.Fatal("replay input mutation was not detected")
	}
	if hash, err := RetainedEventSchemaHash(); err != nil || !validSHA256(hash) {
		t.Fatalf("invalid schema hash %q: %v", hash, err)
	}
	descriptor := RetainedEventSchemaDescriptor()
	originalHash, _ := canonicalDigest(descriptor)
	descriptor.Fields = append(descriptor.Fields, "synthetic_mutation")
	mutatedHash, _ := canonicalDigest(descriptor)
	if originalHash == mutatedHash {
		t.Fatal("schema mutation did not change schema hash")
	}
}

func TestAuthoritativeCapabilityFieldsAreRepresented(t *testing.T) {
	requiredRetained := []string{
		"event_id", "candidate_family", "candidate_version", "implementation_hash", "primary_symbol", "event_timestamp", "decision_timestamp",
		"source_partition_id", "source_snapshot_id", "source_input_hash", "feature_schema_version", "trend_state", "primary_regime", "volatility_bucket",
		"decision_features.close", "decision_features.ema_50", "decision_features.ema_200", "decision_features.trend_slope_20", "decision_features.realized_vol_60",
		"btc_context.symbol", "btc_context.snapshot_id", "btc_context.source_input_hash", "btc_context.available_at", "btc_context.return_60",
		"eth_context.symbol", "eth_context.snapshot_id", "eth_context.source_input_hash", "eth_context.available_at", "eth_context.return_60",
		"reference_price", "evaluation_horizon", "evaluation_horizon_ms", "warmup_sufficient", "deterministic_exclusion_reason",
		"cost_inputs.fee_bps", "cost_inputs.spread_bps", "cost_inputs.slippage_bps", "cost_inputs.funding_bps", "cost_inputs.adverse_selection_bps",
		"attribution.month", "attribution.quarter", "attribution.regime", "replay_input_hash",
	}
	requiredCluster := []string{"cluster_id", "start", "end", "member_event_ids", "member_symbols", "cluster_timestamps", "same_symbol_overlap_identities", "common_market_cluster_identity"}
	assertDescriptorFields(t, RetainedEventSchemaDescriptor(), requiredRetained)
	assertDescriptorFields(t, IndependentClusterSchemaDescriptor(), requiredCluster)
	if hash, err := IndependentClusterSchemaHash(); err != nil || !validSHA256(hash) {
		t.Fatalf("invalid independent-cluster schema hash %q: %v", hash, err)
	}
	for _, field := range RetainedEventSchemaDescriptor().Fields {
		lower := strings.ToLower(field)
		for _, prohibited := range []string{"outcome", "profit", "expectancy", "win_rate", "loss_rate", "drawdown"} {
			if strings.Contains(lower, prohibited) {
				t.Fatalf("outcome-derived field entered retained schema: %s", field)
			}
		}
	}
}

func TestRetainedEventRequiredFieldsFailClosed(t *testing.T) {
	base := syntheticEvent(t, "AAAUSDT", time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC))
	tests := []struct {
		name   string
		mutate func(*RetainedEvent)
	}{
		{"BTC context", func(e *RetainedEvent) { e.BTCContext.SnapshotID = "" }},
		{"ETH context", func(e *RetainedEvent) { e.ETHContext.Symbol = "" }},
		{"warmup", func(e *RetainedEvent) { e.WarmupSufficient = false }},
		{"reference", func(e *RetainedEvent) { e.ReferencePrice = 0 }},
		{"event timestamp", func(e *RetainedEvent) { e.EventTimestamp = time.Time{} }},
		{"source hash", func(e *RetainedEvent) { e.SourceInputHash = "bad" }},
		{"schema", func(e *RetainedEvent) { e.SchemaVersion = RetainedEventSchemaVersion + ".mutated" }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			event := base
			test.mutate(&event)
			if err := ValidateRetainedEvent(event); err == nil {
				t.Fatal("mutation passed validation")
			}
		})
	}
}

func TestDuplicateEventsCannotInflateRawOrIndependentCount(t *testing.T) {
	baseTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	event := syntheticEvent(t, "AAAUSDT", baseTime)
	unique, err := DeduplicateRetainedEvents([]RetainedEvent{event, event})
	if err != nil {
		t.Fatal(err)
	}
	if len(unique) != 1 {
		t.Fatalf("duplicate inflated raw count: %d", len(unique))
	}
	clusters, err := ClusterEvents([]RetainedEvent{event, event}, DefaultIndependencePolicy())
	if err != nil {
		t.Fatal(err)
	}
	if len(clusters) != 1 || len(clusters[0].MemberEventIDs) != 1 {
		t.Fatal("duplicate inflated independent count")
	}
}

func TestConflictingDuplicateEventFails(t *testing.T) {
	base := syntheticEvent(t, "AAAUSDT", time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC))
	mutated := base
	mutated.Features.TrendSlope20 = -2
	mutated, err := SealRetainedEvent(mutated)
	if err != nil {
		t.Fatal(err)
	}
	if mutated.EventID != base.EventID || mutated.ReplayInputHash == base.ReplayInputHash {
		t.Fatal("fixture did not preserve identity and mutate replay input")
	}
	if _, err := DeduplicateRetainedEvents([]RetainedEvent{base, mutated}); err == nil {
		t.Fatal("conflicting duplicate passed")
	}
}

func TestClusteringPolicySyntheticCasesAndOrdering(t *testing.T) {
	base := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	a := syntheticEvent(t, "AAAUSDT", base)
	b := syntheticEvent(t, "AAAUSDT", base.Add(2*time.Hour))
	c := syntheticEvent(t, "BBBUSDT", base.Add(3*time.Hour))
	d := syntheticEvent(t, "AAAUSDT", base.Add(7*time.Hour))
	e := syntheticEvent(t, "AAAUSDT", base.Add(11*time.Hour))
	policy := DefaultIndependencePolicy()
	forward, err := ClusterEvents([]RetainedEvent{a, b, c, d, e}, policy)
	if err != nil {
		t.Fatal(err)
	}
	reverse, err := ClusterEvents([]RetainedEvent{e, d, c, b, a}, policy)
	if err != nil {
		t.Fatal(err)
	}
	left, _ := json.Marshal(forward)
	right, _ := json.Marshal(reverse)
	if string(left) != string(right) {
		t.Fatal("input ordering changed cluster bytes")
	}
	if len(forward) != 3 {
		t.Fatalf("synthetic independent count = %d, want 3", len(forward))
	}
	if len(forward[0].MemberSymbols) != 2 {
		t.Fatal("cross-symbol common-market overlap was not clustered")
	}
	if len(forward[0].SameSymbolOverlapIdentities) != 2 {
		t.Fatal("same-symbol overlap identities missing")
	}
	if forward[1].Start != d.DecisionTimestamp || forward[2].Start != e.DecisionTimestamp {
		t.Fatal("boundary spacing is nondeterministic")
	}
	if concentration, err := ClusterConcentration(forward); err != nil || concentration <= 0 {
		t.Fatalf("invalid structural concentration %v: %v", concentration, err)
	}
	if hash, err := IndependencePolicyHash(policy); err != nil || !validSHA256(hash) {
		t.Fatalf("invalid policy hash %q: %v", hash, err)
	}
}

func TestClusteringMissingTimestampAndContextFail(t *testing.T) {
	event := syntheticEvent(t, "AAAUSDT", time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC))
	event.DecisionTimestamp = time.Time{}
	if _, err := ClusterEvents([]RetainedEvent{event}, DefaultIndependencePolicy()); err == nil {
		t.Fatal("missing timestamp passed")
	}
	event = syntheticEvent(t, "AAAUSDT", time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC))
	event.ETHContext.SnapshotID = ""
	if _, err := ClusterEvents([]RetainedEvent{event}, DefaultIndependencePolicy()); err == nil {
		t.Fatal("missing context passed")
	}
}

func TestProposedUncertaintySyntheticBehavior(t *testing.T) {
	method := ProposedUncertaintyMethod()
	positive := []ClusterObservation{{"a", 1}, {"b", 2}, {"c", 3}, {"d", 4}, {"e", 5}}
	negative := []ClusterObservation{{"a", -1}, {"b", -2}, {"c", -3}, {"d", -4}, {"e", -5}}
	pos, err := EstimateLowerBound(positive, method)
	if err != nil {
		t.Fatal(err)
	}
	neg, err := EstimateLowerBound(negative, method)
	if err != nil {
		t.Fatal(err)
	}
	if pos.LowerBound <= 0 || neg.LowerBound >= 0 {
		t.Fatalf("unexpected synthetic bounds: positive=%v negative=%v", pos.LowerBound, neg.LowerBound)
	}
	duplicate := append(append([]ClusterObservation{}, positive...), positive[0])
	dup, err := EstimateLowerBound(duplicate, method)
	if err != nil {
		t.Fatal(err)
	}
	if dup != pos {
		t.Fatal("cluster duplication changed interval")
	}
	reordered := []ClusterObservation{positive[4], positive[2], positive[0], positive[3], positive[1]}
	reorderResult, err := EstimateLowerBound(reordered, method)
	if err != nil {
		t.Fatal(err)
	}
	left, _ := json.Marshal(pos)
	right, _ := json.Marshal(reorderResult)
	if string(left) != string(right) {
		t.Fatal("ordering or seed changed deterministic result")
	}
	if pos.MethodStatus != PolicyStatusProposedNotAccepted {
		t.Fatal("method was not explicitly proposed")
	}
}

func TestProposedUncertaintyInvalidObservationsFail(t *testing.T) {
	method := ProposedUncertaintyMethod()
	for _, observations := range [][]ClusterObservation{
		{{"", 1}, {"b", 2}}, {{"a", math.NaN()}, {"b", 2}}, {{"a", 1}, {"a", 2}},
	} {
		if _, err := EstimateLowerBound(observations, method); err == nil {
			t.Fatal("invalid observation passed")
		}
	}
	if hash, err := UncertaintyMethodHash(method); err != nil || !validSHA256(hash) {
		t.Fatalf("invalid method hash %q: %v", hash, err)
	}
}

func TestExposureLedgerAndPartitionRules(t *testing.T) {
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	entry := ExposureEntry{
		SourceReport: "historical.json", SourceCommit: strings.Repeat("a", 40), CandidateID: "candidate",
		ExposedWindow: TimeWindow{base, base.AddDate(1, 0, 0)}, GranularityExposed: []string{"month", "aggregate"},
		MetricsExposed: []string{"historical outcome summary"}, MonthLevelExposed: true, EvidenceHash: "sha256:" + strings.Repeat("b", 64),
	}
	ledger, err := SealExposureLedger(ExposureLedger{CandidateID: "candidate", Entries: []ExposureEntry{entry}})
	if err != nil {
		t.Fatal(err)
	}
	coverageEnd := base.AddDate(3, 0, 0)
	plan := PartitionPlan{
		PolicyVersion: PartitionPolicyVersion, PITCoverage: TimeWindow{base, coverageEnd}, WarmupDuration: 0, EmbargoDuration: 24 * time.Hour,
		Development: TimeWindow{base, base.AddDate(1, 0, 0)}, Validation: TimeWindow{base.AddDate(1, 0, 0).Add(24 * time.Hour), base.AddDate(2, 0, 0)},
		FinalHoldout: TimeWindow{base.AddDate(2, 0, 0).Add(24 * time.Hour), coverageEnd}, MinimumIndependentDecisionGate: 300,
	}
	if err := ValidatePartitionPlan(plan, ledger); err != nil {
		t.Fatal(err)
	}
	overlap := plan
	overlap.Validation.Start = base.AddDate(0, 6, 0)
	if err := ValidatePartitionPlan(overlap, ledger); err == nil {
		t.Fatal("exposed period became validation")
	}
	overlap = plan
	overlap.FinalHoldout.Start = base.AddDate(0, 6, 0)
	if err := ValidatePartitionPlan(overlap, ledger); err == nil {
		t.Fatal("exposed period became holdout")
	}
	notLast := plan
	notLast.FinalHoldout.End = coverageEnd.Add(-time.Hour)
	if err := ValidatePartitionPlan(notLast, ledger); err == nil {
		t.Fatal("non-final holdout passed")
	}
	weakened := plan
	weakened.MinimumIndependentDecisionGate = 1
	if err := ValidatePartitionPlan(weakened, ledger); err == nil {
		t.Fatal("weakened sample gate passed")
	}
}

func syntheticEvent(t *testing.T, symbol string, at time.Time) RetainedEvent {
	t.Helper()
	digest := "sha256:" + strings.Repeat("a", 64)
	event, err := SealRetainedEvent(RetainedEvent{
		CandidateFamily: "DowntrendMidVolRelief", CandidateVersion: "v1", ImplementationHash: digest,
		PrimarySymbol: symbol, EventTimestamp: at, DecisionTimestamp: at.Add(time.Minute),
		SourcePartitionID: "partition/" + symbol + "/" + at.Format("2006-01"), SourceSnapshotID: "snapshot/" + symbol, SourceInputHash: digest,
		FeatureSchemaVersion: "synthetic.features.v1", TrendState: "DOWN", PrimaryRegime: "DOWNTREND", VolatilityBucket: "MID",
		Features:       DecisionFeatures{Close: 100, EMA50: 99, EMA200: 101, TrendSlope20: -1, RealizedVol60: 0.003},
		BTCContext:     ContextInput{Symbol: "BTCUSDT", SnapshotID: "snapshot/BTC", SourceInputHash: digest, AvailableAt: at, Return60: 0.01},
		ETHContext:     ContextInput{Symbol: "ETHUSDT", SnapshotID: "snapshot/ETH", SourceInputHash: digest, AvailableAt: at, Return60: 0.02},
		ReferencePrice: 100, EvaluationHorizon: "240m", EvaluationHorizonMS: int64(4 * time.Hour / time.Millisecond), WarmupSufficient: true,
		CostInputs:  CostInputs{FeeBPS: 1, SpreadBPS: 1, SlippageBPS: 1, FundingBPS: 1, AdverseSelectionBPS: 1},
		Attribution: EventAttribution{Month: at.Format("2006-01"), Quarter: at.Format("2006") + "-Q" + string(rune('0'+((int(at.Month())-1)/3+1))), Regime: "DOWNTREND"},
	})
	if err != nil {
		t.Fatal(err)
	}
	return event
}

func assertDescriptorFields(t *testing.T, descriptor SchemaDescriptor, required []string) {
	t.Helper()
	available := make(map[string]struct{}, len(descriptor.Fields))
	for _, field := range descriptor.Fields {
		if _, duplicate := available[field]; duplicate {
			t.Fatalf("duplicate descriptor field %s", field)
		}
		available[field] = struct{}{}
	}
	for _, field := range required {
		if _, ok := available[field]; !ok {
			t.Fatalf("authoritative field %s is absent from %s", field, descriptor.Version)
		}
	}
}
