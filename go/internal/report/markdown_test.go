package report

import (
	"strings"
	"testing"
)

func TestFormatValue(t *testing.T) {
	tests := []struct {
		name string
		val  any
		want string
	}{
		{"nil", nil, ""},
		{"true", true, "Yes"},
		{"false", false, "No"},
		{"int", 1000, "1,000"},
		{"int small", 42, "42"},
		{"int negative", -1500, "-1,500"},
		{"float", 1000.567, "1,000.57"},
		{"float small", 3.14, "3.14"},
		{"string", "hello", "hello"},
		{"string slice", []string{"a", "b", "c"}, "a, b, c"},
		{"any slice", []any{"x", "y"}, "x, y"},
		{"int zero", 0, "0"},
		{"float zero", 0.0, "0.00"},
	}
	for _, tt := range tests {
		got := FormatValue(tt.val)
		if got != tt.want {
			t.Errorf("FormatValue(%s) = %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestGenerateSectionBasic(t *testing.T) {
	columns := []Column{
		{"Name", "name"},
		{"Count", "count"},
	}
	rows := []map[string]any{
		{"name": "Alpha", "count": 10},
		{"name": "Beta", "count": 20},
	}

	result := GenerateSection(columns, rows)

	if !strings.Contains(result, "| Name | Count |") {
		t.Error("missing header row")
	}
	if !strings.Contains(result, "| Alpha | 10 |") {
		t.Error("missing Alpha row")
	}
	if !strings.Contains(result, "| Beta | 20 |") {
		t.Error("missing Beta row")
	}
	if !strings.Contains(result, "|:---|:---|") {
		t.Error("missing alignment row")
	}
}

func TestGenerateSectionWithTitle(t *testing.T) {
	columns := []Column{{"Key", "key"}}
	rows := []map[string]any{{"key": "val"}}

	result := GenerateSection(columns, rows, WithTitle("My Section", 2))

	if !strings.HasPrefix(result, "## My Section\n") {
		t.Errorf("expected title, got: %q", result[:30])
	}
}

func TestGenerateSectionWithDescription(t *testing.T) {
	columns := []Column{{"Key", "key"}}
	rows := []map[string]any{{"key": "val"}}

	result := GenerateSection(columns, rows,
		WithTitle("Title", 3),
		WithDescription("Some description here"),
	)

	if !strings.Contains(result, "Some description here\n") {
		t.Error("missing description")
	}
}

func TestGenerateSectionWithFilter(t *testing.T) {
	columns := []Column{{"Name", "name"}, {"Active", "active"}}
	rows := []map[string]any{
		{"name": "A", "active": true},
		{"name": "B", "active": false},
		{"name": "C", "active": true},
	}

	result := GenerateSection(columns, rows,
		WithFilter(func(r map[string]any) bool {
			v, _ := r["active"].(bool)
			return v
		}),
	)

	if strings.Contains(result, "| B |") {
		t.Error("filtered row B should not appear")
	}
	if !strings.Contains(result, "| A |") || !strings.Contains(result, "| C |") {
		t.Error("active rows should appear")
	}
}

func TestGenerateSectionWithSort(t *testing.T) {
	columns := []Column{{"Name", "name"}, {"Score", "score"}}
	rows := []map[string]any{
		{"name": "Low", "score": 10},
		{"name": "High", "score": 90},
		{"name": "Mid", "score": 50},
	}

	result := GenerateSection(columns, rows, WithSortBy("score", true))

	highIdx := strings.Index(result, "High")
	midIdx := strings.Index(result, "Mid")
	lowIdx := strings.Index(result, "Low")
	if highIdx > midIdx || midIdx > lowIdx {
		t.Errorf("expected desc order: High, Mid, Low")
	}
}

func TestGenerateSectionEmptyRows(t *testing.T) {
	columns := []Column{{"Name", "name"}}
	result := GenerateSection(columns, nil)

	// Should still have header and alignment rows.
	if !strings.Contains(result, "| Name |") {
		t.Error("missing header even with empty rows")
	}
	lines := strings.Split(strings.TrimSpace(result), "\n")
	if len(lines) != 2 {
		t.Errorf("expected 2 lines (header + align), got %d", len(lines))
	}
}

func TestGenerateSectionColumnOrder(t *testing.T) {
	columns := []Column{
		{"First", "a"},
		{"Second", "b"},
		{"Third", "c"},
	}
	rows := []map[string]any{{"a": "1", "b": "2", "c": "3"}}

	result := GenerateSection(columns, rows)

	headerLine := strings.Split(result, "\n")[0]
	firstIdx := strings.Index(headerLine, "First")
	secondIdx := strings.Index(headerLine, "Second")
	thirdIdx := strings.Index(headerLine, "Third")
	if firstIdx >= secondIdx || secondIdx >= thirdIdx {
		t.Errorf("columns out of order: %s", headerLine)
	}
}

func TestFormatValueLargeInt(t *testing.T) {
	got := FormatValue(1234567)
	if got != "1,234,567" {
		t.Errorf("got %q", got)
	}
}

func TestAddCommasEdgeCases(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"0", "0"},
		{"123", "123"},
		{"1234", "1,234"},
		{"1000000", "1,000,000"},
	}
	for _, tt := range tests {
		got := addCommas(tt.input)
		if got != tt.want {
			t.Errorf("addCommas(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
