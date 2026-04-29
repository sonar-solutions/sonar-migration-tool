// Package summary generates a PDF migration summary report from task outputs.
package summary

import "time"

// MigrationSummary holds the collected data for the PDF report.
type MigrationSummary struct {
	RunID       string
	GeneratedAt time.Time
	Sections    []Section
}

// Section represents a category of migrated entities (e.g., Projects, Quality Gates).
type Section struct {
	Name      string
	Succeeded []EntityItem
	Failed    []EntityItem
	Skipped   []EntityItem
}

// EntityItem represents a single entity in the report.
type EntityItem struct {
	Name         string
	Organization string
	Detail       string // cloud key for successes, scan history status, skip reason
	ErrorMessage string // failures only
}

// sectionDef maps a report section to its corresponding task names and analysis entity type.
type sectionDef struct {
	Name           string // display name
	InputTask      string // generate*Mappings task (for computing skips)
	OutputTask     string // create* task (successes)
	AnalysisEntity string // entity type in analysis report (for failures)
	NameField      string // JSONL field to extract entity name
	DetailField    string // JSONL field for detail column (e.g., cloud key)
}

// sectionDefs defines the sections in report order.
var sectionDefs = []sectionDef{
	{
		Name:           "Projects",
		InputTask:      "generateProjectMappings",
		OutputTask:     "createProjects",
		AnalysisEntity: "Project",
		NameField:      "name",
		DetailField:    "cloud_project_key",
	},
	{
		Name:           "Quality Profiles",
		InputTask:      "generateProfileMappings",
		OutputTask:     "createProfiles",
		AnalysisEntity: "Quality Profile",
		NameField:      "name",
		DetailField:    "cloud_profile_key",
	},
	{
		Name:           "Quality Gates",
		InputTask:      "generateGateMappings",
		OutputTask:     "createGates",
		AnalysisEntity: "Quality Gate",
		NameField:      "name",
		DetailField:    "cloud_gate_id",
	},
	{
		Name:           "Groups",
		InputTask:      "generateGroupMappings",
		OutputTask:     "createGroups",
		AnalysisEntity: "Group",
		NameField:      "name",
		DetailField:    "cloud_group_id",
	},
	{
		Name:           "Permission Templates",
		InputTask:      "generateTemplateMappings",
		OutputTask:     "createPermissionTemplates",
		AnalysisEntity: "Permission Template",
		NameField:      "name",
		DetailField:    "cloud_template_id",
	},
	{
		Name:           "Portfolios",
		InputTask:      "generatePortfolioMappings",
		OutputTask:     "createPortfolios",
		AnalysisEntity: "Portfolio",
		NameField:      "name",
		DetailField:    "cloud_portfolio_id",
	},
}

// skippedOrgSentinel matches the value used by the wizard when an org is skipped.
const skippedOrgSentinel = "SKIPPED"
