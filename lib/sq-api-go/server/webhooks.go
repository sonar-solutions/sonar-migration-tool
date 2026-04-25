package server

import (
	"context"

	"github.com/sonar-solutions/sq-api-go/types"
)

// WebhooksClient provides methods for /api/webhooks endpoints.
type WebhooksClient struct{ baseClient }

// List returns all global webhooks from /api/webhooks/list.
func (w *WebhooksClient) List(ctx context.Context) ([]types.Webhook, error) {
	var result types.WebhooksListResponse
	if err := w.get(ctx, "api/webhooks/list", nil, &result); err != nil {
		return nil, err
	}
	return result.Webhooks, nil
}
