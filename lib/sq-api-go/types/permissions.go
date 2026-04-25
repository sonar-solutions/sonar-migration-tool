package types

// TemplatePermission describes the users/groups count for one permission key
// within a permission template.
type TemplatePermission struct {
	Key                string `json:"key"`
	UsersCount         int    `json:"usersCount"`
	GroupsCount        int    `json:"groupsCount"`
	WithProjectCreator bool   `json:"withProjectCreator"`
}

// PermissionTemplate represents a single permission template returned by
// /api/permissions/search_templates.
type PermissionTemplate struct {
	ID                string               `json:"id"`
	Name              string               `json:"name"`
	Description       string               `json:"description"`
	ProjectKeyPattern string               `json:"projectKeyPattern"`
	CreatedAt         string               `json:"createdAt"`
	UpdatedAt         string               `json:"updatedAt"`
	Permissions       []TemplatePermission `json:"permissions"`
}

// PermissionTemplatesResponse is the response envelope for
// /api/permissions/search_templates.
type PermissionTemplatesResponse struct {
	PermissionTemplates []PermissionTemplate `json:"permissionTemplates"`
}

// TemplateGroup is returned by /api/permissions/template_groups.
type TemplateGroup struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions"`
}

// TemplateGroupsResponse is the paged response envelope for
// /api/permissions/template_groups.
type TemplateGroupsResponse struct {
	PagedResponse
	Groups []TemplateGroup `json:"groups"`
}

// TemplateUser is returned by /api/permissions/template_users.
type TemplateUser struct {
	Login       string   `json:"login"`
	Name        string   `json:"name"`
	Email       string   `json:"email"`
	Permissions []string `json:"permissions"`
}

// TemplateUsersResponse is the paged response envelope for
// /api/permissions/template_users.
type TemplateUsersResponse struct {
	PagedResponse
	Users []TemplateUser `json:"users"`
}

// PermissionTemplateCreateResponse is the response envelope for
// /api/permissions/create_template (Cloud).
type PermissionTemplateCreateResponse struct {
	PermissionTemplate PermissionTemplate `json:"permissionTemplate"`
}
