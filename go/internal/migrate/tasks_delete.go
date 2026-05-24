package migrate

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/sonar-solutions/sq-api-go/types"
	"github.com/sonar-solutions/sonar-migration-tool/internal/common"
)

// sonarWayGateName is the canonical name of SonarCloud's built-in
// quality gate. Used as a fallback alongside the IsBuiltIn flag in
// case an API response omits the flag.
const sonarWayGateName = "Sonar way"

// isBuiltInGate reports whether a quality gate is the built-in
// SonarCloud "Sonar way". The IsBuiltIn flag is the source of truth
// when present; the name fallback handles SonarCloud responses that
// omit the flag and accepts the documented variants
// ("Sonar way", "Sonar Way", "Sonar way (built-in)").
func isBuiltInGate(g types.QualityGate) bool {
	if g.IsBuiltIn {
		return true
	}
	name := strings.ToLower(strings.TrimSpace(g.Name))
	return name == "sonar way" || name == "sonar way (built-in)"
}

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
			Name: "deleteGates",
			// deleteGates enumerates the org's gates via the SonarCloud
			// API rather than reading createGates' output — issue #213
			// requires deleting EVERY non-built-in gate, not just the
			// ones the migration created. resetDefaultGates is pinned
			// first via the dependency so the built-in is the current
			// default before any destroy call (SonarCloud refuses to
			// destroy the current default).
			Dependencies: []string{"generateOrganizationMappings", "resetDefaultGates"},
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
			// Restores the built-in "Sonar way" as the org's default
			// gate before deleteGates runs. SonarCloud rejects /api/
			// qualitygates/destroy on whichever gate is currently the
			// default, so without this step the gate that was set as
			// default during migration (and any gate the user later
			// promoted to default) survives reset. Issue #213.
			Name:         "resetDefaultGates",
			Dependencies: []string{"generateOrganizationMappings"},
			Run:          runResetDefaultGates,
		},
		{
			Name:         "resetPermissionTemplates",
			Dependencies: []string{"setDefaultTemplates"},
			Run:          runResetPermissionTemplates,
		},
		{
			// Reverts every org-level setting that has been customized on
			// SonarQube Cloud back to its default. Setting reset is
			// scoped per organization; this task iterates the mapped orgs
			// and resets the union of customized keys in each.
			Name:         "resetGlobalSettings",
			Dependencies: []string{"generateOrganizationMappings"},
			Run:          runResetGlobalSettings,
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
			e.Logger.Debug("project api call: POST /api/projects/delete",
				"project", key)
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

// runDeleteGates enumerates every quality gate in each mapped org via
// /api/qualitygates/list and destroys the non-built-in ones. Issue
// #213 requires reset to delete every non-built-in gate, not just
// those the migration created — including any gates an admin added
// manually. resetDefaultGates is a dependency, so by the time this
// runs the built-in Sonar way is the org's default and the
// previously-default custom gate is destroyable.
func runDeleteGates(ctx context.Context, e *Executor) error {
	counter := NewTaskCounter("deleteGates")
	err := forEachMigrateItem(ctx, e, "deleteGates", "generateOrganizationMappings",
		func(ctx context.Context, item json.RawMessage, w *common.ChunkWriter) error {
			orgKey := extractField(item, "sonarcloud_org_key")
			if shouldSkipOrg(orgKey) {
				return nil
			}
			gates, err := e.Cloud.QualityGates.List(ctx, orgKey)
			if err != nil {
				counter.Fail()
				logAPIWarn(e.Logger, "deleteGates: listing gates failed", err, "org", orgKey)
				return nil
			}
			e.Logger.Info("deleteGates: listed gates",
				"org", orgKey, "count", len(gates), "summary", summariseGates(gates))
			for _, g := range gates {
				if isBuiltInGate(g) {
					e.Logger.Info("deleteGates: keeping built-in gate",
						"org", orgKey, "gate", g.Name, "gate_id", g.ID)
					continue
				}
				e.Logger.Info("deleteGates: destroying non-built-in gate",
					"org", orgKey, "gate", g.Name, "gate_id", g.ID, "isDefault", g.IsDefault)
				if err := e.Cloud.QualityGates.Destroy(ctx, g.ID, orgKey); err != nil {
					counter.Fail()
					logAPIWarn(e.Logger, "deleteGates failed", err,
						"gate", g.Name, "gate_id", g.ID, "org", orgKey, "isDefault", g.IsDefault)
					continue
				}
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

// runResetGlobalSettings reverts every customized org-level setting on
// SonarQube Cloud back to its default. SQC's /api/settings/values only
// returns keys that have been explicitly customized, so the reset key
// list is naturally bounded — no enumeration of all definitions is
// required. Iteration is per-org from generateOrganizationMappings so
// no upstream create*/generate* dependency is pulled into reset's
// plan.
func runResetGlobalSettings(ctx context.Context, e *Executor) error {
	counter := NewTaskCounter("resetGlobalSettings")
	err := forEachMigrateItem(ctx, e, "resetGlobalSettings", "generateOrganizationMappings",
		func(ctx context.Context, item json.RawMessage, w *common.ChunkWriter) error {
			orgKey := extractField(item, "sonarcloud_org_key")
			if shouldSkipOrg(orgKey) {
				return nil
			}

			values, err := e.Cloud.Settings.Values(ctx, "", orgKey)
			if err != nil {
				counter.Fail()
				logAPIWarn(e.Logger, "resetGlobalSettings: listing org settings failed", err, "org", orgKey)
				return nil
			}

			var keys []string
			for _, s := range values {
				// Skip settings that are still at their inherited default
				// — only revert what's been explicitly set at org scope.
				if s.Inherited || s.Key == "" {
					continue
				}
				keys = append(keys, s.Key)
			}
			if len(keys) == 0 {
				counter.Success()
				return nil
			}

			e.Logger.Debug("settings api call: POST /api/settings/reset",
				"org", orgKey, "keys", keys)
			if err := e.Cloud.Settings.Reset(ctx, "", keys, orgKey); err != nil {
				counter.Fail()
				logAPIWarn(e.Logger, "resetGlobalSettings: reset failed", err, "org", orgKey, "keys", keys)
				return nil
			}
			counter.Success()
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

// runResetDefaultGates restores the built-in "Sonar way" as each
// mapped org's default quality gate, so deleteGates can subsequently
// destroy whichever custom gate the migration (or the user) had
// promoted to default. SonarCloud's /api/qualitygates/destroy rejects
// the current default; without this step the custom default gate
// survives reset. Issue #213.
func runResetDefaultGates(ctx context.Context, e *Executor) error {
	counter := NewTaskCounter("resetDefaultGates")
	err := forEachMigrateItem(ctx, e, "resetDefaultGates", "generateOrganizationMappings",
		func(ctx context.Context, item json.RawMessage, w *common.ChunkWriter) error {
			orgKey := extractField(item, "sonarcloud_org_key")
			if shouldSkipOrg(orgKey) {
				return nil
			}
			gates, err := e.Cloud.QualityGates.List(ctx, orgKey)
			if err != nil {
				counter.Fail()
				logAPIWarn(e.Logger, "resetDefaultGates: listing gates failed", err, "org", orgKey)
				return nil
			}
			e.Logger.Info("resetDefaultGates: listed gates",
				"org", orgKey, "count", len(gates), "summary", summariseGates(gates))

			var builtIn *int
			var builtInName string
			for i := range gates {
				if isBuiltInGate(gates[i]) {
					builtIn = &gates[i].ID
					builtInName = gates[i].Name
					if gates[i].IsDefault {
						// Already default — nothing to do.
						e.Logger.Info("resetDefaultGates: built-in is already default",
							"org", orgKey, "gate", builtInName, "gate_id", *builtIn)
						counter.Success()
						return nil
					}
					break
				}
			}
			if builtIn == nil {
				e.Logger.Warn("resetDefaultGates: no built-in gate found in list response; deleteGates may fail to destroy the current default",
					"org", orgKey, "gates_returned", summariseGates(gates))
				counter.Fail()
				return nil
			}
			e.Logger.Info("resetDefaultGates: promoting built-in to default",
				"org", orgKey, "gate", builtInName, "gate_id", *builtIn)
			if err := e.Cloud.QualityGates.SetDefault(ctx, *builtIn, orgKey); err != nil {
				counter.Fail()
				logAPIWarn(e.Logger, "resetDefaultGates: set_as_default failed", err, "org", orgKey, "gate_id", *builtIn)
				return nil
			}
			counter.Success()
			return nil
		})
	counter.LogSummary(e.Logger)
	return err
}

// summariseGates renders a compact, log-friendly summary of an org's
// gates: "<name> (id=N, builtIn=B, default=D)" joined by ", ".
// Used by reset's task logging so an operator can see exactly what
// SonarCloud returned when something goes wrong.
func summariseGates(gates []types.QualityGate) string {
	parts := make([]string, 0, len(gates))
	for _, g := range gates {
		parts = append(parts, fmt.Sprintf("%q (id=%d, builtIn=%t, default=%t)",
			g.Name, g.ID, g.IsBuiltIn, g.IsDefault))
	}
	return strings.Join(parts, ", ")
}

func runResetPermissionTemplates(_ context.Context, e *Executor) error {
	// No-op: Cloud resets defaults when templates are deleted.
	w, _ := e.Store.Writer("resetPermissionTemplates")
	return w.WriteChunk(nil)
}
