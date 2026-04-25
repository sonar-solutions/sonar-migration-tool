package cloud

import (
	"context"
	"net/url"

	"github.com/sonar-solutions/sq-api-go/types"
)

// QualityProfilesClient provides write-path methods for SonarQube Cloud quality profiles.
type QualityProfilesClient struct{ baseClient }

// CreateProfileParams holds the parameters for creating a Cloud quality profile.
type CreateProfileParams struct {
	Name         string
	Language     string
	Organization string
}

// Create creates a new quality profile via /api/qualityprofiles/create.
func (q *QualityProfilesClient) Create(ctx context.Context, params CreateProfileParams) (*types.QualityProfile, error) {
	form := url.Values{}
	form.Set("name", params.Name)
	form.Set("language", params.Language)
	form.Set("organization", params.Organization)

	var result types.QualityProfileCreateResponse
	if err := q.postForm(ctx, "api/qualityprofiles/create", form, &result); err != nil {
		return nil, err
	}
	return &result.Profile, nil
}

// Restore restores a quality profile from an XML backup via /api/qualityprofiles/restore.
// organization is the Cloud org key. xmlBackup is the raw XML from Server's backup endpoint.
func (q *QualityProfilesClient) Restore(ctx context.Context, organization string, xmlBackup []byte) (*types.QualityProfile, error) {
	fields := map[string]string{"organization": organization}
	var result types.QualityProfileRestoreResponse
	if err := q.postMultipart(ctx, "api/qualityprofiles/restore", fields, "backup", "backup.xml", xmlBackup, &result); err != nil {
		return nil, err
	}
	return &result.Profile, nil
}

// Delete deletes a quality profile via /api/qualityprofiles/delete.
func (q *QualityProfilesClient) Delete(ctx context.Context, language, profileName, organization string) error {
	form := url.Values{}
	form.Set("language", language)
	form.Set("qualityProfile", profileName)
	form.Set("organization", organization)
	return q.postForm(ctx, "api/qualityprofiles/delete", form, nil)
}

// SetDefault sets a quality profile as the default for its language via
// /api/qualityprofiles/set_default.
func (q *QualityProfilesClient) SetDefault(ctx context.Context, language, profileName, organization string) error {
	form := url.Values{}
	form.Set("language", language)
	form.Set("qualityProfile", profileName)
	form.Set("organization", organization)
	return q.postForm(ctx, "api/qualityprofiles/set_default", form, nil)
}

// ChangeParent sets the parent of a quality profile via /api/qualityprofiles/change_parent.
func (q *QualityProfilesClient) ChangeParent(ctx context.Context, language, profileName, parentName, organization string) error {
	form := url.Values{}
	form.Set("language", language)
	form.Set("qualityProfile", profileName)
	form.Set("parentQualityProfile", parentName)
	form.Set("organization", organization)
	return q.postForm(ctx, "api/qualityprofiles/change_parent", form, nil)
}

// AddProject associates a quality profile with a project via
// /api/qualityprofiles/add_project.
func (q *QualityProfilesClient) AddProject(ctx context.Context, language, profileName, projectKey, organization string) error {
	form := url.Values{}
	form.Set("language", language)
	form.Set("qualityProfile", profileName)
	form.Set("project", projectKey)
	form.Set("organization", organization)
	return q.postForm(ctx, "api/qualityprofiles/add_project", form, nil)
}

// AddGroup grants a group access to a quality profile via
// /api/qualityprofiles/add_group.
func (q *QualityProfilesClient) AddGroup(ctx context.Context, language, profileName, groupName, organization string) error {
	form := url.Values{}
	form.Set("language", language)
	form.Set("qualityProfile", profileName)
	form.Set("group", groupName)
	form.Set("organization", organization)
	return q.postForm(ctx, "api/qualityprofiles/add_group", form, nil)
}
