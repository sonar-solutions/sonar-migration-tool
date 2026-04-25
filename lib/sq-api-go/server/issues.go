package server

import (
	"context"
	"net/url"

	sqapi "github.com/sonar-solutions/sq-api-go"
	"github.com/sonar-solutions/sq-api-go/types"
)

// IssuesClient provides methods for /api/issues endpoints.
type IssuesClient struct{ baseClient }

// Search returns a Paginator over issues for a project from /api/issues/search.
func (i *IssuesClient) Search(ctx context.Context, projectKey string) *sqapi.Paginator[types.Issue] {
	return sqapi.NewPaginator(func(ctx context.Context, page, pageSize int) ([]types.Issue, int, error) {
		params := url.Values{}
		params.Set("components", projectKey)
		params.Set("p", itoa(page))
		params.Set("ps", itoa(pageSize))

		var result types.IssuesSearchResponse
		if err := i.get(ctx, "api/issues/search", params, &result); err != nil {
			return nil, 0, err
		}
		return result.Issues, result.Paging.Total, nil
	}, 0)
}
