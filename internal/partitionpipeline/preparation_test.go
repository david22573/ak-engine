package partitionpipeline

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"time"
)

var boundaryTestStart = time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)

func TestBoundarySlicingClassesAndHalfOpenEndpoints(t *testing.T) {
	membership := Interval{Start: boundaryTestStart, End: boundaryTestStart.Add(4 * time.Minute)}
	tests := []struct {
		name      string
		start     time.Time
		rows      int
		wantClass string
		wantRows  int
		wantFirst time.Time
		wantLast  time.Time
	}{
		{"01_exact_partition", membership.Start, 4, "EXACT_BOUNDARY", 4, membership.Start, membership.End.Add(-time.Minute)},
		{"02_crosses_left_boundary", membership.Start.Add(-time.Minute), 5, "LEFT_CLIPPED", 4, membership.Start, membership.End.Add(-time.Minute)},
		{"03_crosses_right_boundary", membership.Start, 5, "RIGHT_CLIPPED", 4, membership.Start, membership.End.Add(-time.Minute)},
		{"04_crosses_both_boundaries", membership.Start.Add(-time.Minute), 6, "BOTH_BOUNDARIES", 4, membership.Start, membership.End.Add(-time.Minute)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fragment, parent := boundaryParent(t, "ADAUSDT", test.start, test.rows)
			child, class, err := sliceChild(fragment, parent, "ADAUSDT", membership, true)
			if err != nil {
				t.Fatal(err)
			}
			if class != test.wantClass || len(child.Records) != test.wantRows || time.UnixMilli(child.Records[0].OpenTimeMS).UTC() != test.wantFirst || time.UnixMilli(child.Records[len(child.Records)-1].OpenTimeMS).UTC() != test.wantLast {
				t.Fatalf("unexpected boundary result: class=%s rows=%d first=%s last=%s", class, len(child.Records), time.UnixMilli(child.Records[0].OpenTimeMS).UTC(), time.UnixMilli(child.Records[len(child.Records)-1].OpenTimeMS).UTC())
			}
		})
	}
	t.Run("05_inclusive_start_is_emitted", func(t *testing.T) {
		fragment, parent := boundaryParent(t, "ADAUSDT", membership.Start, 4)
		child, _, err := sliceChild(fragment, parent, "ADAUSDT", membership, true)
		if err != nil || time.UnixMilli(child.Records[0].OpenTimeMS).UTC() != membership.Start {
			t.Fatalf("inclusive start was not emitted: %v", err)
		}
	})
	t.Run("06_exclusive_end_is_not_emitted", func(t *testing.T) {
		fragment, parent := boundaryParent(t, "ADAUSDT", membership.Start, 5)
		child, _, err := sliceChild(fragment, parent, "ADAUSDT", membership, true)
		if err != nil {
			t.Fatal(err)
		}
		for _, row := range child.Records {
			if time.UnixMilli(row.OpenTimeMS).UTC() == membership.End {
				t.Fatal("row at the exclusive end was emitted")
			}
		}
	})
}

