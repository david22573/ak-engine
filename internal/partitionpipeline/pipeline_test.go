package partitionpipeline

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/david22573/ak-engine/internal/qualificationrunner"
)

func TestSyntheticPlanAndMaterializationLifecycle(t *testing.T) {
	root, plan := syntheticSource(t, "DEVELOPMENT")
	copyPlan := plan
	copyPlan.PlanSHA256 = ""
	copyPlan.PlanSHA256, _ = planHash(copyPlan)
	first, err := EncodePlan(plan)
	if err != nil {
		t.Fatal(err)
	}
	second, err := EncodePlan(copyPlan)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("canonical plan is not byte-stable")
	}

	registry := filepath.Join(t.TempDir(), "registry")
	if err := CreateRegistry(registry); err != nil {
		t.Fatal(err)
	}
	if err := RegisterPlan(registry, plan); err != nil {
		t.Fatal(err)
	}
	auth := syntheticAuthorization(t, plan)
	if err := AuthorizeMaterialization(registry, plan.PlanSHA256, auth); err != nil {
		t.Fatal(err)
	}
	artifact, manifest, receipt, err := Materialize(registry, plan.PlanSHA256, time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if artifact.Label != qualificationrunner.SyntheticLabel || artifact.PartitionPlanSHA256 != plan.PlanSHA256 || receipt.RowsOpened != 9*800 || receipt.CandidateInputRows == 0 || receipt.ArtifactSHA256 != artifact.ArtifactSHA256 || receipt.ArtifactManifestSHA256 != manifest.ManifestSHA256 {
		t.Fatalf("materialization evidence incomplete: %#v %#v", manifest, receipt)
	}
	if manifest.ArtifactSHA256 != artifact.ArtifactSHA256 || manifest.PlanSHA256 != plan.PlanSHA256 {
		t.Fatal("artifact manifest binding mismatch")
	}
	consumeAuth, err := SealConsumptionAuthorization(ConsumptionAuthorization{PlanSHA256: plan.PlanSHA256, ArtifactSHA256: artifact.ArtifactSHA256, Partition: plan.PartitionName, VariantID: "V00", RIFAuthorizationID: "synthetic-consume", RIFAccessReceiptSHA256: testHash('d'), AuthorizedAt: time.Date(2030, 1, 1, 0, 0, 1, 0, time.UTC)})
	if err != nil {
		t.Fatal(err)
	}
	if err := AuthorizeConsumption(registry, plan.PlanSHA256, consumeAuth); err != nil {
		t.Fatal(err)
	}
	consumed, consumptionReceipt, err := ConsumeArtifact(registry, plan.PlanSHA256, time.Date(2030, 1, 1, 0, 0, 2, 0, time.UTC))
	if err != nil || len(consumed) == 0 || consumptionReceipt.VariantID != "V00" {
		t.Fatalf("authorized consumption did not seal: %v", err)
	}
	state, err := readRegistry(registry)
	if err != nil {
		t.Fatal(err)
	}
	if state.Entries[plan.PlanSHA256].State != ConsumptionSealed {
		t.Fatal("partition lifecycle did not seal consumption")
	}
	if _, _, _, err := Materialize(registry, plan.PlanSHA256, time.Now()); err == nil {
		t.Fatal("successful materialization replayed")
	}
	_ = root
}

func TestPlanRejectsBoundaryUniverseMembershipAndPathMutation(t *testing.T) {
	_, base := syntheticSource(t, "VALIDATION")
	tests := map[string]func(*Plan){
		"boundary":         func(p *Plan) { p.PartitionInterval.Start = p.PartitionInterval.Start.Add(time.Minute) },
		"missing manifest": func(p *Plan) { p.SourceManifests = p.SourceManifests[:8] },
		"extra manifest":   func(p *Plan) { p.SourceManifests = append(p.SourceManifests, p.SourceManifests[0]) },
		"manifest hash":    func(p *Plan) { p.SourceManifests[0].FileSHA256 = testHash('a') },
		"wrong target":     func(p *Plan) { p.CandidateTargetSymbols = p.CandidateTargetSymbols[:7] },
		"context overlap":  func(p *Plan) { p.ContextOnlySymbols = []string{"ETHUSDT"} },
		"alternate root":   func(p *Plan) { p.SourceRoot = filepath.Dir(p.SourceRoot) },
		"ordering":         func(p *Plan) { p.SourceManifests[0], p.SourceManifests[1] = p.SourceManifests[1], p.SourceManifests[0] },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			candidate := clonePlan(base)
			mutate(&candidate)
			candidate.PlanSHA256 = ""
			candidate.PlanSHA256, _ = planHash(candidate)
			if err := VerifyPlan(candidate); err == nil {
				t.Fatal("mutated plan verified")
			}
		})
	}
}

