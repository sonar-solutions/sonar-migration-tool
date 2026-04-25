package types

// Group represents a single group returned by /api/permissions/groups.
type Group struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	MembersCount int    `json:"membersCount"`
	Default      bool   `json:"default"`
}

// GroupsSearchResponse is the response envelope for /api/permissions/groups.
type GroupsSearchResponse struct {
	PagedResponse
	Groups []Group `json:"groups"`
}

// GroupUser represents a user returned by /api/user_groups/users.
type GroupUser struct {
	Login    string `json:"login"`
	Name     string `json:"name"`
	Selected bool   `json:"selected"`
}

// GroupUsersResponse is the response envelope for /api/user_groups/users.
type GroupUsersResponse struct {
	PagedResponse
	Users []GroupUser `json:"users"`
}

// GroupCreateResponse is the response envelope for /api/user_groups/create (Cloud).
type GroupCreateResponse struct {
	Group Group `json:"group"`
}
