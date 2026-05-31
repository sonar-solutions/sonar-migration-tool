package pipeline

import (
	"context"

	sqapi "github.com/sonar-solutions/sq-api-go"
)

// standardPipeline is an embeddable struct that provides shared method
// implementations for all pipeline versions. Query parameters and batching
// configuration are set per-version in the constructor; extraction methods
// are promoted and can be overridden (SQ 2025 overrides ExtractIssues and
// ExtractGroups).
type standardPipeline struct {
	client            *sqapi.Client
	issueSearchParam  string
	issueStatusValues []string
	metricBatchSize   int
}

func (s standardPipeline) IssueSearchParam() string   { return s.issueSearchParam }
func (s standardPipeline) IssueStatusValues() []string { return s.issueStatusValues }

func (s standardPipeline) SupportsMetricBatching() (bool, int) {
	return s.metricBatchSize > 0, s.metricBatchSize
}

func (s standardPipeline) ExtractIssues(ctx context.Context, projectKey string) ([]Issue, error) {
	return fetchAllIssues(ctx, s.client, projectKey, s.issueSearchParam, s.issueStatusValues)
}

func (s standardPipeline) ExtractMetrics(ctx context.Context, projectKey string, metricKeys []string) ([]ComponentMetrics, error) {
	return fetchAllMetrics(ctx, s.client, projectKey, metricKeys, s.metricBatchSize)
}

// ExtractHotspots paginates /api/hotspots/search and is identical across all
// pipeline versions.
func (s standardPipeline) ExtractHotspots(ctx context.Context, projectKey string) ([]Hotspot, error) {
	return fetchAllHotspots(ctx, s.client, projectKey)
}

// ExtractGroups retrieves groups via /api/user_groups/search. SQ 2025 overrides
// this with the V2 /api/v2/authorizations/groups endpoint.
func (s standardPipeline) ExtractGroups(ctx context.Context) ([]Group, error) {
	return fetchAllGroups(ctx, s.client)
}

// EnrichCleanCode is a no-op for SQ 10.0+ (Clean Code attributes are natively
// present in API responses) and a stub pending SPEC-012 for SQ 9.9.
func (s standardPipeline) EnrichCleanCode(_ context.Context, issues []Issue, _ *sqapi.Client) ([]Issue, error) {
	return issues, nil
}
