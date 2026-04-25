package server

import (
	"context"
	"net/url"

	"github.com/sonar-solutions/sq-api-go/types"
)

// PullRequestsClient provides methods for /api/project_pull_requests endpoints.
type PullRequestsClient struct{ baseClient }

// List returns all pull requests for a project from
// /api/project_pull_requests/list.
func (p *PullRequestsClient) List(ctx context.Context, projectKey string) ([]types.PullRequest, error) {
	params := url.Values{}
	params.Set("project", projectKey)

	var result types.PullRequestsResponse
	if err := p.get(ctx, "api/project_pull_requests/list", params, &result); err != nil {
		return nil, err
	}
	return result.PullRequests, nil
}
