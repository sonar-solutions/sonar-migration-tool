package extract

import (
	"encoding/json"
	"fmt"
	"os"
)

// configFileShape is the union of the three documented config-file
// formats for the extract command (issue #158):
//
//   - examples/config-extract.example.json — flat top-level keys
//   - examples/config.example.json         — "extract" sub-object
//   - examples/migration-config.example.json — "sonarqube" + "settings"
//     sub-objects (the SonarCloud-side blocks are ignored by extract)
//
// Detection at parse time:
//   - sonarqube present -> shape 3 (side-sectioned)
//   - extract present   -> shape 2 (command-sectioned)
//   - else              -> shape 1 (flat)
type configFileShape struct {
	// Shape 1 (flat) fields. Reused inside Shape 2's "extract" object.
	URL                string `json:"url"`
	Token              string `json:"token"`
	ExportDirectory    string `json:"export_directory"`
	ExtractType        string `json:"extract_type"`
	PEMFilePath        string `json:"pem_file_path"`
	KeyFilePath        string `json:"key_file_path"`
	CertPassword       string `json:"cert_password"`
	Concurrency        int    `json:"concurrency"`
	Timeout            int    `json:"timeout"`
	ExtractID          string `json:"extract_id"`
	TargetTask         string `json:"target_task"`
	IncludeScanHistory bool   `json:"include_scan_history"`
	ProjectKey         string `json:"project_key"`

	// Shape 2 (command-sectioned).
	Extract *configFileShape `json:"extract"`

	// Shape 3 (side-sectioned). The SonarCloud side of
	// migration-config.example.json is ignored — extract only consumes
	// the SonarQube-side credentials and shared settings block.
	SonarQube *sonarQubeBlock `json:"sonarqube"`
	Settings  *settingsBlock  `json:"settings"`
}

type sonarQubeBlock struct {
	URL   string `json:"url"`
	Token string `json:"token"`
}

type settingsBlock struct {
	ExportDirectory string `json:"export_directory"`
	Concurrency     int    `json:"concurrency"`
	Timeout         int    `json:"timeout"`
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

func (s configFileShape) toExtractConfig() ExtractConfig {
	var cfg ExtractConfig
	switch {
	case s.SonarQube != nil:
		cfg.URL = s.SonarQube.URL
		cfg.Token = s.SonarQube.Token
		if s.Settings != nil {
			cfg.ExportDirectory = s.Settings.ExportDirectory
			cfg.Concurrency = s.Settings.Concurrency
			cfg.Timeout = s.Settings.Timeout
		}
	case s.Extract != nil:
		return s.Extract.toExtractConfig()
	default:
		cfg.URL = s.URL
		cfg.Token = s.Token
		cfg.ExportDirectory = s.ExportDirectory
		cfg.ExtractType = s.ExtractType
		cfg.PEMFilePath = s.PEMFilePath
		cfg.KeyFilePath = s.KeyFilePath
		cfg.CertPassword = s.CertPassword
		cfg.Concurrency = s.Concurrency
		cfg.Timeout = s.Timeout
		cfg.ExtractID = s.ExtractID
		cfg.TargetTask = s.TargetTask
		cfg.IncludeScanHistory = s.IncludeScanHistory
		cfg.ProjectKey = s.ProjectKey
	}
	return cfg
}

// LoadExtractConfigFile parses a JSON config file in any of the three
// documented shapes and returns the populated ExtractConfig.
func LoadExtractConfigFile(path string) (ExtractConfig, error) {
	shape, err := parseConfigFile(path)
	if err != nil {
		return ExtractConfig{}, err
	}
	return shape.toExtractConfig(), nil
}
