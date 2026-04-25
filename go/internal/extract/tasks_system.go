package extract

import (
	"context"
	"net/url"
)

func systemTasks() []TaskDef {
	return []TaskDef{
		{
			Name:     "getServerInfo",
			Editions: AllEditions,
			Run: func(ctx context.Context, e *Executor) error {
				return fetchAndWriteSingle(ctx, e, "getServerInfo", "api/system/info", nil, "", map[string]any{"serverUrl": e.ServerURL})
			},
		},
		{
			Name:     "getServerSettings",
			Editions: AllEditions,
			Run: func(ctx context.Context, e *Executor) error {
				return fetchAndWriteArray(ctx, e, "getServerSettings", "api/settings/values", "settings", nil, map[string]any{"serverUrl": e.ServerURL})
			},
		},
		{
			Name:     "getPlugins",
			Editions: AllEditions,
			Run: func(ctx context.Context, e *Executor) error {
				return fetchAndWriteArray(ctx, e, "getPlugins", "api/plugins/installed", "plugins",
					url.Values{"type": {"EXTERNAL"}}, map[string]any{"serverUrl": e.ServerURL})
			},
		},
		{
			Name:     "getUsage",
			Editions: []Edition{EditionDeveloper, EditionEnterprise, EditionDatacenter},
			Run: func(ctx context.Context, e *Executor) error {
				return fetchAndWriteArray(ctx, e, "getUsage", "api/projects/license_usage", "projects", nil, nil)
			},
		},
		{
			Name:     "getBindings",
			Editions: AllEditions,
			Run: func(ctx context.Context, e *Executor) error {
				return fetchAndWriteArray(ctx, e, "getBindings", "api/alm_settings/list", "almSettings", nil, map[string]any{"serverUrl": e.ServerURL})
			},
		},
	}
}
