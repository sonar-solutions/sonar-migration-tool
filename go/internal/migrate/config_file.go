package migrate

import (
	"encoding/json"
	"fmt"
	"os"
)

// configFileShape is the union of the three documented config-file formats
// (see examples/config-migrate.example.json, examples/config.example.json,
// and examples/migration-config.example.json). Issue #176.
//
// Detection at parse time:
//   - sonarcloud present -> shape 3 (side-sectioned)
//   - migrate present    -> shape 2 (command-sectioned)
//   - else               -> shape 1 (flat)
type configFileShape struct {
	// Shape 1 (flat) fields. Also reused inside Shape 2's "migrate" object.
	Token              string `json:"token"`
	EnterpriseKey      string `json:"enterprise_key"`
	URL                string `json:"url"`
	Edition            string `json:"edition"`
	ExportDirectory    string `json:"export_directory"`
	Concurrency        int    `json:"concurrency"`
	RunID              string `json:"run_id"`
	TargetTask         string `json:"target_task"`
	SkipProfiles       bool   `json:"skip_profiles"`
	IncludeScanHistory bool   `json:"include_scan_history"`
	Debug              bool   `json:"debug"`

	// Shape 2 (command-sectioned).
	Migrate *configFileShape `json:"migrate"`

	// Shape 3 (side-sectioned). sonarqube + extract blocks are ignored —
	// migrate/reset only consume SonarCloud-side values.
	SonarCloud *sonarCloudBlock `json:"sonarcloud"`
	Settings   *settingsBlock   `json:"settings"`
}

type sonarCloudBlock struct {
	URL           string `json:"url"`
	Token         string `json:"token"`
	EnterpriseKey string `json:"enterprise_key"`
	Edition       string `json:"edition"`
}

type settingsBlock struct {
	ExportDirectory string `json:"export_directory"`
	Concurrency     int    `json:"concurrency"`
}

func parseConfigFile(path string) (configFileShape, error) {
	var shape configFileShape
	data, err := os.ReadFile(path)
	if err != nil {
		return shape, fmt.Errorf("reading config file: %w", err)
	}
	if len(data) == 0 {
		return shape, fmt.Errorf("config file %s is empty", path)
	}
	if err := json.Unmarshal(data, &shape); err != nil {
		return shape, fmt.Errorf("parsing config file: %w", err)
	}
	return shape, nil
}

func (s configFileShape) toMigrateConfig() MigrateConfig {
	var cfg MigrateConfig
	switch {
	case s.SonarCloud != nil:
		cfg.Token = s.SonarCloud.Token
		cfg.URL = s.SonarCloud.URL
		cfg.EnterpriseKey = s.SonarCloud.EnterpriseKey
		cfg.Edition = s.SonarCloud.Edition
		if s.Settings != nil {
			cfg.ExportDirectory = s.Settings.ExportDirectory
			cfg.Concurrency = s.Settings.Concurrency
		}
	case s.Migrate != nil:
		return s.Migrate.toMigrateConfig()
	default:
		cfg.Token = s.Token
		cfg.EnterpriseKey = s.EnterpriseKey
		cfg.URL = s.URL
		cfg.Edition = s.Edition
		cfg.ExportDirectory = s.ExportDirectory
		cfg.Concurrency = s.Concurrency
		cfg.RunID = s.RunID
		cfg.TargetTask = s.TargetTask
		cfg.SkipProfiles = s.SkipProfiles
		cfg.IncludeScanHistory = s.IncludeScanHistory
		cfg.Debug = s.Debug
	}
	return cfg
}

func (s configFileShape) toResetConfig() ResetConfig {
	m := s.toMigrateConfig()
	return ResetConfig{
		Token:           m.Token,
		EnterpriseKey:   m.EnterpriseKey,
		Edition:         m.Edition,
		URL:             m.URL,
		Concurrency:     m.Concurrency,
		ExportDirectory: m.ExportDirectory,
		Debug:           m.Debug,
	}
}

// LoadMigrateConfigFile parses a JSON config file in any of the three
// documented shapes and returns the populated MigrateConfig.
func LoadMigrateConfigFile(path string) (MigrateConfig, error) {
	shape, err := parseConfigFile(path)
	if err != nil {
		return MigrateConfig{}, err
	}
	return shape.toMigrateConfig(), nil
}

// LoadResetConfigFile parses a JSON config file in any of the three
// documented shapes and returns the populated ResetConfig.
func LoadResetConfigFile(path string) (ResetConfig, error) {
	shape, err := parseConfigFile(path)
	if err != nil {
		return ResetConfig{}, err
	}
	return shape.toResetConfig(), nil
}
