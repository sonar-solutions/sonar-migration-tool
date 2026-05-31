package pipeline

import (
	"context"

	sqapi "github.com/sonar-solutions/sq-api-go"
)

// standardPipeline is an embeddable struct that provides shared method
// implementations for pipeline versions that use the standard groups API
// and have no-op Clean Code enrichment (SQ 9.9, 10.0, 10.4, 2025).
// SQ 2025 overrides ExtractGroups with the V2 API.
type standardPipeline struct {
	client *sqapi.Client
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
