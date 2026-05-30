package cloud

import (
	"context"
	"net/url"
)

// WebhooksClient wraps the SonarQube Cloud /api/webhooks/* endpoints.
type WebhooksClient struct{ baseClient }

// CreateWebhookParams carries the per-webhook payload for
// /api/webhooks/create. Project + Organization are both required to
// register a project-scoped webhook on SonarQube Cloud; Secret is
// optional and only sent when non-empty.
type CreateWebhookParams struct {
	Organization string
	Project      string
	Name         string
	URL          string
	Secret       string
}

// Create registers a webhook on a SonarQube Cloud project.
func (c *WebhooksClient) Create(ctx context.Context, params CreateWebhookParams) error {
	form := url.Values{}
	form.Set("organization", params.Organization)
	form.Set("project", params.Project)
	form.Set("name", params.Name)
	form.Set("url", params.URL)
	if params.Secret != "" {
		form.Set("secret", params.Secret)
	}
	return c.postForm(ctx, "api/webhooks/create", form, nil)
}
