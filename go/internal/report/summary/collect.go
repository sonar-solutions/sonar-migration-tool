package summary

import (
	"encoding/json"
	"time"

	"github.com/sonar-solutions/sonar-migration-tool/internal/analysis"
	"github.com/sonar-solutions/sonar-migration-tool/internal/common"
)

// CollectSummary reads task JSONL outputs and the analysis report to build
// a MigrationSummary for PDF rendering.
func CollectSummary(runDir string) (*MigrationSummary, error) {
	store := common.NewDataStore(runDir)
	failuresByType, err := collectFailures(runDir)
	if err != nil {
		return nil, err
	}

	scanHistoryMap := collectScanHistory(store)

	var sections []Section
	for _, def := range sectionDefs {
		section := collectSection(store, def, failuresByType)
		if def.Name == "Projects" {
			attachScanHistory(section.Succeeded, scanHistoryMap)
		}
		sections = append(sections, section)
	}

	runID := extractRunID(runDir)
	return &MigrationSummary{
		RunID:       runID,
		GeneratedAt: time.Now(),
		Sections:    sections,
	}, nil
}

// collectFailures parses the requests.log and groups failures by entity type.
func collectFailures(runDir string) (map[string][]analysis.ReportRow, error) {
	rows, err := analysis.ParseRequestsLog(runDir)
	if err != nil {
		return nil, err
	}
	result := make(map[string][]analysis.ReportRow)
	for _, row := range rows {
		if row.Outcome == "failure" {
			result[row.EntityType] = append(result[row.EntityType], row)
		}
	}
	return result, nil
}

// collectSection builds a Section from task JSONL data and analysis failures.
func collectSection(store *common.DataStore, def sectionDef, failuresByType map[string][]analysis.ReportRow) Section {
	succeeded := collectSucceeded(store, def)
	skipped := collectSkipped(store, def)
	failed := collectFailed(failuresByType, def)

	return Section{
		Name:      def.Name,
		Succeeded: succeeded,
		Failed:    failed,
		Skipped:   skipped,
	}
}

// collectSucceeded reads items from the output (create*) task JSONL.
func collectSucceeded(store *common.DataStore, def sectionDef) []EntityItem {
	items, err := store.ReadAll(def.OutputTask)
	if err != nil {
		return nil
	}
	var result []EntityItem
	for _, item := range items {
		result = append(result, EntityItem{
			Name:         jsonStr(item, def.NameField),
			Organization: jsonStr(item, "sonarcloud_org_key"),
			Detail:       jsonStr(item, def.DetailField),
		})
	}
	return result
}

// collectSkipped reads the input (generate*Mappings) task and finds items
// where sonarcloud_org_key is empty or "SKIPPED".
func collectSkipped(store *common.DataStore, def sectionDef) []EntityItem {
	items, err := store.ReadAll(def.InputTask)
	if err != nil {
		return nil
	}
	var result []EntityItem
	for _, item := range items {
		orgKey := jsonStr(item, "sonarcloud_org_key")
		if orgKey == "" || orgKey == skippedOrgSentinel {
			result = append(result, EntityItem{
				Name:         jsonStr(item, def.NameField),
				Organization: jsonStr(item, "sonarqube_org_key"),
				Detail:       "Organization skipped",
			})
		}
	}
	return result
}

// collectFailed maps analysis report failure rows to EntityItems.
func collectFailed(failuresByType map[string][]analysis.ReportRow, def sectionDef) []EntityItem {
	rows := failuresByType[def.AnalysisEntity]
	var result []EntityItem
	for _, row := range rows {
		result = append(result, EntityItem{
			Name:         row.EntityName,
			Organization: row.Organization,
			ErrorMessage: row.ErrorMessage,
		})
	}
	return result
}

// collectScanHistory reads importScanHistory JSONL and returns a map of
// cloud_project_key -> status ("success", "failed", "skipped").
func collectScanHistory(store *common.DataStore) map[string]string {
	items, err := store.ReadAll("importScanHistory")
	if err != nil || len(items) == 0 {
		return nil
	}
	result := make(map[string]string)
	for _, item := range items {
		key := jsonStr(item, "cloud_project_key")
		status := jsonStr(item, "status")
		if key != "" {
			result[key] = status
		}
	}
	return result
}

// attachScanHistory adds scan history status to project EntityItems.
func attachScanHistory(projects []EntityItem, scanMap map[string]string) {
	if scanMap == nil {
		return
	}
	for i := range projects {
		cloudKey := projects[i].Detail
		if status, ok := scanMap[cloudKey]; ok {
			projects[i].Detail = cloudKey + "|scan:" + status
		}
	}
}

// extractRunID extracts the run ID from a directory path (last path component).
func extractRunID(runDir string) string {
	for i := len(runDir) - 1; i >= 0; i-- {
		if runDir[i] == '/' || runDir[i] == '\\' {
			return runDir[i+1:]
		}
	}
	return runDir
}

// jsonStr extracts a string field from a json.RawMessage.
func jsonStr(raw json.RawMessage, key string) string {
	return common.ExtractField(raw, key)
}