func TestBoundaryParentAndChildAdversarialRows(t *testing.T) {
	membership := Interval{Start: boundaryTestStart, End: boundaryTestStart.Add(4 * time.Minute)}
	t.Run("07_wrong_symbol_inside_selected_interval", func(t *testing.T) {
		fragment, parent := boundaryParent(t, "ADAUSDT", membership.Start, 4)
		fragment.Records[1].Symbol = "BTCUSDT"
		if err := validateParentFragment(fragment, parent, "ADAUSDT", true); err == nil {
			t.Fatal("wrong-symbol selected row passed")
		}
	})
	t.Run("08_wrong_symbol_outside_selected_interval", func(t *testing.T) {
		fragment, parent := boundaryParent(t, "ADAUSDT", membership.Start.Add(-time.Minute), 5)
		fragment.Records[0].Symbol = "BTCUSDT"
		if err := validateParentFragment(fragment, parent, "ADAUSDT", true); err == nil {
			t.Fatal("wrong-symbol parent row outside the slice passed")
		}
	})
	t.Run("09_out_of_order_rows", func(t *testing.T) {
		fragment, parent := boundaryParent(t, "ADAUSDT", membership.Start, 4)
		fragment.Records[1], fragment.Records[2] = fragment.Records[2], fragment.Records[1]
		if err := validateParentFragment(fragment, parent, "ADAUSDT", true); err == nil {
			t.Fatal("out-of-order parent passed")
		}
	})
	t.Run("10_duplicate_timestamps", func(t *testing.T) {
		fragment, parent := boundaryParent(t, "ADAUSDT", membership.Start, 4)
		fragment.Records[2].OpenTimeMS = fragment.Records[1].OpenTimeMS
		fragment.Records[2].CloseTimeMS = fragment.Records[1].CloseTimeMS
		fragment.Records[2].MarketEventTimeUTC = fragment.Records[1].MarketEventTimeUTC
		fragment.Records[2].ProviderCandleCloseTimeUTC = fragment.Records[1].ProviderCandleCloseTimeUTC
		if err := validateParentFragment(fragment, parent, "ADAUSDT", true); err == nil {
			t.Fatal("duplicate parent timestamp passed")
		}
	})
	t.Run("11_empty_authorized_slice", func(t *testing.T) {
		fragment, parent := boundaryParent(t, "ADAUSDT", membership.Start.Add(-8*time.Minute), 4)
		if _, _, err := sliceChild(fragment, parent, "ADAUSDT", membership, true); err == nil {
			t.Fatal("empty authorized slice passed")
		}
	})
	t.Run("12_missing_expected_candle", func(t *testing.T) {
		fragment, parent := boundaryParent(t, "ADAUSDT", membership.Start, 4)
		fragment.Records[2] = normalizedBoundaryRecord("ADAUSDT", membership.Start.Add(3*time.Minute))
		if err := validateParentFragment(fragment, parent, "ADAUSDT", true); err == nil {
			t.Fatal("gapped parent passed")
		}
	})
	t.Run("13_parent_hash_mismatch", func(t *testing.T) {
		fragment, parent := boundaryParent(t, "ADAUSDT", membership.Start, 4)
		parent.FragmentSHA256 = testHash('0')
		if err := validateParentFragment(fragment, parent, "ADAUSDT", true); err == nil {
			t.Fatal("parent hash mismatch passed")
		}
	})
	t.Run("14_child_hash_mismatch", func(t *testing.T) {
		fragment, parent := boundaryParent(t, "ADAUSDT", membership.Start, 4)
		child, _, err := sliceChild(fragment, parent, "ADAUSDT", membership, true)
		if err != nil {
			t.Fatal(err)
		}
		child.ChildSHA256 = testHash('0')
		if err := verifyChildArtifact(child); err == nil {
			t.Fatal("child hash mismatch passed")
		}
	})
	t.Run("19_malformed_final_row", func(t *testing.T) {
		fragment, parent := boundaryParent(t, "ADAUSDT", membership.Start, 4)
		fragment.Records[len(fragment.Records)-1].CloseTimeMS--
		if err := validateParentFragment(fragment, parent, "ADAUSDT", true); err == nil {
			t.Fatal("malformed final row passed")
		}
	})
	t.Run("25_metadata_safe_but_rows_violate", func(t *testing.T) {
		fragment, parent := boundaryParent(t, "ADAUSDT", membership.Start, 4)
		fragment.Records[len(fragment.Records)-1] = normalizedBoundaryRecord("ADAUSDT", membership.End)
		if err := validateParentFragment(fragment, parent, "ADAUSDT", true); err == nil {
			t.Fatal("source rows outside claimed safe metadata passed")
		}
	})
}

