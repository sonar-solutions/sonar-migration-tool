package common

import (
	"time"

	"github.com/sonar-solutions/sonar-migration-tool/internal/report"
	"github.com/sonar-solutions/sonar-migration-tool/internal/structure"
)

// ProjectScans is serverID → ciTool → projectKey → scan data.
type ProjectScans = map[string]map[string]map[string]map[string]any

// ProcessScanDetails extracts CI pipeline scan data from getProjectAnalyses.
func ProcessScanDetails(dir string, mapping structure.ExtractMapping, idMap ServerIDMapping) ProjectScans {
	scans := make(ProjectScans)
	for _, item := range readData(dir, mapping, "getProjectAnalyses") {
		sid := serverID(idMap, item.ServerURL)
		scanDate := report.ExtractString(item.Data, "$.date")
		detectedCI := report.ExtractString(item.Data, "$.detectedCI")
		projectKey := report.ExtractString(item.Data, "$.projectKey")
		if scanDate == "" || detectedCI == "" || projectKey == "" {
			continue
		}
		date, ok := parseSQDate(scanDate)
		if !ok {
			continue
		}
		ensureScanEntry(scans, sid, detectedCI, projectKey)
		scan := scans[sid][detectedCI][projectKey]
		scan["project_key"] = projectKey
		scan["server_id"] = sid
		failed := checkFailedEvents(item.Data, scan)
		updateScanEntry(scan, date, failed)
	}
	return scans
}

func ensureScanEntry(scans ProjectScans, sid, ci, projectKey string) {
	if scans[sid] == nil {
		scans[sid] = make(map[string]map[string]map[string]any)
	}
	if scans[sid][ci] == nil {
		scans[sid][ci] = make(map[string]map[string]any)
	}
	if scans[sid][ci][projectKey] == nil {
		scans[sid][ci][projectKey] = map[string]any{
			"total_scans": 0, "first_scan": nil, "last_scan": nil,
			"scan_count_30_days": 0, "failed_scans": 0, "failed_scans_30_days": 0,
		}
	}
}

func checkFailedEvents(data, scan map[string]any) bool {
	events := report.ExtractPathValue(data, "events", nil)
	arr, ok := events.([]any)
	if !ok {
		return false
	}
	for _, e := range arr {
		em, ok := e.(map[string]any)
		if !ok {
			continue
		}
		category := report.ExtractString(em, "category")
		name := report.ExtractString(em, "name")
		if category == "QUALITY_GATE" && name == "FAILED" {
			scan["failed_scans"] = scan["failed_scans"].(int) + 1
			return true
		}
	}
	return false
}

func updateScanEntry(scan map[string]any, date time.Time, failed bool) {
	scan["total_scans"] = scan["total_scans"].(int) + 1
	if isRecent(date) {
		scan["scan_count_30_days"] = scan["scan_count_30_days"].(int) + 1
		if failed {
			scan["failed_scans_30_days"] = scan["failed_scans_30_days"].(int) + 1
		}
	}
	updateScanDates(scan, date)
}

func updateScanDates(scan map[string]any, date time.Time) {
	if first, ok := scan["first_scan"].(time.Time); !ok || date.Before(first) {
		scan["first_scan"] = date
	}
	if last, ok := scan["last_scan"].(time.Time); !ok || date.After(last) {
		scan["last_scan"] = date
	}
}

// GeneratePipelineMarkdown generates the CI Environment Overview and Project Scan Details sections.
func GeneratePipelineMarkdown(dir string, mapping structure.ExtractMapping, idMap ServerIDMapping) (string, string, ProjectScans) {
	scans := ProcessScanDetails(dir, mapping, idMap)
	overview := buildPipelineOverview(scans)
	details := buildScanDetails(scans)
	return overview, details, scans
}

func buildPipelineOverview(scans ProjectScans) string {
	var rows []map[string]any
	for sid, pipelines := range scans {
		for ciTool, projects := range pipelines {
			row := aggregatePipelineStats(sid, ciTool, projects)
			rows = append(rows, row)
		}
	}
	return report.GenerateSection(
		[]report.Column{
			{"Server ID", "server_id"}, {"CI Tool", "name"}, {"# Projects", "project_count"},
			{"First Run", "first_scan"}, {"Most Recent Run", "last_scan"}, {"Total Scans", "total_scans"},
		},
		rows,
		report.WithTitle("CI Environment Overview", 2),
		report.WithSortBy("total_scans", true),
	)
}

func aggregatePipelineStats(sid, ciTool string, projects map[string]map[string]any) map[string]any {
	var firstScan, lastScan time.Time
	totalScans := 0
	for _, scan := range projects {
		totalScans += scan["total_scans"].(int)
		if f, ok := scan["first_scan"].(time.Time); ok {
			if firstScan.IsZero() || f.Before(firstScan) {
				firstScan = f
			}
		}
		if l, ok := scan["last_scan"].(time.Time); ok {
			if lastScan.IsZero() || l.After(lastScan) {
				lastScan = l
			}
		}
	}
	return map[string]any{
		"server_id": sid, "name": ciTool, "project_count": len(projects),
		"first_scan": firstScan, "last_scan": lastScan, "total_scans": totalScans,
	}
}

func buildScanDetails(scans ProjectScans) string {
	var rows []map[string]any
	for sid, pipelines := range scans {
		for ciTool, projects := range pipelines {
			for projectKey, scan := range projects {
				rows = append(rows, map[string]any{
					"server_id":    sid,
					"project_name": projectKey,
					"ci_tool":      ciTool,
					"total_scans":  scan["total_scans"],
					"first_scan":   scan["first_scan"],
					"last_scan":    scan["last_scan"],
				})
			}
		}
	}
	return report.GenerateSection(
		[]report.Column{
			{"Server ID", "server_id"}, {"Project Name", "project_name"}, {"CI Tool", "ci_tool"},
			{"# of Scans", "total_scans"}, {"First Scan", "first_scan"}, {"Most Recent Scan", "last_scan"},
		},
		rows,
		report.WithTitle("Project Scan Details", 3),
		report.WithSortBy("total_scans", true),
	)
}
