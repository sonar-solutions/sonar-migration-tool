package server

import (
	"context"
	"net/url"
	"strings"

	"github.com/sonar-solutions/sq-api-go/types"
)

// MeasuresClient provides methods for /api/measures endpoints.
type MeasuresClient struct{ baseClient }

// Search returns measures for one or more projects from /api/measures/search.
// projectKeys and metricKeys are passed as comma-separated strings.
func (m *MeasuresClient) Search(ctx context.Context, projectKeys []string, metricKeys []string) ([]types.Measure, error) {
	params := url.Values{}
	params.Set("projectKeys", strings.Join(projectKeys, ","))
	params.Set("metricKeys", strings.Join(metricKeys, ","))

	var result types.MeasuresSearchResponse
	if err := m.get(ctx, "api/measures/search", params, &result); err != nil {
		return nil, err
	}
	return result.Measures, nil
}