func TestMaterializerRejectsChangedSymlinkCrossPartitionAndWrongSymbol(t *testing.T) {
	for name, mutate := range map[string]func(*testing.T, string, Plan){
		"changed manifest": func(t *testing.T, root string, p Plan) {
			if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(p.SourceManifests[0].RelativePath)), []byte("changed\n"), 0o600); err != nil {
				t.Fatal(err)
			}
		},
		"symlink fragment": func(t *testing.T, root string, p Plan) {
			ref := p.SourceManifests[0].FragmentArtifacts[0]
			path := filepath.Join(root, filepath.FromSlash(ref.RelativePath))
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink("missing", path); err != nil {
				t.Fatal(err)
			}
		},
		"cross partition row": func(t *testing.T, root string, p Plan) {
			rewriteFirstFragment(t, root, p, func(row *normalizedRecord) {
				row.OpenTimeMS = p.PartitionInterval.End.UnixMilli()
				row.CloseTimeMS = row.OpenTimeMS + 59999
			})
		},
		"pre-2026 row": func(t *testing.T, root string, p Plan) {
			rewriteFirstFragment(t, root, p, func(row *normalizedRecord) {
				row.OpenTimeMS = time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC).UnixMilli()
				row.CloseTimeMS = row.OpenTimeMS + 59999
			})
		},
		"wrong symbol row": func(t *testing.T, root string, p Plan) {
			rewriteFirstFragment(t, root, p, func(row *normalizedRecord) { row.Symbol = "NOTUSDT" })
		},
	} {
		t.Run(name, func(t *testing.T) {
			root, plan := syntheticSource(t, "DEVELOPMENT")
			registry := filepath.Join(t.TempDir(), "registry")
			if err := CreateRegistry(registry); err != nil {
				t.Fatal(err)
			}
			if err := RegisterPlan(registry, plan); err != nil {
				t.Fatal(err)
			}
			if err := AuthorizeMaterialization(registry, plan.PlanSHA256, syntheticAuthorization(t, plan)); err != nil {
				t.Fatal(err)
			}
			mutate(t, root, plan)
			if _, _, _, err := Materialize(registry, plan.PlanSHA256, time.Now()); err == nil {
				t.Fatal("unsafe source mutation materialized")
			}
		})
	}
}

func TestUnregisteredPlanCacheAndAuthorizationFailClosed(t *testing.T) {
	_, plan := syntheticSource(t, "FINAL_HOLDOUT")
	registry := filepath.Join(t.TempDir(), "registry")
	if err := CreateRegistry(registry); err != nil {
		t.Fatal(err)
	}
	if err := AuthorizeMaterialization(registry, plan.PlanSHA256, syntheticAuthorization(t, plan)); err == nil {
		t.Fatal("unregistered plan authorized")
	}
	if _, _, _, err := Materialize(registry, plan.PlanSHA256, time.Now()); err == nil {
		t.Fatal("unregistered cache materialized")
	}
	if err := RegisterPlan(registry, plan); err != nil {
		t.Fatal(err)
	}
	if _, err := ProveZeroAccess(registry, plan.PlanSHA256); err != nil {
		t.Fatalf("verified plan did not produce durable zero-access proof: %v", err)
	}
	unsealedConsumption, _ := SealConsumptionAuthorization(ConsumptionAuthorization{PlanSHA256: plan.PlanSHA256, ArtifactSHA256: testHash('e'), Partition: plan.PartitionName, VariantID: "V00", RIFAuthorizationID: "unsealed", RIFAccessReceiptSHA256: testHash('d'), AuthorizedAt: time.Now().UTC()})
	if err := AuthorizeConsumption(registry, plan.PlanSHA256, unsealedConsumption); err == nil {
		t.Fatal("unsealed artifact consumption authorized")
	}
	bad := syntheticAuthorization(t, plan)
	bad.CheckpointSHA256 = testHash('f')
	bad.AuthorizationSHA256 = ""
	bad.AuthorizationSHA256, _ = canonicalHash(bad)
	if err := AuthorizeMaterialization(registry, plan.PlanSHA256, bad); err == nil {
		t.Fatal("another checkpoint authorization accepted")
	}
}

