//go:build integration

package server_test

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sqapi "github.com/sonar-solutions/sq-api-go"
	"github.com/sonar-solutions/sq-api-go/server"
)

const skipNoProjectsPriv = "token lacks privilege for /api/projects/search"

// newTestClient creates a server.Client pointed at the SQS instance configured
// via environment variables. The test is skipped if either variable is unset.
//
// Run with:
//
//	SQS_URL=https://... SQS_TOKEN=squ_... go test -tags integration -v ./server/
func newTestClient(t *testing.T) *server.Client {
	t.Helper()
	sqsURL := os.Getenv("SQS_URL")
	sqsToken := os.Getenv("SQS_TOKEN")
	if sqsURL == "" || sqsToken == "" {
		t.Skip("SQS_URL and SQS_TOKEN environment variables not set")
	}
	// Use version 10.7 as an initial estimate; System.Version() will confirm.
	base := sqapi.NewServerClient(sqsURL, sqsToken, 10.7)
	return server.New(base)
}

func TestIntegrationSystemVersion(t *testing.T) {
	sc := newTestClient(t)
	ctx := context.Background()

	version, err := sc.System.Version(ctx)
	require.NoError(t, err)
	assert.Greater(t, version, float64(0), "version should be positive")
	t.Logf("Server version: %v", version)
}

func TestIntegrationSystemInfo(t *testing.T) {
	sc := newTestClient(t)
	ctx := context.Background()

	info, err := sc.System.Info(ctx)
	if sqapi.IsForbidden(err) {
		t.Skip("token lacks admin privilege for /api/system/info")
	}
	require.NoError(t, err)
	assert.NotEmpty(t, info, "system info should not be empty")
	t.Logf("System info keys: %d", len(info))
}

func TestIntegrationProjectsSearch(t *testing.T) {
	sc := newTestClient(t)
	ctx := context.Background()

	projects, err := sc.Projects.Search(ctx).All(ctx)
	if sqapi.IsForbidden(err) {
		t.Skip(skipNoProjectsPriv)
	}
	require.NoError(t, err)
	t.Logf("Projects found: %d", len(projects))
	for _, p := range projects {
		assert.NotEmpty(t, p.Key, "project key should not be empty")
		assert.NotEmpty(t, p.Name, "project name should not be empty")
	}
}

func TestIntegrationUsersSearch(t *testing.T) {
	sc := newTestClient(t)
	ctx := context.Background()

	users, err := sc.Users.Search(ctx).All(ctx)
	if sqapi.IsForbidden(err) {
		t.Skip("token lacks privilege for /api/users/search")
	}
	require.NoError(t, err)
	assert.NotEmpty(t, users, "expected at least one user")
	for _, u := range users {
		assert.NotEmpty(t, u.Login, "user login should not be empty")
	}
}

func TestIntegrationGroupsSearch(t *testing.T) {
	sc := newTestClient(t)
	ctx := context.Background()

	groups, err := sc.Groups.Search(ctx).All(ctx)
	if sqapi.IsForbidden(err) {
		t.Skip("token lacks admin privilege for /api/permissions/groups")
	}
	require.NoError(t, err)
	t.Logf("Groups found: %d", len(groups))
	for _, g := range groups {
		assert.NotEmpty(t, g.Name, "group name should not be empty")
	}
}

func TestIntegrationRulesRepositories(t *testing.T) {
	sc := newTestClient(t)
	ctx := context.Background()

	repos, err := sc.Rules.Repositories(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, repos, "expected at least one rule repository")
	t.Logf("Repositories found: %d", len(repos))
	assert.NotEmpty(t, repos[0].Key, "repository key should not be empty")
}

func TestIntegrationRulesSearch(t *testing.T) {
	sc := newTestClient(t)
	ctx := context.Background()

	rules, err := sc.Rules.Search(ctx).All(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, rules, "expected at least one rule")
	t.Logf("Rules found: %d", len(rules))
	assert.NotEmpty(t, rules[0].Key, "rule key should not be empty")
}

func TestIntegrationQualityGatesList(t *testing.T) {
	sc := newTestClient(t)
	ctx := context.Background()

	gates, err := sc.QualityGates.List(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, gates, "expected at least the built-in quality gate")
	t.Logf("Quality gates found: %d", len(gates))
	for _, g := range gates {
		assert.NotEmpty(t, g.Name, "gate name should not be empty")
	}
}

