package migrate

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/sonar-solutions/sonar-migration-tool/internal/common"
)

// deleteTasks returns tasks for deleting/resetting entities in Cloud.
func deleteTasks() []TaskDef {
	entEditions := []common.Edition{common.EditionEnterprise, common.EditionDatacenter}

	return []TaskDef{
		{
			Name:         "deleteProjects",
			Dependencies: []string{"getCreatedProjects"},
			Run:          runDeleteProjects,
		},
		{
			Name:         "deleteProfiles",
			Dependencies: []string{"createProfiles"},
			Run:          runDeleteProfiles,
		},
		{
			Name:         "deleteGates",
			Dependencies: []string{"createGates"},
			Run:          runDeleteGates,
		},
		{
			Name:         "deleteGroups",
			Dependencies: []string{"createGroups"},
			Run:          runDeleteGroups,
		},
		{
			Name:         "deleteTemplates",
			Dependencies: []string{"createPermissionTemplates"},
			Run:          runDeleteTemplates,
		},
		{
			Name:         "deletePortfolios",
			Editions:     entEditions,
			Dependencies: []string{"createPortfolios"},
			Run:          runDeletePortfolios,
		},
		{
			Name:         "resetDefaultProfiles",
			Dependencies: []string{"setDefaultProfiles"},
			Run:          runResetDefaultProfiles,
		},
		{
			Name:         "resetDefaultGates",
			Dependencies: []string{"setDefaultGates"},
			Run:          runResetDefaultGates,
		},
		{
			Name:         "resetPermissionTemplates",
			Dependencies: []string{"setDefaultTemplates"},
			Run:          runResetPermissionTemplates,
		},
	}
}

func runDeleteProjects(ctx context.Context, e *Executor) error {
	return forEachMigrateItem(ctx, e, "deleteProjects", "getCreatedProjects",
		func(ctx context.Context, item json.RawMessage, w *common.ChunkWriter) error {
			key := extractField(item, "key")
			if key == "" {
				return nil
			}
			err := e.Cloud.Projects.Delete(ctx, key)
			if err != nil {
				e.Logger.Warn("deleteProjects failed", "key", key, "err", err)
			}
			return nil
		})
}

func runDeleteProfiles(ctx context.Context, e *Executor) error {
	return forEachMigrateItemFiltered(ctx, e, "deleteProfiles", "createProfiles",
		func(item json.RawMessage) bool {
			return !extractBool(item, "is_default")
		},
		func(ctx context.Context, item json.RawMessage, w *common.ChunkWriter) error {
			orgKey := extractField(item, "sonarcloud_org_key")
			name := extractField(item, "name")
			lang := extractField(item, "language")

			err := e.Cloud.QualityProfiles.Delete(ctx, lang, name, orgKey)
			if err != nil {
				e.Logger.Warn("deleteProfiles failed", "name", name, "err", err)
			}
			return nil
		})
}

func runDeleteGates(ctx context.Context, e *Executor) error {
	return forEachMigrateItem(ctx, e, "deleteGates", "createGates",
		func(ctx context.Context, item json.RawMessage, w *common.ChunkWriter) error {
			orgKey := extractField(item, "sonarcloud_org_key")
			gateIDStr := extractField(item, "cloud_gate_id")
			gateID, _ := strconv.Atoi(gateIDStr)

			err := e.Cloud.QualityGates.Destroy(ctx, gateID, orgKey)
			if err != nil {
				e.Logger.Warn("deleteGates failed", "gate", gateIDStr, "err", err)
			}
			return nil
		})
}

func runDeleteGroups(ctx context.Context, e *Executor) error {
	return forEachMigrateItem(ctx, e, "deleteGroups", "createGroups",
		func(ctx context.Context, item json.RawMessage, w *common.ChunkWriter) error {
			groupIDStr := extractField(item, "cloud_group_id")
			groupID, _ := strconv.Atoi(groupIDStr)
			if groupID == 0 {
				return nil
			}
			err := e.Cloud.Groups.Delete(ctx, groupID)
			if err != nil {
				e.Logger.Warn("deleteGroups failed", "group", groupIDStr, "err", err)
			}
			return nil
		})
}

func runDeleteTemplates(ctx context.Context, e *Executor) error {
	return forEachMigrateItem(ctx, e, "deleteTemplates", "createPermissionTemplates",
		func(ctx context.Context, item json.RawMessage, w *common.ChunkWriter) error {
			templateID := extractField(item, "cloud_template_id")
			if templateID == "" {
				return nil
			}
			err := e.Cloud.Permissions.DeleteTemplate(ctx, templateID)
			if err != nil {
				e.Logger.Warn("deleteTemplates failed", "template", templateID, "err", err)
			}
			return nil
		})
}

func runDeletePortfolios(ctx context.Context, e *Executor) error {
	return forEachMigrateItem(ctx, e, "deletePortfolios", "createPortfolios",
		func(ctx context.Context, item json.RawMessage, w *common.ChunkWriter) error {
			portfolioID := extractField(item, "cloud_portfolio_id")
			if portfolioID == "" {
				return nil
			}
			err := e.CloudAPI.Enterprises.DeletePortfolio(ctx, portfolioID)
			if err != nil {
				e.Logger.Warn("deletePortfolios failed", "portfolio", portfolioID, "err", err)
			}
			return nil
		})
}

func runResetDefaultProfiles(_ context.Context, e *Executor) error {
	// No-op: Cloud resets defaults when profiles are deleted.
	w, _ := e.Store.Writer("resetDefaultProfiles")
	return w.WriteChunk(nil)
}

func runResetDefaultGates(_ context.Context, e *Executor) error {
	// No-op: Cloud resets defaults when gates are deleted.
	w, _ := e.Store.Writer("resetDefaultGates")
	return w.WriteChunk(nil)
}

func runResetPermissionTemplates(_ context.Context, e *Executor) error {
	// No-op: Cloud resets defaults when templates are deleted.
	w, _ := e.Store.Writer("resetPermissionTemplates")
	return w.WriteChunk(nil)
}
