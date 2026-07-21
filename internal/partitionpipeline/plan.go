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
	SchemaVersion         string    `json:"schema_version"`
	CheckpointHash        string    `json:"checkpoint_hash"`
	CoverageStartUTC      time.Time `json:"coverage_start_utc"`
	CoverageEndUTC        time.Time `json:"coverage_end_utc"`
	RequiredSymbols       []string  `json:"required_symbol_universe"`
	CompleteDays          int       `json:"complete_eligible_utc_days"`
	DailyPartitionHashes  []string  `json:"daily_partition_hashes"`
	DailyPartitionPaths   []string  `json:"daily_partition_relative_paths"`
	SourceSchemaHash      string    `json:"source_schema_hash"`
	SealedBinarySHA256    string    `json:"sealed_binary_sha256"`
	SourceSealCommit      string    `json:"source_seal_commit"`
	RepairCommit          string    `json:"repair_implementation_commit"`
	ProtocolHash          string    `json:"protocol_hash"`
	AbandonedRegistryHash string    `json:"abandoned_evidence_registry_hash"`
	MissingIntervalCount  int       `json:"missing_interval_count"`
	ConflictCount         int       `json:"conflict_count"`
	EvidenceGapCount      int       `json:"evidence_gap_count"`
	ClockErrorCount       int       `json:"clock_error_count"`
	PhysicalComplete      bool      `json:"physical_complete"`
	PITEvidenceComplete   bool      `json:"pit_evidence_complete"`
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
	ReceiptHash  string `json:"receipt_hash"`
	FragmentHash string `json:"normalized_fragment_hash"`
	FragmentPath string `json:"fragment_relative_path"`
}

