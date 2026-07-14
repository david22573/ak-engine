package main

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestPreconditionCommandCannotReadValidationOrHoldoutContent(t *testing.T) {
	_, sourcePath, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate command source")
	}
	file, err := parser.ParseFile(token.NewFileSet(), sourcePath[:len(sourcePath)-len("main_test.go")]+"main.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	allowedFlags := map[string]bool{"historian-gap": false, "out-dir": false, "generated-at": false}
	allowedHistoricalReports := map[string]bool{
		"runs/reports/pr4b0_candidate_inventory.json":     false,
		"runs/reports/pr4b0_candidate_qualification.json": false,
		"runs/reports/pr4b0_r1_variant_results.json":      false,
		"runs/reports/pr4b0_r1_final_decision.json":       false,
		"runs/reports/pr4b0_r1_evidence_supplement.json":  false,
	}
	readIdentifiers := map[string]int{}
	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		if selector, ok := call.Fun.(*ast.SelectorExpr); ok {
			owner, _ := selector.X.(*ast.Ident)
			if owner != nil && ((owner.Name == "os" && (selector.Sel.Name == "Open" || selector.Sel.Name == "OpenFile" || selector.Sel.Name == "ReadDir")) || (owner.Name == "filepath" && (selector.Sel.Name == "Glob" || selector.Sel.Name == "Walk" || selector.Sel.Name == "WalkDir"))) {
				t.Errorf("precondition command contains forbidden content-enumeration primitive %s.%s", owner.Name, selector.Sel.Name)
			}
			if owner != nil && owner.Name == "os" && selector.Sel.Name == "ReadFile" {
				identifier, ok := call.Args[0].(*ast.Ident)
				if !ok || (identifier.Name != "historianPath" && identifier.Name != "path") {
					t.Errorf("unexpected precondition input read: %#v", call.Args[0])
				} else {
					readIdentifiers[identifier.Name]++
				}
			}
			if owner != nil && owner.Name == "flag" && selector.Sel.Name == "StringVar" {
				name := stringLiteral(t, call.Args[1])
				if _, allowed := allowedFlags[name]; !allowed {
					t.Errorf("unexpected precondition input flag %q", name)
				} else {
					allowedFlags[name] = true
				}
			}
		}
		if identifier, ok := call.Fun.(*ast.Ident); ok && identifier.Name == "fileDigest" {
			path := stringLiteral(t, call.Args[0])
			if _, allowed := allowedHistoricalReports[path]; !allowed {
				t.Errorf("unexpected historical evidence read %q", path)
			} else {
				allowedHistoricalReports[path] = true
			}
		}
		return true
	})
	if readIdentifiers["historianPath"] != 1 || readIdentifiers["path"] != 1 {
		t.Fatalf("unexpected read topology: %#v", readIdentifiers)
	}
	for name, found := range allowedFlags {
		if !found {
			t.Errorf("expected flag %q was not declared", name)
		}
	}
	for path, found := range allowedHistoricalReports {
		if !found {
			t.Errorf("expected historical evidence %q was not bound", path)
		}
	}
}

func stringLiteral(t *testing.T, expression ast.Expr) string {
	t.Helper()
	literal, ok := expression.(*ast.BasicLit)
	if !ok || literal.Kind != token.STRING {
		t.Fatalf("expected string literal, got %#v", expression)
	}
	value, err := strconv.Unquote(literal.Value)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func TestHistorianGapConsumerFailsClosedOnSubstitution(t *testing.T) {
	base := syntheticHistorianGap(t)
	if err := validateHistorianGap(base); err != nil {
		t.Fatalf("synthetic authoritative gap did not validate: %v", err)
	}
	tests := map[string]func(*historianGap){
		"dataset identity":   func(gap *historianGap) { gap.DatasetVersion = "sha256:" + strings.Repeat("b", 64) },
		"manifest identity":  func(gap *historianGap) { gap.ManifestHash = "sha256:" + strings.Repeat("c", 64) },
		"candidate identity": func(gap *historianGap) { gap.CandidateVersion = "v2" },
		"missing context":    func(gap *historianGap) { gap.RequiredContextSymbols = []string{"BTCUSDT"} },
		"narrowed partitions": func(gap *historianGap) {
			gap.ExpectedPartitions = gap.ExpectedPartitions[:len(gap.ExpectedPartitions)-1]
		},
		"snapshot path": func(gap *historianGap) {
			var snapshot historianSnapshotIdentity
			if err := json.Unmarshal(gap.Snapshots[0], &snapshot); err != nil {
				t.Fatal(err)
			}
			snapshot.RelativePath = "/synthetic/escape.parquet"
			gap.Snapshots[0], _ = json.Marshal(snapshot)
		},
		"availability substitution": func(gap *historianGap) {
			var snapshot historianSnapshotIdentity
			if err := json.Unmarshal(gap.Snapshots[0], &snapshot); err != nil {
				t.Fatal(err)
			}
			available := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
			snapshot.SourceAvailableAt = &available
			gap.Snapshots[0], _ = json.Marshal(snapshot)
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			gap := cloneHistorianGap(t, base)
			mutate(&gap)
			if err := validateHistorianGap(gap); err == nil {
				t.Fatal("substituted Historian evidence passed")
			}
		})
	}
}

