package server

import (
	"context"
	"net/url"

	sqapi "github.com/sonar-solutions/sq-api-go"
	"github.com/sonar-solutions/sq-api-go/types"
)

const groupsPageSize = 100

// GroupsClient provides methods for group-related endpoints.
type GroupsClient struct{ baseClient }

// Search returns a Paginator over all groups from /api/permissions/groups.
// SonarQube accepts a maximum of 100 groups per page on this endpoint.
func (g *GroupsClient) Search(ctx context.Context) *sqapi.Paginator[types.Group] {
	return sqapi.NewPaginator(func(ctx context.Context, page, pageSize int) ([]types.Group, int, error) {
		params := url.Values{}
		params.Set("p", itoa(page))
		params.Set("ps", itoa(pageSize))

		var result types.GroupsSearchResponse
		if err := g.get(ctx, "api/permissions/groups", params, &result); err != nil {
			return nil, 0, err
		}
		return result.Groups, result.Paging.Total, nil
	}, groupsPageSize)
}

// Users returns a Paginator over the members of a named group from
// /api/user_groups/users.
func (g *GroupsClient) Users(ctx context.Context, groupName string) *sqapi.Paginator[types.GroupUser] {
	return sqapi.NewPaginator(func(ctx context.Context, page, pageSize int) ([]types.GroupUser, int, error) {
		params := url.Values{}
		params.Set("name", groupName)
		params.Set("p", itoa(page))
		params.Set("ps", itoa(pageSize))

		var result types.GroupUsersResponse
		if err := g.get(ctx, "api/user_groups/users", params, &result); err != nil {
			return nil, 0, err
		}
		return result.Users, result.Paging.Total, nil
	}, 0)
}
