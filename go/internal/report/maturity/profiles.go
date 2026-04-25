package maturity

import (
	"strings"

	"github.com/sonar-solutions/sonar-migration-tool/internal/report"
	"github.com/sonar-solutions/sonar-migration-tool/internal/report/common"
)

// GenerateProfileSummary generates the quality profile maturity summary.
func GenerateProfileSummary(profileMap common.ProfileMap, languages map[string]map[string]any) string {
	row := map[string]any{"profiles": 0, "active": 0, "sonar_way": 0, "custom_defaults": 0}

	for _, langProfiles := range profileMap {
		for lang, profilesByName := range langProfiles {
			if _, ok := languages[lang]; !ok {
				continue
			}
			for _, profile := range profilesByName {
				accumulateProfile(row, profile)
			}
		}
	}

	return report.GenerateSection(
		[]report.Column{
			{"Profiles", "profiles"}, {"Active Profiles", "active"},
			{"Inherits Sonar Way", "sonar_way"}, {"Custom Defaults", "custom_defaults"},
		},
		[]map[string]any{row},
		report.WithTitle("Quality Profiles", 3),
	)
}

func accumulateProfile(row, profile map[string]any) {
	row["profiles"] = row["profiles"].(int) + 1
	projects, _ := profile["projects"].(map[string]bool)
	if len(projects) > 0 {
		row["active"] = row["active"].(int) + 1
		if inheritsSonarWay(profile) {
			row["sonar_way"] = row["sonar_way"].(int) + 1
		}
	}
	if isCustomDefault(profile) {
		row["custom_defaults"] = row["custom_defaults"].(int) + 1
	}
}

func inheritsSonarWay(profile map[string]any) bool {
	root, _ := profile["root"].(string)
	name, _ := profile["name"].(string)
	return root != "" && (strings.EqualFold(root, "sonar way") || strings.EqualFold(name, "sonar way"))
}

func isCustomDefault(profile map[string]any) bool {
	isBuiltIn, _ := profile["is_built_in"].(bool)
	isDefault, _ := profile["is_default"].(bool)
	return !isBuiltIn && isDefault
}
