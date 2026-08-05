package researchidentity

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/david22573/ak-engine/internal/canonicalcontract"
)

const (
	historianManifestSchemaName = "ak.historian.research_identity_manifest"
	historianAvailabilitySchema = "ak.historian.availability_policy"
	historianCoverageSchema     = "ak.historian.coverage_policy"
	historianArchiveSchema      = "ak.historian.archive_identity"
	historianStrictCoverageMode = "STRICT_ZERO_DEFECT_FULL_WINDOW"
)

type historianRawObjectReference struct {
	LogicalID    string `json:"logical_id"`
	MediaType    string `json:"media_type"`
	ObjectHash   string `json:"object_hash"`
	RelativePath string `json:"relative_path"`
	SizeBytes    int64  `json:"size_bytes"`
}

type historianArchiveIdentity struct {
	Contract      canonicalcontract.ContractHeader `json:"contract"`
	ArtifactHash  string                           `json:"artifact_hash"`
	ArchiveID     string                           `json:"archive_id"`
	MediaType     string                           `json:"media_type"`
	RawObjectHash string                           `json:"raw_object_hash"`
	RelativePath  string                           `json:"relative_path"`
	SizeBytes     int64                            `json:"size_bytes"`
}

type historianAvailabilityPolicy struct {
	Contract            canonicalcontract.ContractHeader `json:"contract"`
	ArtifactHash        string                           `json:"artifact_hash"`
	PolicyID            string                           `json:"policy_id"`
	PolicyVersion       string                           `json:"policy_version"`
	AvailabilityDelayNS int64                            `json:"availability_delay_ns"`
}

type historianCoveragePolicy struct {
	Contract           canonicalcontract.ContractHeader `json:"contract"`
	ArtifactHash       string                           `json:"artifact_hash"`
	PolicyID           string                           `json:"policy_id"`
	PolicyVersion      string                           `json:"policy_version"`
	Mode               string                           `json:"mode"`
	IntervalNS         int64                            `json:"interval_ns"`
	AllowGaps          bool                             `json:"allow_gaps"`
	AllowDuplicates    bool                             `json:"allow_duplicates"`
	AllowOutOfOrder    bool                             `json:"allow_out_of_order"`
	AllowPartialWindow bool                             `json:"allow_partial_window"`
}

type historianCoverageEvidence struct {
	Contract                canonicalcontract.ContractHeader `json:"contract"`
	ArtifactHash            string                           `json:"artifact_hash"`
	Status                  string                           `json:"status"`
	FullWindow              bool                             `json:"full_window"`
	RequestedStartUTC       string                           `json:"requested_start_utc"`
	RequestedEndUTC         string                           `json:"requested_end_utc"`
	EarliestEventUTC        string                           `json:"earliest_event_utc"`
	LatestEventUTC          string                           `json:"latest_event_utc"`
	RowCount                int64                            `json:"row_count"`
	ExpectedRowCount        int64                            `json:"expected_row_count"`
	SeriesCount             int                              `json:"series_count"`
	GapCount                int64                            `json:"gap_count"`
	DuplicateTimestampCount int64                            `json:"duplicate_timestamp_count"`
	OutOfOrderCount         int64                            `json:"out_of_order_count"`
}

type historianManifest struct {
	Contract                 canonicalcontract.ContractHeader `json:"contract"`
	ArtifactHash             string                           `json:"artifact_hash"`
	ManifestID               string                           `json:"manifest_id"`
	ManifestVersion          string                           `json:"manifest_version"`
	DatasetStartUTC          string                           `json:"dataset_start_utc"`
	DatasetEndUTC            string                           `json:"dataset_end_utc"`
	PointInTimeCutoffUTC     string                           `json:"point_in_time_cutoff_utc"`
	Dataset                  DatasetIdentity                  `json:"dataset"`
	SourceArchive            historianArchiveIdentity         `json:"source_archive"`
	AvailabilityPolicy       historianAvailabilityPolicy      `json:"availability_policy"`
	AvailabilityPolicySource historianRawObjectReference      `json:"availability_policy_source"`
	CoveragePolicy           historianCoveragePolicy          `json:"coverage_policy"`
	CoveragePolicySource     historianRawObjectReference      `json:"coverage_policy_source"`
	CoverageEvidence         historianCoverageEvidence        `json:"coverage_evidence"`
	PITEvidence              PITEvidenceIdentity              `json:"pit_evidence"`
}

