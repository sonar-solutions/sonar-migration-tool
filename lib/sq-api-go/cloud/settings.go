package cloud

import (
	"context"
	"net/url"
	"strings"
)

// SettingsClient provides write-path methods for SonarQube Cloud project settings.
type SettingsClient struct{ baseClient }

// Set sets a single-value project setting via /api/settings/set.
func (s *SettingsClient) Set(ctx context.Context, projectKey, settingKey, value string) error {
	form := url.Values{}
	form.Set("component", projectKey)
	form.Set("key", settingKey)
	form.Set("value", value)
	return s.postForm(ctx, "api/settings/set", form, nil)
}

// SetValues sets a multi-value project setting via /api/settings/set.
// values are sent as repeated "values" form parameters.
func (s *SettingsClient) SetValues(ctx context.Context, projectKey, settingKey string, values []string) error {
	form := url.Values{}
	form.Set("component", projectKey)
	form.Set("key", settingKey)
	// SonarQube accepts multi-value settings as a comma-separated string
	// in the "value" param, or as repeated "values[]" params. The Python
	// implementation joins them; we do the same for consistency.
	form.Set("value", strings.Join(values, ","))
	return s.postForm(ctx, "api/settings/set", form, nil)
}
