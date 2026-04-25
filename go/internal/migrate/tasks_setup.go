package migrate

import "context"

// setupTasks returns tasks that load CSV mappings into JSONL for the migrate pipeline.
func setupTasks() []TaskDef {
	return []TaskDef{
		{
			Name: "generateProjectMappings",
			Run: func(ctx context.Context, e *Executor) error {
				return loadCSVToJSONL(e, "generateProjectMappings", "projects.csv")
			},
		},
		{
			Name: "generateProfileMappings",
			Run: func(ctx context.Context, e *Executor) error {
				return loadCSVToJSONL(e, "generateProfileMappings", "profiles.csv")
			},
		},
		{
			Name: "generateGateMappings",
			Run: func(ctx context.Context, e *Executor) error {
				return loadCSVToJSONL(e, "generateGateMappings", "gates.csv")
			},
		},
		{
			Name: "generateGroupMappings",
			Run: func(ctx context.Context, e *Executor) error {
				return loadCSVToJSONL(e, "generateGroupMappings", "groups.csv")
			},
		},
		{
			Name: "generateTemplateMappings",
			Run: func(ctx context.Context, e *Executor) error {
				return loadCSVToJSONL(e, "generateTemplateMappings", "templates.csv")
			},
		},
		{
			Name: "generatePortfolioMappings",
			Run: func(ctx context.Context, e *Executor) error {
				return loadCSVToJSONL(e, "generatePortfolioMappings", "portfolios.csv")
			},
		},
		{
			Name: "generateOrganizationMappings",
			Run: func(ctx context.Context, e *Executor) error {
				return loadCSVToJSONL(e, "generateOrganizationMappings", "organizations.csv")
			},
		},
	}
}