func TestIntegrationQualityProfilesSearch(t *testing.T) {
	sc := newTestClient(t)
	ctx := context.Background()

	profiles, err := sc.QualityProfiles.Search(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, profiles, "expected at least the built-in quality profile")
	t.Logf("Quality profiles found: %d", len(profiles))
	for _, p := range profiles {
		assert.NotEmpty(t, p.Key, "profile key should not be empty")
		assert.NotEmpty(t, p.Language, "profile language should not be empty")
	}
}

func TestIntegrationPermissionsSearchTemplates(t *testing.T) {
	sc := newTestClient(t)
	ctx := context.Background()

	templates, err := sc.Permissions.SearchTemplates(ctx)
	if sqapi.IsForbidden(err) {
		t.Skip("token lacks admin privilege for /api/permissions/search_templates")
	}
	require.NoError(t, err)
	t.Logf("Permission templates found: %d", len(templates))
	for _, tmpl := range templates {
		assert.NotEmpty(t, tmpl.Name, "template name should not be empty")
	}
}

func TestIntegrationBranchesList(t *testing.T) {
	sc := newTestClient(t)
	ctx := context.Background()

	projects, err := sc.Projects.Search(ctx).All(ctx)
	if sqapi.IsForbidden(err) {
		t.Skip(skipNoProjectsPriv)
	}
	require.NoError(t, err)
	if len(projects) == 0 {
		t.Skip("no projects available to test branch listing")
	}

	branches, err := sc.Branches.List(ctx, projects[0].Key)
	if sqapi.IsForbidden(err) {
		t.Skip("token lacks privilege for /api/project_branches/list")
	}
	require.NoError(t, err)
	t.Logf("Branches for %s: %d", projects[0].Key, len(branches))
	for _, b := range branches {
		assert.NotEmpty(t, b.Name, "branch name should not be empty")
	}
}

func TestIntegrationAnalysesSearch(t *testing.T) {
	sc := newTestClient(t)
	ctx := context.Background()

	projects, err := sc.Projects.Search(ctx).All(ctx)
	if sqapi.IsForbidden(err) {
		t.Skip(skipNoProjectsPriv)
	}
	require.NoError(t, err)
	if len(projects) == 0 {
		t.Skip("no projects available to test analyses")
	}

	analyses, err := sc.Analyses.Search(ctx, projects[0].Key).All(ctx)
	if sqapi.IsForbidden(err) {
		t.Skip("token lacks privilege for /api/project_analyses/search")
	}
	require.NoError(t, err)
	t.Logf("Analyses for %s: %d", projects[0].Key, len(analyses))
}

func TestIntegrationIssuesSearch(t *testing.T) {
	sc := newTestClient(t)
	ctx := context.Background()

	projects, err := sc.Projects.Search(ctx).All(ctx)
	if sqapi.IsForbidden(err) {
		t.Skip(skipNoProjectsPriv)
	}
	require.NoError(t, err)
	if len(projects) == 0 {
		t.Skip("no projects available to test issue search")
	}

	issues, err := sc.Issues.Search(ctx, projects[0].Key).All(ctx)
	if sqapi.IsForbidden(err) {
		t.Skip("token lacks privilege for /api/issues/search")
	}
	require.NoError(t, err)
	t.Logf("Issues for %s: %d", projects[0].Key, len(issues))
}

func TestIntegrationHotspotsSearch(t *testing.T) {
	sc := newTestClient(t)
	ctx := context.Background()

	projects, err := sc.Projects.Search(ctx).All(ctx)
	if sqapi.IsForbidden(err) {
		t.Skip(skipNoProjectsPriv)
	}
	require.NoError(t, err)
	if len(projects) == 0 {
		t.Skip("no projects available to test hotspot search")
	}

	hotspots, err := sc.Hotspots.Search(ctx, projects[0].Key).All(ctx)
	if sqapi.IsForbidden(err) {
		t.Skip("token lacks privilege for /api/hotspots/search")
	}
	require.NoError(t, err)
	t.Logf("Hotspots for %s: %d", projects[0].Key, len(hotspots))
}

