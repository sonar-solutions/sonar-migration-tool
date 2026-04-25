package cloud

import (
	"context"
	"net/url"
)

// BranchesClient provides write-path methods for SonarQube Cloud project branches.
type BranchesClient struct{ baseClient }

// Rename renames the main branch of a project via /api/project_branches/rename.
func (b *BranchesClient) Rename(ctx context.Context, projectKey, name string) error {
	form := url.Values{}
	form.Set("project", projectKey)
	form.Set("name", name)
	return b.postForm(ctx, "api/project_branches/rename", form, nil)
}
