package types

// QualityProfile represents a single quality profile returned by
// /api/qualityprofiles/search.
type QualityProfile struct {
	Key             string `json:"key"`
	Name            string `json:"name"`
	Language        string `json:"language"`
	LanguageName    string `json:"languageName"`
	IsDefault       bool   `json:"isDefault"`
	IsBuiltIn       bool   `json:"isBuiltIn"`
	ActiveRuleCount int    `json:"activeRuleCount"`
	RulesUpdatedAt  string `json:"rulesUpdatedAt"`
	LastUsed        string `json:"lastUsed"`
	ParentKey       string `json:"parentKey"`
	ParentName      string `json:"parentName"`
}

// QualityProfilesSearchResponse is the response envelope for /api/qualityprofiles/search.
type QualityProfilesSearchResponse struct {
	Profiles []QualityProfile `json:"profiles"`
}

// ProfileGroup is returned by /api/qualityprofiles/search_groups.
type ProfileGroup struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Selected    bool   `json:"selected"`
}

// ProfileGroupsResponse is the response envelope for /api/qualityprofiles/search_groups.
type ProfileGroupsResponse struct {
	PagedResponse
	Groups []ProfileGroup `json:"groups"`
}

// ProfileUser is returned by /api/qualityprofiles/search_users.
type ProfileUser struct {
	Login    string `json:"login"`
	Name     string `json:"name"`
	Selected bool   `json:"selected"`
}

// ProfileUsersResponse is the response envelope for /api/qualityprofiles/search_users.
type ProfileUsersResponse struct {
	PagedResponse
	Users []ProfileUser `json:"users"`
}

// QualityProfileCreateResponse is the response envelope for
// /api/qualityprofiles/create (Cloud).
type QualityProfileCreateResponse struct {
	Profile QualityProfile `json:"profile"`
}

// QualityProfileRestoreResponse is the response envelope for
// /api/qualityprofiles/restore (Cloud). The profile field is omitted when
// the backup is for a built-in profile.
type QualityProfileRestoreResponse struct {
	Profile QualityProfile `json:"profile"`
}
