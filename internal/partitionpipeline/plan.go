package partitionpipeline

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/david22573/ak-engine/internal/qualificationrunner"
)

type CheckpointInput struct {
	SourceRoot                string
	ProspectiveSourceRoot     string
	CheckpointRelativePath    string
	CheckpointID              string
	CheckpointSHA256          string
	HistorianCommit           string
	HistorianTree             string
	SourceIdentitySHA256      string
	ReacquisitionProtocol     HashIdentity
	PreAcquisitionSealSHA256  string
	SealedBinarySHA256        string
	AbandonedEvidenceRegistry HashIdentity
	AvailabilityCutoff        time.Time
}

type checkpointManifest struct {
	SchemaVersion           string                     `json:"schema_version"`
	DatasetID               string                     `json:"dataset_id"`
	GenerationID            string                     `json:"generation_id"`
	CreatedAtUTC            time.Time                  `json:"created_at_utc"`
	CoverageStartUTC        time.Time                  `json:"coverage_start_utc"`
	CoverageEndUTC          time.Time                  `json:"coverage_end_utc"`
	EvaluationCutoffFloor   time.Time                  `json:"evaluation_cutoff_floor"`
	RequiredSymbols         []string                   `json:"required_symbol_universe"`
	SourceSchemaHash        string                     `json:"source_schema_hash"`
	AvailabilityPolicyHash  string                     `json:"availability_policy_hash"`
	CoveragePolicyHash      string                     `json:"coverage_policy_hash"`
	ManifestContractHash    string                     `json:"manifest_contract_hash"`
	IngestionReceiptHash    string                     `json:"ingestion_receipt_hash"`
	P4ActivationHash        string                     `json:"p4_activation_hash"`
	P4LiveChainTerminal     string                     `json:"p4_live_receipt_chain_terminal"`
	P4LiveAuthorityTerminal string                     `json:"p4_live_authority_chain_terminal"`
	BackfillChainGenesis    string                     `json:"r1p5r_reacquisition_receipt_chain_genesis"`
	BackfillChainTerminal   string                     `json:"r1p5r_reacquisition_receipt_chain_terminal"`
	RepairCommit            string                     `json:"repair_implementation_commit"`
	SourceSealCommit        string                     `json:"source_seal_commit"`
	SealedBinarySHA256      string                     `json:"sealed_binary_sha256"`
	AbandonedRegistryHash   string                     `json:"abandoned_evidence_registry_hash"`
	P4CollectorCommit       string                     `json:"p4_collector_source_commit"`
	ProtocolHash            string                     `json:"protocol_hash"`
	ExposureLedgerHash      string                     `json:"exposure_ledger_hash"`
	DailyPartitionPaths     []string                   `json:"daily_partition_relative_paths"`
	DailyPartitionHashes    []string                   `json:"daily_partition_hashes"`
	MissingPartitions       []string                   `json:"missing_partitions"`
	PhysicalComplete        bool                       `json:"physical_complete"`
	PITEvidenceComplete     bool                       `json:"pit_evidence_complete"`
	CompleteDays            int                        `json:"complete_eligible_utc_days"`
	MissingIntervalCount    int                        `json:"missing_interval_count"`
	ConflictCount           int                        `json:"conflict_count"`
	EvidenceGapCount        int                        `json:"evidence_gap_count"`
	ClockErrorCount         int                        `json:"clock_error_count"`
	PerSymbol               []checkpointSymbolCoverage `json:"per_symbol_coverage"`
	CheckpointHash          string                     `json:"checkpoint_hash"`
}

type checkpointSymbolCoverage struct {
	Symbol               string    `json:"symbol"`
	StartUTC             time.Time `json:"start_utc"`
	EndUTC               time.Time `json:"end_utc"`
	ObservedCandles      int       `json:"observed_candles"`
	CompleteUTCDateCount int       `json:"complete_utc_day_count"`
	MissingIntervals     int       `json:"missing_interval_count"`
	DuplicateCount       int       `json:"duplicate_count"`
	ConflictCount        int       `json:"conflict_count"`
	EvidenceGapCount     int       `json:"evidence_gap_count"`
}

