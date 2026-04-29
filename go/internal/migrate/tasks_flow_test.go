package migrate

import (
	"context"
	"encoding/json"
	"testing"
)

// setupCreateOutputs populates the Store with mock createX outputs
// that downstream tasks depend on.
func setupCreateOutputs(t *testing.T, e *Executor) {
	t.Helper()
	writeItem := func(task string, data map[string]any) {
		w, _ := e.Store.Writer(task)
		b, _ := json.Marshal(data)
		w.WriteOne(b)
	}

	writeItem("createProjects", map[string]any{
		"key": "proj1", "name": "Project 1", "server_url": testServerURL,
		"cloud_project_key": "cloud-org1_proj1", "sonarcloud_org_key": testCloudOrg,
		"gate_name": testCustomGate,
		"profiles":  []map[string]any{{"key": "prof1", "name": "Custom", "language": "java"}},
	})

	writeItem("createProfiles", map[string]any{
		"name": "Custom", "language": "java", "parent_name": "Sonar way",
		"source_profile_key": "prof1", "sonarcloud_org_key": testCloudOrg,
		"cloud_profile_key": "cloud-prof-1", "is_default": true,
	})

	writeItem("createGates", map[string]any{
		"name": testCustomGate, "source_gate_key": testCustomGate,
		"sonarcloud_org_key": testCloudOrg, "cloud_gate_id": "42", "is_default": true,
		"server_url": testServerURL,
	})

	writeItem("createGroups", map[string]any{
		"name": "sonar-users", "sonarcloud_org_key": testCloudOrg, "cloud_group_id": "101",
	})

	writeItem("createPermissionTemplates", map[string]any{
		"name": "Default", "sonarcloud_org_key": testCloudOrg,
		"cloud_template_id": "tpl-cloud-1", "is_default": true,
	})

	writeItem("generateOrganizationMappings", map[string]any{
		"sonarqube_org_key": "org1", "sonarcloud_org_key": testCloudOrg,
		"server_url": testServerURL,
	})

	writeItem("createMigrationGroups", map[string]any{
		"sonarcloud_org_key": testCloudOrg,
		"groups":             []string{"migration-scanners", "migration-viewers"},
	})

	writeItem("getMigrationUser", map[string]any{
		"login": "migration-user", "name": "Migration User",
	})

	writeItem("generateProjectMappings", map[string]any{
		"key": "proj1", "name": "Project 1", "server_url": testServerURL,
		"sonarcloud_org_key": testCloudOrg, "alm": "github",
		"repository": "myorg/myrepo", "is_cloud_binding": true,
	})
}

func TestSetProfileParent(t *testing.T) {
	cloudSrv := newMockCloudServer()
	defer cloudSrv.Close()
	apiSrv := newMockAPIServer()
	defer apiSrv.Close()
	dir := t.TempDir()
	setupExtractData(dir)
	e := newTestExecutor(cloudSrv, apiSrv, dir)
	setupCreateOutputs(t, e)

	reg := BuildMigrateRegistry(RegisterAll())
	err := reg["setProfileParent"].Run(context.Background(), e)
	if err != nil {
		t.Fatalf("setProfileParent: %v", err)
	}
}

func TestSetDefaultProfiles(t *testing.T) {
	cloudSrv := newMockCloudServer()
	defer cloudSrv.Close()
	apiSrv := newMockAPIServer()
	defer apiSrv.Close()
	dir := t.TempDir()
	setupExtractData(dir)
	e := newTestExecutor(cloudSrv, apiSrv, dir)
	setupCreateOutputs(t, e)

	// restoreProfiles is a dependency — stub it.
	w, _ := e.Store.Writer("restoreProfiles")
	w.WriteChunk(nil)

	reg := BuildMigrateRegistry(RegisterAll())
	err := reg["setDefaultProfiles"].Run(context.Background(), e)
	if err != nil {
		t.Fatalf("setDefaultProfiles: %v", err)
	}
}

