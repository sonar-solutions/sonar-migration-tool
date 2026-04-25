package server

import (
	"context"

	sqapi "github.com/sonar-solutions/sq-api-go"
	"github.com/sonar-solutions/sq-api-go/types"
)

// SystemClient provides methods for the /api/system and /api/server endpoints.
type SystemClient struct{ baseClient }

// Info returns the full system info from /api/system/info.
// The response is a deeply nested, version-varying JSON object returned as a
// map of raw JSON values. Callers can unmarshal specific sub-objects as needed.
func (s *SystemClient) Info(ctx context.Context) (types.SystemInfo, error) {
	var result types.SystemInfo
	if err := s.get(ctx, "api/system/info", nil, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// Version fetches the plain-text server version from /api/server/version and
// parses it into a float64 (e.g. "10.7.0.123" → 10.7) using ParseServerVersion.
func (s *SystemClient) Version(ctx context.Context) (float64, error) {
	data, err := s.getBytes(ctx, "api/server/version", nil)
	if err != nil {
		return 0, err
	}
	return sqapi.ParseServerVersion(string(data))
}
