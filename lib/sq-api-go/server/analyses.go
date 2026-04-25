package server

import (
	"context"
	"net/url"

	sqapi "github.com/sonar-solutions/sq-api-go"
	"github.com/sonar-solutions/sq-api-go/types"
)

// AnalysesClient provides methods for /api/project_analyses endpoints.
type AnalysesClient struct{ baseClient }

// Search returns a Paginator over analyses for a project from
// /api/project_analyses/search.
func (a *AnalysesClient) Search(ctx context.Context, projectKey string) *sqapi.Paginator[types.Analysis] {
	return sqapi.NewPaginator(func(ctx context.Context, page, pageSize int) ([]types.Analysis, int, error) {
		params := url.Values{}
		params.Set("project", projectKey)
		params.Set("p", itoa(page))
		params.Set("ps", itoa(pageSize))

		var result types.AnalysesSearchResponse
		if err := a.get(ctx, "api/project_analyses/search", params, &result); err != nil {
			return nil, 0, err
		}
		return result.Analyses, result.Paging.Total, nil
	}, 0)
}
