// Copyright (C) SonarSource Sàrl
// For more information, see https://sonarsource.com/legal/
// mailto:info AT sonarsource DOT com

package summary

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-pdf/fpdf"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sonar-solutions/sonar-migration-tool/internal/analysis"
	"github.com/sonar-solutions/sonar-migration-tool/internal/migrate"
	sqapi "github.com/sonar-solutions/sq-api-go"
)

func writeRateLimitArtefact(t *testing.T, dir string, state migrate.RateLimitState) {
	t.Helper()
	data, err := json.Marshal(state)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, migrate.RateLimitEventsFile), data, 0o600))
}

func TestCollectRateLimitReport_NoFileReturnsNil(t *testing.T) {
	dir := t.TempDir()
	got := collectRateLimitReport(dir, nil)
	assert.Nil(t, got)
}

func TestCollectRateLimitReport_SingleBlipBelowThreshold(t *testing.T) {
	dir := t.TempDir()
	writeRateLimitArtefact(t, dir, migrate.RateLimitState{
		Total:                  1,
		Counts:                 map[string]int{sqapi.KindSQCRateLimit.String(): 1},
		CumulativePauseSeconds: 5,
		FirstByKind: map[string]migrate.FirstEventSnapshot{
			sqapi.KindSQCRateLimit.String(): {ObservedAt: time.Now()},
		},
	})
	got := collectRateLimitReport(dir, nil)
	assert.Nil(t, got, "single 5s SQC blip below 30s threshold should not surface")
}

func TestCollectRateLimitReport_CumulativeAboveThreshold(t *testing.T) {
	dir := t.TempDir()
	writeRateLimitArtefact(t, dir, migrate.RateLimitState{
		Total:                  3,
		Counts:                 map[string]int{sqapi.KindSQCRateLimit.String(): 3},
		CumulativePauseSeconds: 65,
		FirstByKind: map[string]migrate.FirstEventSnapshot{
			sqapi.KindSQCRateLimit.String(): {ObservedAt: time.Now()},
		},
	})
	got := collectRateLimitReport(dir, nil)
	require.NotNil(t, got)
	assert.Equal(t, 3, got.SQCHits)
	assert.InDelta(t, 65.0, got.CumulativePauseSeconds, 0.001)
	assert.False(t, got.CausedTaskFailure)
}

func TestCollectRateLimitReport_TaskFailureWith429(t *testing.T) {
	dir := t.TempDir()
	writeRateLimitArtefact(t, dir, migrate.RateLimitState{
		Total:                  2,
		Counts:                 map[string]int{sqapi.KindSQCRateLimit.String(): 2},
		CumulativePauseSeconds: 2,
		FirstByKind: map[string]migrate.FirstEventSnapshot{
			sqapi.KindSQCRateLimit.String(): {ObservedAt: time.Now()},
		},
	})
	failures := map[string][]analysis.ReportRow{
		"Project": {{HTTPStatus: "429"}},
	}
	got := collectRateLimitReport(dir, failures)
	require.NotNil(t, got)
	assert.True(t, got.CausedTaskFailure)
}

func TestCollectRateLimitReport_CloudflareHitAlwaysSurfaces(t *testing.T) {
	dir := t.TempDir()
	writeRateLimitArtefact(t, dir, migrate.RateLimitState{
		Total: 1,
		Counts: map[string]int{
			sqapi.KindCloudflareRateLimit.String(): 1,
		},
		CumulativePauseSeconds: 0,
		FirstByKind: map[string]migrate.FirstEventSnapshot{
			sqapi.KindCloudflareRateLimit.String(): {
				ObservedAt:  time.Now(),
				BodySnippet: "<html>1015</html>",
				Headers:     map[string]string{"CF-Ray": "abc", "Server": "cloudflare"},
			},
		},
	})
	got := collectRateLimitReport(dir, nil)
	require.NotNil(t, got, "any non-SQC 429 must always surface, regardless of pause time")
	assert.Equal(t, 1, got.CloudflareHits)
	assert.Equal(t, "<html>1015</html>", got.FirstBodySnippet)
	assert.Contains(t, got.FirstHeadersSummary, "CF-Ray: abc")
	assert.Contains(t, got.FirstHeadersSummary, "Server: cloudflare")
}

