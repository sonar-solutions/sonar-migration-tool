package pipeline

import (
	"context"

	sqapi "github.com/sonar-solutions/sq-api-go"
)

// SQ99Pipeline handles SonarQube Server 9.9 LTS.
//   - Uses legacy "statuses" parameter (OPEN, CONFIRMED, REOPENED, RESOLVED, CLOSED)
//   - Batches metricKeys at 15 per request
//   - Clean Code enrichment from SonarQube Cloud (SPEC-012, not yet implemented)
//
// ExtractHotspots, ExtractGroups, and EnrichCleanCode are promoted from
// standardPipeline (shared across all pipeline versions).
type SQ99Pipeline struct {
	standardPipeline
}

func newSQ99(client *sqapi.Client) *SQ99Pipeline {
	return &SQ99Pipeline{standardPipeline: standardPipeline{client: client}}
}

// Compile-time interface check.
var _ Pipeline = (*SQ99Pipeline)(nil)

func (p *SQ99Pipeline) Version() string { return "sq-9.9" }

func (p *SQ99Pipeline) IssueSearchParam() string { return "statuses" }

func (p *SQ99Pipeline) IssueStatusValues() []string {
	return []string{"OPEN", "CONFIRMED", "REOPENED", "RESOLVED", "CLOSED"}
}

func (p *SQ99Pipeline) SupportsMetricBatching() (bool, int) { return true, 15 }

func (p *SQ99Pipeline) ExtractIssues(ctx context.Context, projectKey string) ([]Issue, error) {
	return fetchAllIssues(ctx, p.client, projectKey, p.IssueSearchParam(), p.IssueStatusValues())
}

func (p *SQ99Pipeline) ExtractMetrics(ctx context.Context, projectKey string, metricKeys []string) ([]ComponentMetrics, error) {
	_, batchSize := p.SupportsMetricBatching()
	return fetchAllMetrics(ctx, p.client, projectKey, metricKeys, batchSize)
}
