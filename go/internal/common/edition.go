package common

import (
	"encoding/json"
	"strings"
)

// Edition represents a SonarQube edition.
type Edition string

const (
	EditionCommunity  Edition = "community"
	EditionDeveloper  Edition = "developer"
	EditionEnterprise Edition = "enterprise"
	EditionDatacenter Edition = "datacenter"
)

// AllEditions is the full set of Server editions (not including Cloud teams/free).
var AllEditions = []Edition{EditionCommunity, EditionDeveloper, EditionEnterprise, EditionDatacenter}

// ParseEdition normalises the string from /api/system/info into an Edition.
// It checks both the top-level "edition" key and the nested "System.Edition" path.
func ParseEdition(raw json.RawMessage) Edition {
	var info map[string]json.RawMessage
	if err := json.Unmarshal(raw, &info); err != nil {
		return EditionCommunity
	}

	// Try top-level "edition" first.
	if ed, ok := parseEditionString(info["edition"]); ok {
		return ed
	}

	// Try nested "System.Edition" (newer API format).
	if sysRaw, ok := info["System"]; ok {
		var sys map[string]json.RawMessage
		if json.Unmarshal(sysRaw, &sys) == nil {
			if ed, ok := parseEditionString(sys["Edition"]); ok {
				return ed
			}
		}
	}

	return EditionCommunity
}

// parseEditionString extracts and normalises an edition from a raw JSON string.
func parseEditionString(raw json.RawMessage) (Edition, bool) {
	if raw == nil {
		return "", false
	}
	var s string
	if json.Unmarshal(raw, &s) != nil {
		return "", false
	}
	ed := Edition(strings.ToLower(s))
	switch ed {
	case EditionCommunity, EditionDeveloper, EditionEnterprise, EditionDatacenter:
		return ed, true
	}
	return "", false
}
