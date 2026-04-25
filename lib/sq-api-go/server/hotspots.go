package server

import (
	"context"
	"net/url"

	sqapi "github.com/sonar-solutions/sq-api-go"
	"github.com/sonar-solutions/sq-api-go/types"
)

// HotspotsClient provides methods for /api/hotspots endpoints.
type HotspotsClient struct{ baseClient }

// Search returns a Paginator over security hotspots for a project from
// /api/hotspots/search.
func (h *HotspotsClient) Search(ctx context.Context, projectKey string) *sqapi.Paginator[types.Hotspot] {
	return sqapi.NewPaginator(func(ctx context.Context, page, pageSize int) ([]types.Hotspot, int, error) {
		params := url.Values{}
		params.Set("projectKey", projectKey)
		params.Set("p", itoa(page))
		params.Set("ps", itoa(pageSize))

		var result types.HotspotsSearchResponse
		if err := h.get(ctx, "api/hotspots/search", params, &result); err != nil {
			return nil, 0, err
		}
		return result.Hotspots, result.Paging.Total, nil
	}, 0)
}
