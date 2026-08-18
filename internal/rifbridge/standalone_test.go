package rifbridge_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/david22573/ak-engine/internal/researchidentity"
	"github.com/david22573/ak-engine/internal/rifbridge"
)

func TestEngineDoesNotDependOnRIFSourceCheckout(t *testing.T) {
	root := repoRoot(t)
	goMod, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatalf("read go.mod: %v", err)
	}
	if strings.Contains(string(goMod), "ak-rif") {
		t.Fatalf("go.mod must not require or replace ak-rif")
	}

	err = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", ".cache", "bin", "runs", "vendor", "epochorchestrator":
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, imported := range file.Imports {
			importPath := strings.Trim(imported.Path.Value, "\"")
			if importPath == "ak-rif" || strings.HasPrefix(importPath, "ak-rif/") ||
				importPath == "github.com/david22573/ak-rif" || strings.HasPrefix(importPath, "github.com/david22573/ak-rif/") {
				t.Fatalf("engine source imports ak-rif implementation package: %s", path)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk source: %v", err)
	}
}

func TestResearchDiagnosticsAPIAcceptsFactsNotGovernance(t *testing.T) {
	method, ok := reflect.TypeOf(rifbridge.NewBridge()).MethodByName("EmitResearchDiagnostics")
	if !ok {
		t.Fatal("EmitResearchDiagnostics method missing")
	}
	if method.Type.NumIn() != 2 || method.Type.In(1) != reflect.TypeOf(rifbridge.ResearchAssessment{}) {
		t.Fatalf("unexpected API signature: %s", method.Type)
	}

	forbidden := []string{"promot", "approv", "frozen", "paper", "runtime", "testnet", "mainnet", "authoriz"}
	assessmentType := reflect.TypeOf(rifbridge.ResearchAssessment{})
	for i := 0; i < assessmentType.NumField(); i++ {
		field := assessmentType.Field(i)
		if field.Type.Kind() == reflect.Bool {
			t.Fatalf("research assessment must not accept a governance boolean: %s", field.Name)
		}
		name := strings.ToLower(field.Name)
		for _, term := range forbidden {
			if strings.Contains(name, term) {
				t.Fatalf("research assessment contains governance field %s", field.Name)
			}
		}
	}
	requestType := reflect.TypeOf(researchidentity.DerivationRequest{})
	for i := 0; i < requestType.NumField(); i++ {
		name := strings.ToLower(requestType.Field(i).Name)
		for _, prohibited := range []string{"observationcount", "validated", "identityhash", "candhash", "confighash", "promot", "approv", "frozen", "paper", "runtime", "testnet", "mainnet", "authoriz"} {
			if strings.Contains(name, prohibited) {
				t.Fatalf("identity derivation request accepts claimed or governance field %s", requestType.Field(i).Name)
			}
		}
	}

	resultType := reflect.TypeOf(rifbridge.ResearchDiagnosticsResult{})
	for i := 0; i < resultType.NumField(); i++ {
		name := strings.ToLower(resultType.Field(i).Name)
		for _, term := range forbidden {
			if strings.Contains(name, term) {
				t.Fatalf("research diagnostics result can represent governance through %s", resultType.Field(i).Name)
			}
		}
	}
}

func TestExactIdentityProductionPathsHaveNoPlaceholdersOrAuthorityMachineValues(t *testing.T) {
	root := repoRoot(t)
	paths := []string{
		filepath.Join(root, "internal", "researchidentity"),
		filepath.Join(root, "internal", "rifbridge"),
		filepath.Join(root, "internal", "app", "evaluate_candidate_deep.go"),
		filepath.Join(root, "internal", "app", "rif_smoke.go"),
	}
	forbiddenFragments := []string{"candhash", "confighash", "unknown-commit", "placeholder-candidate", "placeholder-config"}
	forbiddenMachineValues := map[string]bool{"approved": true, "frozen": true, "paper_ready": true, "paper_eligible": true, "authorized": true}
	for _, scope := range paths {
		err := filepath.WalkDir(scope, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
			if err != nil {
				return err
			}
			ast.Inspect(file, func(node ast.Node) bool {
				literal, ok := node.(*ast.BasicLit)
				if !ok || literal.Kind != token.STRING {
					return true
				}
				value, err := strconv.Unquote(literal.Value)
				if err != nil {
					t.Fatalf("invalid string literal in %s: %v", path, err)
				}
				lower := strings.ToLower(value)
				for _, fragment := range forbiddenFragments {
					if strings.Contains(lower, fragment) {
						t.Fatalf("production identity path contains placeholder %q in %s", fragment, path)
					}
				}
				if forbiddenMachineValues[lower] {
					t.Fatalf("production identity path contains authority machine value %q in %s", value, path)
				}
				return true
			})
			return nil
		})
		if err != nil {
			t.Fatalf("scan %s: %v", scope, err)
		}
	}
}

func TestCurrentResearchMachineFieldsExcludeAuthoritySemantics(t *testing.T) {
	root := filepath.Join(repoRoot(t), "internal", "rifbridge")
	forbiddenTags := []string{
		"is_promoted",
		"promoted",
		"promotion",
		"approved",
		"frozen",
		"paper_ready",
		"paper_eligible",
		"runtime_ready",
		"testnet_ready",
		"mainnet_ready",
		"authorized",
		"passed_integrity_checks",
	}
	legacyIdentifiers := map[string]bool{
		"EvaluateAndEmit":            true,
		"PromotionPacket":            true,
		"PromotionEvidence":          true,
		"ParsePromotionEvidenceJSON": true,
		"ResearchLock":               true,
		"ResearchAudit":              true,
		"StrictPromotionAllowed":     true,
		"PassedIntegrityChecks":      true,
		"isPromoted":                 true,
	}

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") || d.Name() == "research_governance.go" {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			return err
		}
		ast.Inspect(file, func(node ast.Node) bool {
			switch n := node.(type) {
			case *ast.Ident:
				if legacyIdentifiers[n.Name] {
					t.Fatalf("legacy authority identifier %s remains in %s", n.Name, path)
				}
			case *ast.Field:
				if n.Tag == nil {
					return true
				}
				rawTag, err := strconv.Unquote(n.Tag.Value)
				if err != nil {
					t.Fatalf("invalid struct tag in %s: %v", path, err)
				}
				jsonName := strings.Split(reflect.StructTag(rawTag).Get("json"), ",")[0]
				for _, prohibited := range forbiddenTags {
					if jsonName == prohibited || strings.Contains(jsonName, prohibited) {
						t.Fatalf("prohibited machine-readable authority field %q in %s", jsonName, path)
					}
				}
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatalf("scan current research source: %v", err)
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get cwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found")
		}
		dir = parent
	}
}
