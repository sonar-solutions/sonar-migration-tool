package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
)

func newResetTestCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "reset"}
	f := cmd.Flags()
	f.String("config", "", "")
	f.String("edition", "enterprise", "")
	f.String("url", "https://sonarcloud.io/", "")
	f.Int("concurrency", 25, "")
	f.String("export_directory", "/app/files/", "")
	f.Bool("debug", false, "")
	return cmd
}

func writeConfigFile(t *testing.T, contents string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "reset-cfg-*.json")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(contents); err != nil {
		t.Fatal(err)
	}
	f.Close()
	return f.Name()
}

func TestBuildResetConfigFromConfigFile(t *testing.T) {
	path := writeConfigFile(t, `{
		"token": "cfg-token",
		"enterprise_key": "cfg-ent",
		"url": "https://cfg.example.com/",
		"export_directory": "/cfg/files",
		"concurrency": 7
	}`)

	cmd := newResetTestCmd()
	if err := cmd.Flags().Set("config", path); err != nil {
		t.Fatal(err)
	}

	cfg, err := buildResetConfig(cmd, nil)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Token != "cfg-token" {
		t.Errorf("Token: got %q want cfg-token", cfg.Token)
	}
	if cfg.EnterpriseKey != "cfg-ent" {
		t.Errorf("EnterpriseKey: got %q want cfg-ent", cfg.EnterpriseKey)
	}
	if cfg.URL != "https://cfg.example.com/" {
		t.Errorf("URL: got %q", cfg.URL)
	}
	if cfg.ExportDirectory != "/cfg/files" {
		t.Errorf("ExportDirectory: got %q", cfg.ExportDirectory)
	}
	if cfg.Concurrency != 7 {
		t.Errorf("Concurrency: got %d want 7", cfg.Concurrency)
	}
}

func TestBuildResetConfigFlagsOverrideConfig(t *testing.T) {
	path := writeConfigFile(t, `{
		"token": "cfg-token",
		"enterprise_key": "cfg-ent",
		"url": "https://cfg.example.com/",
		"concurrency": 7
	}`)

	cmd := newResetTestCmd()
	_ = cmd.Flags().Set("config", path)
	_ = cmd.Flags().Set("url", "https://flag.example.com/")
	_ = cmd.Flags().Set("concurrency", "99")
	_ = cmd.Flags().Set("debug", "true")

	cfg, err := buildResetConfig(cmd, nil)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.URL != "https://flag.example.com/" {
		t.Errorf("URL: flag should win, got %q", cfg.URL)
	}
	if cfg.Concurrency != 99 {
		t.Errorf("Concurrency: flag should win, got %d", cfg.Concurrency)
	}
	if !cfg.Debug {
		t.Errorf("Debug: flag should win (true)")
	}
	if cfg.Token != "cfg-token" || cfg.EnterpriseKey != "cfg-ent" {
		t.Errorf("config-file values should persist when no flag/arg overrides them, got token=%q ent=%q",
			cfg.Token, cfg.EnterpriseKey)
	}
}

func TestBuildResetConfigPositionalArgsOverrideConfig(t *testing.T) {
	path := writeConfigFile(t, `{
		"token": "cfg-token",
		"enterprise_key": "cfg-ent"
	}`)

	cmd := newResetTestCmd()
	_ = cmd.Flags().Set("config", path)

	cfg, err := buildResetConfig(cmd, []string{"arg-token", "arg-ent"})
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Token != "arg-token" {
		t.Errorf("Token: positional should win, got %q", cfg.Token)
	}
	if cfg.EnterpriseKey != "arg-ent" {
		t.Errorf("EnterpriseKey: positional should win, got %q", cfg.EnterpriseKey)
	}
}

func TestBuildResetConfigNoConfigUsesPositionalAndFlags(t *testing.T) {
	cmd := newResetTestCmd()
	_ = cmd.Flags().Set("url", "https://manual.example.com/")
	_ = cmd.Flags().Set("export_directory", "/manual/dir")

	cfg, err := buildResetConfig(cmd, []string{"tok", "ent"})
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Token != "tok" {
		t.Errorf("Token: got %q", cfg.Token)
	}
	if cfg.EnterpriseKey != "ent" {
		t.Errorf("EnterpriseKey: got %q", cfg.EnterpriseKey)
	}
	if cfg.URL != "https://manual.example.com/" {
		t.Errorf("URL: got %q", cfg.URL)
	}
	if cfg.ExportDirectory != "/manual/dir" {
		t.Errorf("ExportDirectory: got %q", cfg.ExportDirectory)
	}
}

func TestBuildResetConfigShape3FromExampleFile(t *testing.T) {
	// Round-trip the documented side-sectioned example via buildResetConfig.
	path, err := filepath.Abs("../../examples/migration-config.example.json")
	if err != nil {
		t.Fatal(err)
	}

	cmd := newResetTestCmd()
	_ = cmd.Flags().Set("config", path)

	cfg, err := buildResetConfig(cmd, nil)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Token != "YOUR_SONARCLOUD_ADMIN_TOKEN_HERE" {
		t.Errorf("Token: got %q", cfg.Token)
	}
	if cfg.EnterpriseKey != "YOUR_ENTERPRISE_KEY_HERE" {
		t.Errorf("EnterpriseKey: got %q", cfg.EnterpriseKey)
	}
	if cfg.URL != "https://sonarcloud.io/" {
		t.Errorf("URL: got %q", cfg.URL)
	}
	if cfg.ExportDirectory != "./files" {
		t.Errorf("ExportDirectory: got %q", cfg.ExportDirectory)
	}
	if cfg.Concurrency != 10 {
		t.Errorf("Concurrency: got %d", cfg.Concurrency)
	}
}