func TestPreparationFilesystemAdversarialCases(t *testing.T) {
	t.Run("15_parent_mutation_after_plan_generation", func(t *testing.T) {
		root, plan := syntheticParentSource(t, "DEVELOPMENT")
		path := firstParentPath(root, plan)
		if err := os.WriteFile(path, append(mustReadFile(t, path), 0), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, _, _, err := PreparePlan(plan, filepath.Join(t.TempDir(), "prepared")); err == nil {
			t.Fatal("mutated parent prepared")
		}
	})
	t.Run("16_symlink_input", func(t *testing.T) {
		root, plan := syntheticParentSource(t, "DEVELOPMENT")
		path := firstParentPath(root, plan)
		if err := os.Remove(path); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink("missing-parent", path); err != nil {
			t.Fatal(err)
		}
		if _, _, _, err := PreparePlan(plan, filepath.Join(t.TempDir(), "prepared")); !errors.Is(err, errUnsafePath) {
			t.Fatalf("symlink input did not fail with the path guard: %v", err)
		}
	})
	t.Run("17_path_escape", func(t *testing.T) {
		_, plan := syntheticParentSource(t, "DEVELOPMENT")
		plan.SourceManifests[0].FragmentArtifacts[0].RelativePath = "../escape.json.gz"
		plan.PlanSHA256 = ""
		plan.PlanSHA256, _ = planHash(plan)
		if _, _, _, err := PreparePlan(plan, filepath.Join(t.TempDir(), "prepared")); !errors.Is(err, errUnsafePath) {
			t.Fatalf("path escape did not fail with the path guard: %v", err)
		}
	})
	t.Run("18_truncated_fragment", func(t *testing.T) {
		root, plan := syntheticParentSource(t, "DEVELOPMENT")
		path := firstParentPath(root, plan)
		data := mustReadFile(t, path)
		if err := os.WriteFile(path, data[:len(data)/2], 0o600); err != nil {
			t.Fatal(err)
		}
		if _, _, _, err := PreparePlan(plan, filepath.Join(t.TempDir(), "prepared")); err == nil {
			t.Fatal("truncated fragment prepared")
		}
	})
}

func TestStageBoundaryChildrenNeverCross(t *testing.T) {
	boundaries := []struct {
		name       string
		membership Interval
	}{
		{"20_development_validation", Interval{Start: acceptedIntervals["DEVELOPMENT"].End.Add(-10 * time.Minute), End: acceptedIntervals["DEVELOPMENT"].End}},
		{"21_validation_final_holdout", Interval{Start: acceptedIntervals["VALIDATION"].End.Add(-10 * time.Minute), End: acceptedIntervals["VALIDATION"].End}},
	}
	for _, test := range boundaries {
		t.Run(test.name, func(t *testing.T) {
			fragment, parent := boundaryParent(t, "ADAUSDT", test.membership.End.Add(-2*time.Minute), 4)
			child, class, err := sliceChild(fragment, parent, "ADAUSDT", test.membership, true)
			if err != nil || class != "RIGHT_CLIPPED" || child.AuthorizedInterval.End != test.membership.End {
				t.Fatalf("stage crossing was not clipped: class=%s child=%+v err=%v", class, child.AuthorizedInterval, err)
			}
			for _, row := range child.Records {
				if !time.UnixMilli(row.OpenTimeMS).UTC().Before(test.membership.End) {
					t.Fatal("neighbor-stage row entered child")
				}
			}
		})
	}
}

func TestPreparationDeterminismResumeAndConcurrency(t *testing.T) {
	t.Run("22_deterministic_repeated_generation", func(t *testing.T) {
		_, parent := syntheticParentSource(t, "DEVELOPMENT")
		root := filepath.Join(t.TempDir(), "prepared")
		firstPlan, firstManifest, firstAudit, err := PreparePlan(parent, root)
		if err != nil {
			t.Fatal(err)
		}
		secondPlan, secondManifest, secondAudit, err := PreparePlan(parent, root)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(firstPlan, secondPlan) || !reflect.DeepEqual(firstManifest, secondManifest) || !reflect.DeepEqual(firstAudit, secondAudit) {
			t.Fatal("repeated preparation was not deterministic")
		}
	})
	t.Run("23_interrupted_generation_safe_resume", func(t *testing.T) {
		_, parent := syntheticParentSource(t, "DEVELOPMENT")
		root := filepath.Join(t.TempDir(), "prepared")
		canonical, err := ensurePreparedRoot(root)
		if err != nil {
			t.Fatal(err)
		}
		stale := filepath.Join(canonical, "children", ".boundary-tmp-interrupted")
		if err := os.WriteFile(stale, []byte("partial"), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, _, _, err := PreparePlan(parent, root); err != nil {
			t.Fatalf("safe resume failed: %v", err)
		}
		if data := mustReadFile(t, stale); string(data) != "partial" {
			t.Fatal("resume treated an unpublished temporary as an artifact")
		}
	})
	t.Run("24_concurrent_same_child_attempts", func(t *testing.T) {
		_, parent := syntheticParentSource(t, "DEVELOPMENT")
		root := filepath.Join(t.TempDir(), "prepared")
		type result struct {
			plan Plan
			err  error
		}
		start := make(chan struct{})
		results := make(chan result, 2)
		var workers sync.WaitGroup
		for range 2 {
			workers.Add(1)
			go func() {
				defer workers.Done()
				<-start
				plan, _, _, err := PreparePlan(parent, root)
				results <- result{plan: plan, err: err}
			}()
		}
		close(start)
		workers.Wait()
		close(results)
		got := []Plan{}
		for value := range results {
			if value.err != nil {
				t.Fatalf("concurrent preparation failed: %v", value.err)
			}
			got = append(got, value.plan)
		}
		if len(got) != 2 || !reflect.DeepEqual(got[0], got[1]) {
			t.Fatal("concurrent preparation did not converge on one identity")
		}
	})
}

func TestADAUSDTDevelopmentBoundaryRegression(t *testing.T) {
	var fixture struct {
		SchemaVersion             string    `json:"schema_version"`
		Symbol                    string    `json:"symbol"`
		MembershipStartUTC        time.Time `json:"membership_start_utc"`
		MembershipEndUTC          time.Time `json:"membership_end_utc"`
		ParentStartUTC            time.Time `json:"parent_start_utc"`
		ParentEndUTC              time.Time `json:"parent_end_utc"`
		ParentRowCount            int       `json:"parent_row_count"`
		ExpectedChildRowCount     int       `json:"expected_child_row_count"`
		ExpectedExcludedRightRows int       `json:"expected_excluded_right_rows"`
		ExpectedBoundaryClass     string    `json:"expected_boundary_class"`
		ReceiptSHA256             string    `json:"receipt_sha256"`
		FragmentSHA256            string    `json:"fragment_sha256"`
	}
	data := mustReadFile(t, filepath.Join("testdata", "adausdt-development-boundary-regression.json"))
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	records := make([]normalizedRecord, fixture.ParentRowCount)
	for index := range records {
		records[index] = normalizedBoundaryRecord(fixture.Symbol, fixture.ParentStartUTC.Add(time.Duration(index)*time.Minute))
	}
	fragment := sourceFragment{SchemaVersion: "ak-historian.pr4b0-r1p5r.normalized-fragment.v1", Symbol: fixture.Symbol, Records: records, FragmentHash: fixture.FragmentSHA256}
	parent, err := sealParentProvenance(ParentProvenance{SourceRootID: CheckpointSourceRootID, ReceiptSHA256: fixture.ReceiptSHA256, FragmentSHA256: fixture.FragmentSHA256, ParentObjectID: CheckpointSourceRootID + ":" + fixture.FragmentSHA256, ParentStartUTC: fixture.ParentStartUTC, ParentEndUTC: fixture.ParentEndUTC, ParentRowCount: fixture.ParentRowCount, ObservedAvailableAtUTC: time.Date(2026, 7, 17, 7, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatal(err)
	}
	if err := validateParentFragment(fragment, parent, fixture.Symbol, true); err != nil {
		t.Fatal(err)
	}
	child, class, err := sliceChild(fragment, parent, fixture.Symbol, Interval{Start: fixture.MembershipStartUTC, End: fixture.MembershipEndUTC}, true)
	if err != nil {
		t.Fatal(err)
	}
	if class != fixture.ExpectedBoundaryClass || len(child.Records) != fixture.ExpectedChildRowCount || fixture.ParentRowCount-len(child.Records) != fixture.ExpectedExcludedRightRows || child.AuthorizedInterval.End != fixture.MembershipEndUTC {
		t.Fatalf("ADAUSDT regression mismatch: class=%s rows=%d excluded=%d child_end=%s", class, len(child.Records), fixture.ParentRowCount-len(child.Records), child.AuthorizedInterval.End)
	}
}

func boundaryParent(t *testing.T, symbol string, start time.Time, rows int) (sourceFragment, ParentProvenance) {
	t.Helper()
	records := make([]normalizedRecord, rows)
	for index := range records {
		records[index] = normalizedBoundaryRecord(symbol, start.Add(time.Duration(index)*time.Minute))
	}
	fragment := sourceFragment{SchemaVersion: "ak-historian.synthetic.normalized-fragment.v1", RequestID: "boundary-test", Symbol: symbol, SourceSchemaVersion: "synthetic.candle.v1", SourceSchemaFingerprint: testHash('9'), Records: records}
	fragment.FragmentHash, _ = canonicalHash(fragment)
	parent, err := sealParentProvenance(ParentProvenance{SourceRootID: CheckpointSourceRootID, ReceiptSHA256: testHash('7'), FragmentSHA256: fragment.FragmentHash, ParentObjectID: CheckpointSourceRootID + ":" + fragment.FragmentHash, ParentStartUTC: start, ParentEndUTC: start.Add(time.Duration(rows) * time.Minute), ParentRowCount: rows, ObservedAvailableAtUTC: time.Date(2026, 7, 17, 7, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatal(err)
	}
	return fragment, parent
}

func normalizedBoundaryRecord(symbol string, at time.Time) normalizedRecord {
	return normalizedRecord{Market: "futures-um", Symbol: symbol, Interval: "1m", Period: "1m", SourceDate: at.Format("2006-01-02"), OpenTimeMS: at.UnixMilli(), Open: "1", High: "2", Low: "0.5", Close: "1.5", Volume: "10", CloseTimeMS: at.Add(time.Minute - time.Millisecond).UnixMilli(), QuoteAssetVolume: "15", NumberOfTrades: 2, TakerBuyBaseVolume: "5", TakerBuyQuoteVolume: "7.5", MarketEventTimeUTC: at, ProviderCandleCloseTimeUTC: at.Add(time.Minute - time.Millisecond), ObservedAvailableAtUTC: at, AcquiredAtUTC: at, AcquisitionReceiptID: "boundary-test"}
}

func firstParentPath(root string, plan Plan) string {
	return filepath.Join(root, filepath.FromSlash(plan.SourceManifests[0].FragmentArtifacts[0].RelativePath))
}

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func rewriteParentAndReseal(t *testing.T, root string, plan Plan, mutate func([]normalizedRecord)) Plan {
	t.Helper()
	path := firstParentPath(root, plan)
	reader, err := gzip.NewReader(bytes.NewReader(mustReadFile(t, path)))
	if err != nil {
		t.Fatal(err)
	}
	raw, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	if err := reader.Close(); err != nil {
		t.Fatal(err)
	}
	var fragment sourceFragment
	if err := json.Unmarshal(raw, &fragment); err != nil {
		t.Fatal(err)
	}
	mutate(fragment.Records)
	fragment.FragmentHash = ""
	fragment.FragmentHash, _ = canonicalHash(fragment)
	encoded, err := json.Marshal(fragment)
	if err != nil {
		t.Fatal(err)
	}
	var compressed bytes.Buffer
	writer := gzip.NewWriter(&compressed)
	if _, err := writer.Write(encoded); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, compressed.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	plan.SourceManifests[0].FragmentArtifacts[0].CanonicalSHA256 = fragment.FragmentHash
	plan.PlanSHA256 = ""
	plan.PlanSHA256, _ = planHash(plan)
	return plan
}
