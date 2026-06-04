// Copyright (C) SonarSource Sàrl
// For more information, see https://sonarsource.com/legal/
// mailto:info AT sonarsource DOT com

package common

import (
	"github.com/sonar-solutions/sonar-migration-tool/internal/report"
	"github.com/sonar-solutions/sonar-migration-tool/internal/structure"
)

// ProcessPlugins extracts external plugins from getPlugins data.
func ProcessPlugins(dir string, mapping structure.ExtractMapping, idMap ServerIDMapping) map[string][]map[string]any {
	plugins := make(map[string][]map[string]any)
	for _, item := range readData(dir, mapping, "getPlugins") {
		sid := serverID(idMap, item.ServerURL)
		pluginType := report.ExtractString(item.Data, "$.type")
		if pluginType != "EXTERNAL" {
			continue
		}
		plugins[sid] = append(plugins[sid], map[string]any{
			"server_id":    sid,
			"name":         report.ExtractString(item.Data, "$.name"),
			"description":  report.ExtractString(item.Data, "$.description"),
			"version":      report.ExtractString(item.Data, "$.version"),
			"homepage_url": report.ExtractString(item.Data, "$.homepageUrl"),
		})
	}
	return plugins
}

// GeneratePluginMarkdown generates the Installed Plugins markdown section.
func GeneratePluginMarkdown(dir string, mapping structure.ExtractMapping, idMap ServerIDMapping) (string, map[string][]map[string]any) {
	plugins := ProcessPlugins(dir, mapping, idMap)
	var rows []map[string]any
	for _, serverPlugins := range plugins {
		rows = append(rows, serverPlugins...)
	}
	md := report.GenerateSection(
		[]report.Column{
			{Header: "Server ID", Key: "server_id"}, {Header: "Plugin Name", Key: "name"}, {Header: "Description", Key: "description"},
			{Header: "Version", Key: "version"}, {Header: "Home Page URL", Key: "homepage_url"},
		},
		rows,
		report.WithTitle("Installed Plugins", 2),
	)
	return md, plugins
}
