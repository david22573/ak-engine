package research

import (
	"fmt"

	"github.com/david22573/ak-engine/internal/features"
	"github.com/david22573/ak-engine/internal/regime"
	"github.com/david22573/ak-engine/internal/temporal"
)

type LeakageIssue struct {
	Index         int    `json:"index"`
	EventTimeMS   int64  `json:"event_time_ms"`
	AvailableAtMS int64  `json:"available_at_ms"`
	Reason        string `json:"reason"`
}

type LeakageReport struct {
	Status string         `json:"status"`
	Issues []LeakageIssue `json:"issues"`
}

func CheckFeatureRows(rows []features.Row) LeakageReport {
	var issues []LeakageIssue
	for i := 0; i < len(rows); i++ {
		r := rows[i]
		if err := (temporal.Observation{SourceEventMS: r.EventTimeMS, SourceAvailableMS: r.AvailableAtMS}).Validate(); err != nil {
			issues = append(issues, LeakageIssue{
				Index:         i,
				EventTimeMS:   r.EventTimeMS,
				AvailableAtMS: r.AvailableAtMS,
				Reason:        err.Error(),
			})
		}
		if i > 0 && r.EventTimeMS < rows[i-1].EventTimeMS {
			issues = append(issues, LeakageIssue{
				Index:         i,
				EventTimeMS:   r.EventTimeMS,
				AvailableAtMS: r.AvailableAtMS,
				Reason:        fmt.Sprintf("event_time_ms out of order: current %d < previous %d", r.EventTimeMS, rows[i-1].EventTimeMS),
			})
		}
		if i > 0 && r.EventTimeMS == rows[i-1].EventTimeMS {
			issues = append(issues, LeakageIssue{
				Index:         i,
				EventTimeMS:   r.EventTimeMS,
				AvailableAtMS: r.AvailableAtMS,
				Reason:        fmt.Sprintf("duplicate event_time_ms: %d", r.EventTimeMS),
			})
		}
	}
	status := "PASS"
	if len(issues) > 0 {
		status = "FAIL"
	}
	return LeakageReport{
		Status: status,
		Issues: issues,
	}
}

func CheckLabels(labels []regime.Label) LeakageReport {
	var issues []LeakageIssue
	for i := 0; i < len(labels); i++ {
		l := labels[i]
		if err := (temporal.Observation{SourceEventMS: l.EventTimeMS, SourceAvailableMS: l.AvailableAtMS}).Validate(); err != nil {
			issues = append(issues, LeakageIssue{
				Index:         i,
				EventTimeMS:   l.EventTimeMS,
				AvailableAtMS: l.AvailableAtMS,
				Reason:        err.Error(),
			})
		}
		if i > 0 && l.EventTimeMS < labels[i-1].EventTimeMS {
			issues = append(issues, LeakageIssue{
				Index:         i,
				EventTimeMS:   l.EventTimeMS,
				AvailableAtMS: l.AvailableAtMS,
				Reason:        fmt.Sprintf("event_time_ms out of order: current %d < previous %d", l.EventTimeMS, labels[i-1].EventTimeMS),
			})
		}
		if i > 0 && l.EventTimeMS == labels[i-1].EventTimeMS {
			issues = append(issues, LeakageIssue{
				Index:         i,
				EventTimeMS:   l.EventTimeMS,
				AvailableAtMS: l.AvailableAtMS,
				Reason:        fmt.Sprintf("duplicate event_time_ms: %d", l.EventTimeMS),
			})
		}
	}
	status := "PASS"
	if len(issues) > 0 {
		status = "FAIL"
	}
	return LeakageReport{
		Status: status,
		Issues: issues,
	}
}
