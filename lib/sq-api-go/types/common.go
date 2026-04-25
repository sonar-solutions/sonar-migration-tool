// Package types contains shared response types for the sq-api-go library.
package types

// Paging holds the pagination metadata returned by all paginated SonarQube
// API endpoints. Corresponds to the $.paging field in JSON responses.
type Paging struct {
	PageIndex int `json:"pageIndex"`
	PageSize  int `json:"pageSize"`
	Total     int `json:"total"`
}

// PagedResponse is embedded by every paginated endpoint response struct.
// Endpoint-specific response types embed this and add their own result field.
//
// Example:
//
//	type ProjectsSearchResponse struct {
//	    types.PagedResponse
//	    Components []Project `json:"components"`
//	}
type PagedResponse struct {
	Paging Paging `json:"paging"`
}
