// Package predict produces a migration summary PDF before any migrate
// step has run. It synthesises the JSONL outputs the summary pipeline
// expects (generate*Mappings + create* + addGateConditions.notes) from
// the user-edited mapping CSVs and the extract data, then hands the
// resulting "predictive" run directory to summary.RenderPDF.
//
// Two classes of outcomes from a real migrate run cannot be predicted
// and are excluded from the predictive report (#235):
//   - SQC API errors / rate-limiting (no requests.log → no Failed bucket).
//   - Global settings — discovery of SQC-supported settings is dynamic.
package predict

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/sonar-solutions/sonar-migration-tool/internal/common"
	"github.com/sonar-solutions/sonar-migration-tool/internal/structure"
)

// skippedOrgSentinel mirrors migrate.skippedOrgSentinel: a row whose
// sonarcloud_org_key is empty or "SKIPPED" is excluded from the
// synthesized create-task outputs (same rule the real migrate uses).
const skippedOrgSentinel = "SKIPPED"

func shouldSkipOrg(orgKey string) bool {
	return orgKey == "" || orgKey == skippedOrgSentinel
}

// createTaskDef describes how to synthesise one create* task output
// from its upstream mapping task.
type createTaskDef struct {
	MappingsTask string // generate*Mappings task
	CSVFile      string // CSV at exportDir root that the mappings task converts
	OutputTask   string // create* task whose JSONL we synthesise
	IDField      string // synthetic id field name (cloud_gate_id, cloud_project_key, ...)
	NameField    string // entity name field on the mapping row (default "name")
}

// createTasks mirrors the relevant rows from summary.sectionDefs but in
// the predict-package perspective: each entry describes the upstream
// mapping JSONL and the create-task JSONL to fabricate. Global Settings
// is omitted on purpose (#235).
var createTasks = []createTaskDef{
	{MappingsTask: "generateGateMappings", CSVFile: "gates.csv", OutputTask: "createGates", IDField: "cloud_gate_id"},
	{MappingsTask: "generateProfileMappings", CSVFile: "profiles.csv", OutputTask: "createProfiles", IDField: "cloud_profile_key"},
	{MappingsTask: "generateTemplateMappings", CSVFile: "templates.csv", OutputTask: "createPermissionTemplates", IDField: "cloud_template_id"},
	{MappingsTask: "generateGroupMappings", CSVFile: "groups.csv", OutputTask: "createGroups", IDField: "cloud_group_id"},
	{MappingsTask: "generatePortfolioMappings", CSVFile: "portfolios.csv", OutputTask: "createPortfolios", IDField: "cloud_portfolio_id"},
	{MappingsTask: "generateProjectMappings", CSVFile: "projects.csv", OutputTask: "createProjects", IDField: "cloud_project_key", NameField: "key"},
}

// BuildPredictiveRun synthesizes a predictive run directory under
// exportDir and returns its path. The directory contains:
//   - generate*Mappings JSONL (CSV → JSONL, joined to organizations.csv)
//   - create* JSONL with one synthetic row per non-skipped mapping
//   - addGateConditions.notes/ from the extract's getGateConditions data
//     run through the migrate metric-mapping table
//
// No HTTP calls are made; this is purely a local file synthesis. The
// caller hands the returned runDir to summary.GeneratePDFReport.
func BuildPredictiveRun(exportDir string) (string, error) {
	if exportDir == "" {
		return "", fmt.Errorf("export directory is required")
	}

	// Sanity-check: at least one of the mapping CSVs must exist. Otherwise
	// the user hasn't run `structure` yet and a predictive report would
	// just be empty noise.
	atLeastOneCSV := false
	for _, ct := range createTasks {
		if _, err := os.Stat(filepath.Join(exportDir, ct.CSVFile)); err == nil {
			atLeastOneCSV = true
			break
		}
	}
	if !atLeastOneCSV {
		return "", fmt.Errorf("no mapping CSVs found under %s — run `extract` and `structure` first", exportDir)
	}

	runID := "predictive-" + time.Now().UTC().Format("2006-01-02T150405Z")
	runDir := filepath.Join(exportDir, runID)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		return "", fmt.Errorf("creating predictive run directory: %w", err)
	}

	orgLookup, err := buildOrgKeyLookup(exportDir)
	if err != nil {
		return "", fmt.Errorf("loading organizations.csv: %w", err)
	}

	store := common.NewDataStore(runDir)

	// Convert each mapping CSV → JSONL + synthesize the matching create*
	// task output. Missing CSVs are tolerated (the user may not have
	// populated every section for their migration).
	for _, ct := range createTasks {
		rows, err := structure.LoadCSV(exportDir, ct.CSVFile)
		if err != nil {
			return "", fmt.Errorf("loading %s: %w", ct.CSVFile, err)
		}
		if len(rows) == 0 {
			continue
		}
		if err := writeMappingJSONL(store, ct.MappingsTask, rows, orgLookup); err != nil {
			return "", fmt.Errorf("synthesizing %s: %w", ct.MappingsTask, err)
		}
		if err := writeCreateJSONL(store, ct, rows, orgLookup); err != nil {
			return "", fmt.Errorf("synthesizing %s: %w", ct.OutputTask, err)
		}
	}

	// Extract-data-driven synthesizers — both need the extract mapping.
	extractMapping, err := structure.GetUniqueExtracts(exportDir)
	if err == nil && len(extractMapping) > 0 {
		if err := synthesizeAddGateConditionsNotes(exportDir, runDir, extractMapping, orgLookup); err != nil {
			return "", fmt.Errorf("synthesizing addGateConditions.notes: %w", err)
		}
		if err := synthesizeSetGlobalSettings(exportDir, runDir, extractMapping, orgLookup); err != nil {
			return "", fmt.Errorf("synthesizing setGlobalSettings: %w", err)
		}
	}

	return runDir, nil
}

