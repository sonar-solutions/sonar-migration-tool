package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sonar-solutions/sonar-migration-tool/internal/extract"
	"github.com/sonar-solutions/sonar-migration-tool/internal/migrate"
	"github.com/sonar-solutions/sonar-migration-tool/internal/report/summary"
	"github.com/sonar-solutions/sonar-migration-tool/internal/structure"
	"github.com/spf13/cobra"
)

var transferCmd = &cobra.Command{
	Use:   "transfer",
	Short: "Transfer a single project from SonarQube Server to SonarQube Cloud",
	Long: `Transfer migrates a SonarQube Server project to SonarQube Cloud in one command.

It chains extract → structure → mappings → migrate automatically, eliminating
the manual CSV-editing step. Credentials for both sides are required.

Example (flags):
  sonar-migration-tool transfer \
    --sq-url https://sonarqube.example.com \
    --sq-token sqp_xxx \
    --project-key my-project \
    --sc-token squ_xxx \
    --sc-org my-org

Example (config file):
  sonar-migration-tool transfer -c config.json

config.json format:
  {
    "sonarqube": { "url": "...", "token": "...", "projectKey": "..." },
    "sonarcloud": { "token": "...", "organization": "...", "enterpriseKey": "..." }
  }

The enterpriseKey is required for portfolio migration; for projects/gates/profiles
it can be omitted and defaults to the organization key.`,
	RunE: runTransfer,
}

func init() {
	f := transferCmd.Flags()
	f.StringP("config", "c", "", "Path to transfer config file")
	f.String("sq-url", "", "SonarQube Server URL")
	f.String("sq-token", "", "SonarQube Server token")
	f.String("project-key", "", "Project key to transfer (omit to transfer all projects)")
	f.String("sc-token", "", "SonarQube Cloud token")
	f.String("sc-org", "", "SonarQube Cloud organization key")
	f.String("sc-enterprise-key", "", "SonarQube Cloud enterprise key (defaults to --sc-org)")
	f.String("export-dir", "./migration-files/", "Working directory for intermediate files")
	f.Bool("include-scan-history", false, "Extract and import full issue/hotspot scan history")
	f.Int("concurrency", 0, "Max concurrent requests (default: 25)")
	f.Bool("debug", false, "Enable debug-level logging")
}

// transferFileConfig is the transfer-specific config file shape (CloudVoyager-compatible).
// This is intentionally separate from the existing extract/migrate config shapes to avoid
// field-name conflicts with the existing side-sectioned parser.
type transferFileConfig struct {
	SonarQube struct {
		URL        string `json:"url"`
		Token      string `json:"token"`
		ProjectKey string `json:"projectKey"`
	} `json:"sonarqube"`
	SonarCloud struct {
		Token         string `json:"token"`
		Organization  string `json:"organization"`
		EnterpriseKey string `json:"enterpriseKey"`
	} `json:"sonarcloud"`
	Settings struct {
		ExportDirectory    string `json:"exportDirectory"`
		Concurrency        int    `json:"concurrency"`
		IncludeScanHistory bool   `json:"includeScanHistory"`
		Debug              bool   `json:"debug"`
	} `json:"settings"`
}

func loadTransferConfigFile(path string) (transferFileConfig, error) {
	var cfg transferFileConfig
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, fmt.Errorf("reading transfer config %q: %w", path, err)
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parsing transfer config %q: %w", path, err)
	}
	return cfg, nil
}

