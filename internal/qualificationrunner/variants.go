package qualificationrunner

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
)

var acceptedDatasetSymbols = []string{"ADAUSDT", "AVAXUSDT", "BNBUSDT", "BTCUSDT", "DOGEUSDT", "ETHUSDT", "LINKUSDT", "SOLUSDT", "XRPUSDT"}
var acceptedTargetSymbols = []string{"ADAUSDT", "AVAXUSDT", "BNBUSDT", "DOGEUSDT", "ETHUSDT", "LINKUSDT", "SOLUSDT", "XRPUSDT"}
var acceptedContextOnlySymbols = []string{"BTCUSDT"}

func V00Configuration() CandidateConfiguration {
	return CandidateConfiguration{
		SchemaVersion: ConfigurationVersion, CandidateFamily: V00CandidateFamily,
		Side: "LONG", Horizon: "240m", TrendState: "DOWN",
		RealizedVol60Minimum: 0.0015, RealizedVol60Maximum: 0.006,
		ContextAgreement: "REQUIRE_COMPLETE_BTC_ETH_CONTEXT",
		EventQuality:     "BASELINE_DECISION_CLOSE",
		CooldownMinutes:  0,
		Symbols:          append([]string(nil), acceptedTargetSymbols...), DateExclusions: []string{}, QuarterExclusions: []string{},
		TransactionCostBPS: 10, SizingPolicy: "ONE_EQUAL_UNIT_PER_ACCEPTED_EVENT",
		OutcomeDerivedFilters: []string{},
		Indicators:            []string{"EMA50", "EMA200", "TrendSlope20", "RealizedVol60"},
		Features:              []string{"close", "ema_50", "ema_200", "trend_slope_20", "realized_vol_60", "btc_context", "eth_context"},
	}
}

func V00UniverseContract() (UniverseContract, error) {
	contract := UniverseContract{
		SchemaVersion:          UniverseContractVersion,
		DatasetRequiredSymbols: append([]string(nil), acceptedDatasetSymbols...),
		CandidateTargetSymbols: append([]string(nil), acceptedTargetSymbols...),
		ContextOnlySymbols:     append([]string(nil), acceptedContextOnlySymbols...),
		SymbolBlacklists:       []string{}, OutcomeDerivedFilters: []string{},
	}
	return SealUniverseContract(contract)
}

func SealUniverseContract(contract UniverseContract) (UniverseContract, error) {
	contract.ContractSHA256 = ""
	contract.DatasetRequiredSymbols = append([]string(nil), contract.DatasetRequiredSymbols...)
	contract.CandidateTargetSymbols = append([]string(nil), contract.CandidateTargetSymbols...)
	contract.ContextOnlySymbols = append([]string(nil), contract.ContextOnlySymbols...)
	contract.SymbolBlacklists = append([]string{}, contract.SymbolBlacklists...)
	contract.OutcomeDerivedFilters = append([]string{}, contract.OutcomeDerivedFilters...)
	for _, values := range [][]string{contract.DatasetRequiredSymbols, contract.CandidateTargetSymbols, contract.ContextOnlySymbols} {
		sort.Strings(values)
	}
	if err := validateUniverseContract(contract); err != nil {
		return UniverseContract{}, err
	}
	hash, err := canonicalHash(contract)
	if err != nil {
		return UniverseContract{}, err
	}
	contract.ContractSHA256 = hash
	return contract, nil
}

func VerifyUniverseContract(contract UniverseContract) error {
	want, err := SealUniverseContract(contract)
	if err != nil {
		return err
	}
	if contract.ContractSHA256 != want.ContractSHA256 || !reflect.DeepEqual(contract, want) {
		return errors.New("universe contract is noncanonical or mutated")
	}
	return nil
}

func validateUniverseContract(contract UniverseContract) error {
	if contract.SchemaVersion != UniverseContractVersion || len(contract.DatasetRequiredSymbols) == 0 || len(contract.CandidateTargetSymbols) == 0 {
		return errors.New("complete universe contract is required")
	}
	if len(contract.SymbolBlacklists) != 0 || len(contract.OutcomeDerivedFilters) != 0 {
		return errors.New("symbol blacklists and outcome-derived symbol filters are prohibited")
	}
	unique := func(name string, values []string) (map[string]struct{}, error) {
		seen := map[string]struct{}{}
		for _, value := range values {
			if value == "" || value != strings.ToUpper(value) {
				return nil, fmt.Errorf("%s contains a noncanonical symbol", name)
			}
			if _, exists := seen[value]; exists {
				return nil, fmt.Errorf("%s contains a duplicate symbol", name)
			}
			seen[value] = struct{}{}
		}
		return seen, nil
	}
	dataset, err := unique("dataset", contract.DatasetRequiredSymbols)
	if err != nil {
		return err
	}
	targets, err := unique("targets", contract.CandidateTargetSymbols)
	if err != nil {
		return err
	}
	contexts, err := unique("context-only", contract.ContextOnlySymbols)
	if err != nil {
		return err
	}
	if len(targets)+len(contexts) != len(dataset) {
		return errors.New("target/context-only union differs from dataset universe")
	}
	for symbol := range targets {
		if _, overlap := contexts[symbol]; overlap {
			return errors.New("target and context-only symbols overlap")
		}
		if _, exists := dataset[symbol]; !exists {
			return errors.New("target symbol is absent from dataset universe")
		}
	}
	for symbol := range contexts {
		if _, exists := dataset[symbol]; !exists {
			return errors.New("context-only symbol is absent from dataset universe")
		}
	}
	return nil
}