func TestSetDefaultGates(t *testing.T) {
	cloudSrv := newMockCloudServer()
	defer cloudSrv.Close()
	apiSrv := newMockAPIServer()
	defer apiSrv.Close()
	dir := t.TempDir()
	setupExtractData(dir)
	e := newTestExecutor(cloudSrv, apiSrv, dir)
	setupCreateOutputs(t, e)

	// addGateConditions dependency.
	w, _ := e.Store.Writer("addGateConditions")
	w.WriteChunk(nil)

	reg := BuildMigrateRegistry(RegisterAll())
	err := reg["setDefaultGates"].Run(context.Background(), e)
	if err != nil {
		t.Fatalf("setDefaultGates: %v", err)
	}
}

func TestSetDefaultTemplates(t *testing.T) {
	cloudSrv := newMockCloudServer()
	defer cloudSrv.Close()
	apiSrv := newMockAPIServer()
	defer apiSrv.Close()
	dir := t.TempDir()
	setupExtractData(dir)
	e := newTestExecutor(cloudSrv, apiSrv, dir)
	setupCreateOutputs(t, e)

	reg := BuildMigrateRegistry(RegisterAll())
	err := reg["setDefaultTemplates"].Run(context.Background(), e)
	if err != nil {
		t.Fatalf("setDefaultTemplates: %v", err)
	}
}

func TestAddGateConditions(t *testing.T) {
	cloudSrv := newMockCloudServer()
	defer cloudSrv.Close()
	apiSrv := newMockAPIServer()
	defer apiSrv.Close()
	dir := t.TempDir()
	setupExtractData(dir)
	e := newTestExecutor(cloudSrv, apiSrv, dir)
	setupCreateOutputs(t, e)

	// getGateConditions dependency — write mock data with conditions.
	w, _ := e.Store.Writer("getGateConditions")
	b, _ := json.Marshal(map[string]any{"sonarcloud_org_key": testCloudOrg, "cloud_gate_id": "42", "conditions": []map[string]any{{"metric": "coverage", "op": "LT", "error": "80"}}})
	w.WriteOne(b)

	reg := BuildMigrateRegistry(RegisterAll())
	err := reg["addGateConditions"].Run(context.Background(), e)
	if err != nil {
		t.Fatalf("addGateConditions: %v", err)
	}
}

func TestRestoreProfiles(t *testing.T) {
	cloudSrv := newMockCloudServer()
	defer cloudSrv.Close()
	apiSrv := newMockAPIServer()
	defer apiSrv.Close()
	dir := t.TempDir()
	setupExtractData(dir)
	e := newTestExecutor(cloudSrv, apiSrv, dir)
	setupCreateOutputs(t, e)

	// setProfileParent dependency.
	w, _ := e.Store.Writer("setProfileParent")
	w.WriteChunk(nil)

	// getProfileBackups dependency.
	w2, _ := e.Store.Writer("getProfileBackups")
	b2, _ := json.Marshal(map[string]any{"profileKey": "prof1", "sonarcloud_org_key": testCloudOrg, "backup": "<profile><name>Custom</name></profile>"})
	w2.WriteOne(b2)

	reg := BuildMigrateRegistry(RegisterAll())
	err := reg["restoreProfiles"].Run(context.Background(), e)
	if err != nil {
		t.Fatalf("restoreProfiles: %v", err)
	}
}

func TestSetProjectProfiles(t *testing.T) {
	cloudSrv := newMockCloudServer()
	defer cloudSrv.Close()
	apiSrv := newMockAPIServer()
	defer apiSrv.Close()
	dir := t.TempDir()
	setupExtractData(dir)
	e := newTestExecutor(cloudSrv, apiSrv, dir)
	setupCreateOutputs(t, e)

	reg := BuildMigrateRegistry(RegisterAll())
	err := reg["setProjectProfiles"].Run(context.Background(), e)
	if err != nil {
		t.Fatalf("setProjectProfiles: %v", err)
	}
}

func TestSetProjectGates(t *testing.T) {
	cloudSrv := newMockCloudServer()
	defer cloudSrv.Close()
	apiSrv := newMockAPIServer()
	defer apiSrv.Close()
	dir := t.TempDir()
	setupExtractData(dir)
	e := newTestExecutor(cloudSrv, apiSrv, dir)
	setupCreateOutputs(t, e)

	reg := BuildMigrateRegistry(RegisterAll())
	err := reg["setProjectGates"].Run(context.Background(), e)
	if err != nil {
		t.Fatalf("setProjectGates: %v", err)
	}
}