func runTransfer(cmd *cobra.Command, _ []string) error {
	// Start from a zero-valued file config (all fields overridable by flags).
	var fileCfg transferFileConfig

	configFile, _ := cmd.Flags().GetString("config")
	if configFile != "" {
		loaded, err := loadTransferConfigFile(configFile)
		if err != nil {
			return err
		}
		fileCfg = loaded
	}

	// Flags override config file values.
	sqURL := fileCfg.SonarQube.URL
	sqToken := fileCfg.SonarQube.Token
	projectKey := fileCfg.SonarQube.ProjectKey
	scToken := fileCfg.SonarCloud.Token
	scOrg := fileCfg.SonarCloud.Organization
	scEnterpriseKey := fileCfg.SonarCloud.EnterpriseKey
	exportDir := fileCfg.Settings.ExportDirectory
	concurrency := fileCfg.Settings.Concurrency
	includeScanHistory := fileCfg.Settings.IncludeScanHistory
	debug := fileCfg.Settings.Debug

	if cmd.Flags().Changed("sq-url") {
		sqURL, _ = cmd.Flags().GetString("sq-url")
	}
	if cmd.Flags().Changed("sq-token") {
		sqToken, _ = cmd.Flags().GetString("sq-token")
	}
	if cmd.Flags().Changed("project-key") {
		projectKey, _ = cmd.Flags().GetString("project-key")
	}
	if cmd.Flags().Changed("sc-token") {
		scToken, _ = cmd.Flags().GetString("sc-token")
	}
	if cmd.Flags().Changed("sc-org") {
		scOrg, _ = cmd.Flags().GetString("sc-org")
	}
	if cmd.Flags().Changed("sc-enterprise-key") {
		scEnterpriseKey, _ = cmd.Flags().GetString("sc-enterprise-key")
	}
	if cmd.Flags().Changed("export-dir") {
		exportDir, _ = cmd.Flags().GetString("export-dir")
	}
	if cmd.Flags().Changed("concurrency") {
		concurrency, _ = cmd.Flags().GetInt("concurrency")
	}
	if cmd.Flags().Changed("include-scan-history") {
		includeScanHistory, _ = cmd.Flags().GetBool("include-scan-history")
	}
	if cmd.Flags().Changed("debug") {
		debug, _ = cmd.Flags().GetBool("debug")
	}

	// Apply defaults.
	if exportDir == "" {
		exportDir = "./migration-files/"
	}
	// enterpriseKey defaults to org key — sufficient for all non-portfolio operations.
	if scEnterpriseKey == "" {
		scEnterpriseKey = scOrg
	}

	// Validation.
	if sqURL == "" || sqToken == "" {
		return fmt.Errorf("SonarQube Server URL and token are required (--sq-url / --sq-token or config file)")
	}
	if scToken == "" || scOrg == "" {
		return fmt.Errorf("SonarQube Cloud token and organization key are required (--sc-token / --sc-org or config file)")
	}

	ctx := cmd.Context()

	// Phase 1: Extract.
	fmt.Println("[1/4] Extracting from SonarQube Server...")
	var projectKeys []string
	if projectKey != "" {
		projectKeys = []string{projectKey}
	}
	extractCfg := extract.ExtractConfig{
		URL:                sqURL,
		Token:              sqToken,
		ExportDirectory:    exportDir,
		ProjectKeys:        projectKeys,
		Concurrency:        concurrency,
		IncludeScanHistory: includeScanHistory,
	}
	skipped, err := extract.RunExtract(ctx, extractCfg)
	if err != nil {
		return fmt.Errorf("extract failed: %w", err)
	}
	if len(skipped) > 0 {
		fmt.Fprintf(os.Stderr, "Warning: %d project(s) skipped (insufficient privileges):\n", len(skipped))
		for _, k := range skipped {
			fmt.Fprintf(os.Stderr, "  - %s\n", k)
		}
	}

	// Phase 2: Structure.
	fmt.Println("[2/4] Building organization structure...")
	if err := structure.RunStructure(exportDir, scOrg); err != nil {
		return fmt.Errorf("structure failed: %w", err)
	}

	// Phase 3: Mappings.
	fmt.Println("[3/4] Generating entity mappings...")
	if err := structure.RunMappings(exportDir); err != nil {
		return fmt.Errorf("mappings failed: %w", err)
	}

	// Phase 4: Migrate.
	fmt.Println("[4/4] Migrating to SonarQube Cloud...")
	migrateCfg := migrate.MigrateConfig{
		Token:              scToken,
		EnterpriseKey:      scEnterpriseKey,
		ExportDirectory:    exportDir,
		Concurrency:        concurrency,
		IncludeScanHistory: includeScanHistory,
		Debug:              debug,
	}
	runID, err := migrate.RunMigrate(ctx, migrateCfg)
	if err != nil {
		return fmt.Errorf("migrate failed: %w", err)
	}

	// PDF summary report (same as migrate command).
	runDir := filepath.Join(exportDir, runID)
	if pdfPath, pdfErr := summary.GeneratePDFReport(runDir, exportDir, exportDir); pdfErr == nil {
		fmt.Printf("PDF summary report: %s\n", pdfPath)
	}

	fmt.Println("Transfer complete.")
	return nil
}
