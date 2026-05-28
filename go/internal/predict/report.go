package predict

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sonar-solutions/sonar-migration-tool/internal/report/summary"
)

// predictivePDFFilename is the output filename for predictive reports.
// Deliberately distinct from the post-migrate "migration_summary.pdf"
// so an operator can't mistake a prediction for an actual run result.
const predictivePDFFilename = "predictive_migration_summary.pdf"

// GeneratePredictiveReport synthesizes the JSONL outputs a real migrate
// run would have produced under exportDir, then runs the standard
// summary pipeline to build a PDF. Returns the path to the generated
// PDF.
//
// Inputs: extract data + mapping CSVs (organizations.csv, gates.csv,
// projects.csv, ...) under exportDir.
//
// Output: <exportDir>/predictive_migration_summary.pdf. The "Global
// Settings" section is omitted from this report (#235) because
// classifying settings requires runtime SQC support detection that the
// predict pipeline can't perform.
func GeneratePredictiveReport(exportDir string) (string, error) {
	runDir, err := BuildPredictiveRun(exportDir)
	if err != nil {
		return "", fmt.Errorf("building predictive run: %w", err)
	}

	mig, err := summary.CollectSummary(runDir, exportDir)
	if err != nil {
		return "", fmt.Errorf("collecting summary: %w", err)
	}
	mig.OmitSections = map[string]bool{"Global Settings": true}

	pdfBytes, err := summary.RenderPDF(mig)
	if err != nil {
		return "", fmt.Errorf("rendering PDF: %w", err)
	}

	outPath := filepath.Join(exportDir, predictivePDFFilename)
	if err := os.WriteFile(outPath, pdfBytes, 0o644); err != nil {
		return "", fmt.Errorf("writing PDF: %w", err)
	}
	return outPath, nil
}
