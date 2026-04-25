package maturity

import (
	"strings"

	"github.com/sonar-solutions/sonar-migration-tool/internal/report"
	"github.com/sonar-solutions/sonar-migration-tool/internal/report/common"
	"github.com/sonar-solutions/sonar-migration-tool/internal/structure"
)

var severities = []string{"BLOCKER", "CRITICAL", "MAJOR", "MINOR", "INFO"}

var folderMapping = map[string]string{
	"resolved": "getProjectResolvedIssueTypes",
	"all":      "getProjectIssueTypes",
	"recent":   "getProjectRecentIssueTypes",
}

// GenerateIssueMarkdown generates issue overview and detail sections.
func GenerateIssueMarkdown(dir string, mapping structure.ExtractMapping, idMap common.ServerIDMapping) (string, string, string, string) {
	issues := processIssues(dir, mapping, idMap)
	issueRows, detailRows := aggregateIssues(issues)
	overview := buildOverviewSection(issueRows)
	vulnMD := buildDetailSection("Vulnerabilities", "VULNERABILITY", detailRows)
	bugMD := buildDetailSection("Bugs", "BUG", detailRows)
	smellMD := buildDetailSection("Code Smells", "CODE_SMELL", detailRows)
	return overview, vulnMD, bugMD, smellMD
}

func processIssues(dir string, mapping structure.ExtractMapping, idMap common.ServerIDMapping) map[string]map[string]map[string]map[string]map[string]any {
	issues := make(map[string]map[string]map[string]map[string]map[string]any)
	for folder, key := range folderMapping {
		for _, item := range common.ReadDataParsed(dir, mapping, key) {
			sid := common.ServerIDLookup(idMap, item.ServerURL)
			processIssueEntry(issues, sid, folder, item.Data)
		}
	}
	return issues
}

func processIssueEntry(issues map[string]map[string]map[string]map[string]map[string]any, sid, folder string, data map[string]any) {
	projectKey := report.ExtractString(data, "$.projectKey")
	severity := report.ExtractString(data, "$.severity")
	issueType := report.ExtractString(data, "$.issueType")
	total := report.ExtractInt(data, "$.total", 0)
	if projectKey == "" || severity == "" || issueType == "" {
		return
	}
	ensureIssueEntry(issues, sid, projectKey, issueType, severity)
	entry := issues[sid][projectKey][issueType][severity]
	resolution := report.ExtractString(data, "$.resolution")
	if resolution != "" {
		resolution = strings.ReplaceAll(resolution, "-", "_")
	}

	switch folder {
	case "resolved":
		entry[resolution] = total
		entry["open"] = toIntSafe(entry["open"]) - total
	case "all":
		entry["all"] = total
		entry["open"] = toIntSafe(entry["open"]) + total
	case "recent":
		entry["recent"] = total
	}
}

func ensureIssueEntry(issues map[string]map[string]map[string]map[string]map[string]any, sid, projectKey, issueType, severity string) {
	if issues[sid] == nil {
		issues[sid] = make(map[string]map[string]map[string]map[string]any)
	}
	if issues[sid][projectKey] == nil {
		issues[sid][projectKey] = make(map[string]map[string]map[string]any)
	}
	if issues[sid][projectKey][issueType] == nil {
		issues[sid][projectKey][issueType] = make(map[string]map[string]any)
	}
	if issues[sid][projectKey][issueType][severity] == nil {
		issues[sid][projectKey][issueType][severity] = map[string]any{"open": 0, "all": 0, "recent": 0}
	}
}

func aggregateIssues(issues map[string]map[string]map[string]map[string]map[string]any) (map[string]map[string]any, map[string]map[string]map[string]any) {
	issueRows := makeIssueRows()
	detailRows := makeDetailRows()

	for _, projectIssues := range issues {
		for projectKey, typeIssues := range projectIssues {
			for issueType, severityIssues := range typeIssues {
				for severity, issue := range severityIssues {
					accumulateIssue(issueRows, detailRows, issueType, severity, projectKey, issue)
				}
			}
		}
	}
	return issueRows, detailRows
}

