package common

import (
	"github.com/sonar-solutions/sonar-migration-tool/internal/report"
	"github.com/sonar-solutions/sonar-migration-tool/internal/structure"
)

// BuiltinRepos is the set of SonarQube built-in rule repositories.
var BuiltinRepos = map[string]bool{
	"abap": true, "apex": true, "azureresourcemanager": true, "c": true,
	"cloudformation": true, "cobol": true, "cpp": true, "csharpsquid": true,
	"css": true, "docker": true, "dart": true, "flex": true, "go": true,
	"ipython": true, "java": true, "javabugs": true, "javascript": true,
	"javasecurity": true, "jcl": true, "jssecurity": true, "kotlin": true,
	"kubernetes": true, "objc": true, "php": true, "phpsecurity": true,
	"pli": true, "plsql": true, "python": true, "pythonbugs": true,
	"pythonsecurity": true, "roslyn.sonaranalyzer.security.cs": true,
	"rpg": true, "ruby": true, "scala": true, "secrets": true,
	"swift": true, "terraform": true, "text": true, "tsql": true,
	"tssecurity": true, "typescript": true, "vb": true, "vbnet": true,
	"Web": true, "xml": true,
	// External rule repos (still built-in to SQ)
	"external_android-lint": true, "external_bandit": true, "external_cfn-lint": true,
	"external_checkstyle": true, "external_detekt": true, "external_eslint_repo": true,
	"external_fbcontrib": true, "external_findsecbugs": true, "external_flake8": true,
	"external_golint": true, "external_govet": true, "external_hadolint": true,
	"external_ktlint": true, "external_mypy": true, "external_phpstan": true,
	"external_pmd": true, "external_pmd_apex": true, "external_psalm": true,
	"external_pylint": true, "external_rubocop": true, "external_ruff": true,
	"external_scalastyle": true, "external_scapegoat": true, "external_spotbugs": true,
	"external_stylelint": true, "external_swiftlint": true, "external_tflint": true,
	"external_tslint_repo": true, "external_valgrind-c": true, "external_valgrind-cpp": true,
	"external_valgrind-objc": true,
}

// ProfileMap is serverID → language → profileName → profile data.
type ProfileMap = map[string]map[string]map[string]map[string]any

// ProcessRules classifies rules as template or plugin rules.
func ProcessRules(dir string, mapping structure.ExtractMapping, idMap ServerIDMapping, plugins map[string][]map[string]any) (map[string]map[string]bool, map[string]map[string]bool) {
	templateRules := make(map[string]map[string]bool)
	pluginRules := make(map[string]map[string]bool)
	for _, item := range readData(dir, mapping, "getRules") {
		sid := serverID(idMap, item.ServerURL)
		key := report.ExtractString(item.Data, "$.key")
		if key == "" {
			continue
		}
		if report.ExtractString(item.Data, "$.templateKey") != "" {
			ensureStringSet(templateRules, sid)
			templateRules[sid][key] = true
		}
		repo := report.ExtractString(item.Data, "$.repo")
		if !BuiltinRepos[repo] && len(plugins[sid]) > 0 {
			ensureStringSet(pluginRules, sid)
			pluginRules[sid][key] = true
		}
	}
	return templateRules, pluginRules
}

// ProcessProfileRules maps profile keys to their rule sets.
func ProcessProfileRules(dir string, mapping structure.ExtractMapping, idMap ServerIDMapping) map[string]map[string]map[string]bool {
	profiles := make(map[string]map[string]map[string]bool)
	for _, item := range readData(dir, mapping, "getProfileRules") {
		sid := serverID(idMap, item.ServerURL)
		for ruleKey, actives := range item.Data {
			addRuleToProfiles(profiles, sid, ruleKey, actives)
		}
	}
	return profiles
}

func addRuleToProfiles(profiles map[string]map[string]map[string]bool, sid, ruleKey string, actives any) {
	arr, ok := actives.([]any)
	if !ok {
		return
	}
	for _, active := range arr {
		am, ok := active.(map[string]any)
		if !ok {
			continue
		}
		profileKey := report.ExtractString(am, "$.qProfile")
		if profileKey == "" {
			continue
		}
		ensureProfileRules(profiles, sid, profileKey)
		profiles[sid][profileKey][ruleKey] = true
	}
}

// ProcessProfileProjects maps profiles to the projects using them and accumulates rule counts.
func ProcessProfileProjects(projects Projects, profileRules map[string]map[string]map[string]bool, templateRules, pluginRules map[string]map[string]bool) map[string]map[string]map[string]bool {
	profileProjects := make(map[string]map[string]map[string]bool)
	for sid, serverProjects := range projects {
		for _, project := range serverProjects {
			profileList, _ := project["profiles"].([]string)
			for _, profileKey := range profileList {
				ensureProfileProjects(profileProjects, sid, profileKey)
				projectKey, _ := project["key"].(string)
				profileProjects[sid][profileKey][projectKey] = true

				rules := getStringSet(profileRules, sid, profileKey)
				project["rules"] = toInt(project["rules"]) + len(rules)
				project["template_rules"] = toInt(project["template_rules"]) + countOverlap(templateRules[sid], rules)
				project["plugin_rules"] = toInt(project["plugin_rules"]) + countOverlap(pluginRules[sid], rules)
			}
		}
	}
	return profileProjects
}

