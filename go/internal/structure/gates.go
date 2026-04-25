package structure

import (
	"encoding/json"

	"github.com/sonar-solutions/sonar-migration-tool/internal/common"
)

// MapGates maps quality gates to organizations based on project membership.
func MapGates(projectOrgMapping map[string]string, mapping ExtractMapping, directory string) []Gate {
	// Load all non-builtIn gates keyed by serverURL+name.
	gateItems, _ := ReadExtractData(directory, mapping, "getGates")
	gates := make(map[string]gateEntry)
	for _, item := range gateItems {
		var g map[string]any
		json.Unmarshal(item.Data, &g)
		if getBool(g, "isBuiltIn") {
			continue
		}
		name := getString(g, "name")
		gates[item.ServerURL+name] = gateEntry{
			Name:      name,
			IsDefault: getBool(g, "isDefault"),
			ServerURL: item.ServerURL,
		}
	}

	results := make(map[string]Gate)

	// Map gates referenced by projects.
	detailItems, _ := ReadExtractData(directory, mapping, "getProjectDetails")
	mapProjectGates(detailItems, projectOrgMapping, gates, results)

	// Add default gates for all orgs.
	addDefaultGates(projectOrgMapping, gates, results)

	list := make([]Gate, 0, len(results))
	for _, g := range results {
		list = append(list, g)
	}
	return list
}

type gateEntry struct {
	Name      string
	IsDefault bool
	ServerURL string
}

// mapProjectGates adds gates referenced by projects to the results map.
func mapProjectGates(detailItems []ExtractItem, projectOrgMapping map[string]string, gates map[string]gateEntry, results map[string]Gate) {
	for _, item := range detailItems {
		projectKey := common.ExtractField(item.Data, "projectKey")
		orgKey, ok := projectOrgMapping[item.ServerURL+projectKey]
		if !ok {
			continue
		}
		gateName := extractGateName(item.Data)
		if gateName == "" {
			continue
		}
		gateKey := item.ServerURL + gateName
		if _, ok := gates[gateKey]; !ok {
			continue
		}
		uniqueKey := orgKey + gateName
		results[uniqueKey] = Gate{
			Name:            gateName,
			ServerURL:       item.ServerURL,
			SourceGateKey:   gateName,
			IsDefault:       gates[gateKey].IsDefault,
			SonarQubeOrgKey: orgKey,
		}
	}
}

// addDefaultGates adds default gates for all orgs to the results map.
func addDefaultGates(projectOrgMapping map[string]string, gates map[string]gateEntry, results map[string]Gate) {
	orgKeys := uniqueOrgKeys(projectOrgMapping)
	for _, orgKey := range orgKeys {
		for _, gate := range gates {
			if !gate.IsDefault {
				continue
			}
			uniqueKey := orgKey + gate.Name
			results[uniqueKey] = Gate{
				Name:            gate.Name,
				ServerURL:       gate.ServerURL,
				SourceGateKey:   gate.Name,
				IsDefault:       true,
				SonarQubeOrgKey: orgKey,
			}
		}
	}
}

// extractGateName extracts the quality gate name from a project detail.
func extractGateName(raw json.RawMessage) string {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		return ""
	}
	gateRaw, ok := obj["qualityGate"]
	if !ok {
		return ""
	}
	var gate map[string]any
	if err := json.Unmarshal(gateRaw, &gate); err != nil {
		return ""
	}
	return getString(gate, "name")
}
