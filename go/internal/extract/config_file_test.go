package extract

import (
	"os"
	"path/filepath"
	"testing"
)

const examplesDir = "../../../examples"

func TestLoadExtractConfigFileShapes(t *testing.T) {
	cases := []struct {
		name string
		file string
		want ExtractConfig
	}{
		{
			name: "shape 1 - flat",
			file: "config-extract.example.json",
			want: ExtractConfig{
				URL:             "http://localhost:9000",
				Token:           "YOUR_SONARQUBE_TOKEN_HERE",
				ExportDirectory: "./files",
				Concurrency:     10,
				Timeout:         60,
			},
		},
		{
			name: "shape 2 - command-sectioned",
			file: "config.example.json",
			want: ExtractConfig{
				URL:             "http://localhost:9000",
				Token:           "YOUR_SONARQUBE_TOKEN_HERE",
				ExportDirectory: "./files",
				ExtractType:     "all",
				Concurrency:     10,
				Timeout:         60,
			},
		},
		{
			name: "shape 3 - side-sectioned",
			file: "migration-config.example.json",
			want: ExtractConfig{
				URL:             "http://localhost:9000",
				Token:           "YOUR_SONARQUBE_ADMIN_TOKEN_HERE",
				ExportDirectory: "./files",
				Concurrency:     10,
				Timeout:         60,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := LoadExtractConfigFile(filepath.Join(examplesDir, tc.file))
			if err != nil {
				t.Fatalf("LoadExtractConfigFile(%s): %v", tc.file, err)
			}
			if got != tc.want {
				t.Errorf("ExtractConfig mismatch\n got=%+v\nwant=%+v", got, tc.want)
			}
		})
	}
}

func TestLoadExtractConfigFileSnakeCaseFields(t *testing.T) {
	// Regression for issue #158: every snake_case field in the flat
	// shape must round-trip into the corresponding CamelCase struct
	// field. Before the fix, json.Unmarshal silently dropped these
	// because ExtractConfig had no json: tags, so users had to type
	// "exportDirectory" / "extractType" / etc. instead of the
	// documented snake_case keys.
	f, err := os.CreateTemp(t.TempDir(), "extract-cfg-*.json")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.WriteString(`{
		"url": "http://sq.example.com:9000",
		"token": "tok",
		"export_directory": "/data/files",
		"extract_type": "all",
		"pem_file_path": "/certs/client.pem",
		"key_file_path": "/certs/client.key",
		"cert_password": "secret",
		"concurrency": 25,
		"timeout": 90,
		"extract_id": "resume-me",
		"target_task": "getRules",
		"include_scan_history": true
	}`)
	f.Close()

	got, err := LoadExtractConfigFile(f.Name())
	if err != nil {
		t.Fatalf("LoadExtractConfigFile: %v", err)
	}

	want := ExtractConfig{
		URL:                "http://sq.example.com:9000",
		Token:              "tok",
		ExportDirectory:    "/data/files",
		ExtractType:        "all",
		PEMFilePath:        "/certs/client.pem",
		KeyFilePath:        "/certs/client.key",
		CertPassword:       "secret",
		Concurrency:        25,
		Timeout:            90,
		ExtractID:          "resume-me",
		TargetTask:         "getRules",
		IncludeScanHistory: true,
	}
	if got != want {
		t.Errorf("snake_case round-trip mismatch\n got=%+v\nwant=%+v", got, want)
	}
}

func TestLoadExtractConfigFileErrors(t *testing.T) {
	t.Run("missing file", func(t *testing.T) {
		_, err := LoadExtractConfigFile("/nonexistent/path/does-not-exist.json")
		if err == nil {
			t.Fatal("expected error for missing file, got nil")
		}
	})

	t.Run("malformed JSON", func(t *testing.T) {
		f, err := os.CreateTemp(t.TempDir(), "bad-*.json")
		if err != nil {
			t.Fatal(err)
		}
		_, _ = f.WriteString("{not valid")
		f.Close()
		_, err = LoadExtractConfigFile(f.Name())
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
		_, err = LoadExtractConfigFile(f.Name())
		if err == nil {
			t.Fatal("expected error for empty file, got nil")
		}
	})
}
