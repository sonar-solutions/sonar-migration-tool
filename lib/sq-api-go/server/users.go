package server

import (
	"context"
	"net/url"

	sqapi "github.com/sonar-solutions/sq-api-go"
	"github.com/sonar-solutions/sq-api-go/types"
)

const usersPageSize = 100

// UsersClient provides methods for the /api/users endpoints.
type UsersClient struct{ baseClient }

// Search returns a Paginator over all users from /api/users/search.
// SonarQube accepts a maximum of 100 users per page on this endpoint.
func (u *UsersClient) Search(ctx context.Context) *sqapi.Paginator[types.User] {
	return sqapi.NewPaginator(func(ctx context.Context, page, pageSize int) ([]types.User, int, error) {
		params := url.Values{}
		params.Set("p", itoa(page))
		params.Set("ps", itoa(pageSize))

		var result types.UsersSearchResponse
		if err := u.get(ctx, "api/users/search", params, &result); err != nil {
			return nil, 0, err
		}
		return result.Users, result.Paging.Total, nil
	}, usersPageSize)
}
