package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "sonar-migration-tool",
	Short: "Migrate SonarQube Server instances to SonarQube Cloud",
	Long: `CLI tool for migrating SonarQube Server instances to SonarQube Cloud.
Extracts configuration from a Server instance, transforms it, and applies
it to a Cloud organization. Also updates CI/CD pipelines post-migration.`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	addCommands()
}

func addCommands() {
	rootCmd.AddCommand(
		wizardCmd,
		guiCmd,
		extractCmd,
		reportCmd,
		structureCmd,
		mappingsCmd,
		migrateCmd,
		resetCmd,
		analysisReportCmd,
	)
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}
