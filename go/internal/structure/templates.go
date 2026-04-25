package structure

import (
	"encoding/json"
	"regexp"
	"strings"

	"github.com/sonar-solutions/sonar-migration-tool/internal/common"
)

// MapTemplates maps permission templates to organizations.
func MapTemplates(projectOrgMapping map[string]string, mapping ExtractMapping, directory string) []Template {
	orgKeys := uniqueOrgKeys(projectOrgMapping)
	results := make(map[string]Template)

	// Identify default templates.
	defaultTemplates := make(map[string]bool) // serverURL+templateId → true
	defaultItems, _ := ReadExtractData(directory, mapping, "getDefaultTemplates")
	for _, item := range defaultItems {
		templateID := common.ExtractField(item.Data, "templateId")
		defaultTemplates[item.ServerURL+templateID] = true
	}

	// Process templates.
	templateItems, _ := ReadExtractData(directory, mapping, "getTemplates")
	for _, item := range templateItems {
		var t map[string]any
		json.Unmarshal(item.Data, &t)
		templateID := getString(t, "id")
		templateKey := item.ServerURL + templateID
		pattern := getString(t, "projectKeyPattern")
		isDefault := defaultTemplates[templateKey]

		tpl := templateData{
			ID:          templateID,
			Name:        getString(t, "name"),
			Description: getString(t, "description"),
			Pattern:     pattern,
		}

		if isDefault || pattern == "" {
			// Default templates or templates without patterns go to all orgs.
			addTemplateForAllOrgs(results, orgKeys, item.ServerURL, tpl, isDefault)
		} else {
			// Templates with patterns match against project keys.
			addTemplateForMatchingProjects(results, item.ServerURL, tpl, projectOrgMapping)
		}
	}

	list := make([]Template, 0, len(results))
	for _, t := range results {
		list = append(list, t)
	}
	return list
}

type templateData struct {
	ID          string
	Name        string
	Description string
	Pattern     string
}

func addTemplateForAllOrgs(results map[string]Template, orgKeys []string, serverURL string, t templateData, isDefault bool) {
	for _, orgKey := range orgKeys {
		addTemplate(results, orgKey, serverURL, t, isDefault)
	}
}

func addTemplateForMatchingProjects(results map[string]Template, serverURL string, t templateData, projectOrgMapping map[string]string) {
	re, err := regexp.Compile(t.Pattern)
	if err != nil {
		return
	}
	for projectKey, orgKey := range projectOrgMapping {
		// Strip serverURL prefix for regex matching.
		stripped := strings.Replace(projectKey, serverURL, "", 1)
		if re.MatchString(stripped) {
			addTemplate(results, orgKey, serverURL, t, false)
		}
	}
}

func addTemplate(results map[string]Template, orgKey, serverURL string, t templateData, isDefault bool) {
	uniqueKey := orgKey + t.ID
	results[uniqueKey] = Template{
		UniqueKey:         uniqueKey,
		SourceTemplateKey: t.ID,
		Name:              t.Name,
		Description:       t.Description,
		ProjectKeyPattern: t.Pattern,
		ServerURL:         serverURL,
		IsDefault:         isDefault,
		SonarQubeOrgKey:   orgKey,
	}
}
