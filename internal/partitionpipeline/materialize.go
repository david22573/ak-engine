package partitionpipeline

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"time"

	"github.com/david22573/ak-engine/internal/features"
	"github.com/david22573/ak-engine/internal/qualificationrunner"
	"github.com/david22573/ak-engine/pkg/protocol"
)

type normalizedRecord struct {
	Market                     string    `json:"market"`
	Symbol                     string    `json:"symbol"`
	Interval                   string    `json:"interval"`
	Period                     string    `json:"period"`
	SourceDate                 string    `json:"source_date"`
	OpenTimeMS                 int64     `json:"open_time_ms"`
	Open                       string    `json:"open"`
	High                       string    `json:"high"`
	Low                        string    `json:"low"`
	Close                      string    `json:"close"`
	Volume                     string    `json:"volume"`
	CloseTimeMS                int64     `json:"close_time_ms"`
	QuoteAssetVolume           string    `json:"quote_asset_volume"`
	NumberOfTrades             int64     `json:"number_of_trades"`
	TakerBuyBaseVolume         string    `json:"taker_buy_base_volume"`
	TakerBuyQuoteVolume        string    `json:"taker_buy_quote_volume"`
	MarketEventTimeUTC         time.Time `json:"market_event_time_utc"`
	ProviderCandleCloseTimeUTC time.Time `json:"provider_candle_close_time_utc"`
	ObservedAvailableAtUTC     time.Time `json:"observed_available_at_utc"`
	AcquiredAtUTC              time.Time `json:"acquired_at_utc"`
	AcquisitionReceiptID       string    `json:"acquisition_receipt_id"`
}

type sourceFragment struct {
	SchemaVersion           string             `json:"schema_version"`
	RequestID               string             `json:"request_id"`
	Symbol                  string             `json:"symbol"`
	SourceSchemaVersion     string             `json:"source_schema_version"`
	SourceSchemaFingerprint string             `json:"source_schema_fingerprint"`
	Records                 []normalizedRecord `json:"records"`
	FragmentHash            string             `json:"fragment_hash"`
}

type prospectiveRecord struct {
	Market              string `json:"market"`
	Symbol              string `json:"symbol"`
	Interval            string `json:"interval"`
	Period              string `json:"period"`
	SourceDate          string `json:"source_date"`
	OpenTimeMS          int64  `json:"open_time_ms"`
	Open                string `json:"open"`
	High                string `json:"high"`
	Low                 string `json:"low"`
	Close               string `json:"close"`
	Volume              string `json:"volume"`
	CloseTimeMS         int64  `json:"close_time_ms"`
	QuoteAssetVolume    string `json:"quote_asset_volume"`
	NumberOfTrades      int64  `json:"number_of_trades"`
	TakerBuyBaseVolume  string `json:"taker_buy_base_volume"`
	TakerBuyQuoteVolume string `json:"taker_buy_quote_volume"`
}

type prospectiveFragment struct {
	SchemaVersion           string              `json:"schema_version"`
	NormalizationVersion    string              `json:"normalization_version"`
	CycleID                 string              `json:"cycle_id"`
	Symbol                  string              `json:"symbol"`
	SourceSchemaVersion     string              `json:"source_schema_version"`
	SourceSchemaFingerprint string              `json:"source_schema_fingerprint"`
	Records                 []prospectiveRecord `json:"records"`
	FragmentHash            string              `json:"fragment_hash"`
}

func CreateRegistry(root string) error {
	if root == "" || filepath.Clean(root) != root {
		return errUnsafePath
	}
	parent, err := canonicalRealDirectory(filepath.Dir(root))
	if err != nil {
		return err
	}
	if _, err := os.Lstat(root); !errors.Is(err, os.ErrNotExist) {
		return errors.New("registry root must be new")
	}
	if err := os.Mkdir(filepath.Join(parent, filepath.Base(root)), 0o700); err != nil {
		return err
	}
	if err := os.Mkdir(filepath.Join(root, "plans"), 0o700); err != nil {
		return err
	}
	if err := os.Mkdir(filepath.Join(root, "artifacts"), 0o700); err != nil {
		return err
	}
	registry := Registry{SchemaVersion: RegistrySchemaVersion, Entries: map[string]RegistryEntry{}}
	return writeRegistry(root, registry)
}

