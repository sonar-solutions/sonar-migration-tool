package common

import (
	"strings"

	"github.com/sonar-solutions/sonar-migration-tool/internal/report"
	"github.com/sonar-solutions/sonar-migration-tool/internal/structure"
)

// ProcessServerDetails extracts server metadata from getServerInfo.
func ProcessServerDetails(dir string, mapping structure.ExtractMapping) []map[string]any {
	var details []map[string]any
	for _, item := range readData(dir, mapping, "getServerInfo") {
		version := report.ExtractString(item.Data, "System.Version")
		if version == "" {
			version = report.ExtractString(item.Data, "Application Nodes.0.System.Version")
		}
		details = append(details, map[string]any{
			"url":            item.ServerURL,
			"server_id":      report.ExtractString(item.Data, "System.Server ID"),
			"version":        version,
			"edition":        report.ExtractString(item.Data, "System.Edition"),
			"lines_of_code":  report.ExtractInt(item.Data, "System.Lines of Code", 0),
		})
	}
	return details
}

func processSASTConfig(dir string, mapping structure.ExtractMapping, idMap ServerIDMapping) map[string]bool {
	configs := make(map[string]bool)
	for _, item := range readData(dir, mapping, "getProjectSettings") {
		sid := serverID(idMap, item.ServerURL)
		key := report.ExtractString(item.Data, "$.key")
		if strings.Contains(strings.ToLower(key), "security.conf") {
			configs[sid] = true
		}
	}
	return configs
}

func processUserTotals(dir string, mapping structure.ExtractMapping, idMap ServerIDMapping) map[string]int {
	counts := make(map[string]int)
	for _, item := range readData(dir, mapping, "getUsers") {
		sid := serverID(idMap, item.ServerURL)
		counts[sid]++
	}
	return counts
}

// GenerateServerMarkdown generates the Server Details section and builds the ID mapping.
// Returns: (markdown, serverURL→serverID mapping, projects).
func GenerateServerMarkdown(dir string, mapping structure.ExtractMapping) (string, ServerIDMapping, Projects) {
	details := ProcessServerDetails(dir, mapping)
	idMap := make(ServerIDMapping, len(details))
	for _, s := range details {
		idMap[s["url"].(string)] = s["server_id"].(string)
	}

	projects := ProcessProjectDetails(dir, mapping, idMap)
	userTotals := processUserTotals(dir, mapping, idMap)
	sastConfigs := processSASTConfig(dir, mapping, idMap)

	for _, s := range details {
		sid := s["server_id"].(string)
		s["users"] = userTotals[sid]
		s["sast_configured"] = sastConfigs[sid]
		if projects[sid] != nil {
			s["project_count"] = len(projects[sid])
		} else {
			s["project_count"] = 0
		}
	}

	md := report.GenerateSection(
		[]report.Column{
			{"Server ID", "server_id"}, {"Url", "url"}, {"Version", "version"},
			{"Projects", "project_count"}, {"Lines of Code", "lines_of_code"},
			{"Users", "users"}, {"SAST Configured", "sast_configured"},
		},
		details,
		report.WithTitle("Server Details", 2),
	)
	return md, idMap, projects
}