func TestHistorianGapDecoderRejectsUnknownAndTrailingJSON(t *testing.T) {
	data, err := json.Marshal(syntheticHistorianGap(t))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := decodeHistorianGap(data); err != nil {
		t.Fatalf("valid synthetic JSON failed: %v", err)
	}
	var object map[string]any
	if err := json.Unmarshal(data, &object); err != nil {
		t.Fatal(err)
	}
	object["unexpected_substitution"] = true
	unknown, err := json.Marshal(object)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := decodeHistorianGap(unknown); err == nil {
		t.Fatal("unknown Historian field passed")
	}
	if _, err := decodeHistorianGap(append(data, []byte("\n{}")...)); err == nil {
		t.Fatal("trailing Historian JSON passed")
	}
}

func syntheticHistorianGap(t *testing.T) historianGap {
	t.Helper()
	gap := historianGap{
		SchemaVersion: "ak-historian.pit-gap-manifest.v1", Status: "PIT_EVIDENCE_INCOMPLETE",
		DatasetID: "ak-historian-candles-futures-um-1m-pr4b0-r1p", DatasetVersion: historianDatasetVersion,
		ManifestID: "pr4b0-r1p-pit-gap-2023-2025-v1", ManifestHash: historianManifestHash,
		CandidateID: candidateID, CandidateVersion: candidateVersion, ImplementationHash: candidateImplementationHash,
		PhysicalCoverageStart: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), PhysicalCoverageEnd: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		EvaluationCutoff: time.Date(2026, 7, 13, 7, 0, 0, 0, time.UTC), CoveragePolicyVersion: "ak-historian.coverage-policy.v1",
		AvailabilityPolicyVersion: "ak-historian.availability-policy.v1", EventSchemaVersion: "legacy-unversioned",
		RequiredSymbols:        []string{"ADAUSDT", "AVAXUSDT", "BNBUSDT", "DOGEUSDT", "ETHUSDT", "LINKUSDT", "SOLUSDT", "XRPUSDT"},
		RequiredContextSymbols: []string{"BTCUSDT", "ETHUSDT"}, SnapshotSetHash: historianDatasetVersion,
		ManifestCreatedAt: time.Date(2026, 7, 13, 8, 0, 0, 0, time.UTC), HistorianBuild: "synthetic-historian-build-v1",
	}
	symbols := []string{"ADAUSDT", "AVAXUSDT", "BNBUSDT", "BTCUSDT", "DOGEUSDT", "ETHUSDT", "LINKUSDT", "SOLUSDT", "XRPUSDT"}
	for month := gap.PhysicalCoverageStart; month.Before(gap.PhysicalCoverageEnd); month = month.AddDate(0, 1, 0) {
		for _, symbol := range symbols {
			partition := "futures-um/1m/" + symbol + "/" + month.Format("2006-01")
			gap.ExpectedPartitions = append(gap.ExpectedPartitions, partition)
			snapshot, err := json.Marshal(historianSnapshotIdentity{
				PartitionKey: partition,
				RelativePath: "candles/futures-um/1m/symbol=" + symbol + "/year=" + month.Format("2006") + "/month=" + month.Format("01") + "/" + symbol + "-1m-" + month.Format("2006-01") + ".parquet",
				ContentHash:  "sha256:" + strings.Repeat("d", 64), PartitionHash: "sha256:" + strings.Repeat("d", 64),
				EvidenceGaps: []string{"AVAILABILITY_TIMESTAMP_MISSING", "SNAPSHOT_SCHEMA_UNSUPPORTED"},
			})
			if err != nil {
				t.Fatal(err)
			}
			missing, err := json.Marshal(historianMissingEvidence{PartitionKey: partition, Reasons: []string{"AVAILABILITY_TIMESTAMP_MISSING", "SNAPSHOT_SCHEMA_UNSUPPORTED"}})
			if err != nil {
				t.Fatal(err)
			}
			gap.Snapshots = append(gap.Snapshots, snapshot)
			gap.MissingEvidence = append(gap.MissingEvidence, missing)
		}
	}
	return gap
}

func cloneHistorianGap(t *testing.T, value historianGap) historianGap {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	var clone historianGap
	if err := json.Unmarshal(data, &clone); err != nil {
		t.Fatal(err)
	}
	return clone
}
