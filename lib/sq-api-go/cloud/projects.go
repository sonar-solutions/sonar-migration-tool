package cloud

import (
	"context"
	"net/url"

	"github.com/sonar-solutions/sq-api-go/types"
)

// ProjectsClient provides write-path methods for SonarQube Cloud projects.
type ProjectsClient struct{ baseClient }

// CreateProjectParams holds the parameters for creating a Cloud project.
type CreateProjectParams struct {
	// ProjectKey is the project key on Cloud. For org-scoped projects this is
	// typically "<org>_<serverKey>".
	ProjectKey   string
	Name         string
	Organization string
	Visibility   string
	// NewCodeDefinitionType and NewCodeDefinitionValue are optional new-code period settings.
	NewCodeDefinitionType  string
	NewCodeDefinitionValue string
}

// Create creates a new project via /api/projects/create and returns its details.
func (p *ProjectsClient) Create(ctx context.Context, params CreateProjectParams) (*types.Project, error) {
	form := url.Values{}
	form.Set("project", params.ProjectKey)
	form.Set("name", params.Name)
	form.Set("organization", params.Organization)
	if params.Visibility != "" {
		form.Set("visibility", params.Visibility)
	}
	if params.NewCodeDefinitionType != "" {
		form.Set("newCodeDefinitionType", params.NewCodeDefinitionType)
	}
	if params.NewCodeDefinitionValue != "" {
		form.Set("newCodeDefinitionValue", params.NewCodeDefinitionValue)
	}

	var result types.ProjectCreateResponse
	if err := p.postForm(ctx, "api/projects/create", form, &result); err != nil {
		return nil, err
	}
	return &result.Project, nil
}

// Delete deletes a project via /api/projects/delete.
func (p *ProjectsClient) Delete(ctx context.Context, projectKey string) error {
	form := url.Values{}
	form.Set("project", projectKey)
	return p.postForm(ctx, "api/projects/delete", form, nil)
}

// SetTags sets the tags on a project via /api/project_tags/set.
// tags is a comma-separated string (e.g. "java,backend").
func (p *ProjectsClient) SetTags(ctx context.Context, projectKey, tags string) error {
	form := url.Values{}
	form.Set("project", projectKey)
	form.Set("tags", tags)
	return p.postForm(ctx, "api/project_tags/set", form, nil)
}
