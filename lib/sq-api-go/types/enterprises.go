package types

// Enterprise represents a SonarQube Cloud enterprise returned by
// GET /enterprises/enterprises.
type Enterprise struct {
	ID   string `json:"id"`
	Key  string `json:"key"`
	Name string `json:"name"`
}

// EnterprisesListResponse is the response envelope for GET /enterprises/enterprises.
type EnterprisesListResponse struct {
	Enterprises []Enterprise `json:"enterprises"`
}

// Portfolio represents a SonarQube Cloud portfolio returned by the
// /enterprises/portfolios endpoints.
type Portfolio struct {
	ID           string   `json:"id"`
	Key          string   `json:"key"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	EnterpriseID string   `json:"enterpriseId"`
	Selection    string   `json:"selection"`
	Projects     []string `json:"projects"`
}