func historianError(status IdentityStatus, code string, err error) error {
	return &DerivationError{Status: status, Code: code, Err: err}
}

func verifyHistorianIdentity(manifestPath, datasetRoot string, consumedPaths []string, now time.Time) (DatasetIdentity, PITEvidenceIdentity, error) {
	if strings.TrimSpace(manifestPath) == "" {
		return DatasetIdentity{}, PITEvidenceIdentity{}, historianError(StatusDatasetIncomplete, "HISTORIAN_MANIFEST_MISSING", fmt.Errorf("Historian research identity manifest is required"))
	}
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return DatasetIdentity{}, PITEvidenceIdentity{}, historianError(StatusDatasetIncomplete, "HISTORIAN_MANIFEST_UNREADABLE", err)
	}
	validated, err := canonicalcontract.ValidateArtifact(data, true)
	if err != nil {
		status := StatusDatasetIncomplete
		if result, _ := canonicalcontract.Classify(err); result == canonicalcontract.HashMismatch || result == canonicalcontract.CrossReferenceMismatch {
			status = StatusConflict
		}
		return DatasetIdentity{}, PITEvidenceIdentity{}, historianError(status, "HISTORIAN_MANIFEST_INVALID", err)
	}
	if validated.SchemaName != historianManifestSchemaName {
		return DatasetIdentity{}, PITEvidenceIdentity{}, historianError(StatusDatasetIncomplete, "HISTORIAN_MANIFEST_SCHEMA_MISMATCH", fmt.Errorf("unsupported Historian manifest %s", validated.SchemaName))
	}
	var manifest historianManifest
	if err := json.Unmarshal(validated.CanonicalBytes, &manifest); err != nil {
		return DatasetIdentity{}, PITEvidenceIdentity{}, historianError(StatusDatasetIncomplete, "HISTORIAN_MANIFEST_INVALID", err)
	}
	if err := validateHistorianManifestFields(manifest, now); err != nil {
		return DatasetIdentity{}, PITEvidenceIdentity{}, err
	}

	evidenceRoot := filepath.Dir(manifestPath)
	archivePath, err := resolveRelativeRegularFile(evidenceRoot, manifest.SourceArchive.RelativePath)
	if err != nil {
		return DatasetIdentity{}, PITEvidenceIdentity{}, historianError(StatusDatasetIncomplete, "SOURCE_ARCHIVE_MISSING", err)
	}
	if err := compareRawFileIdentity(archivePath, manifest.SourceArchive.SizeBytes, manifest.SourceArchive.RawObjectHash, "source_archive"); err != nil {
		return DatasetIdentity{}, PITEvidenceIdentity{}, historianError(StatusConflict, "SOURCE_ARCHIVE_IDENTITY_CONFLICT", err)
	}
	availabilityPath, err := resolveRelativeRegularFile(evidenceRoot, manifest.AvailabilityPolicySource.RelativePath)
	if err != nil {
		return DatasetIdentity{}, PITEvidenceIdentity{}, historianError(StatusPITIncomplete, "AVAILABILITY_POLICY_MISSING", err)
	}
	coveragePath, err := resolveRelativeRegularFile(evidenceRoot, manifest.CoveragePolicySource.RelativePath)
	if err != nil {
		return DatasetIdentity{}, PITEvidenceIdentity{}, historianError(StatusDatasetIncomplete, "COVERAGE_POLICY_MISSING", err)
	}
	availability, err := verifyAvailabilityPolicy(availabilityPath, manifest.AvailabilityPolicySource)
	if err != nil {
		return DatasetIdentity{}, PITEvidenceIdentity{}, historianError(StatusConflict, "AVAILABILITY_POLICY_CONFLICT", err)
	}
	if !reflect.DeepEqual(availability, manifest.AvailabilityPolicy) {
		return DatasetIdentity{}, PITEvidenceIdentity{}, historianError(StatusConflict, "AVAILABILITY_POLICY_CONFLICT", fmt.Errorf("availability policy bytes differ from embedded policy"))
	}
	coveragePolicy, err := verifyCoveragePolicy(coveragePath, manifest.CoveragePolicySource)
	if err != nil {
		return DatasetIdentity{}, PITEvidenceIdentity{}, historianError(StatusConflict, "COVERAGE_POLICY_CONFLICT", err)
	}
	if !reflect.DeepEqual(coveragePolicy, manifest.CoveragePolicy) {
		return DatasetIdentity{}, PITEvidenceIdentity{}, historianError(StatusConflict, "COVERAGE_POLICY_CONFLICT", fmt.Errorf("coverage policy bytes differ from embedded policy"))
	}
	if availability.AvailabilityDelayNS != manifest.PITEvidence.AvailabilityDelayNS {
		return DatasetIdentity{}, PITEvidenceIdentity{}, historianError(StatusConflict, "PIT_POLICY_CONFLICT", fmt.Errorf("PIT availability delay conflicts with policy"))
	}

	objects := append([]DatasetObjectIdentity(nil), manifest.Dataset.Objects...)
	if !sort.SliceIsSorted(objects, func(i, j int) bool { return objects[i].RelativePath < objects[j].RelativePath }) {
		return DatasetIdentity{}, PITEvidenceIdentity{}, historianError(StatusDatasetIncomplete, "DATASET_OBJECT_INVENTORY_INVALID", fmt.Errorf("Historian object inventory is not sorted"))
	}
	objectByPath := make(map[string]DatasetObjectIdentity, len(objects))
	for _, object := range objects {
		path, err := resolveRelativeRegularFile(datasetRoot, object.RelativePath)
		if err != nil {
			return DatasetIdentity{}, PITEvidenceIdentity{}, historianError(StatusDatasetIncomplete, "DATASET_OBJECT_MISSING", err)
		}
		if _, exists := objectByPath[object.RelativePath]; exists {
			return DatasetIdentity{}, PITEvidenceIdentity{}, historianError(StatusDatasetIncomplete, "DATASET_OBJECT_INVENTORY_INVALID", fmt.Errorf("duplicate dataset object path %s", object.RelativePath))
		}
		if err := compareRawFileIdentity(path, object.SizeBytes, object.SHA256, "dataset_object"); err != nil {
			return DatasetIdentity{}, PITEvidenceIdentity{}, historianError(StatusConflict, "DATASET_OBJECT_IDENTITY_CONFLICT", err)
		}
		objectByPath[object.RelativePath] = object
	}
	if len(objectByPath) == 0 {
		return DatasetIdentity{}, PITEvidenceIdentity{}, historianError(StatusDatasetIncomplete, "DATASET_OBJECT_INVENTORY_EMPTY", fmt.Errorf("Historian dataset object inventory is empty"))
	}
	if len(consumedPaths) == 0 {
		return DatasetIdentity{}, PITEvidenceIdentity{}, historianError(StatusConsumedIncomplete, "CONSUMED_OBJECT_INVENTORY_EMPTY", fmt.Errorf("no consumed dataset objects were recorded"))
	}
	for _, consumed := range consumedPaths {
		relative, err := relativePathWithin(datasetRoot, consumed)
		if err != nil {
			return DatasetIdentity{}, PITEvidenceIdentity{}, historianError(StatusConsumedIncomplete, "CONSUMED_OBJECT_PATH_INVALID", err)
		}
		if _, exists := objectByPath[relative]; !exists {
			return DatasetIdentity{}, PITEvidenceIdentity{}, historianError(StatusConflict, "CONSUMED_OBJECT_MANIFEST_CONFLICT", fmt.Errorf("consumed object %s is absent from manifest", relative))
		}
	}

	manifestRawHash, err := canonicalcontract.HashRaw(rawObjectContractName, canonicalContractVersion, "historian_manifest_source", data)
	if err != nil {
		return DatasetIdentity{}, PITEvidenceIdentity{}, err
	}
	dataset := manifest.Dataset
	dataset.ManifestID = manifest.ManifestID
	dataset.ManifestVersion = manifest.ManifestVersion
	dataset.ManifestHash = manifest.ArtifactHash
	dataset.ManifestRawHash = manifestRawHash
	dataset.DatasetHash = dataset.ArtifactHash
	dataset.SourceArchiveID = manifest.SourceArchive.ArchiveID
	dataset.SourceArchiveHash = manifest.SourceArchive.ArtifactHash
	dataset.PointInTimeCutoffUTC = manifest.PointInTimeCutoffUTC
	dataset.AvailabilityPolicyID = manifest.AvailabilityPolicy.PolicyID
	dataset.AvailabilityPolicyVersion = manifest.AvailabilityPolicy.PolicyVersion
	dataset.AvailabilityPolicyHash = manifest.AvailabilityPolicy.ArtifactHash
	dataset.CoveragePolicyID = manifest.CoveragePolicy.PolicyID
	dataset.CoveragePolicyVersion = manifest.CoveragePolicy.PolicyVersion
	dataset.CoveragePolicyHash = manifest.CoveragePolicy.ArtifactHash
	pit := manifest.PITEvidence
	pit.EvidenceHash = pit.ArtifactHash
	pit.PITPolicyID = manifest.AvailabilityPolicy.PolicyID
	pit.PITPolicyVersion = manifest.AvailabilityPolicy.PolicyVersion
	pit.PITPolicyHash = manifest.AvailabilityPolicy.ArtifactHash
	pit.CoveragePolicyID = manifest.CoveragePolicy.PolicyID
	pit.CoveragePolicyVersion = manifest.CoveragePolicy.PolicyVersion
	pit.SourceArchiveID = manifest.SourceArchive.ArchiveID
	pit.AvailabilityDelayMS = pit.AvailabilityDelayNS / int64(time.Millisecond)
	return dataset, pit, nil
}

