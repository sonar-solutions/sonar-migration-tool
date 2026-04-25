package extract

import (
	"context"
	"encoding/json"
	"net/url"
)

func templateTasks() []TaskDef {
	return []TaskDef{
		{
			Name:     "getTemplates",
			Editions: AllEditions,
			Run: func(ctx context.Context, e *Executor) error {
				return fetchAndWriteArray(ctx, e, "getTemplates", "api/permissions/search_templates", "permissionTemplates", nil, map[string]any{"serverUrl": e.ServerURL})
			},
		},
		{
			Name:     "getDefaultTemplates",
			Editions: AllEditions,
			Run: func(ctx context.Context, e *Executor) error {
				return fetchAndWriteArray(ctx, e, "getDefaultTemplates", "api/permissions/search_templates", "defaultTemplates", nil, map[string]any{"serverUrl": e.ServerURL})
			},
		},
		{
			Name:         "getTemplateGroupsScanners",
			Editions:     AllEditions,
			Dependencies: []string{"getTemplates"},
			Run:          makeTemplatePermissionTask("getTemplateGroupsScanners", "api/permissions/template_groups", "groups", "scan"),
		},
		{
			Name:         "getTemplateGroupsViewers",
			Editions:     AllEditions,
			Dependencies: []string{"getTemplates"},
			Run:          makeTemplatePermissionTask("getTemplateGroupsViewers", "api/permissions/template_groups", "groups", "user"),
		},
		{
			Name:         "getTemplateUsersScanners",
			Editions:     AllEditions,
			Dependencies: []string{"getTemplates"},
			Run:          makeTemplatePermissionTask("getTemplateUsersScanners", "api/permissions/template_users", "users", "scan"),
		},
		{
			Name:         "getTemplateUsersViewers",
			Editions:     AllEditions,
			Dependencies: []string{"getTemplates"},
			Run:          makeTemplatePermissionTask("getTemplateUsersViewers", "api/permissions/template_users", "users", "user"),
		},
	}
}

func makeTemplatePermissionTask(taskName, path, resultKey, permission string) func(ctx context.Context, e *Executor) error {
	return func(ctx context.Context, e *Executor) error {
		return forEachDep(ctx, e, taskName, "getTemplates",
			func(ctx context.Context, item json.RawMessage, w *ChunkWriter) error {
				templateID := extractField(item, "id")
				items, err := e.Raw.GetPaginated(ctx, PaginatedOpts{
					Path: path, ResultKey: resultKey, MaxPageSize: 100,
					Params: url.Values{"templateId": {templateID}, "permission": {permission}},
				})
				if err != nil {
					return err
				}
				items = enrichAll(items, map[string]any{"templateId": templateID, "serverUrl": e.ServerURL})
				return w.WriteChunk(items)
			})
	}
}
