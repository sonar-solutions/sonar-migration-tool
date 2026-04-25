package cmd

import (
	"github.com/sonar-solutions/sonar-migration-tool/internal/structure"
	"github.com/spf13/cobra"
)

var structureCmd = &cobra.Command{
	Use:   "structure",
	Short: "Group projects into organizations",
	Long:  "Groups projects into organizations based on DevOps Bindings and Server Urls. Outputs organizations and projects as CSVs.",
	RunE: func(cmd *cobra.Command, args []string) error {
		exportDir, _ := cmd.Flags().GetString("export_directory")
		return structure.RunStructure(exportDir)
	},
}

func init() {
	structureCmd.Flags().String("export_directory", "/app/files/", "Root directory containing all SonarQube exports")
}