func validateHistorianManifestFields(manifest historianManifest, now time.Time) error {
	start, err := parseUTC(manifest.DatasetStartUTC)
	if err != nil {
		return historianError(StatusDatasetIncomplete, "DATASET_WINDOW_INVALID", err)
	}
	end, err := parseUTC(manifest.DatasetEndUTC)
	if err != nil {
		return historianError(StatusDatasetIncomplete, "DATASET_WINDOW_INVALID", err)
	}
	cutoff, err := parseUTC(manifest.PointInTimeCutoffUTC)
	if err != nil {
		return historianError(StatusPITIncomplete, "PIT_CUTOFF_INVALID", err)
	}
	if !start.Before(end) || cutoff.Before(end) || cutoff.After(now.UTC()) {
		return historianError(StatusDatasetIncomplete, "DATASET_WINDOW_INVALID", fmt.Errorf("invalid dataset window/cutoff order"))
	}
	if manifest.Dataset.StartUTC != manifest.DatasetStartUTC || manifest.Dataset.EndUTC != manifest.DatasetEndUTC || manifest.CoverageEvidence.RequestedStartUTC != manifest.DatasetStartUTC || manifest.CoverageEvidence.RequestedEndUTC != manifest.DatasetEndUTC {
		return historianError(StatusConflict, "HISTORIAN_WINDOW_CONFLICT", fmt.Errorf("Historian window representations disagree"))
	}
	coverage := manifest.CoverageEvidence
	if coverage.Status != "PASS" || !coverage.FullWindow || coverage.RowCount <= 0 || coverage.ExpectedRowCount != coverage.RowCount || coverage.SeriesCount <= 0 || coverage.GapCount != 0 || coverage.DuplicateTimestampCount != 0 || coverage.OutOfOrderCount != 0 {
		return historianError(StatusDatasetIncomplete, "COVERAGE_NOT_STRICT_PASS", fmt.Errorf("coverage is not strict zero-defect full-window PASS"))
	}
	pit := manifest.PITEvidence
	if pit.Status != "PASS" || !pit.FullWindowCoverage || pit.GapCount != 0 || pit.DuplicateTimestampCount != 0 || pit.OutOfOrderCount != 0 || pit.EvaluationCutoffUTC != manifest.PointInTimeCutoffUTC {
		return historianError(StatusPITIncomplete, "PIT_NOT_STRICT_PASS", fmt.Errorf("PIT evidence is not strict PASS"))
	}
	latestAvailable, err := parseUTC(pit.LatestAvailableUTC)
	if err != nil || latestAvailable.After(cutoff) {
		return historianError(StatusPITIncomplete, "PIT_AVAILABILITY_INVALID", fmt.Errorf("latest availability exceeds cutoff: %w", err))
	}
	if manifest.CoveragePolicy.Mode != historianStrictCoverageMode || manifest.CoveragePolicy.IntervalNS <= 0 || manifest.CoveragePolicy.AllowGaps || manifest.CoveragePolicy.AllowDuplicates || manifest.CoveragePolicy.AllowOutOfOrder || manifest.CoveragePolicy.AllowPartialWindow {
		return historianError(StatusDatasetIncomplete, "COVERAGE_POLICY_INVALID", fmt.Errorf("unsupported coverage policy"))
	}
	if manifest.AvailabilityPolicy.AvailabilityDelayNS < 0 || manifest.AvailabilityPolicy.AvailabilityDelayNS%int64(time.Millisecond) != 0 {
		return historianError(StatusPITIncomplete, "AVAILABILITY_POLICY_INVALID", fmt.Errorf("unsupported availability delay"))
	}
	return nil
}

