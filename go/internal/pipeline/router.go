// Copyright (C) SonarSource Sàrl
// For more information, see https://sonarsource.com/legal/
// mailto:info AT sonarsource DOT com

package pipeline

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	sqapi "github.com/sonar-solutions/sq-api-go"
)

// DetectPipeline reads the server version from /api/server/version, selects
// the appropriate Pipeline implementation, logs the selection at INFO level,
// and returns the pipeline. Returns an error if the server version is below
// the minimum supported (9.9).
func DetectPipeline(ctx context.Context, client *sqapi.Client) (Pipeline, error) {
	versionStr, err := detectVersionString(ctx, client)
	if err != nil {
		return nil, fmt.Errorf("detecting server version: %w", err)
	}

	major, minor, err := parseVersion(versionStr)
	if err != nil {
		return nil, fmt.Errorf("parsing version %q: %w", versionStr, err)
	}

	p, err := selectPipeline(major, minor, client)
	if err != nil {
		return nil, err
	}

	slog.Info("Detected SonarQube Server, using pipeline", "version", versionStr, "pipeline", p.Version())
	return p, nil
}

// selectPipeline maps a parsed (major, minor) version pair to the correct
// Pipeline implementation. Extracted for unit testability without HTTP.
func selectPipeline(major, minor int, client *sqapi.Client) (Pipeline, error) {
	switch {
	case major >= 2025:
		return newSQ2025(client), nil
	case major >= 11:
		// Forward compatibility: unexpected major version, fall back to SQ 10.4 pipeline.
		slog.Warn("unexpected SonarQube major version, falling back to SQ 10.4 pipeline",
			"major", major, "minor", minor)
		return newSQ104(client), nil
	case major == 10 && minor >= 4:
		return newSQ104(client), nil
	case major == 10 && minor >= 0:
		return newSQ100(client), nil
	case major == 9 && minor == 9:
		return newSQ99(client), nil
	default:
		return nil, fmt.Errorf("unsupported SonarQube Server version: %d.%d (minimum supported: 9.9)", major, minor)
	}
}

// detectVersionString fetches the raw version string from /api/server/version.
func detectVersionString(ctx context.Context, client *sqapi.Client) (string, error) {
	u := client.BaseURL() + "api/server/version"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return "", fmt.Errorf("building version request: %w", err)
	}
	resp, err := client.HTTPClient().Do(req)
	if err != nil {
		return "", fmt.Errorf("fetching /api/server/version: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("/api/server/version returned HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading version response: %w", err)
	}
	return strings.TrimSpace(string(body)), nil
}

// parseVersion parses a SonarQube version string (e.g. "10.4.1.87632",
// "2025.1.0", "9.9.0.65466", "10.5-SNAPSHOT") into major and minor integers.
func parseVersion(versionStr string) (major, minor int, err error) {
	// Strip any pre-release suffix: "10.5-SNAPSHOT" → "10.5"
	clean := strings.SplitN(versionStr, "-", 2)[0]

	parts := strings.Split(clean, ".")
	if len(parts) < 2 {
		return 0, 0, fmt.Errorf("expected at least major.minor in version string, got %q", versionStr)
	}
	major, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("parsing major component %q from version %q: %w", parts[0], versionStr, err)
	}
	minor, err = strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("parsing minor component %q from version %q: %w", parts[1], versionStr, err)
	}
	return major, minor, nil
}
