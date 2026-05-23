package cloud

import (
	"context"
	"encoding/json"
	"net/url"
)

// SettingsClient provides write-path methods for SonarQube Cloud project settings.
type SettingsClient struct{ baseClient }

// Set sets a single-value project setting via /api/settings/set.
//
// The organization argument is accepted for API symmetry but only included
// in the request when projectKey is empty. SonarQube Cloud rejects requests
// that include both component and organization with HTTP 400
// "Only component or organization can be set, not both"; for project-level
// settings the cloud project key already namespaces by organization.
func (s *SettingsClient) Set(ctx context.Context, projectKey, settingKey, value, organization string) error {
	form := url.Values{}
	form.Set("key", settingKey)
	form.Set("value", value)
	addSettingsScope(form, projectKey, organization)
	return s.postForm(ctx, "api/settings/set", form, nil)
}

// SetValues sets a multi-value project setting via /api/settings/set.
// Each entry is sent as a separate "values" form parameter (the encoding
// the SonarQube/SonarQube Cloud Web API expects for multi-value settings
// such as sonar.exclusions, sonar.coverage.exclusions, etc.).
func (s *SettingsClient) SetValues(ctx context.Context, projectKey, settingKey string, values []string, organization string) error {
	form := url.Values{}
	form.Set("key", settingKey)
	for _, v := range values {
		form.Add("values", v)
	}
	addSettingsScope(form, projectKey, organization)
	return s.postForm(ctx, "api/settings/set", form, nil)
}

// SetFieldValues sets a property-set (multi-field) project setting via
// /api/settings/set. Each entry is JSON-encoded and sent as a separate
// "fieldValues" form parameter — the encoding used by settings such as
// sonar.issue.ignore.allfile, sonar.issue.ignore.multicriteria, etc.
func (s *SettingsClient) SetFieldValues(ctx context.Context, projectKey, settingKey string, fieldValues []map[string]any, organization string) error {
	form := url.Values{}
	form.Set("key", settingKey)
	for _, fv := range fieldValues {
		encoded, err := json.Marshal(fv)
		if err != nil {
			continue
		}
		form.Add("fieldValues", string(encoded))
	}
	addSettingsScope(form, projectKey, organization)
	return s.postForm(ctx, "api/settings/set", form, nil)
}

// addSettingsScope adds exactly one of "component" or "organization" to
// the request form. SonarQube Cloud rejects calls that include both, so a
// non-empty projectKey wins (project-level settings inherit organization
// scope from the cloud project key prefix).
func addSettingsScope(form url.Values, projectKey, organization string) {
	if projectKey != "" {
		form.Set("component", projectKey)
		return
	}
	if organization != "" {
		form.Set("organization", organization)
	}
}
