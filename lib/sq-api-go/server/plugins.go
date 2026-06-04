// Copyright (C) SonarSource Sàrl
// For more information, see https://sonarsource.com/legal/
// mailto:info AT sonarsource DOT com

package server

import (
	"context"

	"github.com/sonar-solutions/sq-api-go/types"
)

// PluginsClient provides methods for /api/plugins endpoints.
type PluginsClient struct{ baseClient }

// Installed returns all installed plugins from /api/plugins/installed.
func (p *PluginsClient) Installed(ctx context.Context) ([]types.Plugin, error) {
	var result types.PluginsInstalledResponse
	if err := p.get(ctx, "api/plugins/installed", nil, &result); err != nil {
		return nil, err
	}
	return result.Plugins, nil
}
