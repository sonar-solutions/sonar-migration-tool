package common

import (
	"github.com/sonar-solutions/sonar-migration-tool/internal/report"
	"github.com/sonar-solutions/sonar-migration-tool/internal/structure"
)

// ProcessQualityGates extracts quality gate details and maps them to projects.
func ProcessQualityGates(dir string, mapping structure.ExtractMapping, idMap ServerIDMapping, projects Projects) map[string]map[string]map[string]any {
	gates := make(map[string]map[string]map[string]any)

	for _, item := range readData(dir, mapping, "getGates") {
		sid := serverID(idMap, item.ServerURL)
		name := report.ExtractString(item.Data, "$.name")
		if name == "" {
			continue
		}
		ensureGateEntry(gates, sid, name)
		isBuiltIn := report.ExtractBool(item.Data, "$.isBuiltIn")
		caycStatus := report.ExtractString(item.Data, "$.caycStatus")
		gates[sid][name]["server_id"] = sid
		gates[sid][name]["name"] = name
		gates[sid][name]["is_built_in"] = isBuiltIn
		gates[sid][name]["is_default"] = report.ExtractBool(item.Data, "$.isDefault")
		gates[sid][name]["is_cayc"] = caycStatus == "compliant" || caycStatus == "over-compliant"
	}

	for sid, serverProjects := range projects {
		for _, project := range serverProjects {
			gateName, _ := project["quality_gate"].(string)
			if gateName == "" || gates[sid] == nil || gates[sid][gateName] == nil {
				continue
			}
			gates[sid][gateName]["project_count"] = gates[sid][gateName]["project_count"].(int) + 1
		}
	}
	return gates
}

func ensureGateEntry(gates map[string]map[string]map[string]any, sid, name string) {
	if gates[sid] == nil {
		gates[sid] = make(map[string]map[string]any)
	}
	if gates[sid][name] == nil {
		gates[sid][name] = map[string]any{
			"project_count": 0,
			"conditions":    []any{},
		}
	}
}

// GenerateGateMarkdown generates active and unused quality gate sections.
func GenerateGateMarkdown(dir string, mapping structure.ExtractMapping, idMap ServerIDMapping, projects Projects) (string, string) {
	gates := ProcessQualityGates(dir, mapping, idMap, projects)
	var rows []map[string]any
	for _, serverGates := range gates {
		for _, gate := range serverGates {
			rows = append(rows, gate)
		}
	}

	active := report.GenerateSection(
		[]report.Column{
			{"Server ID", "server_id"}, {"Quality Gate Name", "name"},
			{"# of Projects using", "project_count"}, {"Is Default", "is_default"},
		},
		rows,
		report.WithTitle("Active Custom Quality Gates", 3),
		report.WithSortBy("project_count", true),
		report.WithFilter(func(r map[string]any) bool {
			count, _ := r["project_count"].(int)
			isDefault, _ := r["is_default"].(bool)
			return count > 0 || isDefault
		}),
	)

	unused := report.GenerateSection(
		[]report.Column{{"Server ID", "server_id"}, {"Quality Gate Name", "name"}},
		rows,
		report.WithTitle("Unused Custom Quality Gates", 3),
		report.WithFilter(func(r map[string]any) bool {
			count, _ := r["project_count"].(int)
			isDefault, _ := r["is_default"].(bool)
			return count == 0 && !isDefault
		}),
	)
	return active, unused
}
