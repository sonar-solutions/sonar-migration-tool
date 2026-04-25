package types

// Analysis represents a single project analysis returned by
// /api/project_analyses/search.
type Analysis struct {
	Key            string `json:"key"`
	Date           string `json:"date"`
	ProjectVersion string `json:"projectVersion"`
	Revision       string `json:"revision"`
	BuildString    string `json:"buildString"`
}

// AnalysesSearchResponse is the paged response envelope for
// /api/project_analyses/search.
type AnalysesSearchResponse struct {
	PagedResponse
	Analyses []Analysis `json:"analyses"`
}
