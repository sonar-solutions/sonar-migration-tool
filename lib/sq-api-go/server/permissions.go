package server

import (
	"context"
	"net/url"

	sqapi "github.com/sonar-solutions/sq-api-go"
	"github.com/sonar-solutions/sq-api-go/types"
)

// PermissionsClient provides methods for /api/permissions endpoints.
type PermissionsClient struct{ baseClient }

// SearchTemplates returns all permission templates from
// /api/permissions/search_templates.
func (p *PermissionsClient) SearchTemplates(ctx context.Context) ([]types.PermissionTemplate, error) {
	var result types.PermissionTemplatesResponse
	if err := p.get(ctx, "api/permissions/search_templates", nil, &result); err != nil {
		return nil, err
	}
	return result.PermissionTemplates, nil
}

// TemplateGroups returns a Paginator over groups associated with a permission
// template from /api/permissions/template_groups.
func (p *PermissionsClient) TemplateGroups(ctx context.Context, templateName string) *sqapi.Paginator[types.TemplateGroup] {
	return sqapi.NewPaginator(func(ctx context.Context, page, pageSize int) ([]types.TemplateGroup, int, error) {
		params := url.Values{}
		params.Set("templateName", templateName)
		params.Set("p", itoa(page))
		params.Set("ps", itoa(pageSize))

		var result types.TemplateGroupsResponse
		if err := p.get(ctx, "api/permissions/template_groups", params, &result); err != nil {
			return nil, 0, err
		}
		return result.Groups, result.Paging.Total, nil
	}, 0)
}

// TemplateUsers returns a Paginator over users associated with a permission
// template from /api/permissions/template_users.
func (p *PermissionsClient) TemplateUsers(ctx context.Context, templateName string) *sqapi.Paginator[types.TemplateUser] {
	return sqapi.NewPaginator(func(ctx context.Context, page, pageSize int) ([]types.TemplateUser, int, error) {
		params := url.Values{}
		params.Set("templateName", templateName)
		params.Set("p", itoa(page))
		params.Set("ps", itoa(pageSize))

		var result types.TemplateUsersResponse
		if err := p.get(ctx, "api/permissions/template_users", params, &result); err != nil {
			return nil, 0, err
		}
		return result.Users, result.Paging.Total, nil
	}, 0)
}
