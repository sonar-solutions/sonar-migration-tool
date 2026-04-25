package types

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
}

// HotspotsSearchResponse is the paged response envelope for /api/hotspots/search.
type HotspotsSearchResponse struct {
	PagedResponse
	Hotspots []Hotspot `json:"hotspots"`
}
