package common

import (
	"regexp"
	"strings"

	"github.com/sonar-solutions/sonar-migration-tool/internal/report"
	"github.com/sonar-solutions/sonar-migration-tool/internal/structure"
)

var qualifierMapping = map[string]string{
	"APP": "Applications",
	"TRK": "Projects",
	"VW":  "Portfolios",
}

// ProcessPermissionTemplates extracts permission templates and default assignments.
func ProcessPermissionTemplates(dir string, mapping structure.ExtractMapping, idMap ServerIDMapping) []map[string]any {
	defaults := processDefaultTemplates(dir, mapping, idMap)
	var templates []map[string]any
	for _, item := range readData(dir, mapping, "getTemplates") {
		sid := serverID(idMap, item.ServerURL)
		templateID := report.ExtractString(item.Data, "$.id")
		templates = append(templates, map[string]any{
			"server_id":     sid,
			"name":          report.ExtractString(item.Data, "$.name"),
			"description":   report.ExtractString(item.Data, "$.description"),
			"pattern":       report.ExtractString(item.Data, "$.projectKeyPattern"),
			"defaults":      joinDefaults(defaults[sid][templateID]),
			"projects":      []any{},
			"project_count": 0,
		})
	}
	return templates
}

func processDefaultTemplates(dir string, mapping structure.ExtractMapping, idMap ServerIDMapping) map[string]map[string]map[string]bool {
	defaults := make(map[string]map[string]map[string]bool)
	for _, item := range readData(dir, mapping, "getDefaultTemplates") {
		sid := serverID(idMap, item.ServerURL)
		templateID := report.ExtractString(item.Data, "$.templateId")
		qualifier := report.ExtractString(item.Data, "$.qualifier")
		if templateID == "" || qualifier == "" {
			continue
		}
		if defaults[sid] == nil {
			defaults[sid] = make(map[string]map[string]bool)
		}
		if defaults[sid][templateID] == nil {
			defaults[sid][templateID] = make(map[string]bool)
		}
		if mapped, ok := qualifierMapping[qualifier]; ok {
			defaults[sid][templateID][mapped] = true
		}
	}
	return defaults
}

func joinDefaults(defaultSet map[string]bool) string {
	if len(defaultSet) == 0 {
		return ""
	}
	var parts []string
	for d := range defaultSet {
		parts = append(parts, d)
	}
	return strings.Join(parts, ", ")
}

// FilterProjects applies pattern matching to assign projects to templates.
func FilterProjects(projects Projects, templates []map[string]any) []map[string]any {
	remaining := buildRemainingProjects(projects)
	matchPatternTemplates(projects, templates, remaining)
	assignDefaultTemplates(templates, remaining)
	return templates
}

func buildRemainingProjects(projects Projects) map[string]map[string]bool {
	remaining := make(map[string]map[string]bool)
	for sid, serverProjects := range projects {
		remaining[sid] = make(map[string]bool)
		for _, p := range serverProjects {
			if key, ok := p["key"].(string); ok {
				remaining[sid][key] = true
			}
		}
	}
	return remaining
}

func matchPatternTemplates(projects Projects, templates []map[string]any, remaining map[string]map[string]bool) {
	for _, tmpl := range templates {
		pattern, _ := tmpl["pattern"].(string)
		sid, _ := tmpl["server_id"].(string)
		if pattern == "" || sid == "" {
			continue
		}
		re, err := regexp.Compile(pattern)
		if err != nil {
			continue
		}
		var matched []string
		for _, p := range projects[sid] {
			key, _ := p["key"].(string)
			if key != "" && re.MatchString(key) {
				matched = append(matched, key)
				delete(remaining[sid], key)
			}
		}
		tmpl["project_count"] = len(matched)
	}
}

func assignDefaultTemplates(templates []map[string]any, remaining map[string]map[string]bool) {
	for _, tmpl := range templates {
		defaults, _ := tmpl["defaults"].(string)
		sid, _ := tmpl["server_id"].(string)
		if strings.Contains(defaults, "Projects") && remaining[sid] != nil {
			tmpl["project_count"] = len(remaining[sid])
		}
	}
}

// GeneratePermissionTemplateMarkdown generates the Permission Templates section.
func GeneratePermissionTemplateMarkdown(dir string, mapping structure.ExtractMapping, idMap ServerIDMapping, projects Projects, onlyActive bool) (string, []map[string]any) {
	templates := ProcessPermissionTemplates(dir, mapping, idMap)
	templates = FilterProjects(projects, templates)

	md := report.GenerateSection(
		[]report.Column{
			{"Server ID", "server_id"}, {"Template Name", "name"}, {"Description", "description"},
			{"Project key pattern", "pattern"}, {"Default For", "defaults"}, {"Projects", "project_count"},
		},
		templates,
		report.WithTitle("Permission Templates", 3),
		report.WithFilter(func(r map[string]any) bool {
			if !onlyActive {
				return true
			}
			count, _ := r["project_count"].(int)
			return count > 0
		}),
	)
	return md, templates
}
