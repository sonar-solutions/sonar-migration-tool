package types

// AlmSetting represents a single ALM (Application Lifecycle Management)
// configuration returned by /api/alm_settings/list.
type AlmSetting struct {
	Key string `json:"key"`
	ALM string `json:"alm"`
	URL string `json:"url"`
}

// AlmSettingsResponse is the response envelope for /api/alm_settings/list.
type AlmSettingsResponse struct {
	AlmSettings []AlmSetting `json:"almSettings"`
}

// AlmBinding represents the ALM binding for a project returned by
// /api/alm_settings/get_binding.
type AlmBinding struct {
	Key                   string `json:"key"`
	ALM                   string `json:"alm"`
	Repository            string `json:"repository"`
	Slug                  string `json:"slug"`
	Monorepo              bool   `json:"monorepo"`
	URL                   string `json:"url"`
	SummaryCommentEnabled bool   `json:"summaryCommentEnabled"`
}