// GenerateProfileMarkdown generates active and inactive quality profile sections.
func GenerateProfileMarkdown(dir string, mapping structure.ExtractMapping, idMap ServerIDMapping, projects Projects, plugins map[string][]map[string]any) (string, string, ProfileMap, Projects) {
	templateRules, pluginRules := ProcessRules(dir, mapping, idMap, plugins)
	profileRules := ProcessProfileRules(dir, mapping, idMap)
	profileProjects := ProcessProfileProjects(projects, profileRules, templateRules, pluginRules)
	profiles, profileMap := processQualityProfiles(dir, mapping, idMap, profileProjects, profileRules, templateRules, pluginRules)

	active := report.GenerateSection(
		[]report.Column{
			{"Server ID", "server_id"}, {"Language", "language"}, {"Quality Profile Name", "name"},
			{"Parent Profile", "parent"}, {"Default Profile", "is_default"}, {"Total Rules", "rule_count"},
			{"Template Rules", "template_rules"}, {"Rules from 3rd party plugins", "plugin_rules"},
			{"# of Projects using", "project_count"},
		},
		profiles,
		report.WithTitle("Quality Profiles", 2),
		report.WithSortBy("project_count", true),
		report.WithFilter(func(r map[string]any) bool {
			count, _ := r["project_count"].(int)
			isDefault, _ := r["is_default"].(bool)
			isBuiltIn, _ := r["is_built_in"].(bool)
			return (count > 0 || isDefault) && !isBuiltIn
		}),
	)

	inactive := report.GenerateSection(
		[]report.Column{
			{"Server ID", "server_id"}, {"Language", "language"}, {"Quality Profile Name", "name"},
			{"Parent Profile", "parent"}, {"Total Rules", "rule_count"},
			{"Template Rules", "template_rules"}, {"Rules from 3rd party plugins", "plugin_rules"},
		},
		profiles,
		report.WithTitle("Quality Profiles", 2),
		report.WithFilter(func(r map[string]any) bool {
			count, _ := r["project_count"].(int)
			isDefault, _ := r["is_default"].(bool)
			isBuiltIn, _ := r["is_built_in"].(bool)
			return count == 0 && !isDefault && !isBuiltIn
		}),
	)
	return active, inactive, profileMap, projects
}

func processQualityProfiles(dir string, mapping structure.ExtractMapping, idMap ServerIDMapping, profileProjects map[string]map[string]map[string]bool, profileRules map[string]map[string]map[string]bool, templateRules, pluginRules map[string]map[string]bool) ([]map[string]any, ProfileMap) {
	var profiles []map[string]any
	profileMap := make(ProfileMap)

	for _, item := range readData(dir, mapping, "getProfiles") {
		sid := serverID(idMap, item.ServerURL)
		profileKey := report.ExtractString(item.Data, "$.key")
		language := report.ExtractString(item.Data, "$.language")
		name := report.ExtractString(item.Data, "$.name")
		if language == "" || name == "" {
			continue
		}

		rules := getStringSet(profileRules, sid, profileKey)
		projectSet := getStringSet(profileProjects, sid, profileKey)

		profile := map[string]any{
			"server_id":      sid,
			"language":       language,
			"key":            profileKey,
			"name":           name,
			"is_built_in":    report.ExtractBool(item.Data, "$.isBuiltIn"),
			"is_default":     report.ExtractBool(item.Data, "$.isDefault"),
			"parent":         report.ExtractString(item.Data, "$.parentName"),
			"rule_count":     len(rules),
			"template_rules": countOverlap(templateRules[sid], rules),
			"plugin_rules":   countOverlap(pluginRules[sid], rules),
			"project_count":  len(projectSet),
			"projects":       projectSet,
		}

		ensureProfileMap(profileMap, sid, language)
		profileMap[sid][language][name] = profile
	}

	// Resolve root parents and collect into flat list.
	for sid, languages := range profileMap {
		for _, profilesByName := range languages {
			for _, profile := range profilesByName {
				profile["root"] = extractRootParent(profile, profilesByName)
				profile["server_id"] = sid
				profiles = append(profiles, profile)
			}
		}
	}
	return profiles, profileMap
}

func extractRootParent(profile map[string]any, profiles map[string]map[string]any) string {
	parent, _ := profile["parent"].(string)
	if parent == "" {
		return ""
	}
	if parentProfile, ok := profiles[parent]; ok {
		grandparent := extractRootParent(parentProfile, profiles)
		if grandparent != "" {
			return grandparent
		}
	}
	return parent
}

// --- helpers ---

func ensureStringSet(m map[string]map[string]bool, key string) {
	if m[key] == nil {
		m[key] = make(map[string]bool)
	}
}

func ensureProfileRules(m map[string]map[string]map[string]bool, sid, profileKey string) {
	if m[sid] == nil {
		m[sid] = make(map[string]map[string]bool)
	}
	if m[sid][profileKey] == nil {
		m[sid][profileKey] = make(map[string]bool)
	}
}

func ensureProfileProjects(m map[string]map[string]map[string]bool, sid, profileKey string) {
	ensureProfileRules(m, sid, profileKey)
}

func ensureProfileMap(m ProfileMap, sid, language string) {
	if m[sid] == nil {
		m[sid] = make(map[string]map[string]map[string]any)
	}
	if m[sid][language] == nil {
		m[sid][language] = make(map[string]map[string]any)
	}
}

func getStringSet(m map[string]map[string]map[string]bool, key1, key2 string) map[string]bool {
	if m[key1] != nil && m[key1][key2] != nil {
		return m[key1][key2]
	}
	return nil
}

func countOverlap(a, b map[string]bool) int {
	count := 0
	for k := range b {
		if a[k] {
			count++
		}
	}
	return count
}

func toInt(v any) int {
	if n, ok := v.(int); ok {
		return n
	}
	return 0
}