type dailyManifest struct {
	SchemaVersion      string   `json:"schema_version"`
	Symbol             string   `json:"symbol"`
	UTCDate            string   `json:"utc_date"`
	ExpectedRows       int      `json:"expected_rows"`
	ObservedRows       int      `json:"observed_rows"`
	MissingIntervals   []any    `json:"missing_intervals"`
	DuplicateCount     int      `json:"duplicate_count"`
	ConflictCount      int      `json:"conflict_count"`
	SchemaFailureCount int      `json:"schema_failure_count"`
	EvidenceGapCount   int      `json:"evidence_gap_count"`
	ClockErrorCount    int      `json:"clock_error_count"`
	ReceiptHashes      []string `json:"receipt_hashes"`
	FragmentHashes     []string `json:"fragment_hashes"`
	PhysicalStatus     string   `json:"physical_status"`
	PITStatus          string   `json:"pit_evidence_status"`
	Eligibility        string   `json:"eligibility_classification"`
	PartitionHash      string   `json:"partition_hash"`
}

type sourceReceipt struct {
	SchemaVersion           string    `json:"schema_version"`
	Symbol                  string    `json:"symbol"`
	ReceiptHash             string    `json:"receipt_hash"`
	NormalizedFragmentHash  string    `json:"normalized_fragment_hash"`
	CurrentReceiptChainHash string    `json:"current_receipt_chain_hash"`
	FragmentHash            string    `json:"fragment_hash"`
	FragmentPath            string    `json:"fragment_relative_path"`
	ObservedAvailableAtUTC  time.Time `json:"observed_available_at_utc"`
	RequestedStartUTC       time.Time `json:"requested_start_utc"`
	RequestedEndUTC         time.Time `json:"requested_end_exclusive_utc"`
	ParsedRowCount          int       `json:"parsed_row_count"`
	ParsedRecordCount       int       `json:"parsed_record_count"`
	FirstCandleOpenUTC      time.Time `json:"first_candle_open_time_utc"`
	LastCandleCloseUTC      time.Time `json:"last_candle_close_time_utc"`
	FinalCandleCloseUTC     time.Time `json:"final_candle_close_time_utc"`
	AuthorityReceipt        struct {
		CoveredPeriodStart time.Time `json:"covered_period_start"`
		CoveredPeriodEnd   time.Time `json:"covered_period_end"`
	} `json:"accepted_authority_receipt"`
}

const (
	backfillReceiptSchema    = "ak-historian.pr4b0-r1p5r.historical-reacquisition-receipt.v1"
	prospectiveReceiptSchema = "ak-historian.pr4b0-r1p4.acquisition-receipt.v1"
)

var acceptedIntervals = map[string]Interval{
	"DEVELOPMENT":   {Start: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), End: time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)},
	"VALIDATION":    {Start: time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC), End: time.Date(2026, 5, 29, 0, 0, 0, 0, time.UTC)},
	"FINAL_HOLDOUT": {Start: time.Date(2026, 5, 29, 0, 0, 0, 0, time.UTC), End: time.Date(2026, 7, 17, 0, 0, 0, 0, time.UTC)},
}

const (
	acceptedCheckpointID             = "r1p5r-checkpoint-20260717T073628Z"
	acceptedCheckpointSHA256         = "sha256:bef53d11aa9ce9a6dad61b89ef7ace063b6da812ff92208d463c6ecfbfe8f29c"
	acceptedHistorianCommit          = "14e0b569cc41e3517d2393d87d39c00559ac408e"
	acceptedHistorianTree            = "74182d50c8f6dc6639c68cd6af9a82656640701c"
	acceptedSourceIdentitySHA256     = "sha256:d99a88a72b4bfe84c2ae43a4a477724b95fde2b45b381f7b75fc4c107d2a161a"
	acceptedReacquisitionProtocolID  = "ak-historian.pr4b0-r1p5r.reacquisition-protocol.v1"
	acceptedReacquisitionProtocolSHA = "sha256:7fd6c667d97a0ff3387e352d8c8e9b25ef5a744d641147ed66b98df75dcc0e1a"
	acceptedPreAcquisitionSealSHA    = "sha256:8046a306c80c127bd631df6eb4ae07ef587c350d4ac3a5f3be33f84a0f4681c9"
	acceptedSealedBinarySHA          = "sha256:c10fdf10255a8c88c817d5189b20ca7411be1fcd2ae64df8c07e2d1934054ae6"
	acceptedAbandonedRegistryID      = "ak-historian.pr4b0-r1p5r.abandoned-evidence-registry.v1"
	acceptedAbandonedRegistrySHA     = "sha256:f8a47626a234544f34ae59846c330682e78943add69edb171c558287b35417ca"
)

var acceptedAvailabilityCutoff = time.Date(2026, 7, 17, 7, 36, 29, 0, time.UTC)

