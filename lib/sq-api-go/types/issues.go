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
	CloseDate        string           `json:"closeDate"`
	Line             int              `json:"line"`
	Message          string           `json:"message"`
	Assignee         string           `json:"assignee"`
	TextRange        *IssueTextRange  `json:"textRange"`
	Comments         []IssueComment   `json:"comments"`
	Flows            []IssueFlow      `json:"flows"`
	Transitions      []string         `json:"transitions"`
}

// IssueTextRange describes the line and character offsets of an issue location.
type IssueTextRange struct {
	StartLine   int `json:"startLine"`
	EndLine     int `json:"endLine"`
	StartOffset int `json:"startOffset"`
	EndOffset   int `json:"endOffset"`
}

// IssueComment represents a single comment on an issue.
type IssueComment struct {
	Key       string `json:"key"`
	Login     string `json:"login"`
	HTMLText  string `json:"htmlText"`
	Markdown  string `json:"markdown"`
	CreatedAt string `json:"createdAt"`
}

// IssueFlow represents a data-flow or execution-flow path for an issue.
type IssueFlow struct {
	Locations []IssueLocation `json:"locations"`
}

// IssueLocation is a single location in an issue flow.
type IssueLocation struct {
	Component string          `json:"component"`
	TextRange *IssueTextRange `json:"textRange"`
	Msg       string          `json:"msg"`
}

// IssuesSearchResponse is the paged response envelope for /api/issues/search.
type IssuesSearchResponse struct {
	PagedResponse
	Issues []Issue `json:"issues"`
}