func RegisterPlan(root string, plan Plan) error {
	if err := VerifyPlan(plan); err != nil {
		return err
	}
	registry, err := readRegistry(root)
	if err != nil {
		return err
	}
	if prior, ok := registry.Entries[plan.PlanSHA256]; ok {
		if prior.State != PlanVerified {
			return errors.New("registered plan already advanced beyond verification")
		}
		return verifyRegisteredPlan(root, plan)
	}
	data, err := EncodePlan(plan)
	if err != nil {
		return err
	}
	path := filepath.Join(root, "plans", hashLeaf(plan.PlanSHA256)+".json")
	if err := atomicWrite(path, data, 0o600); err != nil {
		return err
	}
	registry.Entries[plan.PlanSHA256] = RegistryEntry{PlanSHA256: plan.PlanSHA256, State: PlanCreated}
	if err := writeRegistry(root, registry); err != nil {
		return err
	}
	if err := verifyRegisteredPlan(root, plan); err != nil {
		return err
	}
	entry := registry.Entries[plan.PlanSHA256]
	entry.State = PlanVerified
	registry.Entries[plan.PlanSHA256] = entry
	return writeRegistry(root, registry)
}

func PlanState(root, planSHA string) (LifecycleState, error) {
	registry, err := readRegistry(root)
	if err != nil {
		return "", err
	}
	entry, ok := registry.Entries[planSHA]
	if !ok {
		return "", errors.New("plan is not registered")
	}
	return entry.State, nil
}

func ProveZeroAccess(root, planSHA string) (ZeroAccessProof, error) {
	registry, err := readRegistry(root)
	if err != nil {
		return ZeroAccessProof{}, err
	}
	entry, ok := registry.Entries[planSHA]
	if !ok || (entry.State != PlanVerified && entry.State != MaterializationAuthorized) {
		return ZeroAccessProof{}, errors.New("zero access is not provable after materialization started")
	}
	if entry.ArtifactSHA256 != "" || entry.ArtifactManifestSHA256 != "" || entry.AccessReceiptSHA256 != "" || entry.ConsumptionAuthorization != nil || len(entry.ConsumptionReceipts) != 0 {
		return ZeroAccessProof{}, errors.New("zero access proof conflicts with durable artifact or receipt evidence")
	}
	proof := ZeroAccessProof{SchemaVersion: ZeroAccessProofVersion, PlanSHA256: planSHA, RegistrySHA256: registry.RegistrySHA256, LifecycleState: entry.State}
	proof.ProofSHA256, err = canonicalHash(proof)
	return proof, err
}

func AuthorizeMaterialization(root, planSHA string, authorization MaterializationAuthorization) error {
	registry, err := readRegistry(root)
	if err != nil {
		return err
	}
	entry, ok := registry.Entries[planSHA]
	if !ok || entry.State != PlanVerified {
		return errors.New("only a verified registered plan may be authorized")
	}
	plan, err := loadRegisteredPlan(root, planSHA)
	if err != nil {
		return err
	}
	if authorization.SchemaVersion != AuthorizationSchemaVersion || authorization.PlanSHA256 != planSHA || authorization.CheckpointSHA256 != plan.Checkpoint.SHA256 || authorization.Partition != plan.PartitionName || authorization.RIFAuthorizationID == "" || !validSHA(authorization.RIFAuthorizationSHA256) || !validSHA(authorization.RIFAccessReceiptSHA256) || authorization.AuthorizedAt.IsZero() {
		return errors.New("materialization lacks exact durable RIF authorization")
	}
	want := authorization
	want.AuthorizationSHA256 = ""
	hash, err := canonicalHash(want)
	if err != nil || authorization.AuthorizationSHA256 != hash {
		return errors.New("materialization authorization hash mismatch")
	}
	entry.State, entry.Authorization = MaterializationAuthorized, &authorization
	registry.Entries[planSHA] = entry
	return writeRegistry(root, registry)
}

