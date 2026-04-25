package migrate

import (
	"context"
	"encoding/json"

	"github.com/sonar-solutions/sq-api-go/cloud"
	"github.com/sonar-solutions/sonar-migration-tool/internal/common"
)

// portfolioTasks returns tasks for Enterprise portfolio management.
func portfolioTasks() []TaskDef {
	entEditions := []common.Edition{common.EditionEnterprise, common.EditionDatacenter}

	return []TaskDef{
		{
			Name:         "setPortfolioProjects",
			Editions:     entEditions,
			Dependencies: []string{"createPortfolios", "createProjects"},
			Run: func(ctx context.Context, e *Executor) error {
				// Build project key lookup from created projects.
				projects, _ := e.Store.ReadAll("createProjects")
				projectKeys := make(map[string]string) // server_url+key → cloud_project_key
				for _, p := range projects {
					serverURL := extractField(p, "server_url")
					key := extractField(p, "key")
					cloudKey := extractField(p, "cloud_project_key")
					projectKeys[serverURL+key] = cloudKey
				}

				// Read portfolio project associations from extract data.
				portfolioItems, _ := readExtractItems(e, "getPortfolioProjects")

				// Group by portfolio.
				portfolioProjects := make(map[string][]string) // source_portfolio_key → []cloud_project_key
				for _, item := range portfolioItems {
					portfolioKey := extractField(item.Data, "portfolioKey")
					refKey := extractField(item.Data, "refKey")
					cloudKey, ok := projectKeys[item.ServerURL+refKey]
					if !ok {
						continue
					}
					portfolioProjects[portfolioKey] = append(portfolioProjects[portfolioKey], cloudKey)
				}

				// Match with created portfolios.
				return forEachMigrateItem(ctx, e, "setPortfolioProjects", "createPortfolios",
					func(ctx context.Context, item json.RawMessage, w *common.ChunkWriter) error {
						portfolioID := extractField(item, "cloud_portfolio_id")
						sourceKey := extractField(item, "source_portfolio_key")
						projects, ok := portfolioProjects[sourceKey]
						if !ok || len(projects) == 0 {
							return nil
						}

						err := e.CloudAPI.Enterprises.UpdatePortfolio(ctx, cloud.UpdatePortfolioParams{
							PortfolioID: portfolioID,
							Projects:    projects,
						})
						if err != nil {
							e.Logger.Warn("setPortfolioProjects failed",
								"portfolio", portfolioID, "err", err)
						}
						return nil
					})
			},
		},
	}
}
