package maturity

import (
	"fmt"

	"github.com/sonar-solutions/sonar-migration-tool/internal/report"
	"github.com/sonar-solutions/sonar-migration-tool/internal/report/common"
)

// Tier thresholds for project size classification.
var tierDefs = []struct {
	name   string
	minLOC int
}{
	{"xl", 500000}, {"l", 100000}, {"m", 10000}, {"s", 1000}, {"xs", 0},
}

// GenerateUsageMarkdown generates the project usage summary by LOC tier.
func GenerateUsageMarkdown(projects common.Projects, scans common.ProjectScans) string {
	activeProjects := collectActiveScans(projects, scans)
	rows := buildUsageRows(projects, activeProjects)

	return report.GenerateSection(
		[]report.Column{
			{"Size", "size"}, {"Projects", "projects"}, {"Active Projects (30d)", "active_projects"},
			{"Scanned Code (LOC)", "scanned_code"}, {"Scans/Day", "scans_per_day"},
		},
		rows,
		report.WithTitle("Usage", 3),
	)
}

// collectActiveScans maps tiers to active projects with LOC data.
func collectActiveScans(projects common.Projects, scans common.ProjectScans) map[string]map[string]int {
	active := make(map[string]map[string]int)
	for sid, ciTools := range scans {
		for _, ciProjects := range ciTools {
			for projectKey, scan := range ciProjects {
				addActiveProject(active, projects, sid, projectKey, scan)
			}
		}
	}
	return active
}

func addActiveProject(active map[string]map[string]int, projects common.Projects, sid, projectKey string, scan map[string]any) {
	if projects[sid] == nil || projects[sid][projectKey] == nil {
		return
	}
	s30, _ := scan["scan_count_30_days"].(int)
	if s30 <= 0 {
		return
	}
	project := projects[sid][projectKey]
	tier, _ := project["tier"].(string)
	loc, _ := project["loc"].(int)
	if active[tier] == nil {
		active[tier] = make(map[string]int)
	}
	active[tier][projectKey] = loc
}

func buildUsageRows(projects common.Projects, activeProjects map[string]map[string]int) []map[string]any {
	tierCounts := countProjectsByTier(projects)
	tierScans := countScansByTier(projects)

	var rows []map[string]any
	for _, td := range tierDefs {
		scannedCode := 0
		for _, loc := range activeProjects[td.name] {
			scannedCode += loc
		}
		scansPerDay := float64(0)
		if tierScans[td.name] > 0 {
			scansPerDay = float64(tierScans[td.name]) / 30
		}
		rows = append(rows, map[string]any{
			"size":            fmt.Sprintf("%s > %s LOC", td.name, report.FormatValue(td.minLOC)),
			"projects":        tierCounts[td.name],
			"active_projects": len(activeProjects[td.name]),
			"scanned_code":    scannedCode,
			"scans_per_day":   scansPerDay,
		})
	}
	return rows
}

func countProjectsByTier(projects common.Projects) map[string]int {
	counts := make(map[string]int)
	for _, serverProjects := range projects {
		for _, project := range serverProjects {
			tier, _ := project["tier"].(string)
			counts[tier]++
		}
	}
	return counts
}

func countScansByTier(projects common.Projects) map[string]int {
	scans := make(map[string]int)
	for _, serverProjects := range projects {
		for _, project := range serverProjects {
			tier, _ := project["tier"].(string)
			s30, _ := project["30_day_scans"].(int)
			scans[tier] += s30
		}
	}
	return scans
}
