package structure

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExportCSVEmpty(t *testing.T) {
	dir := t.TempDir()
	err := ExportCSV(dir, "empty", []Organization{})
	if err != nil {
		t.Fatal(err)
	}
	// Should create an empty file.
	_, err = os.Stat(filepath.Join(dir, "empty.csv"))
	if err != nil {
		t.Errorf("expected file to exist: %v", err)
	}
}

func TestLoadCSVMissing(t *testing.T) {
	dir := t.TempDir()
	result, err := LoadCSV(dir, "nonexistent.csv")
	if err != nil {
		t.Errorf("expected nil error for missing file, got %v", err)
	}
	if result != nil {
		t.Errorf("expected nil result, got %v", result)
	}
}

func TestCSVRoundTripProjects(t *testing.T) {
	dir := t.TempDir()
	projects := []Project{
		{
			Key: "proj1", Name: "Project 1", GateName: "Custom",
			Profiles:               []any{map[string]any{"key": "prof1", "language": "java"}},
			ServerURL:              "https://sq.example.com/",
			SonarQubeOrgKey:        "org1",
			MainBranch:             "main",
			IsCloudBinding:         true,
			NewCodeDefinitionType:  "days",
			NewCodeDefinitionValue: 30,
			ALM:                    "github",
			Repository:             "org/repo",
			Monorepo:               false,
		},
	}

	if err := ExportCSV(dir, "projects", projects); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadCSV(dir, "projects.csv")
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded) != 1 {
		t.Fatalf("expected 1 row, got %d", len(loaded))
	}

	row := loaded[0]
	if row["key"] != "proj1" {
		t.Errorf("expected key=proj1, got %v", row["key"])
	}
	if row["is_cloud_binding"] != true {
		t.Errorf("expected is_cloud_binding=true, got %v (%T)", row["is_cloud_binding"], row["is_cloud_binding"])
	}
	if row["main_branch"] != "main" {
		t.Errorf("expected main_branch=main, got %v", row["main_branch"])
	}
}

func TestCSVRoundTripTemplates(t *testing.T) {
	dir := t.TempDir()
	templates := []Template{
		{
			UniqueKey: "org1tpl1", SourceTemplateKey: "tpl1", Name: "Default",
			ProjectKeyPattern: "proj.*", ServerURL: "https://sq.example.com/",
			IsDefault: true, SonarQubeOrgKey: "org1",
		},
	}

	if err := ExportCSV(dir, "templates", templates); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadCSV(dir, "templates.csv")
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded) != 1 {
		t.Fatalf("expected 1 row, got %d", len(loaded))
	}
	if loaded[0]["is_default"] != true {
		t.Errorf("expected is_default=true, got %v", loaded[0]["is_default"])
	}
}

func TestCoerceCSVValue(t *testing.T) {
	tests := []struct {
		input    string
		expected any
	}{
		{"true", true},
		{"false", false},
		{`["a","b"]`, []any{"a", "b"}},
		{`{"key":"val"}`, map[string]any{"key": "val"}},
		{"hello", "hello"},
		{"", ""},
		{"null", nil},
	}
	for _, tt := range tests {
		got := coerceCSVValue(tt.input)
		switch expected := tt.expected.(type) {
		case nil:
			if got != nil {
				t.Errorf("coerceCSVValue(%q) = %v, want nil", tt.input, got)
			}
		case bool:
			if got != expected {
				t.Errorf("coerceCSVValue(%q) = %v, want %v", tt.input, got, expected)
			}
		case string:
			if got != expected {
				t.Errorf("coerceCSVValue(%q) = %v, want %v", tt.input, got, expected)
			}
		default:
			// For slices/maps, just check type.
			if got == nil {
				t.Errorf("coerceCSVValue(%q) = nil, want %v", tt.input, expected)
			}
		}
	}
}
