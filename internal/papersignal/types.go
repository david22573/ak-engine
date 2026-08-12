package papersignal

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// SignalSide represents the side of the trade
type SignalSide string

const (
	SideLong  SignalSide = "LONG"
	SideShort SignalSide = "SHORT"
	SideWait  SignalSide = "WAIT"
)

// SignalStatus represents the status of the generated signal
type SignalStatus string

const (
	StatusAllowed                    SignalStatus = "PAPER_SIGNAL_ALLOWED"
	StatusBlockedByRIF               SignalStatus = "PAPER_SIGNAL_BLOCKED_BY_RIF"
	StatusBlockedByData              SignalStatus = "PAPER_SIGNAL_BLOCKED_BY_DATA"
	StatusBlockedByCandidateContract SignalStatus = "PAPER_SIGNAL_BLOCKED_BY_CANDIDATE_CONTRACT"
	StatusWait                       SignalStatus = "PAPER_SIGNAL_WAIT"
)

// OutcomeStatus represents the grader's evaluation of the signal
type OutcomeStatus string

const (
	OutcomePending           OutcomeStatus = "PENDING"
	OutcomeLongTPFirst       OutcomeStatus = "LONG_TP_FIRST"
	OutcomeLongStopFirst     OutcomeStatus = "LONG_STOP_FIRST"
	OutcomeShortTPFirst      OutcomeStatus = "SHORT_TP_FIRST"
	OutcomeShortStopFirst    OutcomeStatus = "SHORT_STOP_FIRST"
	OutcomeCorrectWait       OutcomeStatus = "CORRECT_WAIT"
	OutcomeBadWaitLongRan    OutcomeStatus = "BAD_WAIT_LONG_RAN"
	OutcomeBadWaitShortRan   OutcomeStatus = "BAD_WAIT_SHORT_RAN"
	OutcomeNoEdgeChop        OutcomeStatus = "NO_EDGE_CHOP"
	OutcomeAmbiguousIntrabar OutcomeStatus = "AMBIGUOUS_INTRABAR"
	OutcomeInsufficientData  OutcomeStatus = "INSUFFICIENT_DATA"
)

const (
	ModeDryRun       = "DRY_RUN"
	ModePaperForward = "PAPER_FORWARD"
	ModePaperReplay  = "PAPER_REPLAY"

	SampleInsufficient = "PAPER_INSUFFICIENT_SAMPLE"
	SampleEarly        = "PAPER_EARLY_SAMPLE"
	SampleReady        = "PAPER_CALIBRATION_READY"

	ReadinessNotReady            = "NOT_READY"
	ReadinessContinuePaper       = "CONTINUE_PAPER"
	ReadinessShadowCandidate     = "SHADOW_CANDIDATE"
	ReadinessBlockedByRIF        = "BLOCKED_BY_RIF"
	ReadinessBlockedBySampleSize = "BLOCKED_BY_SAMPLE_SIZE"
	ReadinessBlockedByResults    = "BLOCKED_BY_RESULTS"
)

// PaperSignal represents a paper signal output
type PaperSignal struct {
	SchemaVersion        string        `json:"schema_version"`
	SignalID             string        `json:"signal_id"`
	GeneratedAtUTC       string        `json:"generated_at_utc"`
	CandidateID          string        `json:"candidate_id"`
	CandidateVersion     string        `json:"candidate_version"`
	CandidateHash        string        `json:"candidate_hash"`
	ConfigurationHash    string        `json:"configuration_hash"`
	ResearchEvidenceHash string        `json:"research_evidence_hash"`
	DecisionInputHash    string        `json:"decision_input_hash"`
	Symbol               string        `json:"symbol"`
	MarketType           string        `json:"market_type"`
	Timeframe            string        `json:"timeframe"`
	Side                 SignalSide    `json:"side"`
	SignalStatus         SignalStatus  `json:"signal_status"`
	SignalReason         string        `json:"signal_reason"`
	DataAsOfUTC          string        `json:"data_as_of_utc"`
	DecisionTimeUTC      string        `json:"decision_time_utc"`
	FillTimeUTC          string        `json:"fill_time_utc"`
	ResearchLockPath     string        `json:"research_lock_path"`
	ResearchLockHash     string        `json:"research_lock_hash"`
	DatasetManifestHash  string        `json:"dataset_manifest_hash"`
	UniverseHash         string        `json:"universe_hash"`
	LifecycleHash        string        `json:"lifecycle_hash"`
	PitCoverageHash      string        `json:"pit_coverage_hash"`
	RIFStatus            string        `json:"rif_status"`
	RIFWarnings          []string      `json:"rif_warnings"`
	EntryModel           string        `json:"entry_model"`
	ExitModel            string        `json:"exit_model"`
	InvalidationModel    string        `json:"invalidation_model"`
	ObservationWindow    int           `json:"observation_window"`
	OutcomeStatus        OutcomeStatus `json:"outcome_status"`
	OutcomeDueAtUTC      string        `json:"outcome_due_at_utc"`
	Notes                string        `json:"notes"`
}

