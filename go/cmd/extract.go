package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/sonar-solutions/sonar-migration-tool/internal/extract"
	"github.com/spf13/cobra"
)

var extractCmd = &cobra.Command{
	Use:   "extract [url] [token]",
	Short: "Extract data from a SonarQube Server instance",
	Long:  "Extracts data from a SonarQube Server instance and stores it in the export directory as new line delimited json files.",
	Args:  cobra.MaximumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := buildExtractConfig(cmd, args)
		if err != nil {
			return err
		}
		if cfg.URL == "" || cfg.Token == "" {
			return fmt.Errorf("URL and TOKEN are required (either as arguments or in config file)")
		}
		skipped, err := extract.RunExtract(cmd.Context(), cfg)
		if err != nil {
			return err
		}
		if len(skipped) > 0 {
			fmt.Fprintf(os.Stderr, "\n%d project(s) skipped (insufficient privileges):\n", len(skipped))
			for _, key := range skipped {
				fmt.Fprintf(os.Stderr, "  - %s\n", key)
			}
		}
		return nil
	},
}

func init() {
	f := extractCmd.Flags()
	f.String("config", "", "Path to JSON configuration file")
	f.String("pem_file_path", "", "Path to client certificate pem file")
	f.String("key_file_path", "", "Path to client certificate key file")
	f.String("cert_password", "", "Password for client certificate")
	f.String("export_directory", "/app/files/", "Root directory to output the export")
	f.String("extract_type", "", "Type of extract to run")
	f.Int("concurrency", 0, "Maximum number of concurrent requests")
	f.Int("timeout", 0, "Number of seconds before a request will timeout")
	f.String("extract_id", "", "ID of an extract to resume in case of failures")
	f.String("target_task", "", "Target task to complete; all dependent tasks will be included")
}

func buildExtractConfig(cmd *cobra.Command, args []string) (extract.ExtractConfig, error) {
	var cfg extract.ExtractConfig

	// Load config file if specified.
	configFile, _ := cmd.Flags().GetString("config")
	if configFile != "" {
		data, err := os.ReadFile(configFile)
		if err != nil {
			return cfg, fmt.Errorf("reading config file: %w", err)
		}
		if err := json.Unmarshal(data, &cfg); err != nil {
			return cfg, fmt.Errorf("parsing config file: %w", err)
		}
	}

	// CLI args override config file.
	if len(args) > 0 && args[0] != "" {
		cfg.URL = args[0]
	}
	if len(args) > 1 && args[1] != "" {
		cfg.Token = args[1]
	}

	// Flags override everything.
	overrideString(cmd, "pem_file_path", &cfg.PEMFilePath)
	overrideString(cmd, "key_file_path", &cfg.KeyFilePath)
	overrideString(cmd, "cert_password", &cfg.CertPassword)
	overrideString(cmd, "export_directory", &cfg.ExportDirectory)
	overrideString(cmd, "extract_type", &cfg.ExtractType)
	overrideString(cmd, "extract_id", &cfg.ExtractID)
	overrideString(cmd, "target_task", &cfg.TargetTask)
	overrideInt(cmd, "concurrency", &cfg.Concurrency)
	overrideInt(cmd, "timeout", &cfg.Timeout)

	return cfg, nil
}

func overrideString(cmd *cobra.Command, flag string, target *string) {
	if cmd.Flags().Changed(flag) {
		val, _ := cmd.Flags().GetString(flag)
		*target = val
	}
}

func overrideInt(cmd *cobra.Command, flag string, target *int) {
	if cmd.Flags().Changed(flag) {
		val, _ := cmd.Flags().GetInt(flag)
		*target = val
	}
}
