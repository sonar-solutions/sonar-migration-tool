package migrate

import (
	"context"
	"encoding/json"
	"net/url"

	"github.com/sonar-solutions/sonar-migration-tool/internal/common"
)

// readTasks returns tasks that fetch existing state from SonarQube Cloud.
func readTasks() []TaskDef {
	return []TaskDef{
		{
			Name:         "getProjectIds",
			Dependencies: []string{"createProjects"},
			Run:          runGetProjectIds,
		},
		{
			Name:         "getOrgRepos",
			Dependencies: []string{"generateOrganizationMappings"},
			Run:          runGetOrgRepos,
		},
		{
			Name:         "getGateConditions",
			Dependencies: []string{"createGates"},
			Run:          runGetGateConditions,
		},
		{
			Name:         "getProfileBackups",
			Dependencies: []string{"createProfiles"},
			Run:          runGetProfileBackups,
		},
		{
			Name:         "getMigrationUser",
			Dependencies: []string{"generateOrganizationMappings"},
			Run:          runGetMigrationUser,
		},
		{
			Name:         "getCreatedProjects",
			Dependencies: []string{"createProjects"},
			Run:          runGetCreatedProjects,
		},
		{
			Name:         "getEnterprises",
			Editions:     []common.Edition{common.EditionEnterprise, common.EditionDatacenter},
			Dependencies: []string{"generateOrganizationMappings"},
			Run:          runGetEnterprises,
		},
	}
}

func runGetProjectIds(ctx context.Context, e *Executor) error {
	return forEachMigrateItem(ctx, e, "getProjectIds", "createProjects",
		func(ctx context.Context, item json.RawMessage, w *common.ChunkWriter) error {
			orgKey := extractField(item, "sonarcloud_org_key")
			projectKey := extractField(item, "cloud_project_key")
			if shouldSkipOrg(orgKey) || projectKey == "" {
				return nil
			}
			if err := common.AcquireSem(ctx, e.Sem); err != nil {
				return err
			}
			raw, err := e.Raw.GetPaginated(ctx, common.PaginatedOpts{
				Path: "api/projects/search", ResultKey: "components",
				Params: url.Values{
					"organization": {orgKey},
					"projects":     {projectKey},
				},
			})
			<-e.Sem
			if err != nil {
				return err
			}
			enriched := common.EnrichAll(raw, map[string]any{
				"sonarcloud_org_key": orgKey,
			})
			return w.WriteChunk(enriched)
		})
}

func runGetOrgRepos(ctx context.Context, e *Executor) error {
	return forEachMigrateItem(ctx, e, "getOrgRepos", "generateOrganizationMappings",
		func(ctx context.Context, item json.RawMessage, w *common.ChunkWriter) error {
			orgKey := extractField(item, "sonarcloud_org_key")
			if shouldSkipOrg(orgKey) {
				return nil
			}
			if err := common.AcquireSem(ctx, e.Sem); err != nil {
				return err
			}
			raw, err := e.Raw.GetArray(ctx,
				"api/alm_integration/list_repositories", "repositories",
				url.Values{"organization": {orgKey}})
			<-e.Sem
			if err != nil {
				e.Logger.Warn("getOrgRepos skipped", "org", orgKey, "err", err)
				return nil
			}
			enriched := common.EnrichAll(raw, map[string]any{
				"sonarcloud_org_key": orgKey,
			})
			return w.WriteChunk(enriched)
		})
}

func runGetGateConditions(ctx context.Context, e *Executor) error {
	return forEachMigrateItem(ctx, e, "getGateConditions", "createGates",
		func(ctx context.Context, item json.RawMessage, w *common.ChunkWriter) error {
			gateName := extractField(item, "source_gate_key")
			orgKey := extractField(item, "sonarcloud_org_key")
			serverURL := extractField(item, "server_url")
			// Read gate conditions from extract data.
			items, _ := readExtractItems(e, "getGateConditions")
			for _, ei := range items {
				eiGate := extractField(ei.Data, "name")
				if eiGate == gateName && ei.ServerURL == serverURL {
					enriched := common.EnrichRaw(ei.Data, map[string]any{
						"sonarcloud_org_key": orgKey,
						"cloud_gate_id":     extractField(item, "cloud_gate_id"),
					})
					if err := w.WriteOne(enriched); err != nil {
						return err
					}
				}
			}
			return nil
		})
}

func runGetProfileBackups(ctx context.Context, e *Executor) error {
	return forEachMigrateItem(ctx, e, "getProfileBackups", "createProfiles",
		func(ctx context.Context, item json.RawMessage, w *common.ChunkWriter) error {
			profileKey := extractField(item, "source_profile_key")
			orgKey := extractField(item, "sonarcloud_org_key")
			// Read backup from extract data.
			items, _ := readExtractItems(e, "getProfileBackups")
			for _, ei := range items {
				eiKey := extractField(ei.Data, "profileKey")
				if eiKey == profileKey {
					enriched := common.EnrichRaw(ei.Data, map[string]any{
						"sonarcloud_org_key": orgKey,
					})
					if err := w.WriteOne(enriched); err != nil {
						return err
					}
				}
			}
			return nil
		})
}

func runGetMigrationUser(ctx context.Context, e *Executor) error {
	w, err := e.Store.Writer("getMigrationUser")
	if err != nil {
		return err
	}
	if err := common.AcquireSem(ctx, e.Sem); err != nil {
		return err
	}
	raw, err := e.Raw.Get(ctx, "api/users/current", nil)
	<-e.Sem
	if err != nil {
		return err
	}
	return w.WriteOne(raw)
}

func runGetCreatedProjects(ctx context.Context, e *Executor) error {
	return forEachMigrateItem(ctx, e, "getCreatedProjects", "createProjects",
		func(ctx context.Context, item json.RawMessage, w *common.ChunkWriter) error {
			orgKey := extractField(item, "sonarcloud_org_key")
			if shouldSkipOrg(orgKey) {
				return nil
			}
			if err := common.AcquireSem(ctx, e.Sem); err != nil {
				return err
			}
			raw, err := e.Raw.GetPaginated(ctx, common.PaginatedOpts{
				Path: "api/projects/search", ResultKey: "components",
				Params: url.Values{"organization": {orgKey}},
			})
			<-e.Sem
			if err != nil {
				return err
			}
			enriched := common.EnrichAll(raw, map[string]any{
				"sonarcloud_org_key": orgKey,
			})
			return w.WriteChunk(enriched)
		})
}

func runGetEnterprises(ctx context.Context, e *Executor) error {
	w, err := e.Store.Writer("getEnterprises")
	if err != nil {
		return err
	}
	if err := common.AcquireSem(ctx, e.Sem); err != nil {
		return err
	}
	raw, err := e.RawAPI.Get(ctx, "enterprises/enterprises", nil)
	<-e.Sem
	if err != nil {
		return err
	}
	return w.WriteOne(raw)
}
