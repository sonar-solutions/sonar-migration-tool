// Copyright (C) SonarSource Sàrl
// For more information, see https://sonarsource.com/legal/
// mailto:info AT sonarsource DOT com

package maturity

import (
	"github.com/sonar-solutions/sonar-migration-tool/internal/report"
	"github.com/sonar-solutions/sonar-migration-tool/internal/report/common"
)

// GenerateScansMarkdown generates the scan activity summary (30-day window).
func GenerateScansMarkdown(projectScans common.ProjectScans) string {
	scans30d, failed30d, projectsFailed := 0, 0, 0
	for _, ciTools := range projectScans {
		for _, projects := range ciTools {
			for _, scan := range projects {
				s30, _ := scan["scan_count_30_days"].(int)
				f30, _ := scan["failed_scans_30_days"].(int)
				scans30d += s30
				failed30d += f30
				if f30 > 0 {
					projectsFailed++
				}
			}
		}
	}
	return report.GenerateSection(
		[]report.Column{
			{Header: "Scans (30 days)", Key: "scans"}, {Header: "Failed Scans (30 days)", Key: "failed"},
			{Header: "Projects with Failed Scans", Key: "projects_failed"},
		},
		[]map[string]any{{"scans": scans30d, "failed": failed30d, "projects_failed": projectsFailed}},
		report.WithTitle("Scan Activity", 3),
	)
}
