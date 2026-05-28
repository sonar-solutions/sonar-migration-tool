package pipeline

import (
	"context"

	sqapi "github.com/sonar-solutions/sq-api-go"
)

// SQ100Pipeline handles SonarQube Server 10.0 through 10.3.
//   - Uses legacy "statuses" parameter (same values as SQ 9.9; in this range
//     FALSE_POSITIVE and ACCEPTED are resolutions, not statuses)
//   - Batches metricKeys at 15 per request
//   - Native Clean Code attributes in API responses (no Cloud enrichment needed)
type SQ100Pipeline struct {
	client *sqapi.Client
}

func newSQ100(client *sqapi.Client) *SQ100Pipeline { return &SQ100Pipeline{client: client} }

var _ Pipeline = (*SQ100Pipeline)(nil)

func (p *SQ100Pipeline) Version() string { return "sq-10.0" }

func (p *SQ100Pipeline) IssueSearchParam() string { return "statuses" }

func (p *SQ100Pipeline) IssueStatusValues() []string {
	return []string{"OPEN", "CONFIRMED", "REOPENED", "RESOLVED", "CLOSED"}
}

func (p *SQ100Pipeline) SupportsMetricBatching() (bool, int) { return true, 15 }

func (p *SQ100Pipeline) ExtractIssues(ctx context.Context, projectKey string) ([]Issue, error) {
	return fetchAllIssues(ctx, p.client, projectKey, p.IssueSearchParam(), p.IssueStatusValues())
}

func (p *SQ100Pipeline) ExtractHotspots(ctx context.Context, projectKey string) ([]Hotspot, error) {
	return fetchAllHotspots(ctx, p.client, projectKey)
}

func (p *SQ100Pipeline) ExtractMetrics(ctx context.Context, projectKey string, metricKeys []string) ([]ComponentMetrics, error) {
	_, batchSize := p.SupportsMetricBatching()
	return fetchAllMetrics(ctx, p.client, projectKey, metricKeys, batchSize)
}

func (p *SQ100Pipeline) ExtractGroups(ctx context.Context) ([]Group, error) {
	return fetchAllGroups(ctx, p.client)
}

// EnrichCleanCode is a no-op for SQ 10.0+: Clean Code attributes are natively
// present in API responses and do not require Cloud-side enrichment.
func (p *SQ100Pipeline) EnrichCleanCode(_ context.Context, issues []Issue, _ *sqapi.Client) ([]Issue, error) {
	return issues, nil
}
