package types

// Project represents a single project returned by /api/projects/search.
type Project struct {
	Key        string   `json:"key"`
	Name       string   `json:"name"`
	Qualifier  string   `json:"qualifier"`
	Visibility string   `json:"visibility"`
	Tags       []string `json:"tags"`
}

// ProjectsSearchResponse is the response envelope for /api/projects/search.
type ProjectsSearchResponse struct {
	PagedResponse
	Components []Project `json:"components"`
}

// ComponentDetails is returned by /api/navigation/component and /api/components/show.
type ComponentDetails struct {
	Key        string   `json:"key"`
	Name       string   `json:"name"`
	Qualifier  string   `json:"qualifier"`
	Visibility string   `json:"visibility"`
	Tags       []string `json:"tags"`
}

// ComponentShowResponse is the response envelope for /api/components/show.
type ComponentShowResponse struct {
	Component ComponentDetails `json:"component"`
}

// NavigationComponentResponse is the response envelope for /api/navigation/component.
type NavigationComponentResponse struct {
	Key        string   `json:"key"`
	Name       string   `json:"name"`
	Qualifier  string   `json:"qualifier"`
	Visibility string   `json:"visibility"`
	Tags       []string `json:"tags"`
}

// ProjectsLicenseUsageResponse is the response envelope for /api/projects/license_usage.
type ProjectsLicenseUsageResponse struct {
	Projects []Project `json:"projects"`
}

// ProjectCreateResponse is the response envelope for /api/projects/create (Cloud).
type ProjectCreateResponse struct {
	Project Project `json:"project"`
}