var acceptedIntervals = map[string]Interval{
	"DEVELOPMENT":   {Start: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), End: time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)},
	"VALIDATION":    {Start: time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC), End: time.Date(2026, 5, 29, 0, 0, 0, 0, time.UTC)},
	"FINAL_HOLDOUT": {Start: time.Date(2026, 5, 29, 0, 0, 0, 0, time.UTC), End: time.Date(2026, 7, 17, 0, 0, 0, 0, time.UTC)},
}

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
	root, err := canonicalRealDirectory(input.SourceRoot)
	if err != nil {
		return Plan{}, err
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
	if err := json.Unmarshal(checkpointBytes, &checkpoint); err != nil {
		return Plan{}, fmt.Errorf("checkpoint manifest: %w", err)
	}
	if checkpoint.SchemaVersion != "ak-historian.pr4b0-r1p5r.coverage-checkpoint.v1" || checkpoint.CheckpointHash != input.CheckpointSHA256 || input.CheckpointID == "" || len(checkpoint.DailyPartitionHashes) != 1773 || len(checkpoint.DailyPartitionPaths) != 1773 || checkpoint.CompleteDays != 197 || checkpoint.MissingIntervalCount != 0 || checkpoint.ConflictCount != 0 || checkpoint.EvidenceGapCount != 0 || checkpoint.ClockErrorCount != 0 || !checkpoint.PhysicalComplete || !checkpoint.PITEvidenceComplete {
		return Plan{}, errors.New("accepted checkpoint structural identity or completeness mismatch")
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
	receipts, err := indexReceipts(root)
	if err != nil {
		return Plan{}, err
	}
	members := make([]SourceManifest, 0, int(interval.End.Sub(interval.Start)/(24*time.Hour))*len(universe.DatasetRequiredSymbols))
	for i, relative := range checkpoint.DailyPartitionPaths {
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
		if daily.PartitionHash != checkpoint.DailyPartitionHashes[i] || daily.SchemaVersion != "ak-historian.pr4b0-r1p5r.daily-partition.v1" || !contains(universe.DatasetRequiredSymbols, daily.Symbol) || daily.ExpectedRows != 1440 || daily.ObservedRows != 1440 || len(daily.MissingIntervals) != 0 || daily.DuplicateCount != 0 || daily.ConflictCount != 0 || daily.SchemaFailureCount != 0 || daily.EvidenceGapCount != 0 || daily.ClockErrorCount != 0 || daily.PhysicalStatus != "PHYSICAL_COMPLETE" || daily.PITStatus != "PIT_EVIDENCE_COMPLETE" || daily.Eligibility != "UNEXPOSED_PIT_EVIDENCE_COMPLETE" {
			return Plan{}, errors.New("daily source manifest identity or completeness mismatch")
		}
		if date.Before(interval.Start) || !date.Before(interval.End) {
			continue
		}
		member := SourceManifest{Symbol: daily.Symbol, UTCDate: daily.UTCDate, RelativePath: relative, FileSHA256: byteHash(data), PartitionSHA256: daily.PartitionHash, ExpectedRows: daily.ExpectedRows}
		if len(daily.ReceiptHashes) != len(daily.FragmentHashes) || len(daily.ReceiptHashes) == 0 {
			return Plan{}, errors.New("daily manifest receipt/fragment membership mismatch")
		}
		for j, receiptHash := range daily.ReceiptHashes {
			indexed, ok := receipts[receiptHash]
			if !ok || indexed.receipt.FragmentHash != daily.FragmentHashes[j] {
				return Plan{}, errors.New("planned receipt or fragment is missing from checkpoint source chain")
			}
			if _, err := secureJoin(root, indexed.receipt.FragmentPath, true); err != nil {
				return Plan{}, err
			}
			member.ReceiptArtifacts = append(member.ReceiptArtifacts, SourceArtifact{indexed.relative, receiptHash})
			member.FragmentArtifacts = append(member.FragmentArtifacts, SourceArtifact{indexed.receipt.FragmentPath, indexed.receipt.FragmentHash})
		}
		members = append(members, member)
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
	plan := Plan{SchemaVersion: PlanSchemaVersion, Checkpoint: HashIdentity{input.CheckpointID, input.CheckpointSHA256}, HistorianCommit: input.HistorianCommit, HistorianTree: input.HistorianTree, SourceIdentitySHA256: input.SourceIdentitySHA256, ReacquisitionProtocol: input.ReacquisitionProtocol, PreAcquisitionSealSHA256: input.PreAcquisitionSealSHA256, SealedBinarySHA256: checkpoint.SealedBinarySHA256, AbandonedEvidenceRegistry: input.AbandonedEvidenceRegistry, DatasetRequiredSymbols: universe.DatasetRequiredSymbols, CandidateTargetSymbols: universe.CandidateTargetSymbols, ContextOnlySymbols: universe.ContextOnlySymbols, UniverseContractSHA256: universe.ContractSHA256, EligibleInterval: Interval{checkpoint.CoverageStartUTC, checkpoint.CoverageEndUTC}, PartitionName: partition, PartitionInterval: interval, SourceManifests: members, ExpectedStructuralDays: days, SchemaIdentitySHA256: checkpoint.SourceSchemaHash, OutputFormat: OutputFormat, OrderingPolicy: OrderingPolicy, OutputPathPolicy: OutputPathPolicy, SymlinkPolicy: SymlinkPolicy, CachePolicy: CachePolicy, AvailabilityCutoff: input.AvailabilityCutoff.UTC(), SourceRoot: root}
	plan.PlanSHA256, err = planHash(plan)
	if err != nil {
		return Plan{}, err
	}
	if err := VerifyPlan(plan); err != nil {
		return Plan{}, err
	}
	return plan, nil
}

type indexedReceipt struct {
	relative string
	receipt  sourceReceipt
}

func indexReceipts(root string) (map[string]indexedReceipt, error) {
	base, err := secureJoin(root, "receipts", false)
	if err != nil {
		return nil, err
	}
	result := map[string]indexedReceipt{}
	err = filepath.WalkDir(base, func(path string, entry os.DirEntry, walkErr error) error {
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
		var receipt sourceReceipt
		if err := json.Unmarshal(data, &receipt); err != nil {
			return err
		}
		if !validSHA(receipt.ReceiptHash) || !validSHA(receipt.FragmentHash) {
			return errors.New("source receipt hash is invalid")
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		if _, duplicate := result[receipt.ReceiptHash]; duplicate {
			return errors.New("duplicate receipt identity")
		}
		result[receipt.ReceiptHash] = indexedReceipt{relative, receipt}
		return nil
	})
	return result, err
}

func VerifyPlan(plan Plan) error {
	if plan.SchemaVersion != PlanSchemaVersion || !validSHA(plan.Checkpoint.SHA256) || len(plan.HistorianCommit) != 40 || len(plan.HistorianTree) != 40 || !validSHA(plan.SourceIdentitySHA256) || !validSHA(plan.PreAcquisitionSealSHA256) || !validSHA(plan.SealedBinarySHA256) || !validSHA(plan.UniverseContractSHA256) || !intervalValid(plan.EligibleInterval) || !intervalValid(plan.PartitionInterval) || !validSHA(plan.SchemaIdentitySHA256) || plan.OutputFormat != OutputFormat || plan.OrderingPolicy != OrderingPolicy || plan.OutputPathPolicy != OutputPathPolicy || plan.SymlinkPolicy != SymlinkPolicy || plan.CachePolicy != CachePolicy || plan.AvailabilityCutoff.IsZero() {
		return errors.New("partition plan is incomplete or uses a noncanonical policy")
	}
	root, err := canonicalRealDirectory(plan.SourceRoot)
	if err != nil || root != plan.SourceRoot {
		return errors.New("partition plan source root is alternate or unsafe")
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
