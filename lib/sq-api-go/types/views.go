package types

// View represents a portfolio or application returned by /api/views/search.
// Qualifier is "VW" for portfolios and "APP" for applications.
type View struct {
	Key           string `json:"key"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	Qualifier     string `json:"qualifier"`
	Visibility    string `json:"visibility"`
	SelectionMode string `json:"selectionMode"`
}

// ViewsSearchResponse is the paged response envelope for /api/views/search.
type ViewsSearchResponse struct {
	PagedResponse
	Views []View `json:"views"`
}

// ViewSubView is a nested sub-portfolio or sub-application inside a view.
type ViewSubView struct {
	Key           string `json:"key"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	Qualifier     string `json:"qualifier"`
	SelectionMode string `json:"selectionMode"`
}

// ViewDetails is the full portfolio/application returned by /api/views/show.
type ViewDetails struct {
	Key              string        `json:"key"`
	Name             string        `json:"name"`
	Description      string        `json:"description"`
	Qualifier        string        `json:"qualifier"`
	Visibility       string        `json:"visibility"`
	SelectionMode    string        `json:"selectionMode"`
	SubViews         []ViewSubView `json:"subViews"`
	SelectedProjects []string      `json:"selectedProjects"`
}