var syntheticAcceptedIntervals = map[string]Interval{
	"DEVELOPMENT":   {Start: time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC), End: time.Date(2031, 1, 1, 0, 0, 0, 0, time.UTC)},
	"VALIDATION":    {Start: time.Date(2031, 1, 1, 0, 0, 0, 0, time.UTC), End: time.Date(2032, 1, 1, 0, 0, 0, 0, time.UTC)},
	"FINAL_HOLDOUT": {Start: time.Date(2032, 1, 1, 0, 0, 0, 0, time.UTC), End: time.Date(2033, 1, 1, 0, 0, 0, 0, time.UTC)},
}

func CreatePlan(input CheckpointInput, partition string) (Plan, error) {
	interval, ok := acceptedIntervals[partition]
	if !ok {
		return Plan{}, errors.New("exact DEVELOPMENT, VALIDATION, or FINAL_HOLDOUT partition is required")
	}
	if input.CheckpointID != acceptedCheckpointID || input.CheckpointSHA256 != acceptedCheckpointSHA256 || input.HistorianCommit != acceptedHistorianCommit || input.HistorianTree != acceptedHistorianTree || input.SourceIdentitySHA256 != acceptedSourceIdentitySHA256 || input.ReacquisitionProtocol.ID != acceptedReacquisitionProtocolID || input.ReacquisitionProtocol.SHA256 != acceptedReacquisitionProtocolSHA || input.PreAcquisitionSealSHA256 != acceptedPreAcquisitionSealSHA || input.SealedBinarySHA256 != acceptedSealedBinarySHA || input.AbandonedEvidenceRegistry.ID != acceptedAbandonedRegistryID || input.AbandonedEvidenceRegistry.SHA256 != acceptedAbandonedRegistrySHA || !input.AvailabilityCutoff.Equal(acceptedAvailabilityCutoff) {
		return Plan{}, errors.New("accepted checkpoint or source-chain identity substitution")
	}
	root, err := canonicalRealDirectory(input.SourceRoot)
	if err != nil {
		return Plan{}, err
	}
	prospectiveRoot, err := canonicalRealDirectory(input.ProspectiveSourceRoot)
	if err != nil || prospectiveRoot == root {
		return Plan{}, errors.New("accepted checkpoint requires distinct canonical backfill and prospective source roots")
	}
	checkpointPath, err := secureJoin(root, input.CheckpointRelativePath, true)
	if err != nil {
		return Plan{}, err
	}
	checkpointBytes, err := os.ReadFile(checkpointPath)
	if err != nil {
		return Plan{}, err
	}
	var checkpoint checkpointManifest
	if err := strictJSON(checkpointBytes, &checkpoint); err != nil {
		return Plan{}, fmt.Errorf("checkpoint manifest: %w", err)
	}
	checkpointHash, hashErr := historianCanonicalHash(checkpoint, "checkpoint_hash")
	canonicalCheckpoint, marshalErr := json.Marshal(checkpoint)
	if checkpoint.SchemaVersion != "ak-historian.pr4b0-r1p5r.coverage-checkpoint.v1" || checkpoint.GenerationID != input.CheckpointID || checkpoint.CheckpointHash != input.CheckpointSHA256 || checkpointHash != checkpoint.CheckpointHash || hashErr != nil || marshalErr != nil || !bytes.Equal(bytes.TrimSpace(checkpointBytes), canonicalCheckpoint) || len(checkpoint.DailyPartitionHashes) != 1773 || len(checkpoint.DailyPartitionPaths) != 1773 || checkpoint.CompleteDays != 197 || checkpoint.MissingIntervalCount != 0 || checkpoint.ConflictCount != 0 || checkpoint.EvidenceGapCount != 0 || checkpoint.ClockErrorCount != 0 || !checkpoint.PhysicalComplete || !checkpoint.PITEvidenceComplete {
		return Plan{}, fmt.Errorf("accepted checkpoint structural identity or completeness mismatch: generation=%t field_hash=%t canonical_hash=%t canonical_bytes=%t hash_error=%v marshal_error=%v hashes=%d paths=%d days=%d physical=%t pit=%t", checkpoint.GenerationID == input.CheckpointID, checkpoint.CheckpointHash == input.CheckpointSHA256, checkpointHash == checkpoint.CheckpointHash, bytes.Equal(bytes.TrimSpace(checkpointBytes), canonicalCheckpoint), hashErr, marshalErr, len(checkpoint.DailyPartitionHashes), len(checkpoint.DailyPartitionPaths), checkpoint.CompleteDays, checkpoint.PhysicalComplete, checkpoint.PITEvidenceComplete)
	}
	if checkpoint.SealedBinarySHA256 == "" || checkpoint.SealedBinarySHA256 != input.SealedBinarySHA256 || checkpoint.AbandonedRegistryHash != input.AbandonedEvidenceRegistry.SHA256 || checkpoint.ProtocolHash != input.ReacquisitionProtocol.SHA256 {
		return Plan{}, errors.New("checkpoint source-chain identity mismatch")
	}
	universe, err := qualificationrunner.V00UniverseContract()
	if err != nil {
		return Plan{}, err
	}
	if !reflect.DeepEqual(checkpoint.RequiredSymbols, universe.DatasetRequiredSymbols) || !input.AvailabilityCutoff.After(checkpoint.CoverageEndUTC) {
		return Plan{}, errors.New("checkpoint universe or availability cutoff mismatch")
	}
	receipts, err := indexReceipts(root, prospectiveRoot)
	if err != nil {
		return Plan{}, err
	}
	members := make([]SourceManifest, 0, int(interval.End.Sub(interval.Start)/(24*time.Hour))*len(universe.DatasetRequiredSymbols))
	partitionHashes := map[string]struct{}{}
	for _, hash := range checkpoint.DailyPartitionHashes {
		if !validSHA(hash) {
			return Plan{}, errors.New("checkpoint partition hash set is invalid")
		}
		partitionHashes[hash] = struct{}{}
	}
	if len(partitionHashes) != len(checkpoint.DailyPartitionHashes) {
		return Plan{}, errors.New("checkpoint partition hash set contains duplicates")
	}
	seenPartitionHashes := map[string]struct{}{}
	for _, relative := range checkpoint.DailyPartitionPaths {
		path, err := secureJoin(root, relative, true)
		if err != nil {
			return Plan{}, err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return Plan{}, err
		}
		var daily dailyManifest
		if err := strictJSON(data, &daily); err != nil {
			return Plan{}, err
		}
		date, err := time.Parse("2006-01-02", daily.UTCDate)
		if err != nil {
			return Plan{}, err
		}
		date = date.UTC()
		dailyHash, hashErr := historianCanonicalHash(daily, "partition_hash")
		_, checkpointMember := partitionHashes[daily.PartitionHash]
		_, duplicatePartition := seenPartitionHashes[daily.PartitionHash]
		if hashErr != nil || dailyHash != daily.PartitionHash || !checkpointMember || duplicatePartition || !strings.HasSuffix(relative, "-"+strings.TrimPrefix(daily.PartitionHash, "sha256:")+".json") || daily.SchemaVersion != "ak-historian.pr4b0-r1p5r.daily-partition.v1" || !contains(universe.DatasetRequiredSymbols, daily.Symbol) || daily.ExpectedRows != 1440 || daily.ObservedRows != 1440 || len(daily.MissingIntervals) != 0 || daily.DuplicateCount != 0 || daily.ConflictCount != 0 || daily.SchemaFailureCount != 0 || daily.EvidenceGapCount != 0 || daily.ClockErrorCount != 0 || daily.PhysicalStatus != "PHYSICAL_COMPLETE" || daily.PITStatus != "PIT_EVIDENCE_COMPLETE" || daily.Eligibility != "UNEXPOSED_PIT_EVIDENCE_COMPLETE" {
			return Plan{}, errors.New("daily source manifest identity or completeness mismatch")
		}
		seenPartitionHashes[daily.PartitionHash] = struct{}{}
		if date.Before(interval.Start) || !date.Before(interval.End) {
			continue
		}
		member := SourceManifest{Symbol: daily.Symbol, UTCDate: daily.UTCDate, RelativePath: relative, FileSHA256: byteHash(data), PartitionSHA256: daily.PartitionHash, ExpectedRows: daily.ExpectedRows}
		if len(daily.ReceiptHashes) != len(daily.FragmentHashes) || len(daily.ReceiptHashes) == 0 {
			return Plan{}, errors.New("daily manifest receipt/fragment membership mismatch")
		}
		fragmentSet := map[string]struct{}{}
		for _, fragmentHash := range daily.FragmentHashes {
			if !validSHA(fragmentHash) {
				return Plan{}, errors.New("daily manifest fragment hash set is invalid")
			}
			fragmentSet[fragmentHash] = struct{}{}
		}
		if len(fragmentSet) != len(daily.FragmentHashes) {
			return Plan{}, errors.New("daily manifest fragment hash set contains duplicates")
		}
		seenFragments := map[string]struct{}{}
		for _, receiptHash := range daily.ReceiptHashes {
			indexed, ok := receipts[receiptHash]
			_, fragmentMember := fragmentSet[indexed.fragmentHash]
			if !ok || !fragmentMember {
				return Plan{}, fmt.Errorf("planned receipt or fragment is missing from checkpoint source chain: manifest=%s receipt=%s receipt_found=%t", relative, receiptHash, ok)
			}
			sourceRoot, err := sourceRootByID(root, prospectiveRoot, indexed.sourceRootID)
			if err != nil {
				return Plan{}, err
			}
			if _, err := secureJoin(sourceRoot, indexed.fragmentPath, true); err != nil {
				return Plan{}, err
			}
			if indexed.observedAvailableAtUTC.IsZero() || indexed.observedAvailableAtUTC.After(input.AvailabilityCutoff) {
				return Plan{}, errors.New("planned source receipt violates the availability cutoff")
			}
			member.ReceiptArtifacts = append(member.ReceiptArtifacts, SourceArtifact{SourceRootID: indexed.sourceRootID, RelativePath: indexed.relative, CanonicalSHA256: receiptHash, ObservedAvailableAtUTC: indexed.observedAvailableAtUTC})
			member.FragmentArtifacts = append(member.FragmentArtifacts, SourceArtifact{SourceRootID: indexed.sourceRootID, RelativePath: indexed.fragmentPath, CanonicalSHA256: indexed.fragmentHash, Encoding: indexed.encoding, ReceiptSHA256: receiptHash, ObservedAvailableAtUTC: indexed.observedAvailableAtUTC})
			seenFragments[indexed.fragmentHash] = struct{}{}
		}
		if len(seenFragments) != len(fragmentSet) {
			return Plan{}, errors.New("daily receipt chain does not cover the exact fragment set")
		}
		members = append(members, member)
	}
	if len(seenPartitionHashes) != len(partitionHashes) {
		return Plan{}, errors.New("checkpoint daily path set does not cover its exact partition hash set")
	}
	sort.Slice(members, func(i, j int) bool {
		if members[i].UTCDate != members[j].UTCDate {
			return members[i].UTCDate < members[j].UTCDate
		}
		return members[i].Symbol < members[j].Symbol
	})
	days := int(interval.End.Sub(interval.Start) / (24 * time.Hour))
	if len(members) != days*len(universe.DatasetRequiredSymbols) {
		return Plan{}, errors.New("partition source-manifest cardinality mismatch")
	}
	plan := Plan{SchemaVersion: PlanSchemaVersion, Checkpoint: HashIdentity{input.CheckpointID, input.CheckpointSHA256}, HistorianCommit: input.HistorianCommit, HistorianTree: input.HistorianTree, SourceIdentitySHA256: input.SourceIdentitySHA256, ReacquisitionProtocol: input.ReacquisitionProtocol, PreAcquisitionSealSHA256: input.PreAcquisitionSealSHA256, SealedBinarySHA256: checkpoint.SealedBinarySHA256, AbandonedEvidenceRegistry: input.AbandonedEvidenceRegistry, DatasetRequiredSymbols: universe.DatasetRequiredSymbols, CandidateTargetSymbols: universe.CandidateTargetSymbols, ContextOnlySymbols: universe.ContextOnlySymbols, UniverseContractSHA256: universe.ContractSHA256, EligibleInterval: Interval{checkpoint.CoverageStartUTC, checkpoint.CoverageEndUTC}, PartitionName: partition, PartitionInterval: interval, SourceManifests: members, ExpectedStructuralDays: days, SchemaIdentitySHA256: checkpoint.SourceSchemaHash, OutputFormat: OutputFormat, OrderingPolicy: OrderingPolicy, OutputPathPolicy: OutputPathPolicy, SymlinkPolicy: SymlinkPolicy, CachePolicy: CachePolicy, AvailabilityCutoff: input.AvailabilityCutoff.UTC(), SourceRoot: root, ProspectiveSourceRoot: prospectiveRoot}
	plan.PlanSHA256, err = planHash(plan)
	if err != nil {
		return Plan{}, err
	}
	if err := VerifyPlan(plan); err != nil {
		return Plan{}, err
	}
	return plan, nil
}