// PaperJournalRow represents a single row in the JSONL journal
type PaperJournalRow struct {
	SignalID                   string        `json:"signal_id"`
	CandidateID                string        `json:"candidate_id"`
	CandidateVersion           string        `json:"candidate_version,omitempty"`
	CandidateHash              string        `json:"candidate_hash,omitempty"`
	ConfigurationHash          string        `json:"configuration_hash,omitempty"`
	ResearchEvidenceHash       string        `json:"research_evidence_hash,omitempty"`
	DecisionInputHash          string        `json:"decision_input_hash,omitempty"`
	GeneratedAtUTC             string        `json:"generated_at_utc"`
	DecisionTimeUTC            string        `json:"decision_time_utc,omitempty"`
	FillTimeUTC                string        `json:"fill_time_utc,omitempty"`
	Symbol                     string        `json:"symbol"`
	MarketType                 string        `json:"market_type,omitempty"`
	Timeframe                  string        `json:"timeframe,omitempty"`
	Side                       SignalSide    `json:"side"`
	SignalStatus               SignalStatus  `json:"signal_status"`
	SignalReason               string        `json:"signal_reason,omitempty"`
	EntryReferencePrice        float64       `json:"entry_reference_price"`
	StopReferencePrice         *float64      `json:"stop_reference_price,omitempty"`
	TargetReferencePrice       *float64      `json:"target_reference_price,omitempty"`
	InvalidationReferencePrice *float64      `json:"invalidation_reference_price,omitempty"`
	ObservationWindow          int           `json:"observation_window,omitempty"`
	OutcomeDueAtUTC            string        `json:"outcome_due_at_utc,omitempty"`
	OutcomeStatus              OutcomeStatus `json:"outcome_status"`
	OutcomeCheckedAtUTC        string        `json:"outcome_checked_at_utc"`
	OutcomeReturnBPS           *float64      `json:"outcome_return_bps,omitempty"`
	MaxAdverseExcursionBPS     *float64      `json:"max_adverse_excursion_bps,omitempty"`
	MaxFavorableExcursionBPS   *float64      `json:"max_favorable_excursion_bps,omitempty"`
	OutcomeReason              string        `json:"outcome_reason,omitempty"`
	SourceSnapshotHash         *string       `json:"source_snapshot_hash,omitempty"`
	ResearchLockHash           string        `json:"research_lock_hash"`
	DatasetHash                string        `json:"dataset_hash"`
	UniverseHash               string        `json:"universe_hash"`
	PitCoverageHash            string        `json:"pit_coverage_hash"`
	RIFStatus                  string        `json:"rif_status,omitempty"`
	RIFWarnings                []string      `json:"rif_warnings,omitempty"`
}

// GenerateSignalID generates a deterministic signal ID
func GenerateSignalID(candidateID, symbol, dataAsOf string) string {
	h := sha256.New()
	h.Write([]byte(candidateID))
	h.Write([]byte(symbol))
	h.Write([]byte(dataAsOf))
	return hex.EncodeToString(h.Sum(nil))[:16]
}

func IsAllowedMode(mode string) bool {
	switch mode {
	case ModeDryRun, ModePaperForward, ModePaperReplay:
		return true
	default:
		return false
	}
}

func IsActionableStatus(status SignalStatus) bool {
	return status == StatusAllowed
}

func IsBlockedStatus(status SignalStatus) bool {
	switch status {
	case StatusBlockedByRIF, StatusBlockedByData, StatusBlockedByCandidateContract:
		return true
	default:
		return false
	}
}

// WritePaperSignal outputs the PaperSignal struct to JSON and Markdown files
func WritePaperSignal(outDir string, signal PaperSignal) error {
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return err
	}

	jsonPath, mdPath := SignalArtifactPaths(outDir, signal.SignalID)
	data, err := json.MarshalIndent(signal, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(jsonPath, data, 0644); err != nil {
		return err
	}

	mdContent := fmt.Sprintf("# Paper Signal: %s\n\n", signal.SignalID)
	mdContent += fmt.Sprintf("- **Candidate**: %s\n", signal.CandidateID)
	mdContent += fmt.Sprintf("- **Symbol**: %s\n", signal.Symbol)
	mdContent += fmt.Sprintf("- **Side**: %s\n", signal.Side)
	mdContent += fmt.Sprintf("- **Status**: %s\n", signal.SignalStatus)
	mdContent += fmt.Sprintf("- **Reason**: %s\n", signal.SignalReason)
	mdContent += fmt.Sprintf("- **RIF Status**: %s\n", signal.RIFStatus)

	if err := os.WriteFile(mdPath, []byte(mdContent), 0644); err != nil {
		return err
	}

	if err := os.WriteFile(filepath.Join(outDir, "paper_signal.json"), data, 0644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(outDir, "paper_signal.md"), []byte(mdContent), 0644)
}

