package common

import (
	"github.com/sonar-solutions/sonar-migration-tool/internal/report"
	"github.com/sonar-solutions/sonar-migration-tool/internal/structure"
)

// ProcessApplications extracts application details from getApplicationDetails data.
func ProcessApplications(dir string, mapping structure.ExtractMapping, idMap ServerIDMapping) []map[string]any {
	var apps []map[string]any
	for _, item := range readData(dir, mapping, "getApplicationDetails") {
		sid := serverID(idMap, item.ServerURL)
		data := item.Data
		// Unwrap 'application' wrapper if present.
		if wrapped, ok := data["application"].(map[string]any); ok {
			data = wrapped
		}
		projects := report.ExtractPathValue(data, "$.projects", []any{})
		projectList, _ := projects.([]any)
		apps = append(apps, map[string]any{
			"server_id":     sid,
			"name":          report.ExtractString(data, "$.name"),
			"project_count": len(projectList),
		})
	}
	return apps
}

// GenerateApplicationMarkdown generates active and inactive application sections.
func GenerateApplicationMarkdown(dir string, mapping structure.ExtractMapping, idMap ServerIDMapping) (string, string) {
	apps := ProcessApplications(dir, mapping, idMap)
	columns := []report.Column{
		{"Server ID", "server_id"}, {"Application Name", "name"}, {"# Projects", "project_count"},
	}
	active := report.GenerateSection(columns, apps,
		report.WithTitle("Active Applications", 3),
		report.WithSortBy("project_count", true),
		report.WithFilter(func(r map[string]any) bool {
			n, _ := r["project_count"].(int)
			return n > 0
		}),
	)
	inactive := report.GenerateSection(
		[]report.Column{{"Server ID", "server_id"}, {"Application Name", "name"}},
		apps,
		report.WithTitle("Inactive Applications", 3),
		report.WithFilter(func(r map[string]any) bool {
			n, _ := r["project_count"].(int)
			return n == 0
		}),
	)
	return active, inactive
}