func historianCanonicalHash(value any, hashField string) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return historianCanonicalJSONHash(data, hashField)
}

func historianCanonicalJSONHash(data []byte, hashField string) (string, error) {
	var object map[string]any
	if err := json.Unmarshal(data, &object); err != nil {
		return "", err
	}
	delete(object, hashField)
	data, err := json.Marshal(object)
	if err != nil {
		return "", err
	}
	return byteHash(data), nil
}

type indexedReceipt struct {
	sourceRootID           string
	relative               string
	canonicalSHA256        string
	fragmentPath           string
	fragmentHash           string
	encoding               string
	observedAvailableAtUTC time.Time
	symbol                 string
	parentStartUTC         time.Time
	parentEndUTC           time.Time
	parentRowCount         int
	firstCandleOpenUTC     time.Time
	lastCandleCloseUTC     time.Time
}

func indexReceipts(backfillRoot, prospectiveRoot string) (map[string]indexedReceipt, error) {
	result := map[string]indexedReceipt{}
	if err := indexReceiptRoot(result, backfillRoot, CheckpointSourceRootID); err != nil {
		return nil, err
	}
	if err := indexReceiptRoot(result, prospectiveRoot, ProspectiveSourceRootID); err != nil {
		return nil, err
	}
	return result, nil
}

