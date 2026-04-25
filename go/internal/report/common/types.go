package common

import (
	"encoding/json"
	"time"

	"github.com/sonar-solutions/sonar-migration-tool/internal/report"
	"github.com/sonar-solutions/sonar-migration-tool/internal/structure"
)

// ServerIDMapping maps server URLs to human-readable server IDs.
type ServerIDMapping = map[string]string

// Projects is a nested map: serverID → projectKey → project details.
type Projects = map[string]map[string]map[string]any

// recentWindow is the 30-day lookback window used across multiple generators.
var recentWindow = 30 * 24 * time.Hour

// SonarQube date format used in JSONL data.
const sqDateFormat = "2006-01-02T15:04:05-0700"

// parseSQDate parses a SonarQube-format date string.
func parseSQDate(s string) (time.Time, bool) {
	if s == "" {
		return time.Time{}, false
	}
	t, err := time.Parse(sqDateFormat, s)
	if err != nil {
		// Try alternate format without timezone offset.
		t, err = time.Parse("2006-01-02T15:04:05Z", s)
		if err != nil {
			return time.Time{}, false
		}
	}
	return t, true
}

// isRecent checks if a date is within the 30-day lookback window.
func isRecent(t time.Time) bool {
	return t.After(time.Now().UTC().Add(-recentWindow))
}

// readData reads JSONL items for a key and parses them into maps.
func readData(dir string, mapping structure.ExtractMapping, key string) []dataItem {
	items, err := structure.ReadExtractData(dir, mapping, key)
	if err != nil {
		return nil
	}
	result := make([]dataItem, 0, len(items))
	for _, item := range items {
		obj := report.ParseJSONObject(item.Data)
		if len(obj) > 0 {
			result = append(result, dataItem{ServerURL: item.ServerURL, Data: obj})
		}
	}
	return result
}

// readDataRaw reads JSONL items for a key and returns raw messages with server URLs.
func readDataRaw(dir string, mapping structure.ExtractMapping, key string) []rawDataItem {
	items, err := structure.ReadExtractData(dir, mapping, key)
	if err != nil {
		return nil
	}
	result := make([]rawDataItem, 0, len(items))
	for _, item := range items {
		result = append(result, rawDataItem{ServerURL: item.ServerURL, Data: item.Data})
	}
	return result
}

type dataItem struct {
	ServerURL string
	Data      map[string]any
}

type rawDataItem struct {
	ServerURL string
	Data      json.RawMessage
}

// DataItem is the exported version of dataItem for use by subpackages.
type DataItem = dataItem

// ReadDataParsed reads JSONL items and returns parsed objects. Exported for maturity subpackage.
func ReadDataParsed(dir string, mapping structure.ExtractMapping, key string) []DataItem {
	return readData(dir, mapping, key)
}

// ServerIDLookup looks up the server ID for a URL, exported for maturity subpackage.
func ServerIDLookup(idMap ServerIDMapping, url string) string {
	return serverID(idMap, url)
}

// serverID looks up the server ID for a URL, falling back to the URL.
func serverID(idMap ServerIDMapping, url string) string {
	if id, ok := idMap[url]; ok {
		return id
	}
	return url
}
