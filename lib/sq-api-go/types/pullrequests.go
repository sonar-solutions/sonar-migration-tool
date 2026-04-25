package types

// PullRequestStatus holds the quality gate result for a pull request.
type PullRequestStatus struct {
	QualityGateStatus string `json:"qualityGateStatus"`
	Bugs              int    `json:"bugs"`
	Vulnerabilities   int    `json:"vulnerabilities"`
	CodeSmells        int    `json:"codeSmells"`
}

// PullRequest represents a single pull request returned by
// /api/project_pull_requests/list.
type PullRequest struct {
	Key          string            `json:"key"`
	Title        string            `json:"title"`
	Branch       string            `json:"branch"`
	Base         string            `json:"base"`
	Status       PullRequestStatus `json:"status"`
	AnalysisDate string            `json:"analysisDate"`
	URL          string            `json:"url"`
}

// PullRequestsResponse is the response envelope for
// /api/project_pull_requests/list.
type PullRequestsResponse struct {
	PullRequests []PullRequest `json:"pullRequests"`
}
