// Copyright (C) SonarSource Sàrl
// For more information, see https://sonarsource.com/legal/
// mailto:info AT sonarsource DOT com

package summary

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/go-pdf/fpdf"
	"github.com/sonar-solutions/sonar-migration-tool/internal/analysis"
	"github.com/sonar-solutions/sonar-migration-tool/internal/migrate"
	sqapi "github.com/sonar-solutions/sq-api-go"
)

// rateLimitImpactPauseThreshold is the cumulative pause time (in
// seconds) above which the orange warning is rendered even when the
// migration completed cleanly. A short blip that recovered in under
// half a minute is not worth a callout — operators see the JSON in the
// run dir if they want details.
const rateLimitImpactPauseThreshold = 30.0

// collectRateLimitReport reads rate_limit_events.json from runDir and
// returns a *RateLimitReport only when rate limiting materially
// impacted the run. Returns nil for clean runs, missing files, or
// recoverable single-blip runs.
//
// The "materially impacted" predicate is satisfied when ANY of the
// following hold:
//   - one or more tasks failed with HTTP 429 as the terminal status,
//   - cumulative pause time exceeded rateLimitImpactPauseThreshold,
//   - any non-SQC-classified 429 was observed (Cloudflare or unknown).
func collectRateLimitReport(runDir string, failuresByType map[string][]analysis.ReportRow) *RateLimitReport {
	state, ok := readRateLimitState(runDir)
	if !ok {
		return nil
	}
	report := buildRateLimitReport(state)
	report.CausedTaskFailure = anyFailureWas429(failuresByType)
	if !rateLimitImpacted(report) {
		return nil
	}
	return report
}

// readRateLimitState loads the JSON artefact written by the migrate
// command. Missing files are not an error — they mean the run had no
// 429s.
func readRateLimitState(runDir string) (migrate.RateLimitState, bool) {
	path := filepath.Join(runDir, migrate.RateLimitEventsFile)
	data, err := os.ReadFile(path)
	if err != nil {
		return migrate.RateLimitState{}, false
	}
	var state migrate.RateLimitState
	if err := json.Unmarshal(data, &state); err != nil {
		return migrate.RateLimitState{}, false
	}
	return state, state.Total > 0
}

// buildRateLimitReport flattens the JSON state into the report shape
// rendered by the PDF. The first body snippet and headers summary are
// taken from the first non-SQC event of any kind — SQC application
// limits produce uninteresting JSON error bodies, so surfacing them
// in the PDF would add no signal.
func buildRateLimitReport(state migrate.RateLimitState) *RateLimitReport {
	report := &RateLimitReport{
		TotalHits:              state.Total,
		SQCHits:                state.Counts[sqapi.KindSQCRateLimit.String()],
		CloudflareHits:         state.Counts[sqapi.KindCloudflareRateLimit.String()],
		UnknownHits:            state.Counts[sqapi.KindUnknown429.String()],
		CumulativePauseSeconds: state.CumulativePauseSeconds,
	}
	for _, kind := range nonSQCKindsInPDFOrder {
		snap, ok := state.FirstByKind[kind.String()]
		if !ok {
			continue
		}
		report.FirstBodySnippet = snap.BodySnippet
		report.FirstHeadersSummary = formatHeaders(snap.Headers)
		break
	}
	return report
}

// nonSQCKindsInPDFOrder lists the non-SQC RateLimitKinds in the order
// the PDF should prefer when picking a body snippet to display. A
// Cloudflare-classified event is more diagnostic than an unknown one,
// so it wins when both kinds appear in the same run.
var nonSQCKindsInPDFOrder = []sqapi.RateLimitKind{
	sqapi.KindCloudflareRateLimit,
	sqapi.KindUnknown429,
}