func CanonicalConfigurationHash(configuration CandidateConfiguration) (string, error) {
	if err := validateCompleteConfiguration(configuration); err != nil {
		return "", err
	}
	data, err := json.Marshal(configuration)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(digest[:]), nil
}

func ResolveVariantLedger(ledger VariantLedger, identity IdentityVariantLedger) (VariantLedger, error) {
	if ledger.SchemaVersion != VariantLedgerVersion || ledger.MaximumVariants <= 0 || ledger.MaximumVariants > 12 || len(ledger.Variants) == 0 || len(ledger.Variants) > ledger.MaximumVariants || ledger.V00ID != "V00" {
		return VariantLedger{}, errors.New("variant ledger schema, maximum, or V00 identity is invalid")
	}
	if ledger.MaximumVariants != identity.MaximumRegisteredVariants || ledger.V00ID != identity.V00ID || len(ledger.Variants) != len(identity.Variants) {
		return VariantLedger{}, errors.New("Engine variant ledger does not match registered RIF ledger")
	}
	copyLedger := ledger
	copyLedger.Variants = append([]RegisteredVariant(nil), ledger.Variants...)
	sort.Slice(copyLedger.Variants, func(i, j int) bool { return copyLedger.Variants[i].ID < copyLedger.Variants[j].ID })
	seen := map[string]struct{}{}
	identityByID := map[string]IdentityVariant{}
	for _, item := range identity.Variants {
		identityByID[item.ID] = item
	}
	for index := range copyLedger.Variants {
		item := &copyLedger.Variants[index]
		if item.ID == "" {
			return VariantLedger{}, errors.New("variant ID is required")
		}
		if _, duplicate := seen[item.ID]; duplicate {
			return VariantLedger{}, fmt.Errorf("duplicate variant %q", item.ID)
		}
		seen[item.ID] = struct{}{}
		sort.Strings(item.Dimensions)
		if err := validateVariantAgainstV00(*item); err != nil {
			return VariantLedger{}, fmt.Errorf("variant %s: %w", item.ID, err)
		}
		hash, err := CanonicalConfigurationHash(item.Configuration)
		if err != nil {
			return VariantLedger{}, err
		}
		if item.ConfigurationSHA256 != hash {
			return VariantLedger{}, fmt.Errorf("variant %s canonical configuration hash mismatch", item.ID)
		}
		registered, ok := identityByID[item.ID]
		registeredDimensions := append([]string{}, registered.Dimensions...)
		sort.Strings(registeredDimensions)
		if !ok || registered.ConfigurationSHA256 != item.ConfigurationSHA256 || !reflect.DeepEqual(registeredDimensions, item.Dimensions) {
			return VariantLedger{}, fmt.Errorf("variant %s differs from RIF registration", item.ID)
		}
	}
	if _, ok := seen["V00"]; !ok {
		return VariantLedger{}, errors.New("V00 is not registered")
	}
	copyLedger.StabilityNeighborhoods = append([]StabilityNeighborhood(nil), ledger.StabilityNeighborhoods...)
	for index := range copyLedger.StabilityNeighborhoods {
		sort.Strings(copyLedger.StabilityNeighborhoods[index].NeighborIDs)
	}
	sort.Slice(copyLedger.StabilityNeighborhoods, func(i, j int) bool {
		return copyLedger.StabilityNeighborhoods[i].VariantID < copyLedger.StabilityNeighborhoods[j].VariantID
	})
	if !reflect.DeepEqual(copyLedger.StabilityNeighborhoods, identity.StabilityNeighborhoods) {
		return VariantLedger{}, errors.New("stability-neighborhood ledger differs from RIF registration")
	}
	wantHash, err := variantLedgerHash(copyLedger)
	if err != nil {
		return VariantLedger{}, err
	}
	if ledger.LedgerSHA256 != wantHash {
		return VariantLedger{}, errors.New("variant ledger hash mismatch or post-registration mutation")
	}
	copyLedger.LedgerSHA256 = wantHash
	return copyLedger, nil
}