func TestConsumptionRejectsPartitionCheckpointAndManifestSubstitution(t *testing.T) {
	_, plan := syntheticSource(t, "DEVELOPMENT")
	registry := filepath.Join(t.TempDir(), "registry")
	if err := CreateRegistry(registry); err != nil {
		t.Fatal(err)
	}
	if err := RegisterPlan(registry, plan); err != nil {
		t.Fatal(err)
	}
	if err := AuthorizeMaterialization(registry, plan.PlanSHA256, syntheticAuthorization(t, plan)); err != nil {
		t.Fatal(err)
	}
	artifact, _, _, err := Materialize(registry, plan.PlanSHA256, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ProveZeroAccess(registry, plan.PlanSHA256); err == nil {
		t.Fatal("zero access was proven after rows were opened")
	}
	wrongPartition, _ := SealConsumptionAuthorization(ConsumptionAuthorization{PlanSHA256: plan.PlanSHA256, ArtifactSHA256: artifact.ArtifactSHA256, Partition: "VALIDATION", VariantID: "V00", RIFAuthorizationID: "wrong-partition", RIFAccessReceiptSHA256: testHash('d'), AuthorizedAt: time.Now().UTC()})
	if err := AuthorizeConsumption(registry, plan.PlanSHA256, wrongPartition); err == nil {
		t.Fatal("another partition authorization was accepted")
	}
	valid, _ := SealConsumptionAuthorization(ConsumptionAuthorization{PlanSHA256: plan.PlanSHA256, ArtifactSHA256: artifact.ArtifactSHA256, Partition: plan.PartitionName, VariantID: "V00", RIFAuthorizationID: "valid", RIFAccessReceiptSHA256: testHash('d'), AuthorizedAt: time.Now().UTC()})
	if err := AuthorizeConsumption(registry, plan.PlanSHA256, valid); err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(registry, "artifacts", strings.TrimPrefix(plan.Checkpoint.SHA256, "sha256:"), plan.PartitionName, strings.TrimPrefix(plan.PlanSHA256, "sha256:")+".manifest.json")
	if err := os.WriteFile(manifestPath, []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := ConsumeArtifact(registry, plan.PlanSHA256, time.Now().UTC()); err == nil {
		t.Fatal("changed artifact manifest was consumed")
	}
}

func syntheticSource(t *testing.T, partition string) (string, Plan) {
	t.Helper()
	root := filepath.Join(t.TempDir(), "source")
	if err := os.MkdirAll(filepath.Join(root, "manifests"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "fragments"), 0o700); err != nil {
		t.Fatal(err)
	}
	universe, err := qualificationrunner.V00UniverseContract()
	if err != nil {
		t.Fatal(err)
	}
	interval := acceptedIntervals[partition]
	members := make([]SourceManifest, 0, 9)
	for _, symbol := range universe.DatasetRequiredSymbols {
		records := syntheticRecords(symbol, interval.Start, 800)
		fragment := sourceFragment{SchemaVersion: "ak-historian.pr4b0-r1p5r.normalized-fragment.v1", RequestID: "synthetic:" + partition + ":" + symbol, Symbol: symbol, SourceSchemaVersion: "synthetic.candle.v1", SourceSchemaFingerprint: testHash('9'), Records: records}
		fragment.FragmentHash, _ = canonicalHash(fragment)
		fragmentData, _ := json.Marshal(fragment)
		var compressed bytes.Buffer
		writer := gzip.NewWriter(&compressed)
		if _, err := writer.Write(fragmentData); err != nil {
			t.Fatal(err)
		}
		if err := writer.Close(); err != nil {
			t.Fatal(err)
		}
		fragmentRel := filepath.ToSlash(filepath.Join("fragments", partition+"-"+symbol+".json.gz"))
		if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(fragmentRel)), compressed.Bytes(), 0o600); err != nil {
			t.Fatal(err)
		}
		manifestRel := filepath.ToSlash(filepath.Join("manifests", partition+"-"+symbol+".json"))
		manifestData := []byte(fmt.Sprintf("{\"partition\":%q,\"symbol\":%q}\n", partition, symbol))
		if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(manifestRel)), manifestData, 0o600); err != nil {
			t.Fatal(err)
		}
		members = append(members, SourceManifest{Symbol: symbol, UTCDate: interval.Start.Format("2006-01-02"), RelativePath: manifestRel, FileSHA256: byteHash(manifestData), PartitionSHA256: testHash('c'), ExpectedRows: len(records), ReceiptArtifacts: []SourceArtifact{{RelativePath: "synthetic-receipt/" + symbol, CanonicalSHA256: testHash('7')}}, FragmentArtifacts: []SourceArtifact{{RelativePath: fragmentRel, CanonicalSHA256: fragment.FragmentHash}}})
	}
	sortMembers(members)
	plan := Plan{SchemaVersion: PlanSchemaVersion, Checkpoint: HashIdentity{"synthetic-checkpoint", testHash('1')}, HistorianCommit: strings.Repeat("2", 40), HistorianTree: strings.Repeat("3", 40), SourceIdentitySHA256: testHash('4'), ReacquisitionProtocol: HashIdentity{"synthetic-protocol", testHash('5')}, PreAcquisitionSealSHA256: testHash('6'), SealedBinarySHA256: testHash('7'), AbandonedEvidenceRegistry: HashIdentity{"synthetic-abandoned", testHash('8')}, DatasetRequiredSymbols: universe.DatasetRequiredSymbols, CandidateTargetSymbols: universe.CandidateTargetSymbols, ContextOnlySymbols: universe.ContextOnlySymbols, UniverseContractSHA256: universe.ContractSHA256, EligibleInterval: Interval{time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 7, 17, 0, 0, 0, 0, time.UTC)}, PartitionName: partition, PartitionInterval: interval, SourceManifests: members, ExpectedStructuralDays: int(interval.End.Sub(interval.Start) / (24 * time.Hour)), SchemaIdentitySHA256: testHash('9'), OutputFormat: OutputFormat, OrderingPolicy: OrderingPolicy, OutputPathPolicy: OutputPathPolicy, SymlinkPolicy: SymlinkPolicy, CachePolicy: CachePolicy, AvailabilityCutoff: time.Date(2026, 7, 17, 7, 36, 29, 0, time.UTC), SourceRoot: root, SyntheticFixture: true}
	plan.PlanSHA256, _ = planHash(plan)
	if err := VerifyPlan(plan); err != nil {
		t.Fatal(err)
	}
	return root, plan
}