func SignalArtifactPaths(outDir, signalID string) (string, string) {
	safeID := strings.NewReplacer("/", "_", "\\", "_", ":", "_").Replace(signalID)
	return filepath.Join(outDir, safeID+"_paper_signal.json"), filepath.Join(outDir, safeID+"_paper_signal.md")
}

// AppendToJournal appends a new row to the JSONL journal
func AppendToJournal(journalPath string, row PaperJournalRow) error {
	dir := filepath.Dir(journalPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	f, err := os.OpenFile(journalPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	data, err := json.Marshal(row)
	if err != nil {
		return err
	}

	_, err = f.Write(append(data, '\n'))
	return err
}

func ReadJournal(journalPath string) ([]PaperJournalRow, error) {
	f, err := os.Open(journalPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var rows []PaperJournalRow
	scanner := bufio.NewScanner(f)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var row PaperJournalRow
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			return nil, fmt.Errorf("invalid journal JSONL at line %d: %w", lineNo, err)
		}
		rows = append(rows, row)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return rows, nil
}

func WriteJournalAtomic(journalPath string, rows []PaperJournalRow) error {
	if err := os.MkdirAll(filepath.Dir(journalPath), 0755); err != nil {
		return err
	}
	tmpPath := journalPath + ".tmp"
	f, err := os.Create(tmpPath)
	if err != nil {
		return err
	}
	ok := false
	defer func() {
		_ = f.Close()
		if !ok {
			_ = os.Remove(tmpPath)
		}
	}()
	for _, row := range rows {
		data, err := json.Marshal(row)
		if err != nil {
			return err
		}
		if _, err := f.Write(append(data, '\n')); err != nil {
			return err
		}
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, journalPath); err != nil {
		return err
	}
	ok = true
	return nil
}

func HashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// ForwardObservationRun represents the summary of a paper-forward generation run
type ForwardObservationRun struct {
	SchemaVersion        string            `json:"schema_version"`
	RunID                string            `json:"run_id"`
	GeneratedAtUTC       string            `json:"generated_at_utc"`
	Mode                 string            `json:"mode"`
	Candidates           []string          `json:"candidates"`
	Symbols              []string          `json:"symbols"`
	Timeframes           []string          `json:"timeframes"`
	DatasetManifestPath  string            `json:"dataset_manifest_path"`
	ResearchEvidencePath string            `json:"research_evidence_path,omitempty"`
	RIFStatus            string            `json:"rif_status"`
	GeneratedSignals     int               `json:"generated_signals"`
	AllowedSignals       int               `json:"allowed_signals,omitempty"`
	BlockedSignals       int               `json:"blocked_signals"`
	WaitObservations     int               `json:"wait_observations"`
	PendingOutcomes      int               `json:"pending_outcomes"`
	GradedOutcomes       int               `json:"graded_outcomes"`
	JournalPath          string            `json:"journal_path"`
	Warnings             []string          `json:"warnings"`
	Hashes               map[string]string `json:"hashes"`
}

// ShadowReadinessReport represents the readiness of a candidate for shadow mode
type ShadowReadinessReport struct {
	SchemaVersion         string         `json:"schema_version"`
	GeneratedAtUTC        string         `json:"generated_at_utc"`
	CandidateID           string         `json:"candidate_id"`
	CandidateVersion      string         `json:"candidate_version,omitempty"`
	CandidateHash         string         `json:"candidate_hash,omitempty"`
	ConfigurationHash     string         `json:"configuration_hash,omitempty"`
	ResearchEvidenceHash  string         `json:"research_evidence_hash,omitempty"`
	TotalSignals          int            `json:"total_signals"`
	AllowedSignals        int            `json:"allowed_signals"`
	BlockedSignals        int            `json:"blocked_signals"`
	WaitObservations      int            `json:"wait_observations"`
	IdentityConflicts     int            `json:"identity_conflicts"`
	AmbiguousOutcomes     int            `json:"ambiguous_outcomes"`
	GradedSignals         int            `json:"graded_signals"`
	PendingSignals        int            `json:"pending_signals"`
	OutcomeDistribution   map[string]int `json:"outcome_distribution"`
	WinRatePaper          float64        `json:"win_rate_paper"`
	ExpectancyPaper       *float64       `json:"expectancy_paper,omitempty"`
	MaxAdverseExcursion   *float64       `json:"max_adverse_excursion,omitempty"`
	MaxFavorableExcursion *float64       `json:"max_favorable_excursion,omitempty"`
	RIFBlockSummary       map[string]int `json:"rif_block_summary"`
	SampleSizeLabel       string         `json:"sample_size_label"`
	ReadinessLabel        string         `json:"readiness_label"`
	Blockers              []string       `json:"blockers"`
	Recommendation        string         `json:"recommendation"`
}
