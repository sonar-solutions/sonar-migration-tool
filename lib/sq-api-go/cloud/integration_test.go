//go:build integration

package cloud_test

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sqapi "github.com/sonar-solutions/sq-api-go"
	"github.com/sonar-solutions/sq-api-go/cloud"
)

// newTestClient creates a cloud.Client pointed at the SonarQube Cloud instance
// configured via environment variables. The test is skipped if either variable
// is unset.
//
// Run with:
//
//	SQC_URL=https://sonarcloud.io SQC_TOKEN=squ_... SQC_ORG=myorg go test -tags integration -v ./cloud/
func newTestClient(t *testing.T) (*cloud.Client, string) {
	t.Helper()
	sqcURL := os.Getenv("SQC_URL")
	sqcToken := os.Getenv("SQC_TOKEN")
	sqcOrg := os.Getenv("SQC_ORG")
	if sqcURL == "" || sqcToken == "" || sqcOrg == "" {
		t.Skip("SQC_URL, SQC_TOKEN, and SQC_ORG environment variables not set")
	}
	base := sqapi.NewCloudClient(sqcURL, sqcToken)
	return cloud.New(base), sqcOrg
}

// newAPITestClient creates a cloud.Client pointed at the SonarQube Cloud API base URL
// for the Enterprises API. Skips if SQC_API_URL is not set.
func newAPITestClient(t *testing.T) (*cloud.Client, string) {
	t.Helper()
	sqcAPIURL := os.Getenv("SQC_API_URL")
	sqcToken := os.Getenv("SQC_TOKEN")
	sqcOrg := os.Getenv("SQC_ORG")
	if sqcAPIURL == "" || sqcToken == "" || sqcOrg == "" {
		t.Skip("SQC_API_URL, SQC_TOKEN, and SQC_ORG environment variables not set")
	}
	base := sqapi.NewCloudClient(sqcAPIURL, sqcToken)
	return cloud.New(base), sqcOrg
}

// TestIntegrationEnterprisesList verifies that listing enterprises returns
// at least one enterprise with non-empty ID and key fields.
func TestIntegrationEnterprisesList(t *testing.T) {
	cc, _ := newAPITestClient(t)
	ctx := context.Background()

	enterprises, err := cc.Enterprises.List(ctx)
	require.NoError(t, err)
	t.Logf("Found %d enterprise(s)", len(enterprises))

	if len(enterprises) == 0 {
		t.Skip("no enterprises found — skipping assertions (account may not have Cloud Enterprise)")
	}

	for _, e := range enterprises {
		assert.NotEmpty(t, e.ID, "enterprise ID should not be empty")
		assert.NotEmpty(t, e.Key, "enterprise key should not be empty")
		t.Logf("Enterprise: id=%s key=%s name=%s", e.ID, e.Key, e.Name)
	}
}

// TestIntegrationProjectsCreateAndDelete creates a temporary project, verifies
// its key, then deletes it. The test key uses a prefix that avoids conflicts
// with production projects.
func TestIntegrationProjectsCreateAndDelete(t *testing.T) {
	cc, org := newTestClient(t)
	ctx := context.Background()

	projectKey := org + "_sqapigo-phase4-test"
	params := cloud.CreateProjectParams{
		ProjectKey:   projectKey,
		Name:         "sq-api-go Phase 4 integration test (delete me)",
		Organization: org,
		Visibility:   "private",
	}

	proj, err := cc.Projects.Create(ctx, params)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = cc.Projects.Delete(context.Background(), projectKey)
	})

	assert.Equal(t, projectKey, proj.Key, "created project should have the expected key")
	t.Logf("Created project: key=%s name=%s", proj.Key, proj.Name)
}

// TestIntegrationGroupsCreateAndDelete creates a temporary group and deletes it.
func TestIntegrationGroupsCreateAndDelete(t *testing.T) {
	cc, org := newTestClient(t)
	ctx := context.Background()

	params := cloud.CreateGroupParams{
		Name:         "sq-api-go-phase4-test-group",
		Description:  "Temporary group created by sq-api-go integration tests",
		Organization: org,
	}

	grp, err := cc.Groups.Create(ctx, params)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = cc.Groups.Delete(context.Background(), grp.ID)
	})

	assert.NotZero(t, grp.ID, "created group should have a non-zero ID")
	assert.Equal(t, params.Name, grp.Name, "created group should have the expected name")
	t.Logf("Created group: id=%d name=%s", grp.ID, grp.Name)
}

// TestIntegrationQualityGatesCreateAndDestroy creates a temporary quality gate
// and destroys it.
func TestIntegrationQualityGatesCreateAndDestroy(t *testing.T) {
	cc, org := newTestClient(t)
	ctx := context.Background()

	gate, err := cc.QualityGates.Create(ctx, "sq-api-go-phase4-test-gate", org)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = cc.QualityGates.Destroy(context.Background(), gate.ID, org)
	})

	assert.NotZero(t, gate.ID, "created gate should have a non-zero ID")
	assert.Equal(t, "sq-api-go-phase4-test-gate", gate.Name)
	t.Logf("Created quality gate: id=%d name=%s", gate.ID, gate.Name)
}

// TestIntegrationQualityProfilesCreateAndDelete creates a temporary quality
// profile and deletes it.
func TestIntegrationQualityProfilesCreateAndDelete(t *testing.T) {
	cc, org := newTestClient(t)
	ctx := context.Background()

	params := cloud.CreateProfileParams{
		Name:         "sq-api-go-phase4-test-profile",
		Language:     "java",
		Organization: org,
	}

	profile, err := cc.QualityProfiles.Create(ctx, params)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = cc.QualityProfiles.Delete(context.Background(), "java", params.Name, org)
	})

	assert.NotEmpty(t, profile.Key, "created profile should have a non-empty key")
	assert.Equal(t, params.Name, profile.Name)
	t.Logf("Created quality profile: key=%s name=%s", profile.Key, profile.Name)
}

// TestIntegrationPermissionsCreateAndDeleteTemplate creates a temporary
// permission template and deletes it.
func TestIntegrationPermissionsCreateAndDeleteTemplate(t *testing.T) {
	cc, org := newTestClient(t)
	ctx := context.Background()

	params := cloud.CreateTemplateParams{
		Name:         "sq-api-go-phase4-test-template",
		Description:  "Temporary template created by sq-api-go integration tests",
		Organization: org,
	}

	tmpl, err := cc.Permissions.CreateTemplate(ctx, params)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = cc.Permissions.DeleteTemplate(context.Background(), tmpl.ID)
	})

	assert.NotEmpty(t, tmpl.ID, "created template should have a non-empty ID")
	assert.Equal(t, params.Name, tmpl.Name)
	t.Logf("Created permission template: id=%s name=%s", tmpl.ID, tmpl.Name)
}
