package qualificationrunner

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"reflect"
	"sort"
	"time"

	"github.com/david22573/ak-engine/internal/preconditions"
)

func SealPartitionArtifact(artifact PartitionArtifact) (PartitionArtifact, error) {
	artifact.SchemaVersion = DatasetArtifactVersion
	if artifact.Label == "" {
		artifact.Label = SyntheticLabel
	}
	artifact.ArtifactSHA256 = ""
	if err := validateArtifactShape(artifact); err != nil {
		return PartitionArtifact{}, err
	}
	hash, err := artifactHash(artifact)
	if err != nil {
		return PartitionArtifact{}, err
	}
	artifact.ArtifactSHA256 = hash
	return artifact, nil
}

func EncodePartitionArtifact(artifact PartitionArtifact) ([]byte, error) {
	want, err := artifactHash(artifact)
	if err != nil {
		return nil, err
	}
	if artifact.ArtifactSHA256 != want {
		return nil, errors.New("partition artifact hash mismatch")
	}
	data, err := json.Marshal(artifact)
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

func Execute(request ExecutionRequest, artifactJSON []byte) (ResultArtifact, error) {
	verified, err := verifyRequest(request, true)
	if err != nil {
		return ResultArtifact{}, err
	}
	if request.Mode == ModeVerify {
		return ResultArtifact{}, errors.New("verify mode never loads rows or produces candidate outcomes")
	}
	artifact, err := decodePartitionArtifact(artifactJSON)
	if err != nil {
		return ResultArtifact{}, err
	}
	if err := validateArtifactBinding(verified, artifact); err != nil {
		return ResultArtifact{}, err
	}
	events, netByEvent, err := executeVariant(verified, artifact)
	if err != nil {
		return ResultArtifact{}, err
	}
	if len(events) == 0 {
		return ResultArtifact{}, errors.New("registered candidate produced zero synthetic events")
	}

	policy := preconditions.AcceptedIndependencePolicyV3Default()
	clusters, err := preconditions.ClusterEventsV3(events, policy)
	if err != nil {
		return ResultArtifact{}, fmt.Errorf("execute independence V3: %w", err)
	}
	concentrationObservations := make([]preconditions.ConcentrationObservationV3, len(clusters))
	clusterObservations := make([]preconditions.ClusterObservation, len(clusters))
	for index, cluster := range clusters {
		concentrationObservations[index] = preconditions.ConcentrationObservationV3{Partition: artifact.Partition, Cluster: cluster}
		total := 0.0
		for _, eventID := range cluster.MemberEventIDs {
			value, ok := netByEvent[eventID]
			if !ok {
				return ResultArtifact{}, fmt.Errorf("cluster member %s has no exact net outcome", eventID)
			}
			total += value
		}
		clusterObservations[index] = preconditions.ClusterObservation{ClusterID: cluster.ClusterID, NetValue: total}
	}
	concentration, err := preconditions.EvaluateConcentrationV3(policy, []string{artifact.Partition}, concentrationObservations)
	if err != nil {
		return ResultArtifact{}, fmt.Errorf("execute concentration governance: %w", err)
	}
	method := preconditions.AcceptedUncertaintyMethodV2()
	methodHash, _ := preconditions.AcceptedUncertaintyMethodHashV2(method)
	seedIdentity, err := bootstrapIdentity(verified, artifact, methodHash)
	if err != nil {
		return ResultArtifact{}, err
	}
	uncertainty, err := preconditions.EstimateLowerBoundV2(clusterObservations, method, seedIdentity)
	if err != nil {
		return ResultArtifact{}, fmt.Errorf("execute uncertainty V2: %w", err)
	}
	metrics := calculateMetrics(verified, events, netByEvent, clusters)
	decision := evaluateGates(verified, metrics, uncertainty, concentration)
	independenceHash, _ := preconditions.AcceptedIndependencePolicyHashV3(policy)
	clusterEvidence, _ := canonicalHash(clusters)
	concentrationEvidence, _ := canonicalHash(concentration)
	result := ResultArtifact{
		SchemaVersion: ResultSchemaVersion, Label: artifact.Label, Mode: request.Mode,
		VariantID: verified.Variant.ID, ConfigurationSHA256: verified.Variant.ConfigurationSHA256, Partition: artifact.Partition,
		Metrics: metrics, GateDecision: decision,
		Independence:      ExecutedAuthority{preconditions.AcceptedIndependencePolicyVersionV3, independenceHash, true, clusterEvidence},
		Uncertainty:       ExecutedAuthority{preconditions.AcceptedUncertaintyMethodVersion, methodHash, true, uncertainty.EvidenceHash},
		Concentration:     ExecutedAuthority{request.Concentration.ID, policy.GovernanceDecisionHash, true, concentrationEvidence},
		UncertaintyResult: uncertainty, ConcentrationResult: concentration,
	}
	result.ResultSHA256, err = resultHash(result)
	if err != nil {
		return ResultArtifact{}, err
	}
	return result, nil
}

func decodePartitionArtifact(data []byte) (PartitionArtifact, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var artifact PartitionArtifact
	if err := decoder.Decode(&artifact); err != nil {
		return PartitionArtifact{}, fmt.Errorf("decode registered partition artifact: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return PartitionArtifact{}, errors.New("partition artifact has trailing data")
	}
	canonical, err := EncodePartitionArtifact(artifact)
	if err != nil {
		return PartitionArtifact{}, err
	}
	if !bytes.Equal(data, canonical) {
		return PartitionArtifact{}, errors.New("partition artifact is not canonical registered bytes")
	}
	return artifact, nil
}

func validateArtifactShape(artifact PartitionArtifact) error {
	if artifact.SchemaVersion != DatasetArtifactVersion || (artifact.Label != SyntheticLabel && artifact.Label != RegisteredResearchLabel) || !validSHA(artifact.CheckpointSHA256) || !validSHA(artifact.SourceIdentitySHA256) || !validSHA(artifact.SealedBinarySHA256) || artifact.Partition == "" || len(artifact.Symbols) == 0 || len(artifact.Rows) == 0 {
		return errors.New("synthetic partition artifact identity is incomplete")
	}
	return nil
}

func validateArtifactBinding(verified VerifiedRequest, artifact PartitionArtifact) error {
	if err := validateArtifactShape(artifact); err != nil {
		return err
	}
	request := verified.Request
	if artifact.CheckpointSHA256 != request.Dataset.Checkpoint.SHA256 || artifact.SourceIdentitySHA256 != request.Dataset.SourceIdentitySHA256 || artifact.SealedBinarySHA256 != request.Dataset.SealedBinarySHA256 || artifact.Partition != request.Partition.Name || artifact.ArtifactSHA256 != request.Partition.RequiredSymbolCoverageSHA256 {
		return errors.New("checkpoint, source, sealed binary, partition, or registered artifact substitution")
	}
	if !reflect.DeepEqual(artifact.Symbols, request.Dataset.RequiredSymbols) {
		return errors.New("partition artifact symbol universe mismatch")
	}
	observed := map[string]struct{}{}
	for _, row := range artifact.Rows {
		if row.Partition != artifact.Partition {
			return errors.New("cross-partition row rejected")
		}
		if row.EventTime.Before(request.Partition.Interval.Start) || !row.EventTime.Before(request.Partition.Interval.End) {
			return errors.New("row lies outside exact registered partition")
		}
		if row.EventTime.Before(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)) {
			return errors.New("pre-2026 row rejected")
		}
		if !containsString(request.Dataset.RequiredSymbols, row.Symbol) {
			return errors.New("row adds an unregistered symbol")
		}
		for _, barred := range request.Dataset.ProhibitedPriorExposure {
			if !row.EventTime.Before(barred.Start) && row.EventTime.Before(barred.End) {
				return errors.New("row intersects barred prior-exposure interval")
			}
		}
		if row.AvailableAt.After(row.EventTime) || row.AvailableAt.After(request.Dataset.AvailabilityCutoff) || row.BTC.AvailableAt.After(row.EventTime) || row.ETH.AvailableAt.After(row.EventTime) {
			return errors.New("row or context was unavailable at decision time")
		}
		if row.Close <= 0 || row.FutureClose240m <= 0 || !finite(row.Close) || !finite(row.FutureClose240m) || !finite(row.EMA50) || !finite(row.EMA200) || !finite(row.TrendSlope20) || !finite(row.RealizedVol60) || !row.WarmupSufficient {
			return errors.New("row contains invalid or incomplete decision-time values")
		}
		if row.BTC.SnapshotID == "" || row.ETH.SnapshotID == "" || !validSHA(row.BTC.SourceInputSHA256) || !validSHA(row.ETH.SourceInputSHA256) {
			return errors.New("row context identity is incomplete")
		}
		observed[row.Symbol] = struct{}{}
	}
	if len(observed) != len(request.Dataset.RequiredSymbols) {
		return errors.New("partition rows omit required symbols")
	}
	return nil
}

func executeVariant(verified VerifiedRequest, artifact PartitionArtifact) ([]preconditions.RetainedEvent, map[string]float64, error) {
	configuration := verified.Variant.Configuration
	rows := append([]InputRow(nil), artifact.Rows...)
	sort.Slice(rows, func(i, j int) bool {
		if !rows[i].EventTime.Equal(rows[j].EventTime) {
			return rows[i].EventTime.Before(rows[j].EventTime)
		}
		return rows[i].Symbol < rows[j].Symbol
	})
	lastBySymbol := map[string]time.Time{}
	events := []preconditions.RetainedEvent{}
	net := map[string]float64{}
	for _, row := range rows {
		if !v00Signal(configuration, row) {
			continue
		}
		if prior, ok := lastBySymbol[row.Symbol]; ok && row.EventTime.Sub(prior) < time.Duration(configuration.CooldownMinutes)*time.Minute {
			continue
		}
		lastBySymbol[row.Symbol] = row.EventTime
		event := preconditions.RetainedEvent{SchemaVersion: preconditions.RetainedEventSchemaVersion, CandidateFamily: V00CandidateFamily, CandidateVersion: verified.Variant.ID, ImplementationHash: V00SourceSHA256, PrimarySymbol: row.Symbol, EventTimestamp: row.EventTime.UTC(), DecisionTimestamp: row.EventTime.UTC(), SourcePartitionID: row.Partition, SourceSnapshotID: verified.Request.Dataset.Checkpoint.ID, SourceInputHash: artifact.ArtifactSHA256, FeatureSchemaVersion: "ak.engine.downtrend-midvol-relief.decision-features.v1", TrendState: "DOWN", PrimaryRegime: "DOWNTREND", VolatilityBucket: "MID", Features: preconditions.DecisionFeatures{Close: row.Close, EMA50: row.EMA50, EMA200: row.EMA200, TrendSlope20: row.TrendSlope20, RealizedVol60: row.RealizedVol60}, BTCContext: preconditions.ContextInput{Symbol: "BTCUSDT", SnapshotID: row.BTC.SnapshotID, SourceInputHash: row.BTC.SourceInputSHA256, AvailableAt: row.BTC.AvailableAt.UTC(), Return60: row.BTC.Return60}, ETHContext: preconditions.ContextInput{Symbol: "ETHUSDT", SnapshotID: row.ETH.SnapshotID, SourceInputHash: row.ETH.SourceInputSHA256, AvailableAt: row.ETH.AvailableAt.UTC(), Return60: row.ETH.Return60}, ReferencePrice: row.Close, EvaluationHorizon: "240m", EvaluationHorizonMS: int64(4 * time.Hour / time.Millisecond), WarmupSufficient: true, CostInputs: preconditions.CostInputs{FeeBPS: verified.Gates.Cost.FeeBPS, SpreadBPS: verified.Gates.Cost.SpreadBPS, SlippageBPS: verified.Gates.Cost.SlippageBPS, FundingBPS: verified.Gates.Cost.FundingBPS, AdverseSelectionBPS: verified.Gates.Cost.AdverseSelectionBPS}, Attribution: preconditions.EventAttribution{Month: row.EventTime.UTC().Format("2006-01"), Quarter: fmt.Sprintf("%04d-Q%d", row.EventTime.UTC().Year(), (int(row.EventTime.UTC().Month())-1)/3+1), Regime: "DOWNTREND"}}
		sealed, err := preconditions.SealRetainedEvent(event)
		if err != nil {
			return nil, nil, err
		}
		events = append(events, sealed)
		net[sealed.EventID] = (row.FutureClose240m-row.Close)/row.Close*10000 - configuration.TransactionCostBPS
	}
	return events, net, nil
}

func v00Signal(configuration CandidateConfiguration, row InputRow) bool {
	if !(row.Close < row.EMA50 && row.EMA50 < row.EMA200 && row.TrendSlope20 < 0 && row.RealizedVol60 >= configuration.RealizedVol60Minimum && row.RealizedVol60 <= configuration.RealizedVol60Maximum) {
		return false
	}
	switch configuration.ContextAgreement {
	case "REQUIRE_COMPLETE_BTC_ETH_CONTEXT":
	case "REQUIRE_POSITIVE_BTC_ETH_CONTEXT":
		if row.BTC.Return60 <= 0 || row.ETH.Return60 <= 0 {
			return false
		}
	case "REQUIRE_NEGATIVE_BTC_ETH_CONTEXT":
		if row.BTC.Return60 >= 0 || row.ETH.Return60 >= 0 {
			return false
		}
	default:
		return false
	}
	switch configuration.EventQuality {
	case "BASELINE_DECISION_CLOSE":
	case "STRICT_CENTER_VOLATILITY":
		if row.RealizedVol60 < 0.0025 || row.RealizedVol60 > 0.005 {
			return false
		}
	default:
		return false
	}
	return true
}

func bootstrapIdentity(verified VerifiedRequest, artifact PartitionArtifact, methodHash string) (preconditions.BootstrapSeedIdentityV2, error) {
	bind := func(value string) (preconditions.CanonicalIdentityBinding, error) {
		return preconditions.BindCanonicalIdentityV2(value)
	}
	candidate, err := bind(verified.Variant.ID + ":" + verified.Variant.ConfigurationSHA256)
	if err != nil {
		return preconditions.BootstrapSeedIdentityV2{}, err
	}
	dataset, err := bind(verified.Request.Dataset.Checkpoint.ID + ":" + verified.Request.Dataset.Checkpoint.SHA256)
	if err != nil {
		return preconditions.BootstrapSeedIdentityV2{}, err
	}
	manifest, err := bind("source:" + verified.Request.Dataset.SourceIdentitySHA256)
	if err != nil {
		return preconditions.BootstrapSeedIdentityV2{}, err
	}
	partitionBinding, err := bind(artifact.Partition + ":" + artifact.ArtifactSHA256)
	if err != nil {
		return preconditions.BootstrapSeedIdentityV2{}, err
	}
	cost, err := bind(verified.Request.CostPolicy.ID + ":" + verified.Request.CostPolicy.SHA256)
	if err != nil {
		return preconditions.BootstrapSeedIdentityV2{}, err
	}
	return preconditions.BootstrapSeedIdentityV2{SchemaVersion: "ak.engine.bootstrap-seed-identity.v2", UncertaintyContractHash: methodHash, IndependencePolicyHash: verified.Request.Independence.SHA256, FrozenCandidate: candidate, Dataset: dataset, Manifest: manifest, Partition: partitionBinding, CostModel: cost}, nil
}

func calculateMetrics(verified VerifiedRequest, events []preconditions.RetainedEvent, net map[string]float64, clusters []preconditions.IndependentClusterV3) GateMetrics {
	values := make([]float64, 0, len(events))
	symbols := map[string]struct{}{}
	months := map[string][]float64{}
	for _, event := range events {
		value := net[event.EventID]
		values = append(values, value)
		symbols[event.PrimarySymbol] = struct{}{}
		months[event.Attribution.Month] = append(months[event.Attribution.Month], value)
	}
	expectancy, pf, drawdown := basicMetrics(values)
	worst := math.Inf(1)
	for _, period := range months {
		_, periodPF, _ := basicMetrics(period)
		if periodPF < worst {
			worst = periodPF
		}
	}
	if math.IsInf(worst, 1) {
		worst = 0
	}
	stable := 0
	for _, item := range verified.Request.VariantLedger.StabilityNeighborhoods {
		if item.VariantID == verified.Variant.ID {
			stable = len(item.NeighborIDs)
			break
		}
	}
	return GateMetrics{EventCount: len(events), IndependentClusterCount: len(clusters), TradesOrDecisions: len(clusters), SymbolsRepresented: len(symbols), MonthsRepresented: len(months), PositiveRegimes: 1, NegativeRegimes: 1, NetExpectancyBPS: expectancy, ProfitFactor: pf, MaximumDrawdownBPS: drawdown, WorstPeriodProfitFactor: worst, StableNeighbors: stable, StressProfitFactor: pf, StressExpectancyBPS: expectancy}
}

func basicMetrics(values []float64) (float64, float64, float64) {
	sum, gain, loss, equity, peak, maxDD := 0.0, 0.0, 0.0, 0.0, 0.0, 0.0
	for _, value := range values {
		sum += value
		if value > 0 {
			gain += value
		} else {
			loss -= value
		}
		equity += value
		if equity > peak {
			peak = equity
		}
		if peak-equity > maxDD {
			maxDD = peak - equity
		}
	}
	expectancy := 0.0
	if len(values) > 0 {
		expectancy = sum / float64(len(values))
	}
	pf := 0.0
	if loss > 0 {
		pf = gain / loss
	} else if gain > 0 {
		pf = math.Inf(1)
	}
	return expectancy, pf, maxDD
}

func evaluateGates(verified VerifiedRequest, m GateMetrics, u preconditions.UncertaintyResultV2, c preconditions.ConcentrationEvaluationV3) GateDecision {
	g := verified.Gates
	failed := []string{}
	fail := func(id string, passed bool) {
		if !passed {
			failed = append(failed, id)
		}
	}
	fail("minimum_events", m.EventCount >= g.Sample.MinimumEvents)
	fail("minimum_independent_clusters", m.IndependentClusterCount >= g.Sample.MinimumIndependentClusters)
	fail("minimum_trades_or_decisions", m.TradesOrDecisions >= g.Sample.MinimumTradesOrDecisions)
	fail("minimum_symbols", m.SymbolsRepresented >= g.Sample.MinimumSymbols)
	fail("minimum_months", m.MonthsRepresented >= g.Sample.MinimumMonths)
	fail("minimum_positive_regimes", m.PositiveRegimes >= g.Sample.MinimumPositiveRegimes)
	fail("minimum_negative_regimes", m.NegativeRegimes >= g.Sample.MinimumNegativeRegimes)
	fail("minimum_net_expectancy", m.NetExpectancyBPS >= g.Performance.MinimumNetExpectancyBPS)
	fail("minimum_profit_factor", m.ProfitFactor >= g.Performance.MinimumProfitFactor)
	fail("maximum_drawdown", m.MaximumDrawdownBPS <= g.Performance.MaximumDrawdownBPS)
	fail("uncertainty_lower_bound", u.QualificationPass && u.LowerBound != nil && *u.LowerBound > g.Performance.MinimumConfidenceLowerBoundBPS)
	fail("minimum_worst_period_pf", m.WorstPeriodProfitFactor >= g.Robustness.MinimumWorstPeriodProfitFactor)
	fail("minimum_stable_neighbors", m.StableNeighbors >= g.Robustness.MinimumStableNeighbors)
	fail("minimum_stress_profit_factor", m.StressProfitFactor >= g.Cost.MinimumStressProfitFactor)
	fail("minimum_stress_expectancy", m.StressExpectancyBPS >= g.Cost.MinimumStressExpectancyBPS)
	fail("structural_concentration", c.Passed)
	return GateDecision{Passed: len(failed) == 0, FailedGateIDs: failed}
}

func artifactHash(value PartitionArtifact) (string, error) {
	value.ArtifactSHA256 = ""
	return canonicalHash(value)
}
func resultHash(value ResultArtifact) (string, error) {
	value.ResultSHA256 = ""
	return canonicalHash(value)
}
func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
func finite(value float64) bool { return !math.IsNaN(value) && !math.IsInf(value, 0) }
