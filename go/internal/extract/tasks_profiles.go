package extract

import (
	"context"
	"encoding/json"
	"net/url"
)

func profileTasks() []TaskDef {
	return []TaskDef{
		{
			Name:     "getProfiles",
			Editions: AllEditions,
			Run: func(ctx context.Context, e *Executor) error {
				return fetchAndWriteArray(ctx, e, "getProfiles", "api/qualityprofiles/search", "profiles", nil, map[string]any{"serverUrl": e.ServerURL})
			},
		},
		{
			Name: "getProfileBackups", Editions: AllEditions, Dependencies: []string{"getProfiles"},
			Run: func(ctx context.Context, e *Executor) error {
				return forEachDep(ctx, e, "getProfileBackups", "getProfiles",
					func(ctx context.Context, item json.RawMessage, w *ChunkWriter) error {
						name := extractField(item, "name")
						lang := extractField(item, "language")
						profileKey := extractField(item, "key")
						raw, err := e.Raw.GetRaw(ctx, "api/qualityprofiles/backup",
							url.Values{"language": {lang}, "qualityProfile": {name}})
						if err != nil {
							return err
						}
						meta := map[string]any{
							"profileName": name, "language": lang,
							"profileKey": profileKey, "serverUrl": e.ServerURL,
							"backup": string(raw),
						}
						b, _ := json.Marshal(meta)
						return w.WriteOne(b)
					})
			},
		},
		{Name: "getProfileGroups", Editions: AllEditions, Dependencies: []string{"getProfiles"},
			Run: perProfilePaginated("getProfileGroups", "api/qualityprofiles/search_groups", "groups")},
		{Name: "getProfileUsers", Editions: AllEditions, Dependencies: []string{"getProfiles"},
			Run: perProfilePaginated("getProfileUsers", "api/qualityprofiles/search_users", "users")},
	}
}

// perProfilePaginated fetches paginated results for non-built-in profiles.
func perProfilePaginated(taskName, path, resultKey string) func(ctx context.Context, e *Executor) error {
	return func(ctx context.Context, e *Executor) error {
		return forEachDepFiltered(ctx, e, taskName, "getProfiles", notBuiltIn,
			func(ctx context.Context, item json.RawMessage, w *ChunkWriter) error {
				name := extractField(item, "name")
				lang := extractField(item, "language")
				profileKey := extractField(item, "key")
				items, err := e.Raw.GetPaginated(ctx, PaginatedOpts{
					Path: path, ResultKey: resultKey,
					Params: url.Values{"qualityProfile": {name}, "language": {lang}},
				})
				if err != nil {
					return err
				}
				return w.WriteChunk(enrichAll(items, map[string]any{"profileKey": profileKey, "serverUrl": e.ServerURL}))
			})
	}
}
