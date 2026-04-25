package server

import (
	"context"
	"net/url"

	"github.com/sonar-solutions/sq-api-go/types"
)

// NewCodeClient provides methods for /api/new_code_periods endpoints.
type NewCodeClient struct{ baseClient }

// List returns all new code period definitions for a project from
// /api/new_code_periods/list.
func (n *NewCodeClient) List(ctx context.Context, projectKey string) ([]types.NewCodePeriod, error) {
	params := url.Values{}
	params.Set("project", projectKey)

	var result types.NewCodePeriodsResponse
	if err := n.get(ctx, "api/new_code_periods/list", params, &result); err != nil {
		return nil, err
	}
	return result.NewCodePeriods, nil
}
