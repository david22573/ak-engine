package governance

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"time"
)

const (
	ProvenanceResolutionSchemaVersion = "ak.engine.pr4b0-r1p2-provenance-resolution.v1"
	InspectionAuditSchemaVersion      = "ak.engine.pr4b0-r1p2-inspection-audit.v1"
)

type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

type ProvenanceEntry struct {
	ArtifactPath            string      `json:"artifact_path"`
	SHA256                  string      `json:"sha256"`
	GitBlobID               string      `json:"git_blob_id"`
	SourceCommit            *string     `json:"source_commit"`
	EarliestKnownAppearance time.Time   `json:"earliest_known_appearance"`
	CandidateID             string      `json:"candidate_id"`
	PossibleExposureRanges  []TimeRange `json:"possible_exposure_ranges"`
	InformationCategories   []string    `json:"information_categories"`
	ByteIdenticalPaths      []string    `json:"byte_identical_paths"`
	ProvenanceEdges         []string    `json:"provenance_edges"`
	Resolution              string      `json:"resolution"`
	ValidationEligible      bool        `json:"validation_eligible"`
	HoldoutEligible         bool        `json:"holdout_eligible"`
}

type ProvenanceResolution struct {
	SchemaVersion  string            `json:"schema_version"`
	Entries        []ProvenanceEntry `json:"entries"`
	ResolutionHash string            `json:"resolution_hash"`
}

type InspectionAudit struct {
	SchemaVersion                          string      `json:"schema_version"`
	Tool                                   string      `json:"tool"`
	Command                                string      `json:"command"`
	Timestamp                              time.Time   `json:"timestamp"`
	Repository                             string      `json:"repository"`
	Commit                                 string      `json:"commit"`
	FilesDisplayed                         []string    `json:"files_displayed"`
	LiteralCategories                      []string    `json:"literal_categories"`
	CandidateFamily                        string      `json:"candidate_family"`
	AffectedPeriods                        []TimeRange `json:"affected_periods"`
	SymbolLevelInformationAppeared         bool        `json:"symbol_level_information_appeared"`
	MonthLevelInformationAppeared          bool        `json:"month_level_information_appeared"`
	QuarterLevelInformationAppeared        bool        `json:"quarter_level_information_appeared"`
	LaterThanKnownResearchPeriodAppeared   bool        `json:"later_than_known_research_period_appeared"`
	ProspectiveValidationExposed           bool        `json:"prospective_validation_exposed"`
	ProspectiveHoldoutExposed              bool        `json:"prospective_holdout_exposed"`
	AffectedImplementationOrPolicyDecision bool        `json:"affected_implementation_or_policy_decision"`
	InspectionCount                        int         `json:"inspection_count"`
	Classification                         string      `json:"classification"`
	ValidationEligible                     bool        `json:"validation_eligible"`
	HoldoutEligible                        bool        `json:"holdout_eligible"`
	FreshPreregistrationRequired           bool        `json:"fresh_preregistration_required"`
	AuditHash                              string      `json:"audit_hash"`
}

func HashCanonical(value any) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(digest[:]), nil
}

