package migration

import (
	"strings"

	"github.com/sonar-solutions/sonar-migration-tool/internal/report/common"
	"github.com/sonar-solutions/sonar-migration-tool/internal/structure"
)

const migrationTemplate = `# SonarQube Utilization Assessment

## Table of Contents

* Instance Overview
* Devops Integrations
* CI Environment Overview
* Permissions
* Governance
    * Installed Plugins
    * Custom Quality Profiles
    * Portfolios
    * Applications
* Installed Plugins
* Project Metrics
* Appendix

{instance_overview}
{devops_integrations}
{pipeline_overview}

## Governance
{permission_templates}

{active_quality_profiles}

{active_quality_gates}

{active_portfolios}

{active_applications}

{plugins}

{project_metrics}

## Appendix
{project_scan_details}

{unused_quality_gates}

{unused_quality_profiles}

{empty_portfolios}

{empty_applications}

`

// GenerateMigrationReport generates the migration/utilization assessment markdown.
func GenerateMigrationReport(exportDir string, mapping structure.ExtractMapping) string {
	serverMD, idMap, projects := common.GenerateServerMarkdown(exportDir, mapping)
	pipelineOverview, scanDetails, projectScans := common.GeneratePipelineMarkdown(exportDir, mapping, idMap)
	devopsMD, _ := common.GenerateDevOpsMarkdown(exportDir, mapping, idMap)
	permissionsMD, _ := common.GeneratePermissionTemplateMarkdown(exportDir, mapping, idMap, projects, false)
	pluginsMD, plugins := common.GeneratePluginMarkdown(exportDir, mapping, idMap)
	activeProfiles, inactiveProfiles, _, projects := common.GenerateProfileMarkdown(exportDir, mapping, idMap, projects, plugins)
	activePortfolios, inactivePortfolios := common.GeneratePortfolioMarkdown(exportDir, mapping, idMap)
	activeGates, inactiveGates := common.GenerateGateMarkdown(exportDir, mapping, idMap, projects)
	activeApps, inactiveApps := common.GenerateApplicationMarkdown(exportDir, mapping, idMap)
	projectMetrics := common.GenerateProjectMetricsMarkdown(projects, projectScans)

	return fillTemplate(migrationTemplate, map[string]string{
		"instance_overview":      serverMD,
		"devops_integrations":    devopsMD,
		"pipeline_overview":      pipelineOverview,
		"permission_templates":   permissionsMD,
		"active_quality_profiles": activeProfiles,
		"active_quality_gates":   activeGates,
		"active_portfolios":      activePortfolios,
		"active_applications":    activeApps,
		"plugins":                pluginsMD,
		"project_metrics":        projectMetrics,
		"project_scan_details":   scanDetails,
		"unused_quality_gates":   inactiveGates,
		"unused_quality_profiles": inactiveProfiles,
		"empty_portfolios":       inactivePortfolios,
		"empty_applications":     inactiveApps,
	})
}

// fillTemplate replaces {key} placeholders in the template with values.
func fillTemplate(template string, values map[string]string) string {
	result := template
	for key, value := range values {
		result = strings.ReplaceAll(result, "{"+key+"}", value)
	}
	return result
}
