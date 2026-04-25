package types

// Issue represents a single code issue returned by /api/issues/search.
type Issue struct {
	Key              string   `json:"key"`
	Rule             string   `json:"rule"`
	Severity         string   `json:"severity"`
	Component        string   `json:"component"`
	Project          string   `json:"project"`
	Status           string   `json:"status"`
	Resolution       string   `json:"resolution"`
	Type             string   `json:"type"`
	Effort           string   `json:"effort"`
	Debt             string   `json:"debt"`
	Tags             []string `json:"tags"`
	Author           string   `json:"author"`
	CreationDate     string   `json:"creationDate"`
	UpdateDate       string   `json:"updateDate"`
	CloseDate        string   `json:"closeDate"`
}

// IssuesSearchResponse is the paged response envelope for /api/issues/search.
type IssuesSearchResponse struct {
	PagedResponse
	Issues []Issue `json:"issues"`
}
