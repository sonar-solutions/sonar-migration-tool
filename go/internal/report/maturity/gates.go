// Copyright (C) SonarSource Sàrl
// For more information, see https://sonarsource.com/legal/
// mailto:info AT sonarsource DOT com

package maturity

import (
	"github.com/sonar-solutions/sonar-migration-tool/internal/report"
	"github.com/sonar-solutions/sonar-migration-tool/internal/report/common"
	"github.com/sonar-solutions/sonar-migration-tool/internal/structure"
)

// GenerateGateMaturityMarkdown generates quality gate summary and detail sections.
func GenerateGateMaturityMarkdown(dir string, mapping structure.ExtractMapping, idMap common.ServerIDMapping, projects common.Projects) (string, string) {
	gates := common.ProcessQualityGates(dir, mapping, idMap, projects)

	totalGates, activeGates, caycGates := 0, 0, 0
	var detailRows []map[string]any

	for _, serverGates := range gates {
		totalGates += len(serverGates)
		for _, gate := range serverGates {
			count, _ := gate["project_count"].(int)
			isDefault, _ := gate["is_default"].(bool)
			isCayc, _ := gate["is_cayc"].(bool)
			if count > 0 || isDefault {
				activeGates++
				collectGateDetail(&detailRows, gate, isCayc)
			}
			if isCayc {
				caycGates++
			}
		}
	}

	summary := report.GenerateSection(
		[]report.Column{{Header: "Gates", Key: "gates"}, {Header: "Active Gates", Key: "active"}, {Header: "CAYC Compliant", Key: "cayc"}},
		[]map[string]any{{"gates": totalGates, "active": activeGates, "cayc": caycGates}},
		report.WithTitle("Quality Gates", 3),
	)

	details := report.GenerateSection(
		[]report.Column{
			{Header: "Server ID", Key: "server_id"}, {Header: "Gate Name", Key: "name"}, {Header: "# Projects", Key: "project_count"},
			{Header: "CAYC", Key: "is_cayc"}, {Header: "New Violations", Key: "new_violations"},
			{Header: "Hotspots Reviewed", Key: "new_security_hotspots_reviewed"},
			{Header: "Coverage", Key: "new_coverage"}, {Header: "Duplicated Lines", Key: "new_duplicated_lines_density"},
			{Header: "Other", Key: "other"},
		},
		detailRows,
		report.WithTitle("Active Gates", 3),
		report.WithSortBy("project_count", true),
	)
	return summary, details
}

func collectGateDetail(rows *[]map[string]any, gate map[string]any, isCayc bool) {
	cayc := "No"
	if isCayc {
		cayc = "Yes"
	}
	*rows = append(*rows, map[string]any{
		"server_id":                        gate["server_id"],
		"name":                             gate["name"],
		"project_count":                    gate["project_count"],
		"is_cayc":                          cayc,
		"new_violations":                   gate["new_violations"],
		"new_security_hotspots_reviewed":   gate["new_security_hotspots_reviewed"],
		"new_coverage":                     gate["new_coverage"],
		"new_duplicated_lines_density":     gate["new_duplicated_lines_density"],
		"other":                            gate["other"],
	})
}
