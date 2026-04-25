package common

import (
	"github.com/sonar-solutions/sonar-migration-tool/internal/report"
	"github.com/sonar-solutions/sonar-migration-tool/internal/structure"
)

// ProcessUsers extracts user data from getUsers, computing activity flags.
func ProcessUsers(dir string, mapping structure.ExtractMapping, idMap ServerIDMapping) map[string][]map[string]any {
	users := make(map[string][]map[string]any)
	for _, item := range readData(dir, mapping, "getUsers") {
		sid := serverID(idMap, item.ServerURL)
		user := processUser(item.Data)
		users[sid] = append(users[sid], user)
	}
	return users
}

func processUser(data map[string]any) map[string]any {
	user := map[string]any{
		"login":                report.ExtractString(data, "$.login"),
		"external_id":         report.ExtractString(data, "$.externalIdentity"),
		"last_connection":     report.ExtractString(data, "$.lastConnectionDate"),
		"external_provider":   report.ExtractString(data, "$.externalProvider"),
		"sonar_lint_connection": report.ExtractString(data, "$.sonarLintLastConnectionDate"),
		"is_active":           false,
		"is_active_sonar_lint": false,
	}

	if connStr, ok := user["last_connection"].(string); ok && connStr != "" {
		if t, ok := parseSQDate(connStr); ok && isRecent(t) {
			user["is_active"] = true
		}
	}
	if slStr, ok := user["sonar_lint_connection"].(string); ok && slStr != "" {
		if t, ok := parseSQDate(slStr); ok && isRecent(t) {
			user["is_active_sonar_lint"] = true
		}
	}
	return user
}

// GenerateUserMarkdown generates the User Management section.
func GenerateUserMarkdown(dir string, mapping structure.ExtractMapping, idMap ServerIDMapping) (string, map[string][]map[string]any, []map[string]any) {
	users := ProcessUsers(dir, mapping, idMap)
	groups := ProcessGroups(dir, mapping, idMap)

	allUsers := flattenUsers(users)
	uniqueIDs := countUniqueUsers(allUsers)
	activeCount := countActiveUsers(allUsers)
	ssoCount := countSSOUsers(allUsers)
	groupNames := countUniqueGroups(groups)

	row := map[string]any{
		"total":  len(allUsers),
		"unique": uniqueIDs,
		"active": activeCount,
		"sso":    ssoCount,
		"groups": groupNames,
	}

	md := report.GenerateSection(
		[]report.Column{
			{"Total Users", "total"}, {"Unique Users", "unique"}, {"Active Users", "active"},
			{"SSO Users", "sso"}, {"Groups", "groups"},
		},
		[]map[string]any{row},
		report.WithTitle("User Management", 3),
	)
	return md, users, groups
}

func flattenUsers(users map[string][]map[string]any) []map[string]any {
	var all []map[string]any
	for _, serverUsers := range users {
		all = append(all, serverUsers...)
	}
	return all
}

func countUniqueUsers(users []map[string]any) int {
	seen := make(map[string]bool)
	for _, u := range users {
		id, _ := u["external_id"].(string)
		if id == "" {
			id, _ = u["login"].(string)
		}
		if id != "" {
			seen[id] = true
		}
	}
	return len(seen)
}

func countActiveUsers(users []map[string]any) int {
	count := 0
	for _, u := range users {
		if active, ok := u["is_active"].(bool); ok && active {
			count++
		}
	}
	return count
}

func countSSOUsers(users []map[string]any) int {
	count := 0
	for _, u := range users {
		if provider, ok := u["external_provider"].(string); ok && provider != "" {
			count++
		}
	}
	return count
}

func countUniqueGroups(groups []map[string]any) int {
	names := make(map[string]bool)
	for _, g := range groups {
		if name, ok := g["name"].(string); ok {
			names[name] = true
		}
	}
	return len(names)
}