func verifyAvailabilityPolicy(path string, source historianRawObjectReference) (historianAvailabilityPolicy, error) {
	var policy historianAvailabilityPolicy
	if err := verifyPolicyArtifact(path, source, historianAvailabilitySchema, "availability_policy_source", &policy); err != nil {
		return policy, err
	}
	return policy, nil
}

func verifyCoveragePolicy(path string, source historianRawObjectReference) (historianCoveragePolicy, error) {
	var policy historianCoveragePolicy
	if err := verifyPolicyArtifact(path, source, historianCoverageSchema, "coverage_policy_source", &policy); err != nil {
		return policy, err
	}
	return policy, nil
}

func verifyPolicyArtifact(path string, source historianRawObjectReference, expectedSchema, rawRole string, target any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if int64(len(data)) != source.SizeBytes {
		return fmt.Errorf("policy size mismatch")
	}
	rawHash, err := canonicalcontract.HashRaw(rawObjectContractName, canonicalContractVersion, rawRole, data)
	if err != nil || !canonicalcontract.EqualHash(rawHash, source.ObjectHash) {
		return fmt.Errorf("policy raw hash mismatch: %w", err)
	}
	validated, err := canonicalcontract.ValidateArtifact(data, true)
	if err != nil {
		return err
	}
	if validated.SchemaName != expectedSchema {
		return fmt.Errorf("policy schema mismatch")
	}
	return json.Unmarshal(validated.CanonicalBytes, target)
}

