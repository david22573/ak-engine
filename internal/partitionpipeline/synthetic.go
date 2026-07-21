package partitionpipeline

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/david22573/ak-engine/internal/qualificationrunner"
)

func CreateSyntheticCheckpointFixture(root string) (map[string]Plan, error) {
	if err := os.MkdirAll(filepath.Join(root, "manifests"), 0o700); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Join(root, "fragments"), 0o700); err != nil {
		return nil, err
	}
	universe, err := qualificationrunner.V00UniverseContract()
	if err != nil {
		return nil, err
	}
	plans := map[string]Plan{}
	for _, partition := range []string{"DEVELOPMENT", "VALIDATION", "FINAL_HOLDOUT"} {
		interval := syntheticAcceptedIntervals[partition]
		members := []SourceManifest{}
		for _, symbol := range universe.DatasetRequiredSymbols {
			records := make([]normalizedRecord, 360)
			for i := range records {
				at := interval.Start.Add(time.Duration(i) * 24 * time.Hour)
				price := 1000 + 5*math.Sin(float64(i))
				records[i] = normalizedRecord{Market: "futures-um", Symbol: symbol, Interval: "1m", Period: "1m", SourceDate: at.Format("2006-01-02"), OpenTimeMS: at.UnixMilli(), Open: fmt.Sprintf("%.8f", price), High: fmt.Sprintf("%.8f", price+1), Low: fmt.Sprintf("%.8f", price-1), Close: fmt.Sprintf("%.8f", price), Volume: "100", CloseTimeMS: at.Add(time.Minute - time.Millisecond).UnixMilli(), QuoteAssetVolume: "100000", NumberOfTrades: 10, TakerBuyBaseVolume: "50", TakerBuyQuoteVolume: "50000", MarketEventTimeUTC: at, ProviderCandleCloseTimeUTC: at.Add(time.Minute - time.Millisecond), ObservedAvailableAtUTC: at, AcquiredAtUTC: at, AcquisitionReceiptID: "synthetic:" + partition}
			}
			fragment := sourceFragment{SchemaVersion: "ak-historian.synthetic.normalized-fragment.v1", RequestID: "synthetic:" + partition + ":" + symbol, Symbol: symbol, SourceSchemaVersion: "synthetic.candle.v1", SourceSchemaFingerprint: repeatHash('9'), Records: records}
			fragment.FragmentHash, _ = canonicalHash(fragment)
			raw, _ := json.Marshal(fragment)
			var compressed bytes.Buffer
			writer, _ := gzip.NewWriterLevel(&compressed, gzip.BestCompression)
			writer.Header.ModTime = time.Unix(0, 0).UTC()
			writer.Header.OS = 255
			_, _ = writer.Write(raw)
			_ = writer.Close()
			fragmentRel := filepath.ToSlash(filepath.Join("fragments", strings.ToLower(partition)+"-"+symbol+".json.gz"))
			if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(fragmentRel)), compressed.Bytes(), 0o600); err != nil {
				return nil, err
			}
			manifestRel := filepath.ToSlash(filepath.Join("manifests", strings.ToLower(partition)+"-"+symbol+".json"))
			manifestBytes := []byte(fmt.Sprintf("{\"partition\":%q,\"symbol\":%q}\n", partition, symbol))
			if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(manifestRel)), manifestBytes, 0o600); err != nil {
				return nil, err
			}
			receiptSHA := repeatHash('d')
			members = append(members, SourceManifest{Symbol: symbol, UTCDate: interval.Start.Format("2006-01-02"), RelativePath: manifestRel, FileSHA256: byteHash(manifestBytes), PartitionSHA256: repeatHash('c'), ExpectedRows: len(records), ReceiptArtifacts: []SourceArtifact{{SourceRootID: CheckpointSourceRootID, RelativePath: "synthetic/receipt/" + symbol, CanonicalSHA256: receiptSHA, ObservedAvailableAtUTC: interval.Start}}, FragmentArtifacts: []SourceArtifact{{SourceRootID: CheckpointSourceRootID, RelativePath: fragmentRel, CanonicalSHA256: fragment.FragmentHash, Encoding: SyntheticFragmentEncoding, ReceiptSHA256: receiptSHA, ObservedAvailableAtUTC: interval.Start}}})
		}
		sort.Slice(members, func(i, j int) bool { return members[i].Symbol < members[j].Symbol })
		plan := Plan{SchemaVersion: PlanSchemaVersion, Checkpoint: HashIdentity{"synthetic-checkpoint", repeatHash('1')}, HistorianCommit: strings.Repeat("2", 40), HistorianTree: strings.Repeat("3", 40), SourceIdentitySHA256: repeatHash('4'), ReacquisitionProtocol: HashIdentity{"synthetic-reacquisition", repeatHash('5')}, PreAcquisitionSealSHA256: repeatHash('6'), SealedBinarySHA256: repeatHash('7'), AbandonedEvidenceRegistry: HashIdentity{"synthetic-abandoned", repeatHash('8')}, DatasetRequiredSymbols: universe.DatasetRequiredSymbols, CandidateTargetSymbols: universe.CandidateTargetSymbols, ContextOnlySymbols: universe.ContextOnlySymbols, UniverseContractSHA256: universe.ContractSHA256, EligibleInterval: Interval{time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2033, 1, 1, 0, 0, 0, 0, time.UTC)}, PartitionName: partition, PartitionInterval: interval, SourceManifests: members, ExpectedStructuralDays: int(interval.End.Sub(interval.Start) / (24 * time.Hour)), SchemaIdentitySHA256: repeatHash('9'), OutputFormat: OutputFormat, OrderingPolicy: OrderingPolicy, OutputPathPolicy: OutputPathPolicy, SymlinkPolicy: SymlinkPolicy, CachePolicy: CachePolicy, AvailabilityCutoff: time.Date(2033, 1, 2, 0, 0, 0, 0, time.UTC), SourceRoot: root, SyntheticFixture: true}
		plan.PlanSHA256, err = planHash(plan)
		if err != nil {
			return nil, err
		}
		if err := VerifyPlan(plan); err != nil {
			return nil, err
		}
		plans[partition] = plan
	}
	return plans, nil
}

func repeatHash(value byte) string { return "sha256:" + strings.Repeat(string(value), 64) }