// rateLimitImpacted is the "Only on impact" predicate gating the PDF
// warning. See the package-level constant for the cumulative-pause
// cutoff; the other branches are unconditional.
func rateLimitImpacted(r *RateLimitReport) bool {
	if r.CloudflareHits > 0 || r.UnknownHits > 0 {
		return true
	}
	if r.CausedTaskFailure {
		return true
	}
	return r.CumulativePauseSeconds > rateLimitImpactPauseThreshold
}

// anyFailureWas429 reports whether any failed request recorded in the
// analysis report ended with HTTP 429 as its terminal status. Used by
// the impact predicate to surface the warning even for short pause
// totals when retries actually exhausted on a 429.
func anyFailureWas429(failuresByType map[string][]analysis.ReportRow) bool {
	for _, rows := range failuresByType {
		for _, row := range rows {
			if row.HTTPStatus == "429" {
				return true
			}
		}
	}
	return false
}

// formatHeaders renders a headers map as "Name: value; Name: value"
// for inclusion in the PDF callout. Sorted by header name so the
// output is deterministic across runs.
func formatHeaders(headers map[string]string) string {
	if len(headers) == 0 {
		return ""
	}
	names := make([]string, 0, len(headers))
	for name := range headers {
		names = append(names, name)
	}
	sort.Strings(names)
	parts := make([]string, 0, len(headers))
	for _, name := range names {
		parts = append(parts, name+": "+headers[name])
	}
	return strings.Join(parts, "; ")
}

// rateLimitMessage assembles the body text shown inside the orange
// callout box. The three variants correspond to the impact branches in
// rateLimitImpacted: clean-but-slow SQC throttling, SQC throttling that
// killed at least one task, and any non-SQC 429 observation.
//
// The function is pure so it can be unit-tested without touching fpdf
// or the disk.
func rateLimitMessage(r *RateLimitReport) string {
	hasNonSQC := r.CloudflareHits > 0 || r.UnknownHits > 0
	switch {
	case hasNonSQC:
		return nonSQCMessage(r)
	case r.CausedTaskFailure:
		return sqcFailureMessage(r)
	default:
		return sqcRecoveredMessage(r)
	}
}

func sqcRecoveredMessage(r *RateLimitReport) string {
	return fmt.Sprintf(
		"SonarQube Cloud's API rate limit was reached %s during this run. "+
			"The tool paused and resumed automatically; total pause time was %s. "+
			"The migration completed successfully but ran slower than usual.",
		pluralHits(r.SQCHits), formatSeconds(r.CumulativePauseSeconds))
}

func sqcFailureMessage(r *RateLimitReport) string {
	return fmt.Sprintf(
		"SonarQube Cloud's API rate limit was reached %s during this run. "+
			"After retries with adaptive backoff the limit did not clear, and one or "+
			"more tasks failed (see Failed rows below). Re-run with `--run-id` to "+
			"resume from the last successful task.",
		pluralHits(r.SQCHits))
}

func nonSQCMessage(r *RateLimitReport) string {
	hits := r.CloudflareHits + r.UnknownHits
	return fmt.Sprintf(
		"A non-standard 429 response was received %s during this run. "+
			"The body and headers suggest an upstream proxy or WAF — not SonarQube "+
			"Cloud's documented application rate limit. The tool did not pause and "+
			"resume for these because the cause may require operator action (IP "+
			"block, bot rule, etc.). First response body snippet: %s",
		pluralHits(hits), snippetForMessage(r.FirstBodySnippet))
}

func pluralHits(n int) string {
	if n == 1 {
		return "1 time"
	}
	return fmt.Sprintf("%d times", n)
}

// formatSeconds prints a duration count as a human-friendly string.
// Sub-minute durations stay in seconds with one decimal; minute+ values
// round to whole minutes so the callout doesn't read like a stopwatch.
func formatSeconds(secs float64) string {
	if secs < 60 {
		return fmt.Sprintf("%.1f seconds", secs)
	}
	mins := int(math.Round(secs / 60))
	if mins == 1 {
		return "about 1 minute"
	}
	return fmt.Sprintf("about %d minutes", mins)
}