func TestSetProjectGroupPermissions(t *testing.T) {
	cloudSrv := newMockCloudServer()
	defer cloudSrv.Close()
	apiSrv := newMockAPIServer()
	defer apiSrv.Close()
	dir := t.TempDir()
	setupExtractData(dir)
	e := newTestExecutor(cloudSrv, apiSrv, dir)
	setupCreateOutputs(t, e)

	reg := BuildMigrateRegistry(RegisterAll())
	err := reg["setProjectGroupPermissions"].Run(context.Background(), e)
	if err != nil {
		t.Fatalf("setProjectGroupPermissions: %v", err)
	}
}

func TestSetProjectSettings(t *testing.T) {
	cloudSrv := newMockCloudServer()
	defer cloudSrv.Close()
	apiSrv := newMockAPIServer()
	defer apiSrv.Close()
	dir := t.TempDir()
	setupExtractData(dir)
	e := newTestExecutor(cloudSrv, apiSrv, dir)
	setupCreateOutputs(t, e)

	reg := BuildMigrateRegistry(RegisterAll())
	err := reg["setProjectSettings"].Run(context.Background(), e)
	if err != nil {
		t.Fatalf("setProjectSettings: %v", err)
	}
}

func TestSetProjectTags(t *testing.T) {
	cloudSrv := newMockCloudServer()
	defer cloudSrv.Close()
	apiSrv := newMockAPIServer()
	defer apiSrv.Close()
	dir := t.TempDir()
	setupExtractData(dir)
	e := newTestExecutor(cloudSrv, apiSrv, dir)
	setupCreateOutputs(t, e)

	reg := BuildMigrateRegistry(RegisterAll())
	err := reg["setProjectTags"].Run(context.Background(), e)
	if err != nil {
		t.Fatalf("setProjectTags: %v", err)
	}
}

func TestCreateMigrationGroups(t *testing.T) {
	cloudSrv := newMockCloudServer()
	defer cloudSrv.Close()
	apiSrv := newMockAPIServer()
	defer apiSrv.Close()
	dir := t.TempDir()
	setupExtractData(dir)
	e := newTestExecutor(cloudSrv, apiSrv, dir)
	setupCreateOutputs(t, e)

	reg := BuildMigrateRegistry(RegisterAll())
	err := reg["createMigrationGroups"].Run(context.Background(), e)
	if err != nil {
		t.Fatalf("createMigrationGroups: %v", err)
	}
}

func TestAddMigrationUserToGroups(t *testing.T) {
	cloudSrv := newMockCloudServer()
	defer cloudSrv.Close()
	apiSrv := newMockAPIServer()
	defer apiSrv.Close()
	dir := t.TempDir()
	setupExtractData(dir)
	e := newTestExecutor(cloudSrv, apiSrv, dir)
	setupCreateOutputs(t, e)

	reg := BuildMigrateRegistry(RegisterAll())
	err := reg["addMigrationUserToMigrationGroups"].Run(context.Background(), e)
	if err != nil {
		t.Fatalf("addMigrationUserToMigrationGroups: %v", err)
	}
}

func TestAddMigrationGroupToTemplates(t *testing.T) {
	cloudSrv := newMockCloudServer()
	defer cloudSrv.Close()
	apiSrv := newMockAPIServer()
	defer apiSrv.Close()
	dir := t.TempDir()
	setupExtractData(dir)
	e := newTestExecutor(cloudSrv, apiSrv, dir)
	setupCreateOutputs(t, e)

	reg := BuildMigrateRegistry(RegisterAll())
	err := reg["addMigrationGroupToTemplates"].Run(context.Background(), e)
	if err != nil {
		t.Fatalf("addMigrationGroupToTemplates: %v", err)
	}
}

