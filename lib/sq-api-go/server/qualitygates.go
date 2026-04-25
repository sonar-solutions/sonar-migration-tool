package server

import (
	"context"
	"net/url"

	sqapi "github.com/sonar-solutions/sq-api-go"
	"github.com/sonar-solutions/sq-api-go/types"
)

const qualityGatesPageSize = 1000

// QualityGatesClient provides methods for the /api/qualitygates endpoints.
type QualityGatesClient struct{ baseClient }

// List returns all quality gates from /api/qualitygates/list.
func (q *QualityGatesClient) List(ctx context.Context) ([]types.QualityGate, error) {
	var result types.QualityGatesListResponse
	if err := q.get(ctx, "api/qualitygates/list", nil, &result); err != nil {
		return nil, err
	}
	return result.QualityGates, nil
}

// Show returns a single quality gate by name from /api/qualitygates/show.
func (q *QualityGatesClient) Show(ctx context.Context, gateName string) (*types.QualityGate, error) {
	params := url.Values{}
	params.Set("name", gateName)

	var result types.QualityGate
	if err := q.get(ctx, "api/qualitygates/show", params, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// SearchGroups returns a Paginator over groups associated with a quality gate
// from /api/qualitygates/search_groups.
func (q *QualityGatesClient) SearchGroups(ctx context.Context, gateName string) *sqapi.Paginator[types.QualityGateGroup] {
	return sqapi.NewPaginator(func(ctx context.Context, page, pageSize int) ([]types.QualityGateGroup, int, error) {
		params := url.Values{}
		params.Set("gateName", gateName)
		params.Set("p", itoa(page))
		params.Set("ps", itoa(pageSize))

		var result types.QualityGateGroupsResponse
		if err := q.get(ctx, "api/qualitygates/search_groups", params, &result); err != nil {
			return nil, 0, err
		}
		return result.Groups, result.Paging.Total, nil
	}, qualityGatesPageSize)
}

// SearchUsers returns a Paginator over users associated with a quality gate
// from /api/qualitygates/search_users.
func (q *QualityGatesClient) SearchUsers(ctx context.Context, gateName string) *sqapi.Paginator[types.QualityGateUser] {
	return sqapi.NewPaginator(func(ctx context.Context, page, pageSize int) ([]types.QualityGateUser, int, error) {
		params := url.Values{}
		params.Set("gateName", gateName)
		params.Set("p", itoa(page))
		params.Set("ps", itoa(pageSize))

		var result types.QualityGateUsersResponse
		if err := q.get(ctx, "api/qualitygates/search_users", params, &result); err != nil {
			return nil, 0, err
		}
		return result.Users, result.Paging.Total, nil
	}, qualityGatesPageSize)
}
