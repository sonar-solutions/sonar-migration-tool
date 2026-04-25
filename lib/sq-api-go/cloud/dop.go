package cloud

import (
	"context"
)

// DOPClient provides methods for the DOP Translation API used to bind
// Cloud projects to DevOps platform repositories.
type DOPClient struct{ baseClient }

// ProjectBindingParams holds the parameters for creating a DOP project binding.
type ProjectBindingParams struct {
	// ProjectID is the internal Cloud project ID (numeric).
	ProjectID string
	// RepositoryID is the DevOps platform repository identifier (integration key).
	RepositoryID string
}

// CreateProjectBinding creates a DOP project binding via
// POST /dop-translation/project-bindings.
func (d *DOPClient) CreateProjectBinding(ctx context.Context, params ProjectBindingParams) error {
	body := map[string]string{
		"projectId":    params.ProjectID,
		"repositoryId": params.RepositoryID,
	}
	return d.postJSON(ctx, "dop-translation/project-bindings", body, nil)
}
