package common

import (
	"github.com/sonar-solutions/sonar-migration-tool/internal/report"
	"github.com/sonar-solutions/sonar-migration-tool/internal/structure"
)

// Measures is a nested map: serverID → projectKey → {metric: value}.
type Measures = map[string]map[string]map[string]any

// ProcessProjectMeasures extracts project metrics from getProjectMeasures data.
func ProcessProjectMeasures(dir string, mapping structure.ExtractMapping, idMap ServerIDMapping) Measures {
	measures := make(Measures)
	for _, item := range readData(dir, mapping, "getProjectMeasures") {
		sid := serverID(idMap, item.ServerURL)
		projectKey := report.ExtractString(item.Data, "$.projectKey")
		metric := report.ExtractString(item.Data, "$.metric")
		if projectKey == "" || metric == "" {
			continue
		}
		ensureNestedMap(measures, sid, projectKey)
		measures[sid][projectKey]["server_id"] = sid
		measures[sid][projectKey]["project_key"] = projectKey
		value := extractMeasureValue(item.Data)
		if value != nil {
			measures[sid][projectKey][metric] = value
		}
	}
	return measures
}

// extractMeasureValue recursively extracts a measure value, handling period nesting.
func extractMeasureValue(measure map[string]any) any {
	if period, ok := measure["period"]; ok {
		if periodMap, ok := period.(map[string]any); ok {
			return extractMeasureValue(periodMap)
		}
	}
	return measure["value"]
}

func ensureNestedMap(m Measures, key1, key2 string) {
	if m[key1] == nil {
		m[key1] = make(map[string]map[string]any)
	}
	if m[key1][key2] == nil {
		m[key1][key2] = make(map[string]any)
	}
}
