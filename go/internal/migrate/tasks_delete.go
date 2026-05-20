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
	counter := NewTaskCounter("deleteProjects")
	err := forEachMigrateItem(ctx, e, "deleteProjects", "getCreatedProjects",
		func(ctx context.Context, item json.RawMessage, w *common.ChunkWriter) error {
			key := extractField(item, "key")
			if key == "" {
				return nil
			}
			err := e.Cloud.Projects.Delete(ctx, key)
			if err != nil {
				counter.Fail()
				logAPIWarn(e.Logger, "deleteProjects failed", err, "key", key)
			} else {
				counter.Success()
			}
			return nil
		})
	counter.LogSummary(e.Logger)
	return err
}

func runDeleteProfiles(ctx context.Context, e *Executor) error {
	counter := NewTaskCounter("deleteProfiles")
	err := forEachMigrateItemFiltered(ctx, e, "deleteProfiles", "createProfiles",
		func(item json.RawMessage) bool {
			return !extractBool(item, "is_default")
		},
		func(ctx context.Context, item json.RawMessage, w *common.ChunkWriter) error {
			orgKey := extractField(item, "sonarcloud_org_key")
			name := extractField(item, "name")
			lang := extractField(item, "language")

			err := e.Cloud.QualityProfiles.Delete(ctx, lang, name, orgKey)
			if err != nil {
				counter.Fail()
				logAPIWarn(e.Logger, "deleteProfiles failed", err, "name", name)
			} else {
				counter.Success()
			}
			return nil
		})
	counter.LogSummary(e.Logger)
	return err
}

func runDeleteGates(ctx context.Context, e *Executor) error {
	counter := NewTaskCounter("deleteGates")
	err := forEachMigrateItem(ctx, e, "deleteGates", "createGates",
		func(ctx context.Context, item json.RawMessage, w *common.ChunkWriter) error {
			orgKey := extractField(item, "sonarcloud_org_key")
			gateIDStr := extractField(item, "cloud_gate_id")
			gateID, _ := strconv.Atoi(gateIDStr)

			err := e.Cloud.QualityGates.Destroy(ctx, gateID, orgKey)
			if err != nil {
				counter.Fail()
				logAPIWarn(e.Logger, "deleteGates failed", err, "gate", gateIDStr)
			} else {
				counter.Success()
			}
			return nil
		})
	counter.LogSummary(e.Logger)
	return err
}

func runDeleteGroups(ctx context.Context, e *Executor) error {
	counter := NewTaskCounter("deleteGroups")
	err := forEachMigrateItem(ctx, e, "deleteGroups", "createGroups",
		func(ctx context.Context, item json.RawMessage, w *common.ChunkWriter) error {
			groupIDStr := extractField(item, "cloud_group_id")
			groupID, _ := strconv.Atoi(groupIDStr)
			if groupID == 0 {
				return nil
			}
			orgKey := extractField(item, "sonarcloud_org_key")
			err := e.Cloud.Groups.Delete(ctx, groupID, orgKey)
			if err != nil {
				counter.Fail()
				logAPIWarn(e.Logger, "deleteGroups failed", err, "group", groupIDStr)
			} else {
				counter.Success()
			}
			return nil
		})
	counter.LogSummary(e.Logger)
	return err
}

func runDeleteTemplates(ctx context.Context, e *Executor) error {
	counter := NewTaskCounter("deleteTemplates")
	err := forEachMigrateItem(ctx, e, "deleteTemplates", "createPermissionTemplates",
		func(ctx context.Context, item json.RawMessage, w *common.ChunkWriter) error {
			templateID := extractField(item, "cloud_template_id")
			if templateID == "" {
				return nil
			}
			orgKey := extractField(item, "sonarcloud_org_key")
			err := e.Cloud.Permissions.DeleteTemplate(ctx, templateID, orgKey)
			if err != nil {
				counter.Fail()
				logAPIWarn(e.Logger, "deleteTemplates failed", err, "template", templateID)
			} else {
				counter.Success()
			}
			return nil
		})
	counter.LogSummary(e.Logger)
	return err
}

func runDeletePortfolios(ctx context.Context, e *Executor) error {
	counter := NewTaskCounter("deletePortfolios")
	err := forEachMigrateItem(ctx, e, "deletePortfolios", "createPortfolios",
		func(ctx context.Context, item json.RawMessage, w *common.ChunkWriter) error {
			portfolioID := extractField(item, "cloud_portfolio_id")
			if portfolioID == "" {
				return nil
			}
			err := e.CloudAPI.Enterprises.DeletePortfolio(ctx, portfolioID)
			if err != nil {
				counter.Fail()
				logAPIWarn(e.Logger, "deletePortfolios failed", err, "portfolio", portfolioID)
			} else {
				counter.Success()
			}
			return nil
		})
	counter.LogSummary(e.Logger)
	return err
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