func Materialize(root, planSHA string, now time.Time) (qualificationrunner.PartitionArtifact, ArtifactManifest, AccessReceipt, error) {
	registry, err := readRegistry(root)
	if err != nil {
		return qualificationrunner.PartitionArtifact{}, ArtifactManifest{}, AccessReceipt{}, err
	}
	entry, ok := registry.Entries[planSHA]
	if !ok || entry.State != MaterializationAuthorized || entry.Authorization == nil {
		return qualificationrunner.PartitionArtifact{}, ArtifactManifest{}, AccessReceipt{}, errors.New("materialization is not authorized")
	}
	plan, err := loadRegisteredPlan(root, planSHA)
	if err != nil {
		return qualificationrunner.PartitionArtifact{}, ArtifactManifest{}, AccessReceipt{}, err
	}
	entry.State = MaterializationStarted
	registry.Entries[planSHA] = entry
	if err := writeRegistry(root, registry); err != nil {
		return qualificationrunner.PartitionArtifact{}, ArtifactManifest{}, AccessReceipt{}, err
	}
	candles := map[string][]protocol.Candle{}
	openedRows, fragmentCount := 0, 0
	for _, member := range plan.SourceManifests {
		manifestPath, err := secureJoin(plan.SourceRoot, member.RelativePath, true)
		if err != nil {
			return qualificationrunner.PartitionArtifact{}, ArtifactManifest{}, AccessReceipt{}, err
		}
		data, err := os.ReadFile(manifestPath)
		if err != nil || byteHash(data) != member.FileSHA256 {
			return qualificationrunner.PartitionArtifact{}, ArtifactManifest{}, AccessReceipt{}, errors.New("planned manifest changed")
		}
		for _, fragmentRef := range member.FragmentArtifacts {
			fragment, err := readFragment(plan, fragmentRef)
			if err != nil {
				return qualificationrunner.PartitionArtifact{}, ArtifactManifest{}, AccessReceipt{}, err
			}
			if fragment.Symbol != member.Symbol {
				return qualificationrunner.PartitionArtifact{}, ArtifactManifest{}, AccessReceipt{}, errors.New("source fragment symbol differs from plan")
			}
			fragmentCount++
			for _, row := range fragment.Records {
				candle, err := toCandle(row)
				if err != nil {
					return qualificationrunner.PartitionArtifact{}, ArtifactManifest{}, AccessReceipt{}, err
				}
				opened := time.UnixMilli(candle.OpenTimeMS).UTC()
				if opened.Before(plan.PartitionInterval.Start) || !opened.Before(plan.PartitionInterval.End) || opened.Before(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)) || candle.Symbol != member.Symbol || !contains(plan.DatasetRequiredSymbols, candle.Symbol) {
					return qualificationrunner.PartitionArtifact{}, ArtifactManifest{}, AccessReceipt{}, errors.New("cross-partition, barred, or wrong-symbol source row rejected")
				}
				candles[candle.Symbol] = append(candles[candle.Symbol], candle)
				openedRows++
			}
		}
	}
	for _, symbol := range plan.DatasetRequiredSymbols {
		if len(candles[symbol]) == 0 {
			return qualificationrunner.PartitionArtifact{}, ArtifactManifest{}, AccessReceipt{}, fmt.Errorf("planned symbol %s has no rows", symbol)
		}
		sort.Slice(candles[symbol], func(i, j int) bool { return candles[symbol][i].OpenTimeMS < candles[symbol][j].OpenTimeMS })
	}
	rows, err := buildInputRows(plan, candles)
	if err != nil {
		return qualificationrunner.PartitionArtifact{}, ArtifactManifest{}, AccessReceipt{}, err
	}
	label := qualificationrunner.RegisteredResearchLabel
	if plan.SyntheticFixture {
		label = qualificationrunner.SyntheticLabel
	}
	artifact := qualificationrunner.PartitionArtifact{Label: label, CheckpointSHA256: plan.Checkpoint.SHA256, SourceIdentitySHA256: plan.SourceIdentitySHA256, SealedBinarySHA256: plan.SealedBinarySHA256, Partition: plan.PartitionName, PartitionPlanSHA256: plan.PlanSHA256, DatasetSymbols: append([]string(nil), plan.DatasetRequiredSymbols...), TargetSymbols: append([]string(nil), plan.CandidateTargetSymbols...), ContextOnlySymbols: append([]string(nil), plan.ContextOnlySymbols...), Rows: rows}
	artifact, err = qualificationrunner.SealPartitionArtifact(artifact)
	if err != nil {
		return qualificationrunner.PartitionArtifact{}, ArtifactManifest{}, AccessReceipt{}, err
	}
	orderedSources := make([]string, len(plan.SourceManifests))
	for i, member := range plan.SourceManifests {
		orderedSources[i] = member.FileSHA256
	}
	receipt := AccessReceipt{SchemaVersion: AccessReceiptSchemaVersion, PlanSHA256: planSHA, CheckpointSHA256: plan.Checkpoint.SHA256, Partition: plan.PartitionName, RIFAuthorizationID: entry.Authorization.RIFAuthorizationID, RIFAccessReceiptSHA256: entry.Authorization.RIFAccessReceiptSHA256, SourceManifestCount: len(plan.SourceManifests), SourceFragmentCount: fragmentCount, RowsOpened: openedRows, CandidateInputRows: len(rows), OpenedAt: now.UTC(), ArtifactSHA256: artifact.ArtifactSHA256}
	manifest := ArtifactManifest{SchemaVersion: ArtifactManifestVersion, PlanSHA256: planSHA, CheckpointSHA256: plan.Checkpoint.SHA256, Partition: plan.PartitionName, UniverseContractSHA256: plan.UniverseContractSHA256, OrderedSourceSHA256: orderedSources, ArtifactSHA256: artifact.ArtifactSHA256}
	manifest.ManifestSHA256, err = manifestHash(manifest)
	if err != nil {
		return qualificationrunner.PartitionArtifact{}, ArtifactManifest{}, AccessReceipt{}, err
	}
	receipt.ArtifactManifestSHA256 = manifest.ManifestSHA256
	receipt.ReceiptSHA256, err = receiptHash(receipt)
	if err != nil {
		return qualificationrunner.PartitionArtifact{}, ArtifactManifest{}, AccessReceipt{}, err
	}
	artifactBytes, _ := qualificationrunner.EncodePartitionArtifact(artifact)
	manifestBytes, _ := json.Marshal(manifest)
	receiptBytes, _ := json.Marshal(receipt)
	base := filepath.Join(root, "artifacts", hashLeaf(plan.Checkpoint.SHA256), plan.PartitionName)
	if err := os.MkdirAll(base, 0o700); err != nil {
		return qualificationrunner.PartitionArtifact{}, ArtifactManifest{}, AccessReceipt{}, err
	}
	for _, out := range []struct {
		name string
		data []byte
	}{{hashLeaf(planSHA) + ".json", artifactBytes}, {hashLeaf(planSHA) + ".manifest.json", append(manifestBytes, '\n')}, {hashLeaf(planSHA) + ".access.json", append(receiptBytes, '\n')}} {
		if err := atomicWrite(filepath.Join(base, out.name), out.data, 0o600); err != nil {
			return qualificationrunner.PartitionArtifact{}, ArtifactManifest{}, AccessReceipt{}, err
		}
	}
	entry.State, entry.ArtifactSHA256, entry.ArtifactManifestSHA256, entry.AccessReceiptSHA256 = MaterializationSealed, artifact.ArtifactSHA256, manifest.ManifestSHA256, receipt.ReceiptSHA256
	registry.Entries[planSHA] = entry
	if err := writeRegistry(root, registry); err != nil {
		return qualificationrunner.PartitionArtifact{}, ArtifactManifest{}, AccessReceipt{}, err
	}
	return artifact, manifest, receipt, nil
}

