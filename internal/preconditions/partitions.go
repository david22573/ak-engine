package preconditions

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

const (
	ExposureLedgerSchemaVersion = "ak.engine.prior-exposure-ledger.v1"
	PartitionPolicyVersion      = "ak.engine.partition-eligibility.v1"
)

type TimeWindow struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

type ExposureEntry struct {
	SourceReport        string     `json:"source_report"`
	SourceCommit        string     `json:"source_commit"`
	CandidateID         string     `json:"candidate_id"`
	ExposedWindow       TimeWindow `json:"exposed_window"`
	GranularityExposed  []string   `json:"granularity_exposed"`
	MetricsExposed      []string   `json:"metrics_exposed"`
	SymbolLevelExposed  bool       `json:"symbol_level_exposed"`
	MonthLevelExposed   bool       `json:"month_level_exposed"`
	QuarterLevelExposed bool       `json:"quarter_level_exposed"`
	EvidenceHash        string     `json:"evidence_hash"`
}

type ExposureLedger struct {
	SchemaVersion string          `json:"schema_version"`
	CandidateID   string          `json:"candidate_id"`
	Entries       []ExposureEntry `json:"entries"`
	LedgerHash    string          `json:"ledger_hash"`
}

type PartitionPlan struct {
	PolicyVersion                  string        `json:"policy_version"`
	PITCoverage                    TimeWindow    `json:"pit_coverage"`
	WarmupDuration                 time.Duration `json:"warmup_duration"`
	EmbargoDuration                time.Duration `json:"embargo_duration"`
	Development                    TimeWindow    `json:"development"`
	Validation                     TimeWindow    `json:"validation"`
	FinalHoldout                   TimeWindow    `json:"final_holdout"`
	MinimumIndependentDecisionGate int           `json:"minimum_independent_decision_gate"`
}

func SealExposureLedger(ledger ExposureLedger) (ExposureLedger, error) {
	if ledger.SchemaVersion == "" {
		ledger.SchemaVersion = ExposureLedgerSchemaVersion
	}
	ledger.LedgerHash = ""
	hash, err := ExposureLedgerHash(ledger)
	if err != nil {
		return ExposureLedger{}, err
	}
	ledger.LedgerHash = hash
	return ledger, nil
}

func ExposureLedgerHash(ledger ExposureLedger) (string, error) {
	if ledger.SchemaVersion != ExposureLedgerSchemaVersion || strings.TrimSpace(ledger.CandidateID) == "" {
		return "", errors.New("invalid exposure ledger identity")
	}
	copyLedger := ledger
	copyLedger.LedgerHash = ""
	copyLedger.Entries = append([]ExposureEntry{}, ledger.Entries...)
	for i := range copyLedger.Entries {
		entry := &copyLedger.Entries[i]
		if entry.SourceReport == "" || entry.SourceCommit == "" || entry.CandidateID != ledger.CandidateID || !validWindow(entry.ExposedWindow) || !validSHA256(entry.EvidenceHash) {
			return "", fmt.Errorf("invalid exposure entry %d", i)
		}
		entry.GranularityExposed = append([]string{}, entry.GranularityExposed...)
		sort.Strings(entry.GranularityExposed)
		entry.MetricsExposed = append([]string{}, entry.MetricsExposed...)
		sort.Strings(entry.MetricsExposed)
	}
	sort.Slice(copyLedger.Entries, func(i, j int) bool {
		if copyLedger.Entries[i].SourceCommit != copyLedger.Entries[j].SourceCommit {
			return copyLedger.Entries[i].SourceCommit < copyLedger.Entries[j].SourceCommit
		}
		return copyLedger.Entries[i].SourceReport < copyLedger.Entries[j].SourceReport
	})
	return canonicalDigest(copyLedger)
}

func ValidatePartitionPlan(plan PartitionPlan, ledger ExposureLedger) error {
	if plan.PolicyVersion != PartitionPolicyVersion {
		return errors.New("unsupported partition policy")
	}
	if plan.MinimumIndependentDecisionGate != 300 {
		return errors.New("the 300-independent-decision gate cannot be weakened")
	}
	for name, window := range map[string]TimeWindow{"PIT coverage": plan.PITCoverage, "development": plan.Development, "validation": plan.Validation, "final holdout": plan.FinalHoldout} {
		if !validWindow(window) {
			return fmt.Errorf("%s window is invalid", name)
		}
	}
	if plan.WarmupDuration < 0 || plan.EmbargoDuration < 0 {
		return errors.New("warm-up and embargo durations cannot be negative")
	}
	if plan.Development.Start.Before(plan.PITCoverage.Start.Add(plan.WarmupDuration)) || plan.Development.End.Add(plan.EmbargoDuration).After(plan.Validation.Start) || plan.Validation.End.Add(plan.EmbargoDuration).After(plan.FinalHoldout.Start) {
		return errors.New("partition chronology, warm-up, or embargo rule failed")
	}
	if !plan.FinalHoldout.End.Equal(plan.PITCoverage.End) || plan.FinalHoldout.Start.Before(plan.Validation.End) {
		return errors.New("final holdout must be the last nonoverlapping PIT partition")
	}
	if plan.Development.Start.Before(plan.PITCoverage.Start) || plan.FinalHoldout.End.After(plan.PITCoverage.End) {
		return errors.New("partitions exceed PIT coverage")
	}
	for _, entry := range ledger.Entries {
		if overlaps(entry.ExposedWindow, plan.Validation) {
			return fmt.Errorf("previous exposure %s overlaps validation", entry.SourceReport)
		}
		if overlaps(entry.ExposedWindow, plan.FinalHoldout) {
			return fmt.Errorf("previous exposure %s overlaps final holdout", entry.SourceReport)
		}
	}
	return nil
}

func PartitionPolicyHash() (string, error) {
	return canonicalDigest(struct{ Version, Ordering, Exposure, Holdout, Gate string }{
		PartitionPolicyVersion, "chronological half-open nonoverlapping windows with deterministic warm-up and embargo gaps",
		"any previously exposed instant is DEVELOPMENT-only", "FINAL_HOLDOUT ends exactly at PIT coverage end", "minimum independent decisions remains 300",
	})
}

func validWindow(window TimeWindow) bool {
	return !window.Start.IsZero() && window.Start.Before(window.End)
}
func overlaps(left, right TimeWindow) bool {
	return left.Start.Before(right.End) && right.Start.Before(left.End)
}