func indexReceiptRoot(result map[string]indexedReceipt, root, rootID string) error {
	base, err := secureJoin(root, "receipts", false)
	if err != nil {
		return err
	}
	return filepath.WalkDir(base, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return errUnsafePath
		}
		if entry.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".json" {
			return errors.New("unregistered receipt cache or file")
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		indexed, err := decodeIndexedReceipt(data, rootID, relative)
		if err != nil {
			return err
		}
		if _, duplicate := result[indexed.canonicalSHA256]; duplicate {
			return errors.New("duplicate receipt identity")
		}
		result[indexed.canonicalSHA256] = indexed
		return nil
	})
}

func decodeIndexedReceipt(data []byte, rootID, relative string) (indexedReceipt, error) {
	var receipt sourceReceipt
	if err := json.Unmarshal(data, &receipt); err != nil {
		return indexedReceipt{}, err
	}
	identity, fragmentHash, encoding, expectedSchema, hashField := receipt.ReceiptHash, receipt.NormalizedFragmentHash, BackfillFragmentEncoding, backfillReceiptSchema, "receipt_hash"
	parentStart, parentEnd, rowCount := receipt.RequestedStartUTC.UTC(), receipt.RequestedEndUTC.UTC(), receipt.ParsedRowCount
	firstOpen, lastClose := receipt.FirstCandleOpenUTC.UTC(), receipt.LastCandleCloseUTC.UTC()
	if rootID == ProspectiveSourceRootID {
		identity, fragmentHash, encoding, expectedSchema, hashField = receipt.CurrentReceiptChainHash, receipt.FragmentHash, ProspectiveFragmentEncoding, prospectiveReceiptSchema, "current_receipt_chain_hash"
		parentStart, parentEnd, rowCount = receipt.AuthorityReceipt.CoveredPeriodStart.UTC(), receipt.AuthorityReceipt.CoveredPeriodEnd.UTC(), receipt.ParsedRecordCount
		firstOpen, lastClose = receipt.FirstCandleOpenUTC.UTC(), receipt.FinalCandleCloseUTC.UTC()
	} else if rootID != CheckpointSourceRootID {
		return indexedReceipt{}, errors.New("unregistered source root identity")
	}
	canonicalIdentity, err := historianCanonicalJSONHash(data, hashField)
	if err != nil || receipt.SchemaVersion != expectedSchema || receipt.Symbol == "" || !validSHA(identity) || canonicalIdentity != identity || !validSHA(fragmentHash) || receipt.FragmentPath == "" || receipt.ObservedAvailableAtUTC.IsZero() || parentStart.IsZero() || !parentStart.Before(parentEnd) || rowCount <= 0 || firstOpen.IsZero() || lastClose.IsZero() || !firstOpen.Equal(parentStart) || !lastClose.Add(time.Millisecond).Equal(parentEnd) || int(parentEnd.Sub(parentStart)/time.Minute) != rowCount {
		return indexedReceipt{}, errors.New("source receipt hash is invalid")
	}
	return indexedReceipt{sourceRootID: rootID, relative: relative, canonicalSHA256: identity, fragmentPath: receipt.FragmentPath, fragmentHash: fragmentHash, encoding: encoding, observedAvailableAtUTC: receipt.ObservedAvailableAtUTC.UTC(), symbol: receipt.Symbol, parentStartUTC: parentStart, parentEndUTC: parentEnd, parentRowCount: rowCount, firstCandleOpenUTC: firstOpen, lastCandleCloseUTC: lastClose}, nil
}

