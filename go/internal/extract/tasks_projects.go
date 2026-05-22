package extract

import (
	"context"
	"encoding/json"
	"net/url"
)

// measureMetricKeys is the set of metric keys extracted for each project.
const measureMetricKeys = "ncloc_language_distribution,new_violations,accepted_issues,alert_status,false_positive_issues,violations,new_lines_to_cover,lines_to_cover,new_coverage,coverage,uncovered_lines,new_uncovered_lines"

// projectPermissions is the set of permissions expanded for getProjectGroupsPermissions.
var projectPermissions = []string{"admin", "codeviewer", "issueadmin", "securityhotspotadmin", "scan", "user"}

func projectTasks() []TaskDef {
	return []TaskDef{
		{
			Name:     "getProjects",
			Editions: AllEditions,
			Run: func(ctx context.Context, e *Executor) error {
				return fetchAndWritePaginated(ctx, e, "getProjects", PaginatedOpts{
					Path: "api/projects/search", ResultKey: "components",
				}, map[string]any{"serverUrl": e.ServerURL})
			},
		},
		{
			Name:         "getProjectTags",
			Editions:     AllEditions,
			Dependencies: []string{"getProjects"},
			Run:          projectTagsTask(),
		},
		{Name: "getProjectDetails", Editions: AllEditions, Dependencies: []string{"getProjects"},
			Run: perProjectSingle("getProjectDetails", "api/navigation/component", "component")},
		{Name: "getProjectSettings", Editions: AllEditions, Dependencies: []string{"getProjects"},
			Run: projectSettingsTask()},
		{Name: "getProjectLinks", Editions: AllEditions, Dependencies: []string{"getProjects"},
			Run: perProjectArray("getProjectLinks", "api/project_links/search", "links", "projectKey", "projectKey")},
		{Name: "getProjectMeasures", Editions: AllEditions, Dependencies: []string{"getProjects"},
			Run: projectMeasuresTask()},
		{Name: "getProjectWebhooks", Editions: AllEditions, Dependencies: []string{"getProjects"},
			Run: perProjectArray("getProjectWebhooks", "api/webhooks/list", "webhooks", "project", "projectKey")},
		{Name: "getProjectBindings", Editions: AllEditions, Dependencies: []string{"getProjects"},
			Run: projectBindingsTask()},
		{Name: "getProjectGroupsPermissions", Editions: AllEditions, Dependencies: []string{"getProjects"},
			Run: projectGroupsPermissionsTask()},
		{Name: "getProjectUsersScanners", Editions: AllEditions, Dependencies: []string{"getProjects"},
			Run: perProjectPermissionUsers("getProjectUsersScanners", "scan")},
		{Name: "getProjectUsersViewers", Editions: AllEditions, Dependencies: []string{"getProjects"},
			Run: perProjectPermissionUsers("getProjectUsersViewers", "user")},
	}
}

// projectTagsTask fetches every project's tags via
// GET /api/projects/search?f=tags and writes one record per project with
// {projectKey, tags, serverUrl}. The plain getProjects task does not pass
// f=tags, so its records always have tags=null — hence the dedicated
// extract here. Projects with no tags are skipped to keep the output lean.
func projectTagsTask() func(ctx context.Context, e *Executor) error {
	return func(ctx context.Context, e *Executor) error {
		items, err := e.Raw.GetPaginated(ctx, PaginatedOpts{
			Path:      "api/projects/search",
			ResultKey: "components",
			Params:    url.Values{"f": {"tags"}},
		})
		if err != nil {
			return err
		}
		out := make([]json.RawMessage, 0, len(items))
		for _, it := range items {
			key := extractField(it, "key")
			if key == "" {
				continue
			}
			tags := extractTagsArray(it)
			if len(tags) == 0 {
				continue
			}
			rec, err := json.Marshal(map[string]any{
				"projectKey": key,
				"tags":       tags,
				"serverUrl":  e.ServerURL,
			})
			if err != nil {
				continue
			}
			out = append(out, rec)
		}
		w, err := e.Store.Writer("getProjectTags")
		if err != nil {
			return err
		}
		return w.WriteChunk(out)
	}
}

