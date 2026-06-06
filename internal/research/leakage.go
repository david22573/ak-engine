package research

import (
	"fmt"

	"github.com/davidmiguel22573/ak-engine/internal/features"
	"github.com/davidmiguel22573/ak-engine/internal/regime"
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
		if r.EventTimeMS <= 0 {
			issues = append(issues, LeakageIssue{
				Index:         i,
				EventTimeMS:   r.EventTimeMS,
				AvailableAtMS: r.AvailableAtMS,
				Reason:        "event_time_ms <= 0",
			})
		}
		if r.AvailableAtMS <= 0 {
			issues = append(issues, LeakageIssue{
				Index:         i,
				EventTimeMS:   r.EventTimeMS,
				AvailableAtMS: r.AvailableAtMS,
				Reason:        "available_at_ms <= 0",
			})
		}
		if r.AvailableAtMS < r.EventTimeMS {
			issues = append(issues, LeakageIssue{
				Index:         i,
				EventTimeMS:   r.EventTimeMS,
				AvailableAtMS: r.AvailableAtMS,
				Reason:        "available_at_ms < event_time_ms",
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
		if l.EventTimeMS <= 0 {
			issues = append(issues, LeakageIssue{
				Index:         i,
				EventTimeMS:   l.EventTimeMS,
				AvailableAtMS: l.AvailableAtMS,
				Reason:        "event_time_ms <= 0",
			})
		}
		if l.AvailableAtMS <= 0 {
			issues = append(issues, LeakageIssue{
				Index:         i,
				EventTimeMS:   l.EventTimeMS,
				AvailableAtMS: l.AvailableAtMS,
				Reason:        "available_at_ms <= 0",
			})
		}
		if l.AvailableAtMS < l.EventTimeMS {
			issues = append(issues, LeakageIssue{
				Index:         i,
				EventTimeMS:   l.EventTimeMS,
				AvailableAtMS: l.AvailableAtMS,
				Reason:        "available_at_ms < event_time_ms",
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