func SealProvenance(value ProvenanceResolution) (ProvenanceResolution, error) {
	value.SchemaVersion, value.ResolutionHash = ProvenanceResolutionSchemaVersion, ""
	value.Entries = append([]ProvenanceEntry{}, value.Entries...)
	for index := range value.Entries {
		entry := &value.Entries[index]
		entry.PossibleExposureRanges = append([]TimeRange{}, entry.PossibleExposureRanges...)
		entry.InformationCategories = append([]string{}, entry.InformationCategories...)
		entry.ByteIdenticalPaths = append([]string{}, entry.ByteIdenticalPaths...)
		entry.ProvenanceEdges = append([]string{}, entry.ProvenanceEdges...)
		sort.Slice(entry.PossibleExposureRanges, func(i, j int) bool {
			return entry.PossibleExposureRanges[i].Start.Before(entry.PossibleExposureRanges[j].Start)
		})
		sort.Strings(entry.InformationCategories)
		sort.Strings(entry.ByteIdenticalPaths)
		sort.Strings(entry.ProvenanceEdges)
		if entry.ArtifactPath == "" || !validDigest(entry.SHA256) || len(entry.GitBlobID) != 40 || entry.CandidateID == "" || entry.EarliestKnownAppearance.IsZero() || len(entry.PossibleExposureRanges) == 0 || entry.Resolution != "UNTRUSTED_PROVENANCE_TREATED_AS_EXPOSED" || entry.ValidationEligible || entry.HoldoutEligible {
			return ProvenanceResolution{}, errors.New("provenance entry is not fail closed")
		}
		for _, period := range entry.PossibleExposureRanges {
			if !period.Start.Before(period.End) {
				return ProvenanceResolution{}, errors.New("provenance exposure range is invalid")
			}
		}
	}
	sort.Slice(value.Entries, func(i, j int) bool { return value.Entries[i].ArtifactPath < value.Entries[j].ArtifactPath })
	hash, err := HashCanonical(value)
	if err != nil {
		return ProvenanceResolution{}, err
	}
	value.ResolutionHash = hash
	return value, nil
}

func SealInspectionAudit(value InspectionAudit) (InspectionAudit, error) {
	value.SchemaVersion, value.AuditHash = InspectionAuditSchemaVersion, ""
	value.FilesDisplayed = append([]string{}, value.FilesDisplayed...)
	sort.Strings(value.FilesDisplayed)
	value.LiteralCategories = append([]string{}, value.LiteralCategories...)
	sort.Strings(value.LiteralCategories)
	value.AffectedPeriods = append([]TimeRange{}, value.AffectedPeriods...)
	sort.Slice(value.AffectedPeriods, func(i, j int) bool { return value.AffectedPeriods[i].Start.Before(value.AffectedPeriods[j].Start) })
	allowed := map[string]bool{"LEGACY_ALREADY_EXPOSED_CONTENT_ONLY": true, "NEW_RESEARCH_PERIOD_CONTENT_EXPOSED": true, "PROSPECTIVE_VALIDATION_CONTENT_EXPOSED": true, "PROSPECTIVE_HOLDOUT_CONTENT_EXPOSED": true, "INSPECTION_SCOPE_UNRESOLVED": true}
	if value.InspectionCount <= 0 || value.Tool == "" || value.Command == "" || value.Timestamp.IsZero() || value.Repository == "" || len(value.Commit) != 40 || len(value.FilesDisplayed) == 0 || len(value.LiteralCategories) == 0 || value.CandidateFamily == "" || len(value.AffectedPeriods) == 0 || !allowed[value.Classification] || value.ValidationEligible || value.HoldoutEligible || !value.FreshPreregistrationRequired {
		return InspectionAudit{}, errors.New("inspection audit is incomplete or not fail closed")
	}
	for _, period := range value.AffectedPeriods {
		if !period.Start.Before(period.End) {
			return InspectionAudit{}, errors.New("inspection period is invalid")
		}
	}
	if value.Classification == "PROSPECTIVE_VALIDATION_CONTENT_EXPOSED" && !value.ProspectiveValidationExposed {
		return InspectionAudit{}, errors.New("validation classification conflicts with scope")
	}
	if value.Classification == "PROSPECTIVE_HOLDOUT_CONTENT_EXPOSED" && !value.ProspectiveHoldoutExposed {
		return InspectionAudit{}, errors.New("holdout classification conflicts with scope")
	}
	hash, err := HashCanonical(value)
	if err != nil {
		return InspectionAudit{}, err
	}
	value.AuditHash = hash
	return value, nil
}

func validDigest(value string) bool {
	if !strings.HasPrefix(value, "sha256:") || len(value) != 71 || value != strings.ToLower(value) {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(value, "sha256:"))
	return err == nil
}