func AuthorizeConsumption(root, planSHA string, authorization ConsumptionAuthorization) error {
	registry, err := readRegistry(root)
	if err != nil {
		return err
	}
	entry, ok := registry.Entries[planSHA]
	if !ok || (entry.State != MaterializationSealed && entry.State != ConsumptionSealed) || entry.ArtifactSHA256 == "" {
		return errors.New("only a sealed registered artifact may be consumed")
	}
	plan, err := loadRegisteredPlan(root, planSHA)
	if err != nil {
		return err
	}
	if authorization.SchemaVersion != ConsumptionAuthorizationVersion || authorization.PlanSHA256 != planSHA || authorization.ArtifactSHA256 != entry.ArtifactSHA256 || authorization.Partition != plan.PartitionName || authorization.VariantID == "" || authorization.RIFAuthorizationID == "" || !validSHA(authorization.RIFAccessReceiptSHA256) || authorization.AuthorizedAt.IsZero() {
		return errors.New("consumption lacks exact durable RIF authorization")
	}
	for _, receipt := range entry.ConsumptionReceipts {
		if receipt.RIFAuthorizationID == authorization.RIFAuthorizationID {
			return errors.New("successful artifact consumption cannot replay")
		}
	}
	want := authorization
	want.AuthorizationSHA256 = ""
	hash, err := canonicalHash(want)
	if err != nil || authorization.AuthorizationSHA256 != hash {
		return errors.New("consumption authorization hash mismatch")
	}
	entry.State, entry.ConsumptionAuthorization = ConsumptionAuthorized, &authorization
	registry.Entries[planSHA] = entry
	return writeRegistry(root, registry)
}

