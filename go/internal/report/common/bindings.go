package common

import (
	"github.com/sonar-solutions/sonar-migration-tool/internal/report"
	"github.com/sonar-solutions/sonar-migration-tool/internal/structure"
)

// GenerateDevOpsMarkdown generates the DevOps Integrations section.
func GenerateDevOpsMarkdown(dir string, mapping structure.ExtractMapping, idMap ServerIDMapping) (string, map[string]map[string]bool) {
	projectBindings := processProjectBindings(dir, mapping, idMap)
	devopsBindings := processDevOpsBindings(dir, mapping, idMap)
	branches := ProcessProjectBranches(dir, mapping, idMap)
	pullRequests := ProcessProjectPullRequests(dir, mapping, idMap)

	var rows []map[string]any
	for sid, bindings := range devopsBindings {
		for _, binding := range bindings {
			bindingKey, _ := binding["key"].(string)
			projectData := getBindingProjects(projectBindings, sid, bindingKey)
			hasMB := hasOverlap(projectData, branches[sid])
			hasPR := hasOverlap(projectData, pullRequests[sid])
			rows = append(rows, map[string]any{
				"server_id":              sid,
				"binding":               bindingKey,
				"type":                  binding["alm"],
				"url":                   binding["url"],
				"projects":              len(projectData),
				"multi_branch_projects": boolToYesNo(hasMB),
				"pr_projects":           boolToYesNo(hasPR),
			})
		}
	}

	md := report.GenerateSection(
		[]report.Column{
			{"Server ID", "server_id"}, {"DevOps Platform Binding", "binding"}, {"Type", "type"},
			{"URL", "url"}, {"# Projects", "projects"},
			{"Multi-branch Projects?", "multi_branch_projects"}, {"PR Projects?", "pr_projects"},
		},
		rows,
		report.WithTitle("DevOps Integrations", 2),
		report.WithSortBy("projects", true),
		report.WithFilter(func(r map[string]any) bool {
			n, _ := r["projects"].(int)
			return n > 0
		}),
	)
	return md, pullRequests
}

func processProjectBindings(dir string, mapping structure.ExtractMapping, idMap ServerIDMapping) map[string]map[string]map[string]bool {
	bindings := make(map[string]map[string]map[string]bool)
	for _, item := range readData(dir, mapping, "getProjectBindings") {
		sid := serverID(idMap, item.ServerURL)
		bindingKey := report.ExtractString(item.Data, "$.key")
		projectKey := report.ExtractString(item.Data, "$.projectKey")
		if bindings[sid] == nil {
			bindings[sid] = make(map[string]map[string]bool)
		}
		if bindings[sid][bindingKey] == nil {
			bindings[sid][bindingKey] = make(map[string]bool)
		}
		bindings[sid][bindingKey][projectKey] = true
	}
	return bindings
}

func processDevOpsBindings(dir string, mapping structure.ExtractMapping, idMap ServerIDMapping) map[string][]map[string]any {
	bindings := make(map[string][]map[string]any)
	for _, item := range readData(dir, mapping, "getBindings") {
		sid := serverID(idMap, item.ServerURL)
		bindings[sid] = append(bindings[sid], map[string]any{
			"key": report.ExtractString(item.Data, "$.key"),
			"alm": report.ExtractString(item.Data, "$.alm"),
			"url": report.ExtractString(item.Data, "$.url"),
		})
	}
	return bindings
}

func getBindingProjects(bindings map[string]map[string]map[string]bool, sid, key string) map[string]bool {
	if bindings[sid] != nil && bindings[sid][key] != nil {
		return bindings[sid][key]
	}
	return nil
}

func hasOverlap(a, b map[string]bool) bool {
	for k := range a {
		if b[k] {
			return true
		}
	}
	return false
}

func boolToYesNo(v bool) string {
	if v {
		return "Yes"
	}
	return "No"
}
