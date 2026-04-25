package server

import (
	"context"
	"net/url"

	sqapi "github.com/sonar-solutions/sq-api-go"
	"github.com/sonar-solutions/sq-api-go/types"
)

// ProjectsClient provides methods for the /api/projects and related endpoints.
type ProjectsClient struct{ baseClient }

// Search returns a Paginator over all projects from /api/projects/search.
func (p *ProjectsClient) Search(ctx context.Context) *sqapi.Paginator[types.Project] {
	return sqapi.NewPaginator(func(ctx context.Context, page, pageSize int) ([]types.Project, int, error) {
		params := url.Values{}
		params.Set("p", itoa(page))
		params.Set("ps", itoa(pageSize))

		var result types.ProjectsSearchResponse
		if err := p.get(ctx, "api/projects/search", params, &result); err != nil {
			return nil, 0, err
		}
		return result.Components, result.Paging.Total, nil
	}, 0)
}

// GetDetails fetches component details from /api/navigation/component.
func (p *ProjectsClient) GetDetails(ctx context.Context, component string) (*types.ComponentDetails, error) {
	params := url.Values{}
	params.Set("component", component)

	var result types.NavigationComponentResponse
	if err := p.get(ctx, "api/navigation/component", params, &result); err != nil {
		return nil, err
	}
	return &types.ComponentDetails{
		Key:        result.Key,
		Name:       result.Name,
		Qualifier:  result.Qualifier,
		Visibility: result.Visibility,
		Tags:       result.Tags,
	}, nil
}

// LicenseUsage returns all projects from /api/projects/license_usage.
func (p *ProjectsClient) LicenseUsage(ctx context.Context) ([]types.Project, error) {
	var result types.ProjectsLicenseUsageResponse
	if err := p.get(ctx, "api/projects/license_usage", nil, &result); err != nil {
		return nil, err
	}
	return result.Projects, nil
}

// Tags returns the tags for a single component from /api/components/show.
func (p *ProjectsClient) Tags(ctx context.Context, component string) ([]string, error) {
	params := url.Values{}
	params.Set("component", component)

	var result types.ComponentShowResponse
	if err := p.get(ctx, "api/components/show", params, &result); err != nil {
		return nil, err
	}
	return result.Component.Tags, nil
}
