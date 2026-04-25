package extract

import (
	"context"
	"encoding/json"
	"net/url"
)

func gateTasks() []TaskDef {
	return []TaskDef{
		{
			Name:     "getGates",
			Editions: AllEditions,
			Run: func(ctx context.Context, e *Executor) error {
				return fetchAndWriteArray(ctx, e, "getGates", "api/qualitygates/list", "qualitygates", nil, map[string]any{"serverUrl": e.ServerURL})
			},
		},
		{
			Name: "getGateConditions", Editions: AllEditions, Dependencies: []string{"getGates"},
			Run: func(ctx context.Context, e *Executor) error {
				return forEachDepFiltered(ctx, e, "getGateConditions", "getGates", notBuiltIn,
					func(ctx context.Context, item json.RawMessage, w *ChunkWriter) error {
						name := extractField(item, "name")
						raw, err := e.Raw.Get(ctx, "api/qualitygates/show", url.Values{"name": {name}})
						if err != nil {
							return err
						}
						conditions, _ := extractArray(raw, "conditions")
						return w.WriteChunk(enrichAll(conditions, map[string]any{"gateName": name, "serverUrl": e.ServerURL}))
					})
			},
		},
		{Name: "getGateGroups", Editions: AllEditions, Dependencies: []string{"getGates"},
			Run: perFilteredDepPaginated("getGateGroups", "getGates", "api/qualitygates/search_groups", "groups", "gateName", "name", 1000)},
		{Name: "getGateUsers", Editions: AllEditions, Dependencies: []string{"getGates"},
			Run: perFilteredDepPaginated("getGateUsers", "getGates", "api/qualitygates/search_users", "users", "gateName", "name", 1000)},
	}
}

// notBuiltIn is a filter func that skips items where isBuiltIn is true.
func notBuiltIn(item json.RawMessage) bool {
	return !extractBool(item, "isBuiltIn")
}

// perFilteredDepPaginated is a helper for per-dep tasks that filter out built-in items
// and fetch paginated results using a name parameter.
func perFilteredDepPaginated(taskName, depTask, path, resultKey, metaKey, nameField string, maxPageSize int) func(ctx context.Context, e *Executor) error {
	return func(ctx context.Context, e *Executor) error {
		return forEachDepFiltered(ctx, e, taskName, depTask, notBuiltIn,
			func(ctx context.Context, item json.RawMessage, w *ChunkWriter) error {
				name := extractField(item, nameField)
				items, err := e.Raw.GetPaginated(ctx, PaginatedOpts{
					Path: path, ResultKey: resultKey, MaxPageSize: maxPageSize,
					Params: url.Values{metaKey: {name}},
				})
				if err != nil {
					return err
				}
				return w.WriteChunk(enrichAll(items, map[string]any{metaKey: name, "serverUrl": e.ServerURL}))
			})
	}
}
