package maturity

import (
	"github.com/sonar-solutions/sonar-migration-tool/internal/report"
	"github.com/sonar-solutions/sonar-migration-tool/internal/report/common"
	"github.com/sonar-solutions/sonar-migration-tool/internal/structure"
)

// GeneratePortfolioSummaryMarkdown generates the portfolio summary for the maturity report.
func GeneratePortfolioSummaryMarkdown(dir string, mapping structure.ExtractMapping, idMap common.ServerIDMapping) string {
	portfolios := common.ProcessPortfolios(dir, mapping, idMap)
	allProjects := make(map[string]bool)
	for _, p := range portfolios {
		if projects, ok := p["projects"].(map[string]bool); ok {
			for k := range projects {
				allProjects[k] = true
			}
		}
	}
	return report.GenerateSection(
		[]report.Column{
			{"Portfolios", "portfolios"}, {"Total Projects in Portfolios", "project_count"},
		},
		[]map[string]any{{"portfolios": len(portfolios), "project_count": len(allProjects)}},
		report.WithTitle("Portfolios", 3),
	)
}
