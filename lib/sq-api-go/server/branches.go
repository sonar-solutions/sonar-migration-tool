package server

import (
	"context"
	"net/url"

	"github.com/sonar-solutions/sq-api-go/types"
)

// BranchesClient provides methods for /api/project_branches endpoints.
type BranchesClient struct{ baseClient }

// List returns all branches for a project from /api/project_branches/list.
func (b *BranchesClient) List(ctx context.Context, projectKey string) ([]types.Branch, error) {
	params := url.Values{}
	params.Set("project", projectKey)

	var result types.BranchesResponse
	if err := b.get(ctx, "api/project_branches/list", params, &result); err != nil {
		return nil, err
	}
	return result.Branches, nil
}
