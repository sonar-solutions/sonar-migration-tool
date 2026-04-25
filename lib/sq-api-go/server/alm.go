package server

import (
	"context"
	"net/url"

	"github.com/sonar-solutions/sq-api-go/types"
)

// AlmClient provides methods for /api/alm_settings endpoints.
type AlmClient struct{ baseClient }

// ListSettings returns all ALM configurations from /api/alm_settings/list.
func (a *AlmClient) ListSettings(ctx context.Context) ([]types.AlmSetting, error) {
	var result types.AlmSettingsResponse
	if err := a.get(ctx, "api/alm_settings/list", nil, &result); err != nil {
		return nil, err
	}
	return result.AlmSettings, nil
}

// GetBinding returns the ALM binding for a project from
// /api/alm_settings/get_binding.
func (a *AlmClient) GetBinding(ctx context.Context, projectKey string) (*types.AlmBinding, error) {
	params := url.Values{}
	params.Set("project", projectKey)

	var result types.AlmBinding
	if err := a.get(ctx, "api/alm_settings/get_binding", params, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