func TestSetOrgGroupPermissions(t *testing.T) {
	cloudSrv := newMockCloudServer()
	defer cloudSrv.Close()
	apiSrv := newMockAPIServer()
	defer apiSrv.Close()
	dir := t.TempDir()
	setupExtractData(dir)
	e := newTestExecutor(cloudSrv, apiSrv, dir)
	setupCreateOutputs(t, e)

	reg := BuildMigrateRegistry(RegisterAll())
	err := reg["setOrgGroupPermissions"].Run(context.Background(), e)
	if err != nil {
		t.Fatalf("setOrgGroupPermissions: %v", err)
	}
}

func TestSetProfileGroupPermissions(t *testing.T) {
	cloudSrv := newMockCloudServer()
	defer cloudSrv.Close()
	apiSrv := newMockAPIServer()
	defer apiSrv.Close()
	dir := t.TempDir()
	setupExtractData(dir)
	e := newTestExecutor(cloudSrv, apiSrv, dir)
	setupCreateOutputs(t, e)

	reg := BuildMigrateRegistry(RegisterAll())
	err := reg["setProfileGroupPermissions"].Run(context.Background(), e)
	if err != nil {
		t.Fatalf("setProfileGroupPermissions: %v", err)
	}
}

func TestUpdateRuleTags(t *testing.T) {
	cloudSrv := newMockCloudServer()
	defer cloudSrv.Close()
	apiSrv := newMockAPIServer()
	defer apiSrv.Close()
	dir := t.TempDir()
	setupExtractData(dir)
	e := newTestExecutor(cloudSrv, apiSrv, dir)
	setupCreateOutputs(t, e)

	reg := BuildMigrateRegistry(RegisterAll())
	err := reg["updateRuleTags"].Run(context.Background(), e)
	if err != nil {
		t.Fatalf("updateRuleTags: %v", err)
	}

	items, _ := e.Store.ReadAll("updateRuleTags")
	if len(items) == 0 {
		t.Error("expected updateRuleTags output")
	}
}

func TestUpdateRuleDescriptions(t *testing.T) {
	cloudSrv := newMockCloudServer()
	defer cloudSrv.Close()
	apiSrv := newMockAPIServer()
	defer apiSrv.Close()
	dir := t.TempDir()
	setupExtractData(dir)
	e := newTestExecutor(cloudSrv, apiSrv, dir)
	setupCreateOutputs(t, e)

	reg := BuildMigrateRegistry(RegisterAll())
	err := reg["updateRuleDescriptions"].Run(context.Background(), e)
	if err != nil {
		t.Fatalf("updateRuleDescriptions: %v", err)
	}

	items, _ := e.Store.ReadAll("updateRuleDescriptions")
	if len(items) == 0 {
		t.Error("expected updateRuleDescriptions output")
	}
}

func TestDeleteTasks(t *testing.T) {
	cloudSrv := newMockCloudServer()
	defer cloudSrv.Close()
	apiSrv := newMockAPIServer()
	defer apiSrv.Close()
	dir := t.TempDir()
	setupExtractData(dir)
	e := newTestExecutor(cloudSrv, apiSrv, dir)
	setupCreateOutputs(t, e)

	// getCreatedProjects dependency for deleteProjects.
	w, _ := e.Store.Writer("getCreatedProjects")
	w.WriteOne(json.RawMessage(`{"key":"cloud-org1_proj1","sonarcloud_org_key":"cloud-org1"}`))

	// setDefaultProfiles/Gates/Templates dependencies for reset tasks.
	for _, task := range []string{"setDefaultProfiles", "setDefaultGates", "setDefaultTemplates"} {
		wt, _ := e.Store.Writer(task)
		wt.WriteChunk(nil)
	}

	reg := BuildMigrateRegistry(RegisterAll())

	deleteTasks := []string{
		"deleteProjects", "deleteProfiles", "deleteGates", "deleteGroups",
		"deleteTemplates", "resetDefaultProfiles", "resetDefaultGates", "resetPermissionTemplates",
	}
	for _, name := range deleteTasks {
		t.Run(name, func(t *testing.T) {
			def, ok := reg[name]
			if !ok {
				t.Skipf("task %q not in registry (may be edition-filtered)", name)
			}
			err := def.Run(context.Background(), e)
			if err != nil {
				t.Fatalf("%s: %v", name, err)
			}
		})
	}
}