func snippetForMessage(snippet string) string {
	snippet = strings.TrimSpace(snippet)
	if snippet == "" {
		return "(empty response body)"
	}
	const maxInMessage = 160
	if len(snippet) > maxInMessage {
		snippet = snippet[:maxInMessage] + "…"
	}
	return strings.Join(strings.Fields(snippet), " ")
}

const (
	rateLimitHeading        = "Rate limiting detected during migration"
	rateLimitBoxPadding     = 3.0
	rateLimitBoxBorderWidth = 0.4
	rateLimitHeadingFontPt  = 11
	rateLimitBodyFontPt     = 9
	rateLimitHeadingLineH   = 6.0
	rateLimitBodyLineH      = 4.5
)

// renderRateLimitWarning draws the amber callout box between the
// executive summary and the per-section tables. The box renders only
// when CollectSummary determined that rate limiting materially
// impacted the run (impact predicate in rateLimitImpacted) — so this
// function does not gate on its argument; nil is a programmer error.
//
// Layout: full-width box, light amber border, amber bold heading on
// the first line, black body text wrapping below. The body is
// pre-measured under the body font so the page-break check reserves
// exactly the space the wrapped text needs — otherwise long bodies
// (notably the non-SQC variant with a 160-char snippet) auto-break
// mid-MultiCell and the border rectangle is drawn across the page
// boundary.
func renderRateLimitWarning(pdf *fpdf.Fpdf, r *RateLimitReport) {
	pdf.Ln(4)
	pageW, _ := pdf.GetPageSize()
	left, _, right, _ := pdf.GetMargins()
	boxW := pageW - left - right
	innerW := boxW - 2*rateLimitBoxPadding

	body := sanitizeForPDF(rateLimitMessage(r))

	pdf.SetFont(pdfFontFamilyBody, "", rateLimitBodyFontPt)
	bodyLines := pdf.SplitLines([]byte(body), innerW)
	bodyLineCount := len(bodyLines)
	if bodyLineCount < 1 {
		bodyLineCount = 1
	}
	boxHeight := rateLimitHeadingLineH + rateLimitBodyLineH*float64(bodyLineCount) + rateLimitBoxPadding*2
	checkPageBreak(pdf, boxHeight)

	startY := pdf.GetY()
	innerX := left + rateLimitBoxPadding

	pdf.SetXY(innerX, startY+rateLimitBoxPadding)
	setColor(pdf, colorAmber)
	pdf.SetFont(pdfFontFamilyHeading, "B", rateLimitHeadingFontPt)
	pdf.CellFormat(innerW, rateLimitHeadingLineH, rateLimitHeading, "", 1, "L", false, 0, "")

	pdf.SetX(innerX)
	setColor(pdf, colorBlack)
	pdf.SetFont(pdfFontFamilyBody, "", rateLimitBodyFontPt)
	pdf.MultiCell(innerW, rateLimitBodyLineH, body, "", "L", false)

	endY := pdf.GetY() + rateLimitBoxPadding

	prevLineWidth := pdf.GetLineWidth()
	pdf.SetLineWidth(rateLimitBoxBorderWidth)
	prevDraw := setDrawColorAmber(pdf)
	pdf.Rect(left, startY, boxW, endY-startY, "D")
	restoreDrawColor(pdf, prevDraw)
	pdf.SetLineWidth(prevLineWidth)

	pdf.SetY(endY + 2)
}

// setDrawColorAmber installs the amber draw colour and returns the
// previous (Sonar-grey) colour as a triple for restoration. The fpdf
// API has no "current draw color" accessor so we cache the constant
// rather than reading state back.
func setDrawColorAmber(pdf *fpdf.Fpdf) [3]int {
	pdf.SetDrawColor(colorAmber[0], colorAmber[1], colorAmber[2])
	return colorSonarGrey
}

func restoreDrawColor(pdf *fpdf.Fpdf, c [3]int) {
	pdf.SetDrawColor(c[0], c[1], c[2])
}
