package pipeline

import (
	"context"

	sqapi "github.com/sonar-solutions/sq-api-go"
)

// SQ99Pipeline handles SonarQube Server 9.9 LTS.
//   - Uses legacy "statuses" parameter (OPEN, CONFIRMED, REOPENED, RESOLVED, CLOSED)
//   - Batches metricKeys at 15 per request
//   - Clean Code enrichment from SonarQube Cloud (SPEC-012, not yet implemented)
type SQ99Pipeline struct {
	client *sqapi.Client
}

func newSQ99(client *sqapi.Client) *SQ99Pipeline { return &SQ99Pipeline{client: client} }

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

func (p *SQ99Pipeline) ExtractHotspots(ctx context.Context, projectKey string) ([]Hotspot, error) {
	return fetchAllHotspots(ctx, p.client, projectKey)
}

func (p *SQ99Pipeline) ExtractMetrics(ctx context.Context, projectKey string, metricKeys []string) ([]ComponentMetrics, error) {
	_, batchSize := p.SupportsMetricBatching()
	return fetchAllMetrics(ctx, p.client, projectKey, metricKeys, batchSize)
}

func (p *SQ99Pipeline) ExtractGroups(ctx context.Context) ([]Group, error) {
	return fetchAllGroups(ctx, p.client)
}

// EnrichCleanCode enriches issues with Clean Code attributes from SonarQube Cloud.
// SQ 9.9 does not expose Clean Code attributes natively; they must be fetched
// from the equivalent rules on SonarQube Cloud (SPEC-012).
// This is a stub pending SPEC-012 implementation.
func (p *SQ99Pipeline) EnrichCleanCode(_ context.Context, issues []Issue, _ *sqapi.Client) ([]Issue, error) {
	return issues, nil
}
