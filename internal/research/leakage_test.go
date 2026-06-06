package research

import (
	"testing"

	"github.com/davidmiguel22573/ak-engine/internal/features"
	"github.com/davidmiguel22573/ak-engine/internal/regime"
)

func TestLeakageChecker(t *testing.T) {
	// 1. Duplicate timestamps in features
	rowsDup := []features.Row{
		{EventTimeMS: 1000, AvailableAtMS: 2000},
		{EventTimeMS: 1000, AvailableAtMS: 2000},
	}
	repDup := CheckFeatureRows(rowsDup)
	if repDup.Status != "FAIL" {
		t.Error("expected CheckFeatureRows to fail for duplicate timestamps")
	}

	// 2. available_at_ms < event_time_ms in features
	rowsEarly := []features.Row{
		{EventTimeMS: 2000, AvailableAtMS: 1000},
	}
	repEarly := CheckFeatureRows(rowsEarly)
	if repEarly.Status != "FAIL" {
		t.Error("expected CheckFeatureRows to fail for available_at < event_time")
	}

	// 3. Duplicate timestamps in labels
	labelsDup := []regime.Label{
		{EventTimeMS: 1000, AvailableAtMS: 2000},
		{EventTimeMS: 1000, AvailableAtMS: 2000},
	}
	repLabelsDup := CheckLabels(labelsDup)
	if repLabelsDup.Status != "FAIL" {
		t.Error("expected CheckLabels to fail for duplicate timestamps")
	}

	// 4. available_at_ms < event_time_ms in labels
	labelsEarly := []regime.Label{
		{EventTimeMS: 2000, AvailableAtMS: 1000},
	}
	repLabelsEarly := CheckLabels(labelsEarly)
	if repLabelsEarly.Status != "FAIL" {
		t.Error("expected CheckLabels to fail for available_at < event_time")
	}
}