// buildOrgKeyLookup reads organizations.csv and returns a map from
// sonarqube_org_key → sonarcloud_org_key. Mirrors what migrate does in
// helpers.go.
func buildOrgKeyLookup(exportDir string) (map[string]string, error) {
	rows, err := structure.LoadCSV(exportDir, "organizations.csv")
	if err != nil {
		return nil, err
	}
	out := make(map[string]string, len(rows))
	for _, row := range rows {
		sqKey, _ := row["sonarqube_org_key"].(string)
		scKey, _ := row["sonarcloud_org_key"].(string)
		if sqKey != "" {
			out[sqKey] = scKey
		}
	}
	return out, nil
}

// writeMappingJSONL writes one JSONL row per CSV row, enriched with the
// sonarcloud_org_key looked up from organizations.csv. Mirrors
// migrate.loadCSVToJSONL.
func writeMappingJSONL(store *common.DataStore, taskName string, rows []map[string]any, orgLookup map[string]string) error {
	w, err := store.Writer(taskName)
	if err != nil {
		return err
	}
	out := make([]json.RawMessage, 0, len(rows))
	for _, row := range rows {
		if sqKey, ok := row["sonarqube_org_key"].(string); ok && sqKey != "" {
			if scKey, found := orgLookup[sqKey]; found {
				row["sonarcloud_org_key"] = scKey
			}
		}
		b, err := json.Marshal(row)
		if err != nil {
			continue
		}
		out = append(out, b)
	}
	return w.WriteChunk(out)
}

// writeCreateJSONL writes one synthetic create-task row per non-skipped
// mapping. The row carries:
//   - every field from the original mapping row
//   - the synthetic cloud id (predict-<entity>-<name>-<org>) under IDField
//   - was_preexisting=false (a real migrate would discover this at runtime)
//
// The synthetic id is stable per (name, org) so the summary collector's
// dedup-by-composite-key still works for entities migrated across
// multiple source orgs.
func writeCreateJSONL(store *common.DataStore, ct createTaskDef, rows []map[string]any, orgLookup map[string]string) error {
	w, err := store.Writer(ct.OutputTask)
	if err != nil {
		return err
	}
	nameField := ct.NameField
	if nameField == "" {
		nameField = "name"
	}
	out := make([]json.RawMessage, 0, len(rows))
	for _, row := range rows {
		// Enrich with sonarcloud_org_key (mirrors migrate behaviour).
		if sqKey, ok := row["sonarqube_org_key"].(string); ok && sqKey != "" {
			if scKey, found := orgLookup[sqKey]; found {
				row["sonarcloud_org_key"] = scKey
			}
		}
		orgKey, _ := row["sonarcloud_org_key"].(string)
		if shouldSkipOrg(orgKey) {
			continue
		}
		name, _ := row[nameField].(string)
		if name == "" {
			continue
		}
		enriched := make(map[string]any, len(row)+2)
		for k, v := range row {
			enriched[k] = v
		}
		enriched[ct.IDField] = syntheticCloudID(ct.OutputTask, name, orgKey)
		enriched["was_preexisting"] = false
		b, err := json.Marshal(enriched)
		if err != nil {
			continue
		}
		out = append(out, b)
	}
	return w.WriteChunk(out)
}

// syntheticCloudID returns a stable placeholder cloud-side identifier
// for a predicted entity. The format is "predict:<task>:<org>:<name>"
// — distinguishable from a real cloud id at a glance, deterministic so
// the summary collector dedupes the same entity across source orgs.
func syntheticCloudID(outputTask, name, orgKey string) string {
	return "predict:" + outputTask + ":" + orgKey + ":" + name
}