// extractTagsArray pulls a non-empty "tags" string array out of a JSON
// record, tolerating absent or non-array shapes.
func extractTagsArray(raw json.RawMessage) []string {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil
	}
	arrRaw, ok := obj["tags"]
	if !ok {
		return nil
	}
	var arr []string
	if json.Unmarshal(arrRaw, &arr) != nil {
		return nil
	}
	return arr
}

func projectSettingsTask() func(ctx context.Context, e *Executor) error {
	return func(ctx context.Context, e *Executor) error {
		return forEachDep(ctx, e, "getProjectSettings", "getProjects",
			func(ctx context.Context, item json.RawMessage, w *ChunkWriter) error {
				key := extractField(item, "key")
				items, err := e.Raw.GetArray(ctx, "api/settings/values", "settings",
					url.Values{"component": {key}})
				if err != nil {
					return err
				}
				filtered := filterNonInherited(items)
				return w.WriteChunk(enrichAll(filtered, map[string]any{"project": key, "serverUrl": e.ServerURL}))
			})
	}
}

func filterNonInherited(items []json.RawMessage) []json.RawMessage {
	var filtered []json.RawMessage
	for _, s := range items {
		if !extractBool(s, "inherited") {
			filtered = append(filtered, s)
		}
	}
	return filtered
}

func projectMeasuresTask() func(ctx context.Context, e *Executor) error {
	return func(ctx context.Context, e *Executor) error {
		return forEachDep(ctx, e, "getProjectMeasures", "getProjects",
			func(ctx context.Context, item json.RawMessage, w *ChunkWriter) error {
				key := extractField(item, "key")
				items, err := e.Raw.GetArray(ctx, "api/measures/search", "measures",
					url.Values{"projectKeys": {key}, "metricKeys": {measureMetricKeys}})
				if err != nil {
					return err
				}
				return w.WriteChunk(enrichAll(items, map[string]any{"projectKey": key, "serverUrl": e.ServerURL}))
			})
	}
}

func projectBindingsTask() func(ctx context.Context, e *Executor) error {
	return func(ctx context.Context, e *Executor) error {
		return forEachDep(ctx, e, "getProjectBindings", "getProjects",
			func(ctx context.Context, item json.RawMessage, w *ChunkWriter) error {
				key := extractField(item, "key")
				raw, err := e.Raw.Get(ctx, "api/alm_settings/get_binding",
					url.Values{"project": {key}})
				if err != nil {
					return nil // Binding may not exist; skip gracefully.
				}
				return w.WriteOne(EnrichRaw(raw, map[string]any{"projectKey": key, "serverUrl": e.ServerURL}))
			})
	}
}

func projectGroupsPermissionsTask() func(ctx context.Context, e *Executor) error {
	return func(ctx context.Context, e *Executor) error {
		return forEachDep(ctx, e, "getProjectGroupsPermissions", "getProjects",
			func(ctx context.Context, item json.RawMessage, w *ChunkWriter) error {
				key := extractField(item, "key")
				meta := map[string]any{"project": key, "serverUrl": e.ServerURL}
				return fetchAllProjectPermissions(ctx, e, w, key, meta)
			})
	}
}

func fetchAllProjectPermissions(ctx context.Context, e *Executor, w *ChunkWriter, key string, meta map[string]any) error {
	for _, perm := range projectPermissions {
		items, err := e.Raw.GetPaginated(ctx, PaginatedOpts{
			Path: "api/permissions/groups", ResultKey: "groups", MaxPageSize: 100,
			Params: url.Values{"projectKey": {key}, "permission": {perm}},
		})
		if err != nil {
			return err
		}
		if err := w.WriteChunk(enrichAll(items, meta)); err != nil {
			return err
		}
	}
	return nil
}