func VerifyPlan(plan Plan) error {
	if plan.SchemaVersion == PreparedPlanSchemaVersion {
		return verifyPreparedPlan(plan)
	}
	return verifyParentPlan(plan)
}

func verifyParentPlan(plan Plan) error {
	if plan.SchemaVersion != PlanSchemaVersion || !validSHA(plan.Checkpoint.SHA256) || len(plan.HistorianCommit) != 40 || len(plan.HistorianTree) != 40 || !validSHA(plan.SourceIdentitySHA256) || !validSHA(plan.PreAcquisitionSealSHA256) || !validSHA(plan.SealedBinarySHA256) || !validSHA(plan.UniverseContractSHA256) || !intervalValid(plan.EligibleInterval) || !intervalValid(plan.PartitionInterval) || !validSHA(plan.SchemaIdentitySHA256) || plan.OutputFormat != OutputFormat || plan.OrderingPolicy != OrderingPolicy || plan.OutputPathPolicy != OutputPathPolicy || plan.SymlinkPolicy != SymlinkPolicy || plan.CachePolicy != CachePolicy || plan.AvailabilityCutoff.IsZero() {
		return errors.New("partition plan is incomplete or uses a noncanonical policy")
	}
	root, err := canonicalRealDirectory(plan.SourceRoot)
	if err != nil || root != plan.SourceRoot {
		return errors.New("partition plan source root is alternate or unsafe")
	}
	prospectiveRoot := ""
	if plan.SyntheticFixture {
		if plan.ProspectiveSourceRoot != "" {
			return errors.New("synthetic partition plan may not bind a prospective source root")
		}
	} else {
		prospectiveRoot, err = canonicalRealDirectory(plan.ProspectiveSourceRoot)
		if err != nil || prospectiveRoot != plan.ProspectiveSourceRoot || prospectiveRoot == root {
			return errors.New("partition plan prospective source root is alternate or unsafe")
		}
	}
	wantInterval, ok := acceptedIntervals[plan.PartitionName]
	if !ok {
		return errors.New("partition plan name substitution")
	}
	if plan.SyntheticFixture {
		wantSynthetic, exists := syntheticAcceptedIntervals[plan.PartitionName]
		if !exists || (!reflect.DeepEqual(plan.PartitionInterval, wantSynthetic) && !reflect.DeepEqual(plan.PartitionInterval, wantInterval)) {
			return errors.New("synthetic partition plan boundary substitution")
		}
	} else if !reflect.DeepEqual(plan.PartitionInterval, wantInterval) {
		return errors.New("partition plan boundary substitution")
	}
	universe, err := qualificationrunner.V00UniverseContract()
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(plan.DatasetRequiredSymbols, universe.DatasetRequiredSymbols) || !reflect.DeepEqual(plan.CandidateTargetSymbols, universe.CandidateTargetSymbols) || !reflect.DeepEqual(plan.ContextOnlySymbols, universe.ContextOnlySymbols) || plan.UniverseContractSHA256 != universe.ContractSHA256 {
		return errors.New("partition plan universe substitution")
	}
	if !plan.SyntheticFixture && len(plan.SourceManifests) != plan.ExpectedStructuralDays*len(plan.DatasetRequiredSymbols) {
		return errors.New("partition plan manifest membership is incomplete")
	}
	if plan.SyntheticFixture && len(plan.SourceManifests) != len(plan.DatasetRequiredSymbols) {
		return errors.New("synthetic partition plan has missing or extra symbol membership")
	}
	for i, member := range plan.SourceManifests {
		if !contains(plan.DatasetRequiredSymbols, member.Symbol) || !validSHA(member.FileSHA256) || !validSHA(member.PartitionSHA256) || len(member.ReceiptArtifacts) == 0 || len(member.ReceiptArtifacts) != len(member.FragmentArtifacts) {
			return errors.New("partition plan source membership invalid")
		}
		if i > 0 && (plan.SourceManifests[i-1].UTCDate > member.UTCDate || (plan.SourceManifests[i-1].UTCDate == member.UTCDate && plan.SourceManifests[i-1].Symbol >= member.Symbol)) {
			return errors.New("partition plan manifest order is noncanonical")
		}
		path, err := secureJoin(root, member.RelativePath, true)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil || byteHash(data) != member.FileSHA256 {
			return errors.New("planned source manifest hash changed")
		}
		for artifactIndex := range member.ReceiptArtifacts {
			receiptRef := member.ReceiptArtifacts[artifactIndex]
			fragmentRef := member.FragmentArtifacts[artifactIndex]
			if !validSHA(receiptRef.CanonicalSHA256) || !validSHA(fragmentRef.CanonicalSHA256) || receiptRef.SourceRootID == "" || fragmentRef.SourceRootID != receiptRef.SourceRootID || fragmentRef.ReceiptSHA256 != receiptRef.CanonicalSHA256 || receiptRef.Encoding != "" || receiptRef.ReceiptSHA256 != "" || receiptRef.ObservedAvailableAtUTC.IsZero() || !receiptRef.ObservedAvailableAtUTC.Equal(fragmentRef.ObservedAvailableAtUTC) || receiptRef.ObservedAvailableAtUTC.After(plan.AvailabilityCutoff) {
				return errors.New("partition plan receipt-to-fragment binding is invalid")
			}
			artifactRoot, rootErr := sourceRootByID(root, prospectiveRoot, receiptRef.SourceRootID)
			if rootErr != nil {
				return rootErr
			}
			if plan.SyntheticFixture {
				if receiptRef.SourceRootID != CheckpointSourceRootID || fragmentRef.Encoding != SyntheticFragmentEncoding {
					return errors.New("synthetic source artifact identity is invalid")
				}
			} else {
				receiptPath, pathErr := secureJoin(artifactRoot, receiptRef.RelativePath, true)
				if pathErr != nil {
					return pathErr
				}
				receiptData, readErr := os.ReadFile(receiptPath)
				if readErr != nil {
					return readErr
				}
				indexed, decodeErr := decodeIndexedReceipt(receiptData, receiptRef.SourceRootID, receiptRef.RelativePath)
				if decodeErr != nil || indexed.canonicalSHA256 != receiptRef.CanonicalSHA256 || indexed.fragmentPath != fragmentRef.RelativePath || indexed.fragmentHash != fragmentRef.CanonicalSHA256 || indexed.encoding != fragmentRef.Encoding || !indexed.observedAvailableAtUTC.Equal(receiptRef.ObservedAvailableAtUTC) {
					return errors.New("planned source receipt binding changed")
				}
			}
			if _, pathErr := secureJoin(artifactRoot, fragmentRef.RelativePath, true); pathErr != nil {
				return pathErr
			}
		}
	}
	want, err := planHash(plan)
	if err != nil {
		return err
	}
	if plan.PlanSHA256 != want {
		return errors.New("partition plan canonical hash mismatch")
	}
	return nil
}

