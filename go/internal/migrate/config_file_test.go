package migrate

import (
	"os"
	"path/filepath"
	"testing"
)

const examplesDir = "../../../examples"

func TestLoadMigrateConfigFileShapes(t *testing.T) {
	cases := []struct {
		name string
		file string
		want MigrateConfig
	}{
		{
			name: "shape 1 - flat",
			file: "config-migrate.example.json",
			want: MigrateConfig{
				Token:           "YOUR_SONARCLOUD_TOKEN_HERE",
				EnterpriseKey:   "YOUR_ENTERPRISE_KEY",
				URL:             "https://sonarcloud.io/",
				ExportDirectory: "./files",
				Concurrency:     10,
			},
		},
		{
			name: "shape 2 - command-sectioned",
			file: "config.example.json",
			want: MigrateConfig{
				Token:           "YOUR_SONARCLOUD_TOKEN_HERE",
				EnterpriseKey:   "YOUR_ENTERPRISE_KEY",
				URL:             "https://sonarcloud.io/",
				Edition:         "enterprise",
				ExportDirectory: "./files",
				Concurrency:     10,
			},
		},
		{
			name: "shape 3 - side-sectioned",
			file: "migration-config.example.json",
			want: MigrateConfig{
				Token:           "YOUR_SONARCLOUD_ADMIN_TOKEN_HERE",
				EnterpriseKey:   "YOUR_ENTERPRISE_KEY_HERE",
				URL:             "https://sonarcloud.io/",
				ExportDirectory: "./files",
				Concurrency:     10,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := LoadMigrateConfigFile(filepath.Join(examplesDir, tc.file))
			if err != nil {
				t.Fatalf("LoadMigrateConfigFile(%s): %v", tc.file, err)
			}
			if got != tc.want {
				t.Errorf("MigrateConfig mismatch\n got=%+v\nwant=%+v", got, tc.want)
			}
		})
	}
}

func TestLoadResetConfigFileShapes(t *testing.T) {
	cases := []struct {
		name string
		file string
		want ResetConfig
	}{
		{
			name: "shape 1 - flat",
			file: "config-migrate.example.json",
			want: ResetConfig{
				Token:           "YOUR_SONARCLOUD_TOKEN_HERE",
				EnterpriseKey:   "YOUR_ENTERPRISE_KEY",
				URL:             "https://sonarcloud.io/",
				ExportDirectory: "./files",
				Concurrency:     10,
			},
		},
		{
			name: "shape 2 - command-sectioned",
			file: "config.example.json",
			want: ResetConfig{
				Token:           "YOUR_SONARCLOUD_TOKEN_HERE",
				EnterpriseKey:   "YOUR_ENTERPRISE_KEY",
				URL:             "https://sonarcloud.io/",
				Edition:         "enterprise",
				ExportDirectory: "./files",
				Concurrency:     10,
			},
		},
		{
			name: "shape 3 - side-sectioned",
			file: "migration-config.example.json",
			want: ResetConfig{
				Token:           "YOUR_SONARCLOUD_ADMIN_TOKEN_HERE",
				EnterpriseKey:   "YOUR_ENTERPRISE_KEY_HERE",
				URL:             "https://sonarcloud.io/",
				ExportDirectory: "./files",
				Concurrency:     10,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := LoadResetConfigFile(filepath.Join(examplesDir, tc.file))
			if err != nil {
				t.Fatalf("LoadResetConfigFile(%s): %v", tc.file, err)
			}
			if got != tc.want {
				t.Errorf("ResetConfig mismatch\n got=%+v\nwant=%+v", got, tc.want)
			}
		})
	}
}

func TestLoadConfigFileErrors(t *testing.T) {
	t.Run("missing file", func(t *testing.T) {
		_, err := LoadMigrateConfigFile("/nonexistent/path/does-not-exist.json")
		if err == nil {
			t.Fatal("expected error for missing file, got nil")
		}
	})

	t.Run("malformed JSON", func(t *testing.T) {
		f, err := os.CreateTemp(t.TempDir(), "bad-*.json")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.WriteString("{not valid json"); err != nil {
			t.Fatal(err)
		}
		f.Close()
		_, err = LoadMigrateConfigFile(f.Name())
		if err == nil {
			t.Fatal("expected error for malformed JSON, got nil")
		}
	})

	t.Run("empty file", func(t *testing.T) {
		f, err := os.CreateTemp(t.TempDir(), "empty-*.json")
		if err != nil {
			t.Fatal(err)
		}
		f.Close()
		_, err = LoadResetConfigFile(f.Name())
		if err == nil {
			t.Fatal("expected error for empty file, got nil")
		}
	})
}