func ConsumeArtifact(root, planSHA string, now time.Time) ([]byte, ConsumptionReceipt, error) {
	registry, err := readRegistry(root)
	if err != nil {
		return nil, ConsumptionReceipt{}, err
	}
	entry, ok := registry.Entries[planSHA]
	if !ok || entry.State != ConsumptionAuthorized || entry.ConsumptionAuthorization == nil {
		return nil, ConsumptionReceipt{}, errors.New("artifact consumption is not authorized")
	}
	plan, err := loadRegisteredPlan(root, planSHA)
	if err != nil {
		return nil, ConsumptionReceipt{}, err
	}
	path := filepath.Join(root, "artifacts", hashLeaf(plan.Checkpoint.SHA256), plan.PartitionName, hashLeaf(planSHA)+".json")
	if err := rejectSymlinkComponents(path); err != nil {
		return nil, ConsumptionReceipt{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, ConsumptionReceipt{}, err
	}
	var artifact qualificationrunner.PartitionArtifact
	if err := strictJSON(data, &artifact); err != nil {
		return nil, ConsumptionReceipt{}, err
	}
	encoded, err := qualificationrunner.EncodePartitionArtifact(artifact)
	if err != nil || !bytes.Equal(data, encoded) || artifact.ArtifactSHA256 != entry.ArtifactSHA256 || artifact.CheckpointSHA256 != plan.Checkpoint.SHA256 || artifact.Partition != plan.PartitionName || artifact.PartitionPlanSHA256 != plan.PlanSHA256 || !reflect.DeepEqual(artifact.DatasetSymbols, plan.DatasetRequiredSymbols) || !reflect.DeepEqual(artifact.TargetSymbols, plan.CandidateTargetSymbols) || !reflect.DeepEqual(artifact.ContextOnlySymbols, plan.ContextOnlySymbols) {
		return nil, ConsumptionReceipt{}, errors.New("sealed artifact hash or canonical bytes changed")
	}
	manifestPath := filepath.Join(root, "artifacts", hashLeaf(plan.Checkpoint.SHA256), plan.PartitionName, hashLeaf(planSHA)+".manifest.json")
	if err := rejectSymlinkComponents(manifestPath); err != nil {
		return nil, ConsumptionReceipt{}, err
	}
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, ConsumptionReceipt{}, err
	}
	var manifest ArtifactManifest
	if err := strictJSON(manifestData, &manifest); err != nil {
		return nil, ConsumptionReceipt{}, err
	}
	wantManifest, err := manifestHash(manifest)
	if err != nil || manifest.ManifestSHA256 != wantManifest || manifest.ManifestSHA256 != entry.ArtifactManifestSHA256 || manifest.PlanSHA256 != planSHA || manifest.CheckpointSHA256 != plan.Checkpoint.SHA256 || manifest.Partition != plan.PartitionName || manifest.UniverseContractSHA256 != plan.UniverseContractSHA256 || manifest.ArtifactSHA256 != artifact.ArtifactSHA256 {
		return nil, ConsumptionReceipt{}, errors.New("sealed artifact manifest identity changed")
	}
	previous := ""
	if len(entry.ConsumptionReceipts) > 0 {
		previous = entry.ConsumptionReceipts[len(entry.ConsumptionReceipts)-1].ReceiptSHA256
	}
	auth := entry.ConsumptionAuthorization
	receipt := ConsumptionReceipt{SchemaVersion: ConsumptionReceiptVersion, PlanSHA256: planSHA, ArtifactSHA256: artifact.ArtifactSHA256, Partition: plan.PartitionName, VariantID: auth.VariantID, RIFAuthorizationID: auth.RIFAuthorizationID, RIFAccessReceiptSHA256: auth.RIFAccessReceiptSHA256, ConsumedAt: now.UTC(), PreviousReceiptSHA256: previous}
	receipt.ReceiptSHA256, err = consumptionReceiptHash(receipt)
	if err != nil {
		return nil, ConsumptionReceipt{}, err
	}
	entry.State, entry.ConsumptionAuthorization = ConsumptionSealed, nil
	entry.ConsumptionReceipts = append(entry.ConsumptionReceipts, receipt)
	registry.Entries[planSHA] = entry
	if err := writeRegistry(root, registry); err != nil {
		return nil, ConsumptionReceipt{}, err
	}
	return data, receipt, nil
}

