package migrate

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/sonar-solutions/sq-api-go/cloud"
	"github.com/sonar-solutions/sonar-migration-tool/internal/common"
)

// configureTasks returns tasks that configure profiles, gates, and defaults.
func configureTasks() []TaskDef {
	return []TaskDef{
		{
			Name:         "setProfileParent",
			Dependencies: []string{"createProfiles"},
			Run:          runSetProfileParent,
		},
		{
			Name:         "restoreProfiles",
			Dependencies: []string{"createProfiles", "setProfileParent", "getProfileBackups"},
			Run:          runRestoreProfiles,
		},
		{
			Name:         "addGateConditions",
			Dependencies: []string{"createGates", "getGateConditions"},
			Run:          runAddGateConditions,
		},
		{
			Name:         "setDefaultProfiles",
			Dependencies: []string{"createProfiles", "restoreProfiles"},
			Run:          runSetDefaultProfiles,
		},
		{
			Name:         "setDefaultGates",
			Dependencies: []string{"createGates", "addGateConditions"},
			Run:          runSetDefaultGates,
		},
		{
			Name:         "setDefaultTemplates",
			Dependencies: []string{"createPermissionTemplates"},
			Run:          runSetDefaultTemplates,
		},
	}
}

func runSetProfileParent(ctx context.Context, e *Executor) error {
	return forEachMigrateItemFiltered(ctx, e, "setProfileParent", "createProfiles",
		func(item json.RawMessage) bool {
			return extractField(item, "parent_name") != ""
		},
		func(ctx context.Context, item json.RawMessage, w *common.ChunkWriter) error {
			orgKey := extractField(item, "sonarcloud_org_key")
			name := extractField(item, "name")
			lang := extractField(item, "language")
			parent := extractField(item, "parent_name")

			err := e.Cloud.QualityProfiles.ChangeParent(ctx, lang, name, parent, orgKey)
			if err != nil {
				e.Logger.Warn("setProfileParent failed", "name", name, "err", err)
			}
			return nil
		})
}

func runRestoreProfiles(ctx context.Context, e *Executor) error {
	return forEachMigrateItem(ctx, e, "restoreProfiles", "getProfileBackups",
		func(ctx context.Context, item json.RawMessage, w *common.ChunkWriter) error {
			orgKey := extractField(item, "sonarcloud_org_key")
			profileKey := extractField(item, "profileKey")
			if shouldSkipOrg(orgKey) || profileKey == "" {
				return nil
			}

			// Read the XML backup from extract data.
			items, _ := readExtractItems(e, "getProfileBackups")
			for _, ei := range items {
				eiKey := extractField(ei.Data, "profileKey")
				if eiKey != profileKey {
					continue
				}
				// The backup data is stored as XML in the "backup" field.
				backup := extractField(ei.Data, "backup")
				if backup == "" {
					continue
				}
				_, err := e.Cloud.QualityProfiles.Restore(ctx, orgKey, []byte(backup))
				if err != nil {
					e.Logger.Warn("restoreProfiles failed", "profile", profileKey, "err", err)
				}
				return nil
			}
			return nil
		})
}

func runAddGateConditions(ctx context.Context, e *Executor) error {
	return forEachMigrateItem(ctx, e, "addGateConditions", "getGateConditions",
		func(ctx context.Context, item json.RawMessage, w *common.ChunkWriter) error {
			orgKey := extractField(item, "sonarcloud_org_key")
			gateIDStr := extractField(item, "cloud_gate_id")
			if shouldSkipOrg(orgKey) || gateIDStr == "" {
				return nil
			}
			gateID, _ := strconv.Atoi(gateIDStr)

			// Extract conditions from the gate data.
			var obj map[string]json.RawMessage
			json.Unmarshal(item, &obj)
			conditionsRaw, ok := obj["conditions"]
			if !ok {
				return nil
			}
			var conditions []map[string]any
			json.Unmarshal(conditionsRaw, &conditions)

			for _, cond := range conditions {
				metric, _ := cond["metric"].(string)
				op, _ := cond["op"].(string)
				errorVal, _ := cond["error"].(string)
				if metric == "" || op == "" {
					continue
				}
				_, err := e.Cloud.QualityGates.CreateCondition(ctx, cloud.CreateConditionParams{
					GateID: gateID, Organization: orgKey,
					Metric: metric, Op: op, Error: errorVal,
				})
				if err != nil {
					e.Logger.Warn("addGateConditions failed", "metric", metric, "err", err)
				}
			}
			return nil
		})
}

func runSetDefaultProfiles(ctx context.Context, e *Executor) error {
	return forEachMigrateItemFiltered(ctx, e, "setDefaultProfiles", "createProfiles",
		func(item json.RawMessage) bool {
			return extractBool(item, "is_default")
		},
		func(ctx context.Context, item json.RawMessage, w *common.ChunkWriter) error {
			orgKey := extractField(item, "sonarcloud_org_key")
			name := extractField(item, "name")
			lang := extractField(item, "language")

			err := e.Cloud.QualityProfiles.SetDefault(ctx, lang, name, orgKey)
			if err != nil {
				e.Logger.Warn("setDefaultProfiles failed", "name", name, "err", err)
			}
			return nil
		})
}

func runSetDefaultGates(ctx context.Context, e *Executor) error {
	return forEachMigrateItemFiltered(ctx, e, "setDefaultGates", "createGates",
		func(item json.RawMessage) bool {
			return extractBool(item, "is_default")
		},
		func(ctx context.Context, item json.RawMessage, w *common.ChunkWriter) error {
			orgKey := extractField(item, "sonarcloud_org_key")
			gateIDStr := extractField(item, "cloud_gate_id")
			gateID, _ := strconv.Atoi(gateIDStr)

			err := e.Cloud.QualityGates.SetDefault(ctx, gateID, orgKey)
			if err != nil {
				e.Logger.Warn("setDefaultGates failed", "gate", gateIDStr, "err", err)
			}
			return nil
		})
}

func runSetDefaultTemplates(ctx context.Context, e *Executor) error {
	return forEachMigrateItemFiltered(ctx, e, "setDefaultTemplates", "createPermissionTemplates",
		func(item json.RawMessage) bool {
			return extractBool(item, "is_default")
		},
		func(ctx context.Context, item json.RawMessage, w *common.ChunkWriter) error {
			templateID := extractField(item, "cloud_template_id")

			err := e.Cloud.Permissions.SetDefaultTemplate(ctx, templateID, "TRK")
			if err != nil {
				e.Logger.Warn("setDefaultTemplates failed", "template", templateID, "err", err)
			}
			return nil
		})
}
