package structure

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/sonar-solutions/sonar-migration-tool/internal/common"
)

// ExtractMapping maps server URLs to their latest extract IDs.
type ExtractMapping map[string]string

// GetUniqueExtracts scans the export directory for extract runs and returns
// a mapping of server URL → latest extract ID.
func GetUniqueExtracts(directory string) (ExtractMapping, error) {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return nil, err
	}

	urlMappings := buildURLMappings(directory, entries)

	result := make(ExtractMapping, len(urlMappings))
	for url, ids := range urlMappings {
		result[url] = latestID(ids)
	}
	return result, nil
}

// buildURLMappings scans extract directories and groups extract IDs by server URL.
func buildURLMappings(directory string, entries []os.DirEntry) map[string]map[string]bool {
	urlMappings := make(map[string]map[string]bool)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		metaPath := filepath.Join(directory, entry.Name(), "extract.json")
		data, err := os.ReadFile(metaPath)
		if err != nil {
			continue
		}
		var meta struct {
			URL string `json:"url"`
		}
		if err := json.Unmarshal(data, &meta); err != nil || meta.URL == "" {
			continue
		}
		if urlMappings[meta.URL] == nil {
			urlMappings[meta.URL] = make(map[string]bool)
		}
		urlMappings[meta.URL][entry.Name()] = true
	}
	return urlMappings
}

// latestID returns the lexicographically greatest key from a set.
func latestID(ids map[string]bool) string {
	var latest string
	for id := range ids {
		if id > latest {
			latest = id
		}
	}
	return latest
}

// MultiExtractReader reads JSONL objects from the named task across all extract
// runs in the mapping. It yields (serverURL, rawObject) pairs.
// Yields (serverURL, rawObject) pairs across all extract runs.
type ExtractItem struct {
	ServerURL string
	Data      json.RawMessage
}

// ReadExtractData reads all JSONL items for a given task key across all extracts.
func ReadExtractData(directory string, mapping ExtractMapping, key string) ([]ExtractItem, error) {
	var items []ExtractItem
	for serverURL, extractID := range mapping {
		taskDir := filepath.Join(directory, extractID, key)
		raw, err := readTaskDir(taskDir)
		if err != nil {
			continue // task may not exist for this extract
		}
		for _, r := range raw {
			items = append(items, ExtractItem{ServerURL: serverURL, Data: r})
		}
	}
	return items, nil
}

// readTaskDir reads all JSONL files from a task directory.
func readTaskDir(dir string) ([]json.RawMessage, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var all []json.RawMessage
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".jsonl" {
			continue
		}
		items, err := common.ReadJSONLFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, err
		}
		all = append(all, items...)
	}
	return all, nil
}