func readFragment(plan Plan, ref SourceArtifact) (sourceFragment, error) {
	root, err := sourceRootByID(plan.SourceRoot, plan.ProspectiveSourceRoot, ref.SourceRootID)
	if err != nil {
		return sourceFragment{}, err
	}
	path, err := secureJoin(root, ref.RelativePath, true)
	if err != nil {
		return sourceFragment{}, err
	}
	encoded, err := os.ReadFile(path)
	if err != nil {
		return sourceFragment{}, err
	}
	data := encoded
	if ref.Encoding == BackfillFragmentEncoding || ref.Encoding == SyntheticFragmentEncoding {
		reader, gzipErr := gzip.NewReader(bytes.NewReader(encoded))
		if gzipErr != nil {
			return sourceFragment{}, gzipErr
		}
		decompressed, readErr := io.ReadAll(io.LimitReader(reader, 64*1024*1024+1))
		closeErr := reader.Close()
		if readErr != nil || closeErr != nil || len(decompressed) > 64*1024*1024 {
			return sourceFragment{}, errors.New("fragment exceeds bounded size or is not a complete gzip stream")
		}
		data = decompressed
	} else if ref.Encoding != ProspectiveFragmentEncoding {
		return sourceFragment{}, errors.New("unregistered source fragment encoding")
	}
	if len(data) > 64*1024*1024 {
		return sourceFragment{}, errors.New("fragment exceeds bounded size")
	}
	if ref.Encoding == ProspectiveFragmentEncoding {
		return decodeProspectiveFragment(data, ref)
	}
	var fragment sourceFragment
	if err := strictJSON(data, &fragment); err != nil {
		return sourceFragment{}, err
	}
	var hash string
	if ref.Encoding == BackfillFragmentEncoding {
		hash, err = historianCanonicalJSONHash(data, "fragment_hash")
		if fragment.SchemaVersion != "ak-historian.pr4b0-r1p5r.normalized-fragment.v1" {
			return sourceFragment{}, errors.New("backfill fragment schema substitution")
		}
	} else {
		want := fragment
		want.FragmentHash = ""
		hash, err = canonicalHash(want)
	}
	if err != nil || fragment.FragmentHash != ref.CanonicalSHA256 || fragment.FragmentHash != hash {
		return sourceFragment{}, errors.New("source fragment canonical hash changed")
	}
	return fragment, nil
}

func decodeProspectiveFragment(data []byte, ref SourceArtifact) (sourceFragment, error) {
	var prospective prospectiveFragment
	if err := strictJSON(data, &prospective); err != nil {
		return sourceFragment{}, err
	}
	hash, err := historianCanonicalJSONHash(data, "fragment_hash")
	if err != nil || prospective.SchemaVersion != "ak-historian.pr4b0-r1p4.normalized-fragment.v1" || prospective.NormalizationVersion == "" || prospective.CycleID == "" || prospective.FragmentHash != ref.CanonicalSHA256 || prospective.FragmentHash != hash || !validSHA(ref.ReceiptSHA256) || ref.ObservedAvailableAtUTC.IsZero() {
		return sourceFragment{}, errors.New("prospective source fragment canonical identity changed")
	}
	fragment := sourceFragment{SchemaVersion: prospective.SchemaVersion, RequestID: ref.ReceiptSHA256, Symbol: prospective.Symbol, SourceSchemaVersion: prospective.SourceSchemaVersion, SourceSchemaFingerprint: prospective.SourceSchemaFingerprint, FragmentHash: prospective.FragmentHash, Records: make([]normalizedRecord, len(prospective.Records))}
	for index, row := range prospective.Records {
		fragment.Records[index] = normalizedRecord{Market: row.Market, Symbol: row.Symbol, Interval: row.Interval, Period: row.Period, SourceDate: row.SourceDate, OpenTimeMS: row.OpenTimeMS, Open: row.Open, High: row.High, Low: row.Low, Close: row.Close, Volume: row.Volume, CloseTimeMS: row.CloseTimeMS, QuoteAssetVolume: row.QuoteAssetVolume, NumberOfTrades: row.NumberOfTrades, TakerBuyBaseVolume: row.TakerBuyBaseVolume, TakerBuyQuoteVolume: row.TakerBuyQuoteVolume, MarketEventTimeUTC: time.UnixMilli(row.OpenTimeMS).UTC(), ProviderCandleCloseTimeUTC: time.UnixMilli(row.CloseTimeMS).UTC(), ObservedAvailableAtUTC: ref.ObservedAvailableAtUTC.UTC(), AcquiredAtUTC: ref.ObservedAvailableAtUTC.UTC(), AcquisitionReceiptID: ref.ReceiptSHA256}
	}
	return fragment, nil
}

