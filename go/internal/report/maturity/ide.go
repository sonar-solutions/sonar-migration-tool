package maturity

import "github.com/sonar-solutions/sonar-migration-tool/internal/report"

// GenerateIDEMarkdown generates the IDE integration (SonarLint) summary.
func GenerateIDEMarkdown(users map[string][]map[string]any) string {
	total, active := 0, 0
	for _, serverUsers := range users {
		for _, user := range serverUsers {
			if conn, ok := user["sonar_lint_connection"].(string); ok && conn != "" {
				total++
			}
			if isActive, ok := user["is_active_sonar_lint"].(bool); ok && isActive {
				active++
			}
		}
	}
	return report.GenerateSection(
		[]report.Column{{"Total SonarLint Users", "total"}, {"Active SonarLint Users (30d)", "active"}},
		[]map[string]any{{"total": total, "active": active}},
		report.WithTitle("IDE Integration", 3),
	)
}
