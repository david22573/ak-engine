package qualificationrunner

import (
	"reflect"
	"testing"
	"time"
)

func TestAcceptedV00UniverseAndConfigurationIdentities(t *testing.T) {
	universe, err := V00UniverseContract()
	if err != nil {
		t.Fatal(err)
	}
	wantDataset := []string{"ADAUSDT", "AVAXUSDT", "BNBUSDT", "BTCUSDT", "DOGEUSDT", "ETHUSDT", "LINKUSDT", "SOLUSDT", "XRPUSDT"}
	wantTargets := []string{"ADAUSDT", "AVAXUSDT", "BNBUSDT", "DOGEUSDT", "ETHUSDT", "LINKUSDT", "SOLUSDT", "XRPUSDT"}
	if !reflect.DeepEqual(universe.DatasetRequiredSymbols, wantDataset) || !reflect.DeepEqual(universe.CandidateTargetSymbols, wantTargets) || !reflect.DeepEqual(universe.ContextOnlySymbols, []string{"BTCUSDT"}) {
		t.Fatalf("accepted V00 universe changed: %#v", universe)
	}
	configuration := V00Configuration()
	if !reflect.DeepEqual(configuration.Symbols, wantTargets) {
		t.Fatal("V00 target defaults changed")
	}
	hash, err := CanonicalConfigurationHash(configuration)
	if err != nil || hash != "sha256:9a3b4d2797daedac643491b8b420b033d05ca46bf051f9fa42b656eb29ede4de" || V00SourceSHA256 != "sha256:3c2e20fd5bf615864aebc5be35ce86c15a6ed8f83de33b2f1d33b00dae6fbfa1" {
		t.Fatalf("accepted V00 identity changed: config=%s source=%s err=%v", hash, V00SourceSHA256, err)
	}
}

func TestUniverseMutationAndOutcomeSelectionFailClosed(t *testing.T) {
	base, err := V00UniverseContract()
	if err != nil {
		t.Fatal(err)
	}
	tests := map[string]func(*UniverseContract){
		"duplicate":       func(value *UniverseContract) { value.DatasetRequiredSymbols[1] = value.DatasetRequiredSymbols[0] },
		"overlap":         func(value *UniverseContract) { value.ContextOnlySymbols = []string{"ETHUSDT"} },
		"missing target":  func(value *UniverseContract) { value.CandidateTargetSymbols = value.CandidateTargetSymbols[:7] },
		"missing context": func(value *UniverseContract) { value.ContextOnlySymbols = nil },
		"blacklist":       func(value *UniverseContract) { value.SymbolBlacklists = []string{"DOGEUSDT"} },
		"outcome filter":  func(value *UniverseContract) { value.OutcomeDerivedFilters = []string{"positive_only"} },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			candidate := base
			candidate.DatasetRequiredSymbols = append([]string(nil), base.DatasetRequiredSymbols...)
			candidate.CandidateTargetSymbols = append([]string(nil), base.CandidateTargetSymbols...)
			candidate.ContextOnlySymbols = append([]string(nil), base.ContextOnlySymbols...)
			mutate(&candidate)
			candidate.ContractSHA256 = ""
			if _, err := SealUniverseContract(candidate); err == nil {
				t.Fatal("mutated universe contract sealed")
			}
		})
	}
	unsorted := base
	unsorted.DatasetRequiredSymbols = append([]string(nil), base.DatasetRequiredSymbols...)
	unsorted.DatasetRequiredSymbols[0], unsorted.DatasetRequiredSymbols[1] = unsorted.DatasetRequiredSymbols[1], unsorted.DatasetRequiredSymbols[0]
	if VerifyUniverseContract(unsorted) == nil {
		t.Fatal("noncanonical per-partition universe ordering verified")
	}
}

func TestVariantsCannotTuneOrDemoteTargetSymbols(t *testing.T) {
	baseline := V00Configuration()
	mutated := baseline
	mutated.Symbols = append([]string(nil), baseline.Symbols[:7]...)
	if err := validateVariantAgainstV00(RegisteredVariant{ID: "V01", Dimensions: []string{"context-agreement"}, Configuration: mutated}); err == nil {
		t.Fatal("variant-level target removal was accepted")
	}
	mutated = baseline
	mutated.Symbols = append([]string{"BTCUSDT"}, baseline.Symbols...)
	if err := validateVariantAgainstV00(RegisteredVariant{ID: "V01", Dimensions: []string{"context-agreement"}, Configuration: mutated}); err == nil {
		t.Fatal("context-only BTC promotion was accepted")
	}
}

func TestBTCContextIsAvailableButNeverScoredAsTarget(t *testing.T) {
	at := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	row := func(symbol string) InputRow {
		return InputRow{Partition: "DEVELOPMENT", Symbol: symbol, EventTime: at, AvailableAt: at, Close: 100, FutureClose240m: 102, EMA50: 105, EMA200: 110, TrendSlope20: -0.1, RealizedVol60: 0.003, WarmupSufficient: true, BTC: Context{SnapshotID: "btc-context", SourceInputSHA256: testHash('b'), AvailableAt: at, Return60: 0.01}, ETH: Context{SnapshotID: "eth-context", SourceInputSHA256: testHash('e'), AvailableAt: at, Return60: 0.01}}
	}
	verified := VerifiedRequest{Request: ExecutionRequest{Dataset: DatasetBinding{Checkpoint: HashIdentity{ID: "synthetic-checkpoint", SHA256: testHash('c')}}}, Variant: RegisteredVariant{ID: "V00", Configuration: V00Configuration()}}
	events, _, err := executeVariant(verified, PartitionArtifact{ArtifactSHA256: testHash('a'), Rows: []InputRow{row("BTCUSDT"), row("ADAUSDT")}})
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].PrimarySymbol != "ADAUSDT" || events[0].BTCContext.Symbol != "BTCUSDT" || events[0].BTCContext.SnapshotID != "btc-context" {
		t.Fatalf("target/context semantics changed: %#v", events)
	}
}