func TestIntegrationMeasuresSearch(t *testing.T) {
	sc := newTestClient(t)
	ctx := context.Background()

	projects, err := sc.Projects.Search(ctx).All(ctx)
	if sqapi.IsForbidden(err) {
		t.Skip(skipNoProjectsPriv)
	}
	require.NoError(t, err)
	if len(projects) == 0 {
		t.Skip("no projects available to test measures")
	}

	metricKeys := []string{"ncloc_language_distribution", "coverage", "violations"}
	measures, err := sc.Measures.Search(ctx, []string{projects[0].Key}, metricKeys)
	if sqapi.IsForbidden(err) {
		t.Skip("token lacks privilege for /api/measures/search")
	}
	require.NoError(t, err)
	t.Logf("Measures for %s: %d", projects[0].Key, len(measures))
}

func TestIntegrationSettingsValues(t *testing.T) {
	sc := newTestClient(t)
	ctx := context.Background()

	settings, err := sc.Settings.Values(ctx, "", "")
	if sqapi.IsForbidden(err) {
		t.Skip("token lacks privilege for /api/settings/values")
	}
	require.NoError(t, err)
	t.Logf("Global settings found: %d", len(settings))
}

func TestIntegrationPluginsInstalled(t *testing.T) {
	sc := newTestClient(t)
	ctx := context.Background()

	plugins, err := sc.Plugins.Installed(ctx)
	if sqapi.IsForbidden(err) {
		t.Skip("token lacks privilege for /api/plugins/installed")
	}
	require.NoError(t, err)
	t.Logf("Installed plugins: %d", len(plugins))
	for _, p := range plugins {
		assert.NotEmpty(t, p.Key, "plugin key should not be empty")
	}
}

func TestIntegrationViewsSearchEnterpriseOnly(t *testing.T) {
	sc := newTestClient(t)
	ctx := context.Background()

	views, err := sc.Views.Search(ctx, "").All(ctx)
	if sqapi.IsNotFound(err) || sqapi.IsForbidden(err) {
		t.Skip("views endpoint not available on this edition or token lacks privilege")
	}
	require.NoError(t, err)
	t.Logf("Views found: %d", len(views))
}

func TestIntegrationWebhooksList(t *testing.T) {
	sc := newTestClient(t)
	ctx := context.Background()

	webhooks, err := sc.Webhooks.List(ctx)
	if sqapi.IsForbidden(err) {
		t.Skip("token lacks admin privilege for /api/webhooks/list")
	}
	require.NoError(t, err)
	t.Logf("Global webhooks found: %d", len(webhooks))
}

func TestIntegrationALMListSettings(t *testing.T) {
	sc := newTestClient(t)
	ctx := context.Background()

	settings, err := sc.ALM.ListSettings(ctx)
	if sqapi.IsForbidden(err) {
		t.Skip("token lacks admin privilege for /api/alm_settings/list")
	}
	require.NoError(t, err)
	t.Logf("ALM settings found: %d", len(settings))
}

func TestIntegrationNewCodeList(t *testing.T) {
	sc := newTestClient(t)
	ctx := context.Background()

	projects, err := sc.Projects.Search(ctx).All(ctx)
	if sqapi.IsForbidden(err) {
		t.Skip(skipNoProjectsPriv)
	}
	require.NoError(t, err)
	if len(projects) == 0 {
		t.Skip("no projects available to test new code periods")
	}

	periods, err := sc.NewCode.List(ctx, projects[0].Key)
	if sqapi.IsForbidden(err) {
		t.Skip("token lacks privilege for /api/new_code_periods/list")
	}
	require.NoError(t, err)
	t.Logf("New code periods for %s: %d", projects[0].Key, len(periods))
}

func TestIntegrationTokensSearch(t *testing.T) {
	sc := newTestClient(t)
	ctx := context.Background()

	users, err := sc.Users.Search(ctx).All(ctx)
	if sqapi.IsForbidden(err) {
		t.Skip("token lacks privilege for /api/users/search")
	}
	require.NoError(t, err)
	if len(users) == 0 {
		t.Skip("no users available to test token search")
	}

	tokens, err := sc.Tokens.Search(ctx, users[0].Login)
	if sqapi.IsForbidden(err) {
		t.Skip("token lacks admin privilege for /api/user_tokens/search")
	}
	require.NoError(t, err)
	t.Logf("Tokens for %s: %d", users[0].Login, len(tokens))
}