func EncodePlan(plan Plan) ([]byte, error) {
	if err := VerifyPlan(plan); err != nil {
		return nil, err
	}
	data, err := json.Marshal(plan)
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}
func DecodePlan(data []byte) (Plan, error) {
	var plan Plan
	if err := strictJSON(data, &plan); err != nil {
		return Plan{}, err
	}
	canonical, err := EncodePlan(plan)
	if err != nil {
		return Plan{}, err
	}
	if !bytes.Equal(data, canonical) {
		return Plan{}, errors.New("partition plan bytes are not canonical")
	}
	return plan, nil
}
func planHash(plan Plan) (string, error) { plan.PlanSHA256 = ""; return canonicalHash(plan) }

func sourceRootByID(backfillRoot, prospectiveRoot, rootID string) (string, error) {
	switch rootID {
	case CheckpointSourceRootID:
		return backfillRoot, nil
	case ProspectiveSourceRootID:
		if prospectiveRoot != "" {
			return prospectiveRoot, nil
		}
	}
	return "", errors.New("unregistered source root identity")
}

func canonicalRealDirectory(path string) (string, error) {
	if path == "" || filepath.Clean(path) != path {
		return "", errUnsafePath
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	if err := rejectSymlinkComponents(abs); err != nil {
		return "", err
	}
	info, err := os.Stat(abs)
	if err != nil || !info.IsDir() {
		return "", errUnsafePath
	}
	return abs, nil
}

func secureJoin(root, relative string, regular bool) (string, error) {
	if relative == "" || filepath.IsAbs(relative) || strings.Contains(relative, "..") || strings.Contains(relative, `\`) {
		return "", errUnsafePath
	}
	path := filepath.Join(root, filepath.FromSlash(relative))
	rel, err := filepath.Rel(root, path)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", errUnsafePath
	}
	if err := rejectSymlinkComponents(path); err != nil {
		return "", err
	}
	info, err := os.Lstat(path)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || (regular && !info.Mode().IsRegular()) || (!regular && !info.IsDir()) {
		return "", errUnsafePath
	}
	return path, nil
}

func rejectSymlinkComponents(path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	current := string(filepath.Separator)
	for _, component := range strings.Split(strings.TrimPrefix(abs, string(filepath.Separator)), string(filepath.Separator)) {
		if component == "" {
			continue
		}
		current = filepath.Join(current, component)
		info, err := os.Lstat(current)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return errUnsafePath
		}
	}
	return nil
}

func strictJSON(data []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return errors.New("trailing JSON data")
	}
	return nil
}
func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
