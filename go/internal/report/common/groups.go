package common

import (
	"github.com/sonar-solutions/sonar-migration-tool/internal/report"
	"github.com/sonar-solutions/sonar-migration-tool/internal/structure"
)

// ProcessGroups extracts user groups from getGroups data.
func ProcessGroups(dir string, mapping structure.ExtractMapping, idMap ServerIDMapping) []map[string]any {
	var groups []map[string]any
	for _, item := range readData(dir, mapping, "getGroups") {
		sid := serverID(idMap, item.ServerURL)
		groups = append(groups, map[string]any{
			"server_id":   sid,
			"name":        report.ExtractString(item.Data, "$.name"),
			"permissions": report.ExtractPathValue(item.Data, "$.permissions", nil),
			"is_managed":  report.ExtractBool(item.Data, "$.managed"),
		})
	}
	return groups
}
