// Copyright (C) SonarSource Sàrl
// For more information, see https://sonarsource.com/legal/
// mailto:info AT sonarsource DOT com

package types

import "encoding/json"

// Hotspot represents a single security hotspot returned by /api/hotspots/search.
type Hotspot struct {
	Key                      string `json:"key"`
	Component                string `json:"component"`
	Project                  string `json:"project"`
	SecurityCategory         string `json:"securityCategory"`
	VulnerabilityProbability string `json:"vulnerabilityProbability"`
	Status                   string `json:"status"`
	Resolution               string `json:"resolution"`
	Line                     int    `json:"line"`
	Message                  string `json:"message"`
	Author                   string `json:"author"`
	CreationDate             string `json:"creationDate"`
	UpdateDate               string `json:"updateDate"`
	RuleKey                  string `json:"ruleKey"`
}

// HotspotDetail is the full detail returned by /api/hotspots/show.
//
// Unlike /api/hotspots/search (where component and project are plain key
// strings), /api/hotspots/show returns them as nested objects. They are kept
// as json.RawMessage so the response unmarshals regardless of shape — the
// migrate-side hotspot sync only consumes Comment and Rule from this type.
type HotspotDetail struct {
	Key                      string           `json:"key"`
	Component                json.RawMessage  `json:"component"`
	Project                  json.RawMessage  `json:"project"`
	SecurityCategory         string           `json:"securityCategory"`
	VulnerabilityProbability string           `json:"vulnerabilityProbability"`
	Status                   string           `json:"status"`
	Resolution               string           `json:"resolution"`
	Line                     int              `json:"line"`
	Message                  string           `json:"message"`
	Author                   string           `json:"author"`
	RuleKey                  string           `json:"ruleKey"`
	CreationDate             string           `json:"creationDate"`
	UpdateDate               string           `json:"updateDate"`
	Comment                  []HotspotComment `json:"comment"`
	Rule                     HotspotRule      `json:"rule"`
}

// HotspotComment represents a single comment on a hotspot.
type HotspotComment struct {
	Key       string `json:"key"`
	Login     string `json:"login"`
	HTMLText  string `json:"htmlText"`
	Markdown  string `json:"markdown"`
	CreatedAt string `json:"createdAt"`
}

// HotspotRule carries the rule metadata embedded in a hotspot detail response.
type HotspotRule struct {
	Key                      string `json:"key"`
	Name                     string `json:"name"`
	SecurityCategory         string `json:"securityCategory"`
	VulnerabilityProbability string `json:"vulnerabilityProbability"`
}

// HotspotsSearchResponse is the paged response envelope for /api/hotspots/search.
type HotspotsSearchResponse struct {
	PagedResponse
	Hotspots []Hotspot `json:"hotspots"`
}
