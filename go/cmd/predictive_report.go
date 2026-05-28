package cmd

import (
	"fmt"

	"github.com/sonar-solutions/sonar-migration-tool/internal/predict"
	"github.com/spf13/cobra"
)

var predictiveReportCmd = &cobra.Command{
	Use:   "predictive-report",
	Short: "Generate a predictive migration PDF report without running migrate",
	Long: `Generates a PDF migration summary from the data produced by extract,
structure, and the user-edited mapping CSVs alone — no calls to SonarQube
Cloud, no actual migration. Two classes of outcomes from a real migrate
run cannot be predicted and are omitted (#235):
  - SonarQube Cloud API errors / rate limiting.
  - Global settings (SQC-support detection is dynamic).

The PDF is written to <export_directory>/predictive_migration_summary.pdf.`,
	RunE: runPredictiveReport,
}

func init() {
	f := predictiveReportCmd.Flags()
	f.String("export_directory", "/app/files/", "Root directory containing extract data and the mapping CSVs")
}

func runPredictiveReport(cmd *cobra.Command, args []string) error {
	exportDir, _ := cmd.Flags().GetString("export_directory")
	pdfPath, err := predict.GeneratePredictiveReport(exportDir)
	if err != nil {
		return err
	}
	fmt.Printf("Predictive PDF report: %s\n", pdfPath)
	return nil
}