func TestRateLimitMessageVariants(t *testing.T) {
	cases := []struct {
		name     string
		report   *RateLimitReport
		contains []string
	}{
		{
			name: "sqc recovered slow",
			report: &RateLimitReport{
				TotalHits: 4, SQCHits: 4,
				CumulativePauseSeconds: 95,
			},
			contains: []string{"SonarQube Cloud", "4 times", "paused and resumed", "minute"},
		},
		{
			name: "sqc task failure",
			report: &RateLimitReport{
				TotalHits: 2, SQCHits: 2,
				CausedTaskFailure: true,
			},
			contains: []string{"limit did not clear", "tasks failed", "--run-id"},
		},
		{
			name: "cloudflare hit",
			report: &RateLimitReport{
				TotalHits: 1, CloudflareHits: 1,
				FirstBodySnippet: "<html>1015</html>",
			},
			contains: []string{"non-standard 429", "upstream proxy or WAF", "operator action", "1015"},
		},
		{
			name: "unknown 429",
			report: &RateLimitReport{
				TotalHits: 1, UnknownHits: 1,
				FirstBodySnippet: "  body  with  spaces  ",
			},
			contains: []string{"non-standard 429", "body with spaces"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			msg := rateLimitMessage(tc.report)
			for _, want := range tc.contains {
				assert.Contains(t, msg, want, "expected %q in message", want)
			}
		})
	}
}

func TestFormatHeaders(t *testing.T) {
	got := formatHeaders(map[string]string{
		"Server": "cloudflare",
		"CF-Ray": "abc",
	})
	assert.Equal(t, "CF-Ray: abc; Server: cloudflare", got)
	assert.Empty(t, formatHeaders(nil))
	assert.Empty(t, formatHeaders(map[string]string{}))
}

func TestFormatSeconds(t *testing.T) {
	assert.Equal(t, "5.0 seconds", formatSeconds(5))
	assert.Equal(t, "29.5 seconds", formatSeconds(29.5))
	assert.Equal(t, "55.0 seconds", formatSeconds(55))   // still under the 60s threshold
	assert.Equal(t, "about 1 minute", formatSeconds(70)) // rounds to 1 min
	assert.Equal(t, "about 5 minutes", formatSeconds(300))
}

func TestSnippetForMessage(t *testing.T) {
	assert.Equal(t, "(empty response body)", snippetForMessage(""))
	assert.Equal(t, "(empty response body)", snippetForMessage("   "))
	assert.Equal(t, "compact body", snippetForMessage("  compact   body  "))

	long := strings.Repeat("a", 250)
	got := snippetForMessage(long)
	assert.LessOrEqual(t, len(got), 165, "long snippets should be truncated with an ellipsis")
}

func TestRenderRateLimitWarningProducesContent(t *testing.T) {
	pdf := fpdf.New("P", "mm", "Letter", "")
	pdf.SetAutoPageBreak(true, 20)
	registerUnicodeFont(pdf)
	pdf.AddPage()
	pdf.SetY(40)

	report := &RateLimitReport{
		TotalHits: 5, SQCHits: 5,
		CumulativePauseSeconds: 120, CausedTaskFailure: true,
	}
	startY := pdf.GetY()
	renderRateLimitWarning(pdf, report)
	endY := pdf.GetY()

	assert.Greater(t, endY, startY, "rendering must advance the Y cursor")

	var buf bytes.Buffer
	require.NoError(t, pdf.Output(&buf))
	assert.Greater(t, buf.Len(), 1000, "PDF must be non-trivial after rendering the warning")
}

// TestRenderRateLimitWarningNearPageBottomBreaksCleanly verifies that
// when the warning is rendered near the bottom of a page, the entire
// box (including a long non-SQC body that wraps to many lines) moves to
// a fresh page rather than being drawn with its border spanning the
// page break.
func TestRenderRateLimitWarningNearPageBottomBreaksCleanly(t *testing.T) {
	pdf := fpdf.New("P", "mm", "Letter", "")
	pdf.SetAutoPageBreak(true, 20)
	registerUnicodeFont(pdf)
	pdf.AddPage()

	_, pageH := pdf.GetPageSize()
	// Park the cursor near the bottom so the old "reserve 4 lines"
	// constant would fit-and-overflow, while a long-body box correctly
	// computed should push to a new page.
	pdf.SetY(pageH - 40)
	startPage := pdf.PageNo()

	report := &RateLimitReport{
		TotalHits:           1,
		CloudflareHits:      1,
		FirstBodySnippet:    strings.Repeat("Cloudflare 1015 Error Encountered ", 8),
		FirstHeadersSummary: "CF-Ray: abc; Retry-After: 60",
	}
	renderRateLimitWarning(pdf, report)

	assert.Greater(t, pdf.PageNo(), startPage,
		"long body near page bottom must trigger a page break before drawing the box")

	var buf bytes.Buffer
	require.NoError(t, pdf.Output(&buf), "rendering must produce a valid PDF")
}