func toCandle(row normalizedRecord) (protocol.Candle, error) {
	parse := func(value string) (float64, error) { return strconv.ParseFloat(value, 64) }
	open, err := parse(row.Open)
	if err != nil {
		return protocol.Candle{}, err
	}
	high, err := parse(row.High)
	if err != nil {
		return protocol.Candle{}, err
	}
	low, err := parse(row.Low)
	if err != nil {
		return protocol.Candle{}, err
	}
	closeValue, err := parse(row.Close)
	if err != nil {
		return protocol.Candle{}, err
	}
	volume, err := parse(row.Volume)
	if err != nil {
		return protocol.Candle{}, err
	}
	quote, err := parse(row.QuoteAssetVolume)
	if err != nil {
		return protocol.Candle{}, err
	}
	takerBase, err := parse(row.TakerBuyBaseVolume)
	if err != nil {
		return protocol.Candle{}, err
	}
	takerQuote, err := parse(row.TakerBuyQuoteVolume)
	if err != nil {
		return protocol.Candle{}, err
	}
	return protocol.Candle{Market: row.Market, Symbol: row.Symbol, Interval: row.Interval, OpenTimeMS: row.OpenTimeMS, Open: open, High: high, Low: low, Close: closeValue, Volume: volume, CloseTimeMS: row.CloseTimeMS, QuoteAssetVolume: quote, NumberOfTrades: row.NumberOfTrades, TakerBuyBaseVolume: takerBase, TakerBuyQuoteVolume: takerQuote}, nil
}

func buildInputRows(plan Plan, candles map[string][]protocol.Candle) ([]qualificationrunner.InputRow, error) {
	if plan.SyntheticFixture {
		return buildSyntheticInputRows(plan, candles), nil
	}
	btc, eth := candles["BTCUSDT"], candles["ETHUSDT"]
	result := []qualificationrunner.InputRow{}
	for _, symbol := range plan.CandidateTargetSymbols {
		primary := candles[symbol]
		featureRows, err := features.BuildRows(primary, features.BuildOptions{Market: "futures-um", Symbol: symbol, Interval: "1m", ContextBTC: btc, ContextETH: eth})
		if err != nil {
			return nil, err
		}
		for i, row := range featureRows {
			if row.Warmup || i+240 >= len(primary) {
				continue
			}
			decision := time.UnixMilli(primary[i].CloseTimeMS).UTC()
			if decision.Before(plan.PartitionInterval.Start) || !decision.Before(plan.PartitionInterval.End) {
				continue
			}
			sourceHash := plan.PlanSHA256
			result = append(result, qualificationrunner.InputRow{Partition: plan.PartitionName, Symbol: symbol, EventTime: decision, AvailableAt: decision, Close: row.Close, FutureClose240m: primary[i+240].Close, EMA50: row.EMA50, EMA200: row.EMA200, TrendSlope20: row.TrendSlope20, RealizedVol60: row.RealizedVol60, WarmupSufficient: true, BTC: qualificationrunner.Context{SnapshotID: fmt.Sprintf("%s:BTCUSDT:%d", plan.PartitionName, primary[i].OpenTimeMS), SourceInputSHA256: sourceHash, AvailableAt: decision, Return60: row.BTCReturn60}, ETH: qualificationrunner.Context{SnapshotID: fmt.Sprintf("%s:ETHUSDT:%d", plan.PartitionName, primary[i].OpenTimeMS), SourceInputSHA256: sourceHash, AvailableAt: decision, Return60: row.ETHReturn60}})
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if !result[i].EventTime.Equal(result[j].EventTime) {
			return result[i].EventTime.Before(result[j].EventTime)
		}
		return result[i].Symbol < result[j].Symbol
	})
	return result, nil
}

