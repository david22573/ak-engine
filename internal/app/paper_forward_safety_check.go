package app

import (
	"encoding/json"
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var (
	pfscOut     string
	pfscJSONOut string
)

type PaperForwardSafetyReport struct {
	SchemaVersion    string               `json:"schema_version"`
	GeneratedAtUTC   string               `json:"generated_at_utc"`
	Status           string               `json:"status"`
	CheckedFiles     []string             `json:"checked_files"`
	ForbiddenImports []string             `json:"forbidden_imports"`
	Findings         []PaperSafetyFinding `json:"findings"`
}

type PaperSafetyFinding struct {
	File       string `json:"file"`
	ImportPath string `json:"import_path"`
	Reason     string `json:"reason"`
}

var paperForwardSafetyCheckCmd = &cobra.Command{
	Use:   "paper-forward-safety-check",
	Short: "Verify paper-forward code has no execution or broker imports",
	RunE: func(cmd *cobra.Command, args []string) error {
		report, err := runPaperForwardSafetyCheck(defaultPaperForwardSafetyFiles())
		if err != nil {
			return err
		}
		if err := writePaperForwardSafetyReport(report, pfscOut, pfscJSONOut); err != nil {
			return err
		}
		if report.Status != "PASS" {
			return fmt.Errorf("paper forward safety check failed with %d findings", len(report.Findings))
		}
		fmt.Printf("Paper forward safety check: %s\n", report.Status)
		return nil
	},
}

func defaultPaperForwardSafetyFiles() []string {
	return []string{
		filepath.Join("internal", "app", "paper_forward.go"),
		filepath.Join("internal", "app", "paper_forward_common.go"),
		filepath.Join("internal", "app", "paper_forward_canonical.go"),
		filepath.Join("internal", "app", "paper_forward_grade_pending.go"),
		filepath.Join("internal", "app", "paper_shadow_readiness.go"),
		filepath.Join("internal", "app", "paper_signal.go"),
		filepath.Join("internal", "app", "paper_signal_grade.go"),
		filepath.Join("internal", "app", "paper_signal_review.go"),
		filepath.Join("internal", "papersignal", "types.go"),
	}
}

func defaultPaperForwardForbiddenImports() []string {
	return []string{
		"broker",
		"execution",
		"order",
		"signing",
		"secret",
		"secrets",
		"credential",
		"credentials",
		"ak-trader",
	}
}

func runPaperForwardSafetyCheck(files []string) (PaperForwardSafetyReport, error) {
	report := PaperForwardSafetyReport{
		SchemaVersion:    "1.0",
		GeneratedAtUTC:   time.Now().UTC().Format(time.RFC3339),
		Status:           "PASS",
		CheckedFiles:     append([]string(nil), files...),
		ForbiddenImports: defaultPaperForwardForbiddenImports(),
		Findings:         []PaperSafetyFinding{},
	}
	for _, file := range files {
		imports, err := parseGoImportPaths(file)
		if err != nil {
			report.Findings = append(report.Findings, PaperSafetyFinding{File: file, Reason: err.Error()})
			continue
		}
		for _, importPath := range imports {
			if reason := forbiddenPaperImportReason(importPath, report.ForbiddenImports); reason != "" {
				report.Findings = append(report.Findings, PaperSafetyFinding{
					File:       file,
					ImportPath: importPath,
					Reason:     reason,
				})
			}
		}
	}
	if len(report.Findings) > 0 {
		report.Status = "FAIL"
	}
	return report, nil
}

func parseGoImportPaths(path string) ([]string, error) {
	fset := token.NewFileSet()
	parsed, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
	if err != nil {
		return nil, err
	}
	imports := make([]string, 0, len(parsed.Imports))
	for _, imp := range parsed.Imports {
		imports = append(imports, strings.Trim(imp.Path.Value, `"`))
	}
	return imports, nil
}

func forbiddenPaperImportReason(importPath string, forbidden []string) string {
	lower := strings.ToLower(importPath)
	for _, term := range forbidden {
		term = strings.ToLower(term)
		if lower == term || strings.Contains(lower, "/"+term+"/") || strings.HasSuffix(lower, "/"+term) || strings.Contains(lower, term+".") {
			return "forbidden paper-only import contains " + term
		}
	}
	return ""
}

func writePaperForwardSafetyReport(report PaperForwardSafetyReport, mdOut, jsonOut string) error {
	if jsonOut != "" {
		if err := os.MkdirAll(filepath.Dir(jsonOut), 0755); err != nil {
			return err
		}
		data, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			return err
		}
		if err := os.WriteFile(jsonOut, data, 0644); err != nil {
			return err
		}
	}
	if mdOut != "" {
		if err := os.MkdirAll(filepath.Dir(mdOut), 0755); err != nil {
			return err
		}
		var md strings.Builder
		md.WriteString("# Forward Paper Safety Check\n\n")
		md.WriteString(fmt.Sprintf("- Result: `%s`\n", report.Status))
		md.WriteString("- Scope: paper-forward runner, pending grader, shadow-readiness report, and paper signal schema\n")
		md.WriteString("- Forbidden imports: " + strings.Join(report.ForbiddenImports, ", ") + "\n\n")
		if len(report.Findings) > 0 {
			md.WriteString("## Findings\n")
			for _, finding := range report.Findings {
				md.WriteString(fmt.Sprintf("- %s imports `%s`: %s\n", finding.File, finding.ImportPath, finding.Reason))
			}
		} else {
			md.WriteString("No broker, execution, order, signing, secrets, credentials, or ak-trader imports were found.\n")
		}
		if err := os.WriteFile(mdOut, []byte(md.String()), 0644); err != nil {
			return err
		}
	}
	return nil
}

func init() {
	paperForwardSafetyCheckCmd.Flags().StringVar(&pfscOut, "out", "runs/reports/forward_paper_safety_check.md", "Markdown output")
	paperForwardSafetyCheckCmd.Flags().StringVar(&pfscJSONOut, "json-out", "runs/reports/forward_paper_safety_check.json", "JSON output")
	rootCmd.AddCommand(paperForwardSafetyCheckCmd)
}
