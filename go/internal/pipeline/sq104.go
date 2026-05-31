package pipeline

import (
	"context"

	sqapi "github.com/sonar-solutions/sq-api-go"
)

// SQ104Pipeline handles SonarQube Server 10.4 through 10.8.
//   - Uses modern "issueStatuses" parameter (OPEN, CONFIRMED, FALSE_POSITIVE, ACCEPTED, FIXED)
//   - Batches metricKeys at 15 per request
//   - Native Clean Code attributes in API responses
//
// Note: "issueStatuses" was introduced in SQ 10.2 but became the recommended
// parameter in 10.4. For simplicity, the SQ 10.0-10.3 pipeline uses the legacy
// "statuses" parameter throughout that range.
//
// ExtractHotspots, ExtractGroups, and EnrichCleanCode are promoted from
// standardPipeline (shared across all pipeline versions).
type SQ104Pipeline struct {
	standardPipeline
}

func newSQ104(client *sqapi.Client) *SQ104Pipeline {
	return &SQ104Pipeline{standardPipeline: standardPipeline{client: client}}
}

var _ Pipeline = (*SQ104Pipeline)(nil)

func (p *SQ104Pipeline) Version() string { return "sq-10.4" }

func (p *SQ104Pipeline) IssueSearchParam() string { return "issueStatuses" }

func (p *SQ104Pipeline) IssueStatusValues() []string {
	return []string{"OPEN", "CONFIRMED", "FALSE_POSITIVE", "ACCEPTED", "FIXED"}
}

func (p *SQ104Pipeline) SupportsMetricBatching() (bool, int) { return true, 15 }

func (p *SQ104Pipeline) ExtractIssues(ctx context.Context, projectKey string) ([]Issue, error) {
	return fetchAllIssues(ctx, p.client, projectKey, p.IssueSearchParam(), p.IssueStatusValues())
}

func (p *SQ104Pipeline) ExtractMetrics(ctx context.Context, projectKey string, metricKeys []string) ([]ComponentMetrics, error) {
	_, batchSize := p.SupportsMetricBatching()
	return fetchAllMetrics(ctx, p.client, projectKey, metricKeys, batchSize)
}
