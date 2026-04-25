package server

import (
	"context"
	"net/url"

	sqapi "github.com/sonar-solutions/sq-api-go"
	"github.com/sonar-solutions/sq-api-go/types"
)

// ViewsClient provides methods for /api/views endpoints.
// These endpoints are only available on Enterprise Edition and above.
type ViewsClient struct{ baseClient }

// Search returns a Paginator over portfolios and/or applications from
// /api/views/search.
// qualifier filters by type: "VW" for portfolios, "APP" for applications.
// Pass empty string to return both.
func (v *ViewsClient) Search(ctx context.Context, qualifier string) *sqapi.Paginator[types.View] {
	return sqapi.NewPaginator(func(ctx context.Context, page, pageSize int) ([]types.View, int, error) {
		params := url.Values{}
		if qualifier != "" {
			params.Set("qualifiers", qualifier)
		}
		params.Set("p", itoa(page))
		params.Set("ps", itoa(pageSize))

		var result types.ViewsSearchResponse
		if err := v.get(ctx, "api/views/search", params, &result); err != nil {
			return nil, 0, err
		}
		return result.Views, result.Paging.Total, nil
	}, 0)
}

// Show returns the full details of a portfolio or application from
// /api/views/show.
func (v *ViewsClient) Show(ctx context.Context, key string) (*types.ViewDetails, error) {
	params := url.Values{}
	params.Set("key", key)

	var result types.ViewDetails
	if err := v.get(ctx, "api/views/show", params, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
