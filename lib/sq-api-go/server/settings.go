package server

import (
	"context"
	"net/url"

	"github.com/sonar-solutions/sq-api-go/types"
)

// SettingsClient provides methods for /api/settings endpoints.
type SettingsClient struct{ baseClient }

// Values returns settings values from /api/settings/values.
// component is optional (pass empty string for global settings).
// keys is an optional comma-separated list of setting keys to filter.
func (s *SettingsClient) Values(ctx context.Context, component string, keys string) ([]types.Setting, error) {
	params := url.Values{}
	if component != "" {
		params.Set("component", component)
	}
	if keys != "" {
		params.Set("keys", keys)
	}

	var result types.SettingsValuesResponse
	if err := s.get(ctx, "api/settings/values", params, &result); err != nil {
		return nil, err
	}
	return result.Settings, nil
}
