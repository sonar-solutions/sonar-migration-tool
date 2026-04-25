package maturity

import (
	"strconv"

	"github.com/sonar-solutions/sonar-migration-tool/internal/report"
	"github.com/sonar-solutions/sonar-migration-tool/internal/report/common"
)

// GenerateCoverageMarkdown generates the test coverage summary.
func GenerateCoverageMarkdown(measures common.Measures) string {
	var linesToCover, uncoveredLines, newLinesToCover, newUncoveredLines int

	for _, serverProjects := range measures {
		for _, project := range serverProjects {
			linesToCover += toIntMetric(project, "lines_to_cover")
			uncoveredLines += toIntMetric(project, "uncovered_lines")
			newLinesToCover += toIntMetric(project, "new_lines_to_cover")
			newUncoveredLines += toIntMetric(project, "new_uncovered_lines")
		}
	}

	coveredLines := linesToCover - uncoveredLines
	newCoveredLines := newLinesToCover - newUncoveredLines
	coverage := calcPercentage(linesToCover, uncoveredLines)
	newCoverage := calcPercentage(newLinesToCover, newUncoveredLines)

	row := map[string]any{
		"lines_to_cover":     linesToCover,
		"covered_lines":      coveredLines,
		"uncovered_lines":    uncoveredLines,
		"coverage":           coverage,
		"new_lines_to_cover": newLinesToCover,
		"new_covered_lines":  newCoveredLines,
		"new_uncovered_lines": newUncoveredLines,
		"new_coverage":       newCoverage,
	}
	return report.GenerateSection(
		[]report.Column{
			{"Lines to Cover", "lines_to_cover"}, {"Covered Lines", "covered_lines"},
			{"Uncovered Lines", "uncovered_lines"}, {"Coverage %", "coverage"},
			{"New Lines to Cover", "new_lines_to_cover"}, {"New Covered Lines", "new_covered_lines"},
			{"New Uncovered Lines", "new_uncovered_lines"}, {"New Code Coverage %", "new_coverage"},
		},
		[]map[string]any{row},
		report.WithTitle("Test Coverage", 3),
	)
}

func toIntMetric(project map[string]any, key string) int {
	v, ok := project[key]
	if !ok {
		return 0
	}
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case string:
		if i, err := strconv.Atoi(n); err == nil {
			return i
		}
	}
	return 0
}

func calcPercentage(total, uncovered int) float64 {
	if total == 0 {
		return 0
	}
	return float64(total-uncovered) / float64(total) * 100
}