func buildSyntheticInputRows(plan Plan, candles map[string][]protocol.Candle) []qualificationrunner.InputRow {
	result := []qualificationrunner.InputRow{}
	for _, symbol := range plan.CandidateTargetSymbols {
		for index, candle := range candles[symbol] {
			decision := time.UnixMilli(candle.CloseTimeMS).UTC()
			future := candle.Close * 1.02
			if index%10 == 0 {
				future = candle.Close * 0.99
			}
			result = append(result, qualificationrunner.InputRow{Partition: plan.PartitionName, Symbol: symbol, EventTime: decision, AvailableAt: decision, Close: candle.Close, FutureClose240m: future, EMA50: candle.Close * 1.05, EMA200: candle.Close * 1.10, TrendSlope20: -0.02, RealizedVol60: 0.003, WarmupSufficient: true, BTC: qualificationrunner.Context{SnapshotID: fmt.Sprintf("%s:BTCUSDT:%d", plan.PartitionName, candle.OpenTimeMS), SourceInputSHA256: plan.PlanSHA256, AvailableAt: decision, Return60: 0.01}, ETH: qualificationrunner.Context{SnapshotID: fmt.Sprintf("%s:ETHUSDT:%d", plan.PartitionName, candle.OpenTimeMS), SourceInputSHA256: plan.PlanSHA256, AvailableAt: decision, Return60: 0.01}})
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if !result[i].EventTime.Equal(result[j].EventTime) {
			return result[i].EventTime.Before(result[j].EventTime)
		}
		return result[i].Symbol < result[j].Symbol
	})
	return result
}

func readRegistry(root string) (Registry, error) {
	canonical, err := canonicalRealDirectory(root)
	if err != nil {
		return Registry{}, err
	}
	data, err := os.ReadFile(filepath.Join(canonical, "registry.json"))
	if err != nil {
		return Registry{}, err
	}
	var registry Registry
	if err := strictJSON(data, &registry); err != nil {
		return Registry{}, err
	}
	want := registry
	want.RegistrySHA256 = ""
	hash, err := canonicalHash(want)
	if err != nil || registry.SchemaVersion != RegistrySchemaVersion || registry.RegistrySHA256 != hash {
		return Registry{}, errors.New("registry integrity mismatch")
	}
	return registry, nil
}
func writeRegistry(root string, registry Registry) error {
	registry.SchemaVersion = RegistrySchemaVersion
	if registry.Entries == nil {
		registry.Entries = map[string]RegistryEntry{}
	}
	registry.RegistrySHA256 = ""
	hash, err := canonicalHash(registry)
	if err != nil {
		return err
	}
	registry.RegistrySHA256 = hash
	data, err := json.Marshal(registry)
	if err != nil {
		return err
	}
	return atomicWrite(filepath.Join(root, "registry.json"), append(data, '\n'), 0o600)
}
func loadRegisteredPlan(root, sha string) (Plan, error) {
	data, err := os.ReadFile(filepath.Join(root, "plans", hashLeaf(sha)+".json"))
	if err != nil {
		return Plan{}, err
	}
	return DecodePlan(data)
}
func verifyRegisteredPlan(root string, plan Plan) error {
	stored, err := loadRegisteredPlan(root, plan.PlanSHA256)
	if err != nil {
		return err
	}
	if !bytes.Equal(mustJSON(stored), mustJSON(plan)) {
		return errors.New("registered plan bytes differ")
	}
	return nil
}
func hashLeaf(sha string) string {
	if len(sha) == 71 {
		return sha[7:]
	}
	return sha
}
func manifestHash(value ArtifactManifest) (string, error) {
	value.ManifestSHA256 = ""
	return canonicalHash(value)
}
func receiptHash(value AccessReceipt) (string, error) {
	value.ReceiptSHA256 = ""
	return canonicalHash(value)
}
func consumptionReceiptHash(value ConsumptionReceipt) (string, error) {
	value.ReceiptSHA256 = ""
	return canonicalHash(value)
}
func mustJSON(value any) []byte { data, _ := json.Marshal(value); return data }
func atomicWrite(path string, data []byte, mode os.FileMode) error {
	if err := rejectSymlinkComponents(filepath.Dir(path)); err != nil {
		return err
	}
	if info, err := os.Lstat(path); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return errUnsafePath
	}
	temp, err := os.CreateTemp(filepath.Dir(path), ".tmp-")
	if err != nil {
		return err
	}
	name := temp.Name()
	defer os.Remove(name)
	if err := temp.Chmod(mode); err != nil {
		temp.Close()
		return err
	}
	if _, err := temp.Write(data); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Sync(); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return os.Rename(name, path)
}
