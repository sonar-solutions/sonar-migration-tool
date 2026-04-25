package types

// Webhook represents a single webhook returned by /api/webhooks/list.
type Webhook struct {
	Key    string `json:"key"`
	Name   string `json:"name"`
	URL    string `json:"url"`
	Secret string `json:"secret"`
}

// WebhooksListResponse is the response envelope for /api/webhooks/list.
type WebhooksListResponse struct {
	Webhooks []Webhook `json:"webhooks"`
}