func syntheticRecords(symbol string, start time.Time, count int) []normalizedRecord {
	rows := make([]normalizedRecord, count)
	for i := range rows {
		at := start.Add(time.Duration(i) * time.Minute)
		closeValue := 1000 - float64(i)*0.08 + 3*math.Sin(float64(i)/2)
		if closeValue <= 0 {
			closeValue = 100
		}
		value := strconvFloat(closeValue)
		rows[i] = normalizedRecord{Market: "futures-um", Symbol: symbol, Interval: "1m", Period: "1m", SourceDate: at.Format("2006-01-02"), OpenTimeMS: at.UnixMilli(), Open: value, High: strconvFloat(closeValue + 1), Low: strconvFloat(closeValue - 1), Close: value, Volume: "100", CloseTimeMS: at.Add(time.Minute - time.Millisecond).UnixMilli(), QuoteAssetVolume: "100000", NumberOfTrades: 10, TakerBuyBaseVolume: "50", TakerBuyQuoteVolume: "50000", MarketEventTimeUTC: at, ProviderCandleCloseTimeUTC: at.Add(time.Minute - time.Millisecond), ObservedAvailableAtUTC: at, AcquiredAtUTC: at, AcquisitionReceiptID: "synthetic"}
	}
	return rows
}
func syntheticAuthorization(t *testing.T, plan Plan) MaterializationAuthorization {
	t.Helper()
	value := MaterializationAuthorization{SchemaVersion: AuthorizationSchemaVersion, PlanSHA256: plan.PlanSHA256, CheckpointSHA256: plan.Checkpoint.SHA256, Partition: plan.PartitionName, RIFAuthorizationID: "synthetic-rif-authorization", RIFAuthorizationSHA256: testHash('a'), RIFAccessReceiptSHA256: testHash('b'), AuthorizedAt: time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)}
	value.AuthorizationSHA256, _ = canonicalHash(value)
	return value
}
func rewriteFirstFragment(t *testing.T, root string, plan Plan, mutate func(*normalizedRecord)) {
	t.Helper()
	ref := plan.SourceManifests[0].FragmentArtifacts[0]
	path := filepath.Join(root, filepath.FromSlash(ref.RelativePath))
	compressed, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	reader, err := gzip.NewReader(bytes.NewReader(compressed))
	if err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	reader.Close()
	var fragment sourceFragment
	if err := json.Unmarshal(data, &fragment); err != nil {
		t.Fatal(err)
	}
	mutate(&fragment.Records[0])
	fragment.FragmentHash = ""
	fragment.FragmentHash, _ = canonicalHash(fragment)
	out, _ := json.Marshal(fragment)
	var buffer bytes.Buffer
	writer := gzip.NewWriter(&buffer)
	writer.Write(out)
	writer.Close()
	if err := os.WriteFile(path, buffer.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
}
func clonePlan(value Plan) Plan {
	data, _ := json.Marshal(value)
	var out Plan
	json.Unmarshal(data, &out)
	return out
}
func sortMembers(values []SourceManifest) {
	sort.Slice(values, func(i, j int) bool {
		if values[i].UTCDate != values[j].UTCDate {
			return values[i].UTCDate < values[j].UTCDate
		}
		return values[i].Symbol < values[j].Symbol
	})
}
func strconvFloat(value float64) string { return fmt.Sprintf("%.8f", value) }
func testHash(value byte) string        { return "sha256:" + strings.Repeat(string(value), 64) }