func compareRawFileIdentity(path string, size int64, expectedHash, role string) error {
	hash, actualSize, err := hashFileRole(path, role)
	if err != nil {
		return err
	}
	if actualSize != size || !canonicalcontract.EqualHash(hash, expectedHash) {
		return fmt.Errorf("raw identity mismatch")
	}
	return nil
}

func resolveRelativeRegularFile(root, relative string) (string, error) {
	if err := canonicalcontract.ValidateRelativePath(relative); err != nil {
		return "", err
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	path := filepath.Join(rootAbs, filepath.FromSlash(relative))
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", err
	}
	if resolved != path {
		return "", fmt.Errorf("symlinks are forbidden")
	}
	rel, err := filepath.Rel(rootAbs, resolved)
	if err != nil || filepath.IsAbs(rel) || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path escapes root")
	}
	info, err := os.Lstat(resolved)
	if err != nil || !info.Mode().IsRegular() {
		return "", fmt.Errorf("path is not a regular file: %w", err)
	}
	return resolved, nil
}

func relativePathWithin(root, path string) (string, error) {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	pathAbs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	relative, err := filepath.Rel(rootAbs, pathAbs)
	if err != nil {
		return "", err
	}
	relative = filepath.ToSlash(filepath.Clean(relative))
	if err := canonicalcontract.ValidateRelativePath(relative); err != nil {
		return "", err
	}
	return relative, nil
}

func parseUTC(value string) (time.Time, error) {
	if err := canonicalcontract.ValidateTimestamp(value); err != nil {
		return time.Time{}, err
	}
	return time.Parse("2006-01-02T15:04:05.000000000Z", value)
}
