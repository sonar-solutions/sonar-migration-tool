package maturity

import (
	"github.com/sonar-solutions/sonar-migration-tool/internal/report"
	"github.com/sonar-solutions/sonar-migration-tool/internal/structure"
)

// GeneratePermissionsMarkdown generates the permissions summary for the maturity report.
func GeneratePermissionsMarkdown(dir string, mapping structure.ExtractMapping) (string, map[string]any) {
	globalPerms := processGlobalPermissions(dir, mapping)
	entityPerms := processEntityPermissions(dir, mapping)

	row := map[string]any{
		"global_user_profile":      globalPerms["user"]["profileadmin"],
		"global_user_quality_gate": globalPerms["user"]["gateadmin"],
		"global_group_profile":     globalPerms["groups"]["profileadmin"],
		"global_group_quality_gate": globalPerms["groups"]["gateadmin"],
		"specific_user_profile":     len(entityPerms["profile"]["users"]),
		"specific_user_quality_gate": len(entityPerms["gate"]["users"]),
		"specific_group_profile":    len(entityPerms["profile"]["groups"]),
		"specific_group_quality_gate": len(entityPerms["gate"]["groups"]),
	}

	md := report.GenerateSection(
		[]report.Column{
			{"Global User Profile Admins", "global_user_profile"},
			{"Global User Gate Admins", "global_user_quality_gate"},
			{"Global Group Profile Admins", "global_group_profile"},
			{"Global Group Gate Admins", "global_group_quality_gate"},
			{"Specific User Profile Perms", "specific_user_profile"},
			{"Specific User Gate Perms", "specific_user_quality_gate"},
			{"Specific Group Profile Perms", "specific_group_profile"},
			{"Specific Group Gate Perms", "specific_group_quality_gate"},
		},
		[]map[string]any{row},
		report.WithTitle("Permissions", 3),
	)
	return md, row
}

func processGlobalPermissions(dir string, mapping structure.ExtractMapping) map[string]map[string]int {
	result := map[string]map[string]int{
		"user":   {"profileadmin": 0, "gateadmin": 0},
		"groups": {"profileadmin": 0, "gateadmin": 0},
	}
	countPermissions(dir, mapping, "user", "getUserPermissions", result)
	countPermissions(dir, mapping, "groups", "getGroupPermissions", result)
	return result
}

func countPermissions(dir string, mapping structure.ExtractMapping, entityType, key string, result map[string]map[string]int) {
	items, err := structure.ReadExtractData(dir, mapping, key)
	if err != nil {
		return
	}
	for _, item := range items {
		obj := report.ParseJSONObject(item.Data)
		countEntityPermissions(obj, entityType, result)
	}
}

func countEntityPermissions(obj map[string]any, entityType string, result map[string]map[string]int) {
	perms := report.ExtractPathValue(obj, "$.permissions", nil)
	arr, ok := perms.([]any)
	if !ok {
		return
	}
	for _, p := range arr {
		if ps, ok := p.(string); ok {
			if _, exists := result[entityType][ps]; exists {
				result[entityType][ps]++
			}
		}
	}
}

func processEntityPermissions(dir string, mapping structure.ExtractMapping) map[string]map[string]map[string]bool {
	result := map[string]map[string]map[string]bool{
		"profile": {"users": {}, "groups": {}},
		"gate":    {"users": {}, "groups": {}},
	}
	entityKeys := map[string]map[string]string{
		"profile": {"users": "getProfileUsers", "groups": "getProfileGroups"},
		"gate":    {"users": "getGateUsers", "groups": "getGateGroups"},
	}
	for entity, keys := range entityKeys {
		for kind, key := range keys {
			collectEntityIDs(dir, mapping, kind, key, result[entity][kind])
		}
	}
	return result
}

func collectEntityIDs(dir string, mapping structure.ExtractMapping, kind, key string, dest map[string]bool) {
	items, err := structure.ReadExtractData(dir, mapping, key)
	if err != nil {
		return
	}
	for _, item := range items {
		obj := report.ParseJSONObject(item.Data)
		id := extractEntityID(obj, kind)
		if id != "" {
			dest[id] = true
		}
	}
}

func extractEntityID(obj map[string]any, kind string) string {
	if kind == "users" {
		return report.ExtractString(obj, "$.login")
	}
	return report.ExtractString(obj, "$.name")
}
