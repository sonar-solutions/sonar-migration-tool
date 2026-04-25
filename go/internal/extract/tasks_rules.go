package extract

import (
	"context"
	"encoding/json"
	"net/url"
)

// standardRepos lists built-in rule repositories to exclude from plugin rules.
var standardRepos = map[string]bool{
	"common-cs": true, "common-java": true, "common-js": true, "common-ts": true,
	"common-php": true, "common-py": true, "common-web": true,
	"csharpsquid": true, "flex": true, "go": true, "java": true,
	"javascript": true, "kotlin": true, "php": true, "python": true,
	"ruby": true, "scala": true, "swift": true, "typescript": true,
	"vbnet": true, "web": true, "xml": true, "css": true,
	"cloudformation": true, "docker": true, "kubernetes": true,
	"terraform": true, "azureresourcemanager": true, "text": true,
	"secrets": true, "jssecurity": true, "javasecurity": true,
	"phpsecurity": true, "pythonsecurity": true, "roslyn.sonaranalyzer.security.cs": true,
	"common-abap": true, "abap": true, "common-apex": true, "apex": true,
	"common-cobol": true, "cobol": true, "common-pli": true, "pli": true,
	"common-rpg": true, "rpg": true, "common-vb": true, "vb": true,
	"common-tsql": true, "tsql": true, "plsql": true, "common-objc": true,
	"objc": true, "common-c": true, "c": true, "cpp": true,
}

func ruleTasks() []TaskDef {
	return []TaskDef{
		{Name: "getRules", Editions: AllEditions,
			Run: getRulesTask()},
		{Name: "getRepos", Editions: AllEditions,
			Run: getReposTask()},
		{Name: "getProfileRules", Editions: AllEditions,
			Run: getProfileRulesTask()},
		{Name: "getRuleDetails", Editions: AllEditions, Dependencies: []string{"getRules"},
			Run: getRuleDetailsTask()},
		{Name: "getPluginRules", Editions: AllEditions, Dependencies: []string{"getRules"},
			Run: getPluginRulesTask()},
		{Name: "getTemplateRules", Editions: AllEditions, Dependencies: []string{"getRules"},
			Run: getTemplateRulesTask()},
		{Name: "getActiveProfileRules", Editions: AllEditions, Dependencies: []string{"getProfiles"},
			Run: getActiveProfileRulesTask()},
		{Name: "getDeactivatedProfileRules", Editions: AllEditions, Dependencies: []string{"getProfiles"},
			Run: getDeactivatedProfileRulesTask()},
	}
}

func getRulesTask() func(ctx context.Context, e *Executor) error {
	return func(ctx context.Context, e *Executor) error {
		return fetchAndWritePaginated(ctx, e, "getRules", PaginatedOpts{
			Path: rulesSearchAPI, ResultKey: "rules", MaxPageSize: 500,
		}, map[string]any{"serverUrl": e.ServerURL})
	}
}

func getReposTask() func(ctx context.Context, e *Executor) error {
	return func(ctx context.Context, e *Executor) error {
		return fetchAndWriteArray(ctx, e, "getRepos", "api/rules/repositories", "repositories", nil, map[string]any{"serverUrl": e.ServerURL})
	}
}

func getProfileRulesTask() func(ctx context.Context, e *Executor) error {
	return func(ctx context.Context, e *Executor) error {
		return fetchAndWritePaginated(ctx, e, "getProfileRules", PaginatedOpts{
			Path: rulesSearchAPI, ResultKey: "actives", MaxPageSize: 500,
			Params: url.Values{"f": {"actives"}, "activation": {"true"}},
		}, map[string]any{"serverUrl": e.ServerURL})
	}
}

func getRuleDetailsTask() func(ctx context.Context, e *Executor) error {
	return func(ctx context.Context, e *Executor) error {
		return forEachDep(ctx, e, "getRuleDetails", "getRules",
			func(ctx context.Context, item json.RawMessage, w *ChunkWriter) error {
				ruleKey := extractField(item, "key")
				raw, err := e.Raw.Get(ctx, "api/rules/show",
					url.Values{"key": {ruleKey}})
				if err != nil {
					return err
				}
				raw = extractSubKey(raw, "rule")
				return w.WriteOne(EnrichRaw(raw, map[string]any{"serverUrl": e.ServerURL}))
			})
	}
}

func getPluginRulesTask() func(ctx context.Context, e *Executor) error {
	return func(ctx context.Context, e *Executor) error {
		return forEachDepFiltered(ctx, e, "getPluginRules", "getRules",
			func(item json.RawMessage) bool {
				return !standardRepos[extractField(item, "repo")]
			},
			func(ctx context.Context, item json.RawMessage, w *ChunkWriter) error {
				ruleKey := extractField(item, "key")
				return w.WriteOne(EnrichRaw(item, map[string]any{"key": ruleKey, "serverUrl": e.ServerURL}))
			})
	}
}

func getTemplateRulesTask() func(ctx context.Context, e *Executor) error {
	return func(ctx context.Context, e *Executor) error {
		return forEachDepFiltered(ctx, e, "getTemplateRules", "getRules",
			func(item json.RawMessage) bool {
				return extractField(item, "templateKey") != ""
			},
			func(ctx context.Context, item json.RawMessage, w *ChunkWriter) error {
				ruleKey := extractField(item, "key")
				return w.WriteOne(EnrichRaw(item, map[string]any{"key": ruleKey, "serverUrl": e.ServerURL}))
			})
	}
}

func getActiveProfileRulesTask() func(ctx context.Context, e *Executor) error {
	return func(ctx context.Context, e *Executor) error {
		return forEachDep(ctx, e, "getActiveProfileRules", "getProfiles",
			func(ctx context.Context, item json.RawMessage, w *ChunkWriter) error {
				profileKey := extractField(item, "key")
				items, err := e.Raw.GetPaginated(ctx, PaginatedOpts{
					Path: rulesSearchAPI, ResultKey: "rules", MaxPageSize: 500,
					Params: url.Values{
						"inheritance": {"NONE"},
						"qprofile":    {profileKey},
						"activation":  {"true"},
					},
				})
				if err != nil {
					return err
				}
				return w.WriteChunk(enrichAll(items, map[string]any{"profileKey": profileKey, "serverUrl": e.ServerURL}))
			})
	}
}

func getDeactivatedProfileRulesTask() func(ctx context.Context, e *Executor) error {
	return func(ctx context.Context, e *Executor) error {
		return forEachDepFiltered(ctx, e, "getDeactivatedProfileRules", "getProfiles",
			func(item json.RawMessage) bool {
				return extractField(item, "parentKey") != ""
			},
			func(ctx context.Context, item json.RawMessage, w *ChunkWriter) error {
				profileKey := extractField(item, "key")
				parentKey := extractField(item, "parentKey")
				items, err := e.Raw.GetPaginated(ctx, PaginatedOpts{
					Path: rulesSearchAPI, ResultKey: "rules", MaxPageSize: 500,
					Params: url.Values{
						"compareToProfile": {parentKey},
						"qprofile":         {profileKey},
						"activation":       {"false"},
					},
				})
				if err != nil {
					return err
				}
				return w.WriteChunk(enrichAll(items, map[string]any{"profileKey": profileKey, "serverUrl": e.ServerURL}))
			})
	}
}

// extractSubKey extracts a sub-object from raw JSON by key, returning the original if not found.
func extractSubKey(raw json.RawMessage, key string) json.RawMessage {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err == nil {
		if sub, ok := obj[key]; ok {
			return sub
		}
	}
	return raw
}
