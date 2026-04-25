package migrate

import (
	"context"
	"encoding/json"
	"testing"
)

func TestGetGateConditions(t *testing.T) {
	cloudSrv := newMockCloudServer()
	defer cloudSrv.Close()
	apiSrv := newMockAPIServer()
	defer apiSrv.Close()
	dir := t.TempDir()
	setupExtractData(dir)
	e := newTestExecutor(cloudSrv, apiSrv, dir)
	setupCreateOutputs(t, e)

	reg := BuildMigrateRegistry(RegisterAll())
	err := reg["getGateConditions"].Run(context.Background(), e)
	if err != nil {
		t.Fatalf("getGateConditions: %v", err)
	}

	items, _ := e.Store.ReadAll("getGateConditions")
	if len(items) == 0 {
		t.Error("expected getGateConditions output")
	}
}

func TestGetProfileBackups(t *testing.T) {
	cloudSrv := newMockCloudServer()
	defer cloudSrv.Close()
	apiSrv := newMockAPIServer()
	defer apiSrv.Close()
	dir := t.TempDir()
	setupExtractData(dir)
	e := newTestExecutor(cloudSrv, apiSrv, dir)
	setupCreateOutputs(t, e)

	reg := BuildMigrateRegistry(RegisterAll())
	err := reg["getProfileBackups"].Run(context.Background(), e)
	if err != nil {
		t.Fatalf("getProfileBackups: %v", err)
	}

	items, _ := e.Store.ReadAll("getProfileBackups")
	if len(items) == 0 {
		t.Error("expected getProfileBackups output")
	}
}

func TestGetCreatedProjects(t *testing.T) {
	cloudSrv := newMockCloudServer()
	defer cloudSrv.Close()
	apiSrv := newMockAPIServer()
	defer apiSrv.Close()
	dir := t.TempDir()
	setupExtractData(dir)
	e := newTestExecutor(cloudSrv, apiSrv, dir)
	setupCreateOutputs(t, e)

	reg := BuildMigrateRegistry(RegisterAll())
	err := reg["getCreatedProjects"].Run(context.Background(), e)
	if err != nil {
		t.Fatalf("getCreatedProjects: %v", err)
	}

	items, _ := e.Store.ReadAll("getCreatedProjects")
	if len(items) == 0 {
		t.Error("expected getCreatedProjects output")
	}
}

func TestGetProjectIdsEmptyOrg(t *testing.T) {
	cloudSrv := newMockCloudServer()
	defer cloudSrv.Close()
	apiSrv := newMockAPIServer()
	defer apiSrv.Close()
	dir := t.TempDir()
	setupExtractData(dir)
	e := newTestExecutor(cloudSrv, apiSrv, dir)

	// createProjects with empty org key should be skipped.
	w, _ := e.Store.Writer("createProjects")
	b, _ := json.Marshal(map[string]any{"cloud_project_key": "x", "sonarcloud_org_key": ""})
	w.WriteOne(b)

	reg := BuildMigrateRegistry(RegisterAll())
	err := reg["getProjectIds"].Run(context.Background(), e)
	if err != nil {
		t.Fatalf("getProjectIds: %v", err)
	}
}

func TestGetOrgReposEmptyOrg(t *testing.T) {
	cloudSrv := newMockCloudServer()
	defer cloudSrv.Close()
	apiSrv := newMockAPIServer()
	defer apiSrv.Close()
	dir := t.TempDir()
	setupExtractData(dir)
	e := newTestExecutor(cloudSrv, apiSrv, dir)

	w, _ := e.Store.Writer("generateOrganizationMappings")
	b, _ := json.Marshal(map[string]any{"sonarcloud_org_key": ""})
	w.WriteOne(b)

	reg := BuildMigrateRegistry(RegisterAll())
	err := reg["getOrgRepos"].Run(context.Background(), e)
	if err != nil {
		t.Fatalf("getOrgRepos: %v", err)
	}
}
