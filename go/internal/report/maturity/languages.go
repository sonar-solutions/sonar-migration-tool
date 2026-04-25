package maturity

import (
	"strconv"
	"strings"

	"github.com/sonar-solutions/sonar-migration-tool/internal/report"
	"github.com/sonar-solutions/sonar-migration-tool/internal/report/common"
)

// GenerateLanguageMarkdown generates the language distribution summary.
func GenerateLanguageMarkdown(measures common.Measures, profileMap common.ProfileMap) (string, map[string]map[string]any) {
	languages := make(map[string]map[string]any)
	collectLanguageUsage(measures, languages)
	collectProfileData(profileMap, languages)

	var rows []map[string]any
	for _, lang := range languages {
		rows = append(rows, lang)
	}

	md := report.GenerateSection(
		[]report.Column{
			{"Language", "language"}, {"Projects", "projects"}, {"Lines of Code", "loc"},
			{"Profiles", "profiles"}, {"Custom Profiles", "custom_profiles"},
			{"Base Rules", "base_rules"}, {"Min Rules", "minimum_rules"},
			{"Max Rules", "maximum_rules"}, {"Avg Rules", "average_rules"},
		},
		rows,
		report.WithTitle("Languages", 3),
		report.WithSortBy("loc", true),
	)
	return md, languages
}

func collectLanguageUsage(measures common.Measures, languages map[string]map[string]any) {
	for _, serverProjects := range measures {
		for _, project := range serverProjects {
			parseLangUsage(project, languages)
		}
	}
}

func parseLangUsage(project map[string]any, languages map[string]map[string]any) {
	dist, ok := project["ncloc_language_distribution"].(string)
	if !ok || dist == "" {
		return
	}
	for _, pair := range strings.Split(dist, ";") {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) != 2 {
			continue
		}
		lang := parts[0]
		loc, err := strconv.Atoi(parts[1])
		if err != nil {
			continue
		}
		ensureLang(languages, lang)
		languages[lang]["projects"] = languages[lang]["projects"].(int) + 1
		languages[lang]["loc"] = languages[lang]["loc"].(int) + loc
	}
}

func collectProfileData(profileMap common.ProfileMap, languages map[string]map[string]any) {
	for _, langProfiles := range profileMap {
		for lang, profilesByName := range langProfiles {
			for name, profile := range profilesByName {
				processProfile(languages, lang, name, profile)
			}
		}
	}
}

func processProfile(languages map[string]map[string]any, lang, name string, profile map[string]any) {
	ensureLang(languages, lang)
	entry := languages[lang]
	if strings.EqualFold(name, "sonar way") {
		ruleCount := countRules(profile)
		entry["base_rules"] = ruleCount
	}
	projects, _ := profile["projects"].(map[string]bool)
	if len(projects) > 0 {
		updateProfileRules(entry, profile)
	}
}

func updateProfileRules(entry, profile map[string]any) {
	isBuiltIn, _ := profile["is_built_in"].(bool)
	ruleCount := countRules(profile)
	entry["profiles"] = entry["profiles"].(int) + 1
	if !isBuiltIn {
		entry["custom_profiles"] = entry["custom_profiles"].(int) + 1
	}
	minR := entry["minimum_rules"].(int)
	maxR := entry["maximum_rules"].(int)
	totalR := entry["total_rules"].(int) + ruleCount
	profiles := entry["profiles"].(int)
	if minR == 0 || ruleCount < minR {
		entry["minimum_rules"] = ruleCount
	}
	if ruleCount > maxR {
		entry["maximum_rules"] = ruleCount
	}
	entry["total_rules"] = totalR
	if profiles > 0 {
		entry["average_rules"] = totalR / profiles
	}
}

func countRules(profile map[string]any) int {
	if rc, ok := profile["rule_count"].(int); ok {
		return rc
	}
	if rules, ok := profile["rules"].(map[string]bool); ok {
		return len(rules)
	}
	return 0
}

func ensureLang(languages map[string]map[string]any, lang string) {
	if languages[lang] == nil {
		languages[lang] = map[string]any{
			"language": lang, "projects": 0, "loc": 0,
			"custom_profiles": 0, "profiles": 0, "base_rules": 0,
			"minimum_rules": 0, "maximum_rules": 0, "average_rules": 0, "total_rules": 0,
		}
	}
}
