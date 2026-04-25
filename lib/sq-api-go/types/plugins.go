package types

// Plugin represents a single installed plugin returned by
// /api/plugins/installed.
type Plugin struct {
	Key                 string `json:"key"`
	Name                string `json:"name"`
	Description         string `json:"description"`
	Version             string `json:"version"`
	License             string `json:"license"`
	OrganizationName    string `json:"organizationName"`
	DocumentationPath   string `json:"documentationPath"`
	ImplementationBuild string `json:"implementationBuild"`
}

// PluginsInstalledResponse is the response envelope for /api/plugins/installed.
type PluginsInstalledResponse struct {
	Plugins []Plugin `json:"plugins"`
}