func TestDeletePortfolios(t *testing.T) {
	cloudSrv := newMockCloudServer()
	defer cloudSrv.Close()
	apiSrv := newMockAPIServer()
	defer apiSrv.Close()
	dir := t.TempDir()
	setupExtractData(dir)
	e := newTestExecutor(cloudSrv, apiSrv, dir)

	// createPortfolios dependency.
	w, _ := e.Store.Writer("createPortfolios")
	w.WriteOne(json.RawMessage(`{"cloud_portfolio_id":"portfolio-1","name":"Test"}`))

	reg := BuildMigrateRegistry(RegisterAll())
	err := reg["deletePortfolios"].Run(context.Background(), e)
	if err != nil {
		t.Fatalf("deletePortfolios: %v", err)
	}
}

func TestMatchProjectReposAndBind(t *testing.T) {
	cloudSrv := newMockCloudServer()
	defer cloudSrv.Close()
	apiSrv := newMockAPIServer()
	defer apiSrv.Close()
	dir := t.TempDir()
	setupExtractData(dir)
	e := newTestExecutor(cloudSrv, apiSrv, dir)
	setupCreateOutputs(t, e)

	// getProjectIds dependency.
	writeItem := func(task string, data map[string]any) {
		w, _ := e.Store.Writer(task)
		b, _ := json.Marshal(data)
		w.WriteOne(b)
	}

	writeItem("getProjectIds", map[string]any{
		"key": "cloud-org1_proj1", "id": "proj-id-1", "sonarcloud_org_key": testCloudOrg,
	})
	writeItem("getOrgRepos", map[string]any{
		"id": "repo-123", "slug": "myorg/myrepo", "label": "myrepo", "sonarcloud_org_key": testCloudOrg,
	})

	reg := BuildMigrateRegistry(RegisterAll())

	// matchProjectRepos.
	err := reg["matchProjectRepos"].Run(context.Background(), e)
	if err != nil {
		t.Fatalf("matchProjectRepos: %v", err)
	}

	items, _ := e.Store.ReadAll("matchProjectRepos")
	if len(items) == 0 {
		t.Fatal("expected matchProjectRepos output")
	}
	repoID := extractField(items[0], "repository_id")
	if repoID != "repo-123" {
		t.Errorf("expected repo-123, got %q", repoID)
	}

	// setProjectBinding.
	err = reg["setProjectBinding"].Run(context.Background(), e)
	if err != nil {
		t.Fatalf("setProjectBinding: %v", err)
	}
}

func TestSetPortfolioProjects(t *testing.T) {
	cloudSrv := newMockCloudServer()
	defer cloudSrv.Close()
	apiSrv := newMockAPIServer()
	defer apiSrv.Close()
	dir := t.TempDir()
	setupExtractData(dir)
	e := newTestExecutor(cloudSrv, apiSrv, dir)
	setupCreateOutputs(t, e)

	// createPortfolios dependency.
	w, _ := e.Store.Writer("createPortfolios")
	bp, _ := json.Marshal(map[string]any{"cloud_portfolio_id": "portfolio-1", "source_portfolio_key": "pf1", "name": "Test"})
	w.WriteOne(bp)

	reg := BuildMigrateRegistry(RegisterAll())
	err := reg["setPortfolioProjects"].Run(context.Background(), e)
	if err != nil {
		t.Fatalf("setPortfolioProjects: %v", err)
	}
}

func TestRunMigrateIntegration(t *testing.T) {
	cloudSrv := newMockCloudServer()
	defer cloudSrv.Close()
	apiSrv := newMockAPIServer()
	defer apiSrv.Close()
	dir := t.TempDir()
	setupExtractData(dir)
	setupCSVs(t, dir)

	cfg := MigrateConfig{
		Token:           "test-token",
		EnterpriseKey:   "test-enterprise",
		Edition:         "enterprise",
		URL:             cloudSrv.URL + "/",
		Concurrency:     5,
		ExportDirectory: dir,
		TargetTask:      "createProjects", // Only run one task + deps.
	}

	_, err := RunMigrate(context.Background(), cfg)
	if err != nil {
		t.Fatalf("RunMigrate: %v", err)
	}
}
