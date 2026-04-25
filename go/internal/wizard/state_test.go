package wizard

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestNewWizardState(t *testing.T) {
	s := NewWizardState()
	if s.Phase != PhaseInit {
		t.Errorf("expected phase %q, got %q", PhaseInit, s.Phase)
	}
	if s.ExtractID != nil || s.SourceURL != nil || s.TargetURL != nil ||
		s.EnterpriseKey != nil || s.MigrationRunID != nil {
		t.Error("expected all pointer fields to be nil")
	}
	if s.OrganizationsMapped || s.ValidationPassed {
		t.Error("expected bool fields to be false")
	}
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()

	original := &WizardState{
		Phase:              PhaseStructure,
		ExtractID:          strPtr("abc-123"),
		SourceURL:          strPtr("https://sonar.example.com"),
		TargetURL:          strPtr("https://sonarcloud.io"),
		EnterpriseKey:      strPtr("my-enterprise"),
		OrganizationsMapped: true,
		ValidationPassed:   false,
		MigrationRunID:     nil,
	}

	if err := original.Save(dir); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.Phase != original.Phase {
		t.Errorf("phase: got %q, want %q", loaded.Phase, original.Phase)
	}
	if *loaded.ExtractID != *original.ExtractID {
		t.Errorf("extract_id: got %q, want %q", *loaded.ExtractID, *original.ExtractID)
	}
	if *loaded.SourceURL != *original.SourceURL {
		t.Errorf("source_url: got %q, want %q", *loaded.SourceURL, *original.SourceURL)
	}
	if loaded.OrganizationsMapped != original.OrganizationsMapped {
		t.Errorf("organizations_mapped: got %v, want %v", loaded.OrganizationsMapped, original.OrganizationsMapped)
	}
	if loaded.MigrationRunID != nil {
		t.Error("expected migration_run_id to be nil")
	}
}

func TestLoadMissingFile(t *testing.T) {
	dir := t.TempDir()
	state, err := Load(dir)
	if err != nil {
		t.Fatalf("Load missing file: %v", err)
	}
	if state.Phase != PhaseInit {
		t.Errorf("expected phase %q for missing file, got %q", PhaseInit, state.Phase)
	}
}

func TestJSONFormat(t *testing.T) {
	state := &WizardState{
		Phase:              PhaseExtract,
		ExtractID:          strPtr("run-42"),
		SourceURL:          strPtr("https://sq.local"),
		TargetURL:          nil,
		EnterpriseKey:      nil,
		OrganizationsMapped: false,
		ValidationPassed:   false,
		MigrationRunID:     nil,
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	// Verify the exact JSON format for state persistence.
	expected := `{
  "phase": "extract",
  "extract_id": "run-42",
  "source_url": "https://sq.local",
  "target_url": null,
  "enterprise_key": null,
  "organizations_mapped": false,
  "validation_passed": false,
  "migration_run_id": null
}`

	if string(data) != expected {
		t.Errorf("JSON mismatch.\nGot:\n%s\n\nWant:\n%s", string(data), expected)
	}
}

func TestSaveCreatesFile(t *testing.T) {
	dir := t.TempDir()
	state := NewWizardState()
	if err := state.Save(dir); err != nil {
		t.Fatalf("Save: %v", err)
	}

	path := filepath.Join(dir, stateFileName)
	if _, err := os.Stat(path); err != nil {
		t.Errorf("state file not created: %v", err)
	}
}
