package summary

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestCollectSummaryEmpty(t *testing.T) {
	dir := t.TempDir()
	summary, err := CollectSummary(dir)
	if err != nil {
		t.Fatalf("CollectSummary: %v", err)
	}
	if summary == nil {
		t.Fatal("expected non-nil summary")
	}
	if len(summary.Sections) != len(sectionDefs) {
		t.Errorf("expected %d sections, got %d", len(sectionDefs), len(summary.Sections))
	}
}

func TestCollectSummaryWithData(t *testing.T) {
	dir := t.TempDir()

	// Write createProjects output (successes)
	writeTaskJSONL(t, dir, "createProjects", []map[string]any{
		{"name": "Project A", "sonarcloud_org_key": "org1", "cloud_project_key": "org1_projA"},
		{"name": "Project B", "sonarcloud_org_key": "org1", "cloud_project_key": "org1_projB"},
	})

	// Write generateProjectMappings (includes a skipped item)
	writeTaskJSONL(t, dir, "generateProjectMappings", []map[string]any{
		{"name": "Project A", "sonarcloud_org_key": "org1"},
		{"name": "Project B", "sonarcloud_org_key": "org1"},
		{"name": "Project C", "sonarcloud_org_key": "SKIPPED", "sonarqube_org_key": "old-org"},
	})

	// Write createGates output
	writeTaskJSONL(t, dir, "createGates", []map[string]any{
		{"name": "Custom Gate", "sonarcloud_org_key": "org1", "cloud_gate_id": "gate-1"},
	})

	summary, err := CollectSummary(dir)
	if err != nil {
		t.Fatalf("CollectSummary: %v", err)
	}

	// Projects section
	projSection := findSection(summary, "Projects")
	if projSection == nil {
		t.Fatal("missing Projects section")
	}
	if len(projSection.Succeeded) != 2 {
		t.Errorf("expected 2 succeeded projects, got %d", len(projSection.Succeeded))
	}
	if len(projSection.Skipped) != 1 {
		t.Errorf("expected 1 skipped project, got %d", len(projSection.Skipped))
	}
	if projSection.Skipped[0].Name != "Project C" {
		t.Errorf("expected skipped project 'Project C', got %q", projSection.Skipped[0].Name)
	}

	// Gates section
	gateSection := findSection(summary, "Quality Gates")
	if gateSection == nil {
		t.Fatal("missing Quality Gates section")
	}
	if len(gateSection.Succeeded) != 1 {
		t.Errorf("expected 1 succeeded gate, got %d", len(gateSection.Succeeded))
	}
}

func TestCollectSummaryWithScanHistory(t *testing.T) {
	dir := t.TempDir()

	writeTaskJSONL(t, dir, "createProjects", []map[string]any{
		{"name": "Proj1", "sonarcloud_org_key": "org1", "cloud_project_key": "org1_proj1"},
		{"name": "Proj2", "sonarcloud_org_key": "org1", "cloud_project_key": "org1_proj2"},
	})

	writeTaskJSONL(t, dir, "importScanHistory", []map[string]any{
		{"cloud_project_key": "org1_proj1", "status": "success", "branch": "main"},
		{"cloud_project_key": "org1_proj2", "status": "failed", "branch": "main", "error": "CE failed"},
	})

	summary, err := CollectSummary(dir)
	if err != nil {
		t.Fatalf("CollectSummary: %v", err)
	}

	projSection := findSection(summary, "Projects")
	if projSection == nil {
		t.Fatal("missing Projects section")
	}

	// Check scan history is attached
	for _, item := range projSection.Succeeded {
		if item.Name == "Proj1" {
			if item.Detail != "org1_proj1|scan:success" {
				t.Errorf("expected scan:success in detail, got %q", item.Detail)
			}
		}
		if item.Name == "Proj2" {
			if item.Detail != "org1_proj2|scan:failed" {
				t.Errorf("expected scan:failed in detail, got %q", item.Detail)
			}
		}
	}
}

func TestCollectSummaryWithFailures(t *testing.T) {
	dir := t.TempDir()

	// Write a requests.log with a failure entry
	logEntry := map[string]any{
		"process_type": "request_completed",
		"status":       "failure",
		"payload": map[string]any{
			"method": "POST",
			"url":    "/api/projects/create",
			"status": float64(400),
			"data": map[string]any{
				"name":         "FailProj",
				"organization": "org1",
			},
			"response": `{"errors":[{"msg":"already exists"}]}`,
		},
	}
	logBytes, _ := json.Marshal(logEntry)
	os.WriteFile(filepath.Join(dir, "requests.log"), logBytes, 0o644)

	summary, err := CollectSummary(dir)
	if err != nil {
		t.Fatalf("CollectSummary: %v", err)
	}

	projSection := findSection(summary, "Projects")
	if len(projSection.Failed) != 1 {
		t.Errorf("expected 1 failed project, got %d", len(projSection.Failed))
	}
	if projSection.Failed[0].Name != "FailProj" {
		t.Errorf("expected 'FailProj', got %q", projSection.Failed[0].Name)
	}
}

func TestExtractRunID(t *testing.T) {
	cases := []struct {
		path string
		want string
	}{
		{"/app/files/04-27-2026-02", "04-27-2026-02"},
		{"files/run-01", "run-01"},
		{"single", "single"},
	}
	for _, tc := range cases {
		got := extractRunID(tc.path)
		if got != tc.want {
			t.Errorf("extractRunID(%q): got %q, want %q", tc.path, got, tc.want)
		}
	}
}

func TestParseScanHistory(t *testing.T) {
	detail, status := parseScanHistory("org1_proj|scan:success")
	if detail != "org1_proj" || status != "success" {
		t.Errorf("got detail=%q status=%q", detail, status)
	}

	detail2, status2 := parseScanHistory("org1_proj")
	if detail2 != "org1_proj" || status2 != "" {
		t.Errorf("no scan: got detail=%q status=%q", detail2, status2)
	}
}

func TestScanStatusLabel(t *testing.T) {
	if scanStatusLabel("success") != "Yes" {
		t.Error("expected Yes")
	}
	if scanStatusLabel("failed") != "Failed" {
		t.Error("expected Failed")
	}
	if scanStatusLabel("skipped") != "No" {
		t.Error("expected No")
	}
	if scanStatusLabel("") != "" {
		t.Error("expected empty")
	}
}

// --- helpers ---

func findSection(s *MigrationSummary, name string) *Section {
	for i := range s.Sections {
		if s.Sections[i].Name == name {
			return &s.Sections[i]
		}
	}
	return nil
}

func writeTaskJSONL(t *testing.T, dir, taskName string, items []map[string]any) {
	t.Helper()
	taskDir := filepath.Join(dir, taskName)
	os.MkdirAll(taskDir, 0o755)
	f, err := os.Create(filepath.Join(taskDir, "results.1.jsonl"))
	if err != nil {
		t.Fatalf("create %s: %v", taskName, err)
	}
	defer f.Close()
	for _, item := range items {
		b, _ := json.Marshal(item)
		f.Write(b)
		f.Write([]byte("\n"))
	}
}
