package summary

import (
	"fmt"
	"os"
	"path/filepath"
)

// GeneratePDFReport collects migration data from runDir and writes a PDF
// summary report to outputDir. Returns the path to the generated PDF.
func GeneratePDFReport(runDir, outputDir string) (string, error) {
	if outputDir == "" {
		outputDir = filepath.Dir(runDir)
	}

	summary, err := CollectSummary(runDir)
	if err != nil {
		return "", fmt.Errorf("collecting summary: %w", err)
	}

	pdfBytes, err := RenderPDF(summary)
	if err != nil {
		return "", fmt.Errorf("rendering PDF: %w", err)
	}

	outPath := filepath.Join(outputDir, "migration_summary.pdf")
	if err := os.WriteFile(outPath, pdfBytes, 0o644); err != nil {
		return "", fmt.Errorf("writing PDF: %w", err)
	}

	return outPath, nil
}
