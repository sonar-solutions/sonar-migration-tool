package common

import (
	"github.com/sonar-solutions/sonar-migration-tool/internal/report"
	"github.com/sonar-solutions/sonar-migration-tool/internal/structure"
)

// ProcessPortfolios extracts portfolio details and project counts.
func ProcessPortfolios(dir string, mapping structure.ExtractMapping, idMap ServerIDMapping) []map[string]any {
	// Intermediate storage: serverID → portfolioKey → portfolio data.
	portfolios := make(map[string]map[string]map[string]any)

	for _, item := range readData(dir, mapping, "getPortfolioDetails") {
		sid := serverID(idMap, item.ServerURL)
		key := report.ExtractString(item.Data, "$.key")
		if key == "" {
			continue
		}
		ensurePortfolio(portfolios, sid, key)
		children := report.ExtractPathValue(item.Data, "$.subViews", nil)
		portfolios[sid][key]["name"] = report.ExtractString(item.Data, "$.name")
		portfolios[sid][key]["server_id"] = sid
		portfolios[sid][key]["selection"] = extractSelectionModes(item.Data)
		portfolios[sid][key]["children"] = children != nil
	}

	for _, item := range readData(dir, mapping, "getPortfolioProjects") {
		sid := serverID(idMap, item.ServerURL)
		portfolioKey := report.ExtractString(item.Data, "$.portfolioKey")
		projectKey := report.ExtractString(item.Data, "$.refKey")
		if portfolioKey == "" || projectKey == "" {
			continue
		}
		if portfolios[sid] == nil || portfolios[sid][portfolioKey] == nil {
			continue
		}
		p := portfolios[sid][portfolioKey]
		projects := toStringSet(p["projects"])
		projects[projectKey] = true
		p["projects"] = projects
		p["project_count"] = len(projects)
	}

	return flattenPortfolios(portfolios)
}

func ensurePortfolio(m map[string]map[string]map[string]any, sid, key string) {
	if m[sid] == nil {
		m[sid] = make(map[string]map[string]any)
	}
	if m[sid][key] == nil {
		m[sid][key] = map[string]any{
			"projects":      make(map[string]bool),
			"project_count": 0,
		}
	}
}

func toStringSet(v any) map[string]bool {
	if m, ok := v.(map[string]bool); ok {
		return m
	}
	return make(map[string]bool)
}

func flattenPortfolios(m map[string]map[string]map[string]any) []map[string]any {
	var result []map[string]any
	for _, serverPortfolios := range m {
		for _, p := range serverPortfolios {
			result = append(result, p)
		}
	}
	return result
}

// extractSelectionModes recursively collects selection modes from a portfolio.
func extractSelectionModes(portfolio map[string]any) []string {
	modes := make(map[string]bool)
	collectSelectionModes(portfolio, modes)
	var result []string
	for mode := range modes {
		result = append(result, mode)
	}
	return result
}

func collectSelectionModes(portfolio map[string]any, modes map[string]bool) {
	mode := report.ExtractString(portfolio, "$.selectionMode")
	if mode != "" {
		modes[mode] = true
	}
	subViews := report.ExtractPathValue(portfolio, "$.subViews", nil)
	if arr, ok := subViews.([]any); ok {
		for _, child := range arr {
			if childMap, ok := child.(map[string]any); ok {
				collectSelectionModes(childMap, modes)
			}
		}
	}
}

// GeneratePortfolioMarkdown generates active and inactive portfolio sections.
func GeneratePortfolioMarkdown(dir string, mapping structure.ExtractMapping, idMap ServerIDMapping) (string, string) {
	portfolios := ProcessPortfolios(dir, mapping, idMap)
	active := report.GenerateSection(
		[]report.Column{
			{"Server ID", "server_id"}, {"Portfolio Name", "name"}, {"Project selection type", "selection"},
			{"Contains Nested Portfolios", "children"}, {"# of Projects", "project_count"},
		},
		portfolios,
		report.WithTitle("Active Portfolios", 3),
		report.WithSortBy("project_count", true),
		report.WithFilter(func(r map[string]any) bool {
			n, _ := r["project_count"].(int)
			return n > 0
		}),
	)
	inactive := report.GenerateSection(
		[]report.Column{
			{"Server ID", "server_id"}, {"Portfolio Name", "name"}, {"Project selection type", "selection"},
		},
		portfolios,
		report.WithTitle("Inactive Portfolios", 3),
		report.WithFilter(func(r map[string]any) bool {
			n, _ := r["project_count"].(int)
			return n == 0
		}),
	)
	return active, inactive
}
