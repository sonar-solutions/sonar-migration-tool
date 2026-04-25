package report

import (
	"fmt"
	"sort"
	"strings"
)

// Column defines a column in a markdown table.
type Column struct {
	Header string // Display name
	Key    string // Row map key
}

// sectionConfig holds options for GenerateSection.
type sectionConfig struct {
	title       string
	level       int
	description string
	sortKey     string
	sortDesc    bool
	filterFn    func(map[string]any) bool
}

// SectionOption configures GenerateSection behavior.
type SectionOption func(*sectionConfig)

// WithTitle adds a markdown heading to the section.
func WithTitle(title string, level int) SectionOption {
	return func(c *sectionConfig) {
		c.title = title
		c.level = level
	}
}

// WithDescription adds a description line below the title.
func WithDescription(desc string) SectionOption {
	return func(c *sectionConfig) { c.description = desc }
}

// WithSortBy sorts rows by the given key. Set desc=true for descending.
func WithSortBy(key string, desc bool) SectionOption {
	return func(c *sectionConfig) {
		c.sortKey = key
		c.sortDesc = desc
	}
}

// WithFilter applies a predicate to include only matching rows.
func WithFilter(fn func(map[string]any) bool) SectionOption {
	return func(c *sectionConfig) { c.filterFn = fn }
}

// GenerateSection produces a markdown table section.
func GenerateSection(columns []Column, rows []map[string]any, opts ...SectionOption) string {
	cfg := &sectionConfig{level: 3}
	for _, opt := range opts {
		opt(cfg)
	}

	var sb strings.Builder

	writeHeader(&sb, cfg)
	writeTableHeader(&sb, columns)
	writeTableRows(&sb, columns, applyFiltersAndSort(rows, cfg))

	return sb.String()
}

func writeHeader(sb *strings.Builder, cfg *sectionConfig) {
	if cfg.title != "" {
		sb.WriteString(strings.Repeat("#", cfg.level))
		sb.WriteString(" ")
		sb.WriteString(cfg.title)
		sb.WriteString("\n")
	}
	if cfg.description != "" {
		sb.WriteString(cfg.description)
		sb.WriteString("\n")
	}
}

func writeTableHeader(sb *strings.Builder, columns []Column) {
	sb.WriteString("| ")
	for i, col := range columns {
		if i > 0 {
			sb.WriteString(" | ")
		}
		sb.WriteString(col.Header)
	}
	sb.WriteString(" |\n")

	sb.WriteString("|")
	for range columns {
		sb.WriteString(":---|")
	}
	sb.WriteString("\n")
}

func writeTableRows(sb *strings.Builder, columns []Column, rows []map[string]any) {
	for _, row := range rows {
		sb.WriteString("| ")
		for i, col := range columns {
			if i > 0 {
				sb.WriteString(" | ")
			}
			sb.WriteString(FormatValue(row[col.Key]))
		}
		sb.WriteString(" |\n")
	}
}

func applyFiltersAndSort(rows []map[string]any, cfg *sectionConfig) []map[string]any {
	filtered := rows
	if cfg.filterFn != nil {
		filtered = filterRows(rows, cfg.filterFn)
	}
	if cfg.sortKey != "" {
		sortRows(filtered, cfg.sortKey, cfg.sortDesc)
	}
	return filtered
}

func filterRows(rows []map[string]any, fn func(map[string]any) bool) []map[string]any {
	var result []map[string]any
	for _, row := range rows {
		if fn(row) {
			result = append(result, row)
		}
	}
	return result
}

func sortRows(rows []map[string]any, key string, desc bool) {
	sort.SliceStable(rows, func(i, j int) bool {
		vi := sortValue(rows[i][key])
		vj := sortValue(rows[j][key])
		if desc {
			return vi > vj
		}
		return vi < vj
	})
}

func sortValue(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case int:
		return float64(n)
	case bool:
		if n {
			return 1
		}
		return 0
	}
	return 0
}

// FormatValue converts a value to a display string for markdown tables.
func FormatValue(v any) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case bool:
		if val {
			return "Yes"
		}
		return "No"
	case int:
		return formatIntCommas(val)
	case float64:
		return formatFloatCommas(val)
	case []string:
		return strings.Join(val, ", ")
	case []any:
		return formatSlice(val)
	case string:
		return val
	default:
		return fmt.Sprintf("%v", v)
	}
}

func formatIntCommas(n int) string {
	s := fmt.Sprintf("%d", n)
	return addCommas(s)
}

func formatFloatCommas(f float64) string {
	intPart := int(f)
	frac := fmt.Sprintf("%.2f", f-float64(intPart))
	// frac is like "0.57" — take the ".57" part
	decimal := frac[1:] // includes the dot
	return addCommas(fmt.Sprintf("%d", intPart)) + decimal
}

func addCommas(s string) string {
	if len(s) <= 3 {
		return s
	}
	neg := ""
	if s[0] == '-' {
		neg = "-"
		s = s[1:]
	}
	var parts []string
	for len(s) > 3 {
		parts = append([]string{s[len(s)-3:]}, parts...)
		s = s[:len(s)-3]
	}
	parts = append([]string{s}, parts...)
	return neg + strings.Join(parts, ",")
}

func formatSlice(arr []any) string {
	strs := make([]string, 0, len(arr))
	for _, item := range arr {
		strs = append(strs, fmt.Sprintf("%v", item))
	}
	return strings.Join(strs, ", ")
}
