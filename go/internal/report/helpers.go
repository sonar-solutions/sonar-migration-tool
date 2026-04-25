package report

import (
	"encoding/json"

	"github.com/sonar-solutions/sonar-migration-tool/internal/structure"
)

// ParseJSONObject unmarshals a json.RawMessage into map[string]any.
// Returns an empty map on error.
func ParseJSONObject(raw json.RawMessage) map[string]any {
	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err != nil {
		return make(map[string]any)
	}
	return obj
}

// BuildServerIDMapping creates a server URL → server ID mapping
// by reading getServerInfo data from all extracts.
func BuildServerIDMapping(directory string, mapping structure.ExtractMapping) map[string]string {
	items, err := structure.ReadExtractData(directory, mapping, "getServerInfo")
	if err != nil {
		return make(map[string]string)
	}
	idMap := make(map[string]string, len(items))
	for _, item := range items {
		obj := ParseJSONObject(item.Data)
		serverID := ExtractString(obj, "System.Server ID")
		if serverID != "" {
			idMap[item.ServerURL] = serverID
		}
	}
	return idMap
}

// ServerIDFromURL looks up the server ID for a URL, falling back to the URL itself.
func ServerIDFromURL(idMapping map[string]string, url string) string {
	if id, ok := idMapping[url]; ok {
		return id
	}
	return url
}
