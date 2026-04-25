package maturity

import (
	"strings"

	"github.com/sonar-solutions/sonar-migration-tool/internal/report/common"
	"github.com/sonar-solutions/sonar-migration-tool/internal/structure"
)

const maturityTemplate = `# SonarQube Maturity Assessment

## Table of Contents

* Adoption
    * Instances
    * DevOps Integrations
    * CI Pipeline Overview
    * Usage
    * User Management
* Governance
    * Detected Project Groupings
    * Languages
    * Profiles
    * Gates
    * Active Gates
    * Permissions
    * Portfolios
* Workflow Integration
    * Scans
    * Issues
    * Testing
    * IDE
* Automation
    * API Usage
    * Webhooks

## Adoption
{instances}

{devops}

{pipeline}

{usage}

{user_management}

## Governance
{project_groupings}

{languages}

{profiles}

{gates}

{active_gates}

{permissions}

{portfolios}

## Workflow Integration

{scans}

{issues}

{vulnerabilities}

{bugs}

{code_smells}

{testing}

{ide}

## Automation

{tokens}

{webhooks}

## Appendix

{plugins}

{tasks}
`

// GenerateMaturityReport generates the maturity assessment markdown.
func GenerateMaturityReport(exportDir string, mapping structure.ExtractMapping) string {
	serverMD, idMap, projects := common.GenerateServerMarkdown(exportDir, mapping)
	pipelineOverview, _, scans := common.GeneratePipelineMarkdown(exportDir, mapping, idMap)
	pluginMD, plugins := common.GeneratePluginMarkdown(exportDir, mapping, idMap)
	devopsMD, _ := common.GenerateDevOpsMarkdown(exportDir, mapping, idMap)
	measures := common.ProcessProjectMeasures(exportDir, mapping, idMap)
	projectGroupings, _ := common.GeneratePermissionTemplateMarkdown(exportDir, mapping, idMap, projects, true)
	_, _, profileMap, _ := common.GenerateProfileMarkdown(exportDir, mapping, idMap, projects, plugins)
	gateSummary, gateDetails := GenerateGateMaturityMarkdown(exportDir, mapping, idMap, projects)
	languageMD, languages := GenerateLanguageMarkdown(measures, profileMap)
	profileMD := GenerateProfileSummary(profileMap, languages)
	userMD, users, _ := common.GenerateUserMarkdown(exportDir, mapping, idMap)
	permissionsMD, _ := GeneratePermissionsMarkdown(exportDir, mapping)
	usageMD := GenerateUsageMarkdown(projects, scans)
	scanMD := GenerateScansMarkdown(scans)
	issueOverview, vulnMD, bugMD, smellMD := GenerateIssueMarkdown(exportDir, mapping, idMap)
	ideMD := GenerateIDEMarkdown(users)
	coverageMD := GenerateCoverageMarkdown(measures)
	portfoliosMD := GeneratePortfolioSummaryMarkdown(exportDir, mapping, idMap)
	tokenMD := common.GenerateTokenMarkdown(exportDir, mapping, idMap)
	taskMD := common.GenerateTaskMarkdown(exportDir, mapping, idMap)
	webhooksMD := common.GenerateWebhookMarkdown(exportDir, mapping, idMap)

	return fillTemplate(maturityTemplate, map[string]string{
		"instances":          serverMD,
		"devops":             devopsMD,
		"pipeline":           pipelineOverview,
		"usage":              usageMD,
		"user_management":    userMD,
		"project_groupings":  projectGroupings,
		"languages":          languageMD,
		"profiles":           profileMD,
		"gates":              gateSummary,
		"active_gates":       gateDetails,
		"permissions":        permissionsMD,
		"portfolios":         portfoliosMD,
		"scans":              scanMD,
		"issues":             issueOverview,
		"vulnerabilities":    vulnMD,
		"bugs":               bugMD,
		"code_smells":        smellMD,
		"testing":            coverageMD,
		"ide":                ideMD,
		"tokens":             tokenMD,
		"webhooks":           webhooksMD,
		"plugins":            pluginMD,
		"tasks":              taskMD,
	})
}

func fillTemplate(template string, values map[string]string) string {
	result := template
	for key, value := range values {
		result = strings.ReplaceAll(result, "{"+key+"}", value)
	}
	return result
}
