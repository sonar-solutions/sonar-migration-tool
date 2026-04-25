package structure

import (
	"encoding/json"

	"github.com/sonar-solutions/sonar-migration-tool/internal/common"
)

// MapProfiles maps quality profiles to organizations based on project membership.
func MapProfiles(projectOrgMapping map[string]string, mapping ExtractMapping, directory string) []Profile {
	// Load all profiles keyed by profile key.
	profileItems, _ := ReadExtractData(directory, mapping, "getProfiles")
	profileMap := make(map[string]profileEntry)
	for _, item := range profileItems {
		var p map[string]any
		json.Unmarshal(item.Data, &p)
		key := getString(p, "key")
		profileMap[key] = profileEntry{
			Key:       key,
			Name:      getString(p, "name"),
			Language:  getString(p, "language"),
			ParentKey: getString(p, "parentKey"),
			IsDefault: getBool(p, "isDefault"),
			IsBuiltIn: getBool(p, "isBuiltIn"),
			ServerURL: item.ServerURL,
		}
	}

	results := make(map[string]Profile)

	// Add profiles referenced by projects.
	detailItems, _ := ReadExtractData(directory, mapping, "getProjectDetails")
	mapProjectProfiles(detailItems, projectOrgMapping, profileMap, results)

	// Add default profiles for all orgs.
	addDefaultProfiles(projectOrgMapping, profileMap, results)

	list := make([]Profile, 0, len(results))
	for _, p := range results {
		list = append(list, p)
	}
	return list
}

type profileEntry struct {
	Key       string
	Name      string
	Language  string
	ParentKey string
	IsDefault bool
	IsBuiltIn bool
	ServerURL string
}

// mapProjectProfiles adds profiles referenced by projects to the results map.
func mapProjectProfiles(detailItems []ExtractItem, projectOrgMapping map[string]string, profileMap map[string]profileEntry, results map[string]Profile) {
	for _, item := range detailItems {
		projectKey := common.ExtractField(item.Data, "projectKey")
		orgKey, ok := projectOrgMapping[item.ServerURL+projectKey]
		if !ok {
			continue
		}
		profiles := extractProfilesArray(item.Data)
		for _, profile := range profiles {
			if getBool(profile, "deleted") {
				continue
			}
			profileKey := getString(profile, "key")
			addProfile(profileMap, results, orgKey, item.ServerURL, profileKey)
		}
	}
}

// addDefaultProfiles adds default profiles for all orgs to the results map.
func addDefaultProfiles(projectOrgMapping map[string]string, profileMap map[string]profileEntry, results map[string]Profile) {
	orgKeys := uniqueOrgKeys(projectOrgMapping)
	for _, orgKey := range orgKeys {
		for _, profile := range profileMap {
			if !profile.IsDefault {
				continue
			}
			addProfile(profileMap, results, orgKey, profile.ServerURL, profile.Key)
		}
	}
}

// addProfile adds a profile and its parent chain to the results (skipping builtIn).
func addProfile(profileMap map[string]profileEntry, results map[string]Profile, orgKey, serverURL, profileKey string) {
	p, ok := profileMap[profileKey]
	if !ok || p.IsBuiltIn {
		return
	}
	uniqueKey := orgKey + profileKey
	if _, exists := results[uniqueKey]; exists {
		return
	}
	parentName := ""
	if parent, ok := profileMap[p.ParentKey]; ok {
		parentName = parent.Name
	}
	results[uniqueKey] = Profile{
		UniqueKey:        uniqueKey,
		Name:             p.Name,
		Language:         p.Language,
		ParentName:       parentName,
		ServerURL:        serverURL,
		IsDefault:        p.IsDefault,
		SourceProfileKey: profileKey,
		SonarQubeOrgKey:  orgKey,
	}
	if p.ParentKey != "" {
		addProfile(profileMap, results, orgKey, serverURL, p.ParentKey)
	}
}

// extractProfilesArray extracts the qualityProfiles array from a project detail.
func extractProfilesArray(raw json.RawMessage) []map[string]any {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil
	}
	profilesRaw, ok := obj["qualityProfiles"]
	if !ok {
		return nil
	}
	var profiles []map[string]any
	json.Unmarshal(profilesRaw, &profiles)
	return profiles
}

func getBool(m map[string]any, key string) bool {
	if m == nil {
		return false
	}
	v, ok := m[key]
	if !ok || v == nil {
		return false
	}
	b, ok := v.(bool)
	return ok && b
}

func uniqueOrgKeys(mapping map[string]string) []string {
	seen := make(map[string]bool)
	for _, v := range mapping {
		seen[v] = true
	}
	keys := make([]string, 0, len(seen))
	for k := range seen {
		keys = append(keys, k)
	}
	return keys
}