func accumulateIssue(issueRows map[string]map[string]any, detailRows map[string]map[string]map[string]any, issueType, severity, projectKey string, issue map[string]any) {
	openCount := toIntSafe(issue["open"])
	recentCount := toIntSafe(issue["recent"])

	ensureDetailRow(detailRows, issueType, severity)
	detail := detailRows[issueType][severity]
	detail["open"] = toIntSafe(detail["open"]) + openCount
	detail["recent"] = toIntSafe(detail["recent"]) + recentCount
	if openCount > 0 {
		addAffectedProject(detail, projectKey)
	}

	row := issueRows[severity]
	row["open"] = toIntSafe(row["open"]) + openCount
	row["recent"] = toIntSafe(row["recent"]) + recentCount
	if openCount > 0 {
		addAffectedProject(row, projectKey)
	}
	updateIssueTypeCounts(row, issueType, openCount, recentCount)
}

func updateIssueTypeCounts(row map[string]any, issueType string, open, recent int) {
	switch issueType {
	case "VULNERABILITY":
		row["open_vulnerabilities"] = toIntSafe(row["open_vulnerabilities"]) + open
		row["recent_vulnerabilities"] = toIntSafe(row["recent_vulnerabilities"]) + recent
	case "BUG":
		row["open_bugs"] = toIntSafe(row["open_bugs"]) + open
		row["recent_bugs"] = toIntSafe(row["recent_bugs"]) + recent
	case "CODE_SMELL":
		row["open_code_smells"] = toIntSafe(row["open_code_smells"]) + open
		row["recent_code_smells"] = toIntSafe(row["recent_code_smells"]) + recent
	}
}

func addAffectedProject(row map[string]any, projectKey string) {
	projects, _ := row["affected_projects"].(map[string]bool)
	if projects == nil {
		projects = make(map[string]bool)
		row["affected_projects"] = projects
	}
	projects[projectKey] = true
	row["affected_project_count"] = len(projects)
}

func makeIssueRows() map[string]map[string]any {
	rows := make(map[string]map[string]any)
	for _, s := range severities {
		rows[s] = map[string]any{
			"severity": s, "open": 0, "recent": 0, "affected_projects": make(map[string]bool), "affected_project_count": 0,
			"open_vulnerabilities": 0, "recent_vulnerabilities": 0,
			"open_bugs": 0, "recent_bugs": 0,
			"open_code_smells": 0, "recent_code_smells": 0,
		}
	}
	return rows
}

func makeDetailRows() map[string]map[string]map[string]any {
	rows := make(map[string]map[string]map[string]any)
	for _, t := range []string{"VULNERABILITY", "BUG", "CODE_SMELL"} {
		rows[t] = make(map[string]map[string]any)
		for _, s := range severities {
			rows[t][s] = map[string]any{
				"severity": s, "open": 0, "recent": 0,
				"affected_projects": make(map[string]bool), "affected_project_count": 0,
			}
		}
	}
	return rows
}

func ensureDetailRow(rows map[string]map[string]map[string]any, issueType, severity string) {
	if rows[issueType] == nil {
		rows[issueType] = make(map[string]map[string]any)
	}
	if rows[issueType][severity] == nil {
		rows[issueType][severity] = map[string]any{
			"severity": severity, "open": 0, "recent": 0,
			"affected_projects": make(map[string]bool), "affected_project_count": 0,
		}
	}
}

func buildOverviewSection(issueRows map[string]map[string]any) string {
	var rows []map[string]any
	for _, s := range severities {
		rows = append(rows, issueRows[s])
	}
	return report.GenerateSection(
		[]report.Column{
			{"Severity", "severity"}, {"Open Issues", "open"}, {"Affected Projects", "affected_project_count"},
			{"Open Vulnerabilities", "open_vulnerabilities"}, {"Open Bugs", "open_bugs"}, {"Open Code Smells", "open_code_smells"},
		},
		rows,
		report.WithTitle("Issues Overview", 3),
	)
}

func buildDetailSection(title, issueType string, detailRows map[string]map[string]map[string]any) string {
	typeRows := detailRows[issueType]
	var rows []map[string]any
	for _, s := range severities {
		if r, ok := typeRows[s]; ok {
			rows = append(rows, r)
		}
	}
	return report.GenerateSection(
		[]report.Column{
			{"Severity", "severity"}, {"Open", "open"}, {"Recent (30d)", "recent"},
			{"Affected Projects", "affected_project_count"},
		},
		rows,
		report.WithTitle(title, 3),
	)
}

func toIntSafe(v any) int {
	if n, ok := v.(int); ok {
		return n
	}
	return 0
}