func SealVariantLedger(ledger VariantLedger) (VariantLedger, error) {
	copyLedger := ledger
	copyLedger.LedgerSHA256 = ""
	for index := range copyLedger.Variants {
		hash, err := CanonicalConfigurationHash(copyLedger.Variants[index].Configuration)
		if err != nil {
			return VariantLedger{}, err
		}
		copyLedger.Variants[index].ConfigurationSHA256 = hash
		sort.Strings(copyLedger.Variants[index].Dimensions)
	}
	sort.Slice(copyLedger.Variants, func(i, j int) bool { return copyLedger.Variants[i].ID < copyLedger.Variants[j].ID })
	for index := range copyLedger.StabilityNeighborhoods {
		sort.Strings(copyLedger.StabilityNeighborhoods[index].NeighborIDs)
	}
	sort.Slice(copyLedger.StabilityNeighborhoods, func(i, j int) bool {
		return copyLedger.StabilityNeighborhoods[i].VariantID < copyLedger.StabilityNeighborhoods[j].VariantID
	})
	hash, err := variantLedgerHash(copyLedger)
	if err != nil {
		return VariantLedger{}, err
	}
	copyLedger.LedgerSHA256 = hash
	return copyLedger, nil
}

func variantLedgerHash(ledger VariantLedger) (string, error) {
	ledger.LedgerSHA256 = ""
	return canonicalHash(ledger)
}

func validateCompleteConfiguration(configuration CandidateConfiguration) error {
	if configuration.SchemaVersion != ConfigurationVersion || configuration.CandidateFamily == "" || configuration.Side == "" || configuration.Horizon == "" || configuration.TrendState == "" || configuration.ContextAgreement == "" || configuration.EventQuality == "" || configuration.SizingPolicy == "" || len(configuration.Symbols) == 0 || len(configuration.Indicators) == 0 || len(configuration.Features) == 0 {
		return errors.New("configuration contains absent or unknown defaults")
	}
	if configuration.RealizedVol60Minimum <= 0 || configuration.RealizedVol60Maximum < configuration.RealizedVol60Minimum || configuration.TransactionCostBPS < 0 || configuration.CooldownMinutes < 0 {
		return errors.New("configuration numeric defaults are invalid")
	}
	return nil
}

func validateVariantAgainstV00(variant RegisteredVariant) error {
	baseline := V00Configuration()
	configuration := variant.Configuration
	if variant.ID == "V00" {
		if len(variant.Dimensions) != 0 || !reflect.DeepEqual(configuration, baseline) {
			return errors.New("V00 must resolve to the exact accepted executable baseline")
		}
		return nil
	}
	if configuration.SchemaVersion != baseline.SchemaVersion || configuration.CandidateFamily != baseline.CandidateFamily || configuration.Side != baseline.Side || configuration.Horizon != baseline.Horizon || configuration.TrendState != baseline.TrendState || configuration.RealizedVol60Minimum != baseline.RealizedVol60Minimum || configuration.RealizedVol60Maximum != baseline.RealizedVol60Maximum || !reflect.DeepEqual(configuration.Symbols, baseline.Symbols) || len(configuration.DateExclusions) != 0 || len(configuration.QuarterExclusions) != 0 || configuration.TransactionCostBPS != baseline.TransactionCostBPS || configuration.SizingPolicy != baseline.SizingPolicy || len(configuration.OutcomeDerivedFilters) != 0 || !reflect.DeepEqual(configuration.Indicators, baseline.Indicators) || !reflect.DeepEqual(configuration.Features, baseline.Features) {
		return errors.New("variant changes a prohibited candidate-semantic, symbol, partition, exclusion, cost, sizing, filter, indicator, or feature dimension")
	}
	changed := []string{}
	if configuration.ContextAgreement != baseline.ContextAgreement {
		changed = append(changed, "context-agreement")
	}
	if configuration.EventQuality != baseline.EventQuality {
		changed = append(changed, "event-quality")
	}
	if configuration.CooldownMinutes != baseline.CooldownMinutes {
		changed = append(changed, "cooldown/independence")
	}
	sort.Strings(changed)
	declared := append([]string(nil), variant.Dimensions...)
	sort.Strings(declared)
	if len(changed) == 0 || !reflect.DeepEqual(changed, declared) {
		return errors.New("declared permitted dimensions do not match canonical configuration changes")
	}
	for _, dimension := range declared {
		if dimension != "context-agreement" && dimension != "event-quality" && dimension != "cooldown/independence" {
			return fmt.Errorf("unsupported dimension %q", dimension)
		}
	}
	return validateCompleteConfiguration(configuration)
}

func canonicalHash(value any) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(digest[:]), nil
}
