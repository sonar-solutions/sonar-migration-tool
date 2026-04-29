package summary

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRenderPDFMinimal(t *testing.T) {
	summary := &MigrationSummary{
		RunID:       "test-run-01",
		GeneratedAt: time.Now(),
		Sections: []Section{
			{Name: "Projects"},
			{Name: "Quality Profiles"},
		},
	}

	pdfBytes, err := RenderPDF(summary)
	if err != nil {
		t.Fatalf("RenderPDF: %v", err)
	}
	if len(pdfBytes) == 0 {
		t.Fatal("expected non-empty PDF")
	}
	// Check PDF magic header
	if string(pdfBytes[:5]) != "%PDF-" {
		t.Errorf("expected PDF header, got %q", string(pdfBytes[:5]))
	}
}

func TestRenderPDFWithData(t *testing.T) {
	summary := &MigrationSummary{
		RunID:       "04-27-2026-02",
		GeneratedAt: time.Now(),
		Sections: []Section{
			{
				Name: "Projects",
				Succeeded: []EntityItem{
					{Name: "Project A", Organization: "org1", Detail: "org1_projA|scan:success"},
					{Name: "Project B", Organization: "org1", Detail: "org1_projB|scan:failed"},
				},
				Failed: []EntityItem{
					{Name: "Project C", Organization: "org1", ErrorMessage: "already exists"},
				},
				Skipped: []EntityItem{
					{Name: "Project D", Organization: "old-org", Detail: "Organization skipped"},
				},
			},
			{
				Name: "Quality Gates",
				Succeeded: []EntityItem{
					{Name: "Custom Gate", Organization: "org1", Detail: "gate-1"},
				},
			},
			{
				Name: "Groups",
				Succeeded: []EntityItem{
					{Name: "DevTeam", Organization: "org1"},
					{Name: "QATeam", Organization: "org1"},
				},
				Failed: []EntityItem{
					{Name: "AdminGroup", Organization: "org1", ErrorMessage: "unauthorized"},
				},
			},
		},
	}

	pdfBytes, err := RenderPDF(summary)
	if err != nil {
		t.Fatalf("RenderPDF: %v", err)
	}
	if len(pdfBytes) < 100 {
		t.Errorf("PDF seems too small: %d bytes", len(pdfBytes))
	}
}

func TestRenderPDFEmptySections(t *testing.T) {
	summary := &MigrationSummary{
		RunID:       "empty-run",
		GeneratedAt: time.Now(),
		Sections:    []Section{{Name: "Projects"}, {Name: "Quality Gates"}},
	}

	pdfBytes, err := RenderPDF(summary)
	if err != nil {
		t.Fatalf("RenderPDF: %v", err)
	}
	if string(pdfBytes[:5]) != "%PDF-" {
		t.Error("expected valid PDF")
	}
}

func TestGeneratePDFReport(t *testing.T) {
	dir := t.TempDir()
	runDir := filepath.Join(dir, "test-run-01")
	os.MkdirAll(runDir, 0o755)

	// Write some task data
	writeTaskJSONL(t, runDir, "createProjects", []map[string]any{
		{"name": "Proj1", "sonarcloud_org_key": "org1", "cloud_project_key": "org1_proj1"},
	})

	pdfPath, err := GeneratePDFReport(runDir, dir)
	if err != nil {
		t.Fatalf("GeneratePDFReport: %v", err)
	}

	if filepath.Base(pdfPath) != "migration_summary.pdf" {
		t.Errorf("expected migration_summary.pdf, got %s", filepath.Base(pdfPath))
	}

	data, err := os.ReadFile(pdfPath)
	if err != nil {
		t.Fatalf("reading PDF: %v", err)
	}
	if string(data[:5]) != "%PDF-" {
		t.Error("output file is not a valid PDF")
	}
}

func TestTruncate(t *testing.T) {
	if truncate("short", 10) != "short" {
		t.Error("should not truncate short string")
	}
	got := truncate("this is a long string", 10)
	if got != "this is..." {
		t.Errorf("expected 'this is...', got %q", got)
	}
}

func TestSubsectionColumnsWithScanHistory(t *testing.T) {
	items := []EntityItem{
		{Detail: "proj1|scan:success"},
	}
	headers, widths := subsectionColumns(false, items)
	if len(headers) != 4 {
		t.Errorf("expected 4 columns with scan history, got %d", len(headers))
	}
	if headers[3] != "Scan History" {
		t.Errorf("expected 'Scan History' column, got %q", headers[3])
	}
	if len(widths) != 4 {
		t.Errorf("expected 4 widths, got %d", len(widths))
	}
}

func TestSubsectionColumnsWithoutScanHistory(t *testing.T) {
	items := []EntityItem{
		{Detail: "proj1"},
	}
	headers, _ := subsectionColumns(false, items)
	if len(headers) != 3 {
		t.Errorf("expected 3 columns without scan history, got %d", len(headers))
	}
}

func TestSubsectionColumnsFailure(t *testing.T) {
	items := []EntityItem{
		{ErrorMessage: "error"},
	}
	headers, _ := subsectionColumns(true, items)
	if len(headers) != 3 || headers[2] != "Error" {
		t.Errorf("expected failure columns, got %v", headers)
	}
}
