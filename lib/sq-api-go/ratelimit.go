// Copyright (C) SonarSource Sàrl
// For more information, see https://sonarsource.com/legal/
// mailto:info AT sonarsource DOT com

package sqapi

import (
	"net/http"
	"strconv"
	"strings"
	"time"
)

// RateLimitKind classifies an HTTP 429 response by its likely source.
//
// SonarQube Cloud's documented application rate limit returns a JSON body
// matching the SonarQube error shape, with or without a Retry-After hint.
// Upstream proxies (notably Cloudflare) can also return 429 — those
// responses typically carry HTML bodies and Cloudflare-specific headers,
// and may indicate an IP-level WAF rule rather than a steady-state quota.
// Treating both identically risks pausing pointlessly when a WAF block is
// active; the classification lets the transport pick an appropriate
// retry strategy and lets the report surface non-standard 429s for
// operator review.
type RateLimitKind int

const (
	// KindUnknown429 is the zero value; used when a 429 cannot be
	// classified, e.g. an empty body with no recognisable headers.
	KindUnknown429 RateLimitKind = iota
	// KindSQCRateLimit is SonarQube Cloud's application-layer rate limit.
	KindSQCRateLimit
	// KindCloudflareRateLimit is a 429 served by Cloudflare in front
	// of SQC — either a Cloudflare-managed rate-limit rule or a
	// WAF challenge that uses 429 as the action.
	KindCloudflareRateLimit
)

// String returns the kind as a short stable identifier suitable for
// logging and JSON serialisation.
func (k RateLimitKind) String() string {
	switch k {
	case KindSQCRateLimit:
		return "sqc-rate-limit"
	case KindCloudflareRateLimit:
		return "cloudflare-rate-limit"
	default:
		return "unknown-429"
	}
}

// RateLimitEvent describes a single 429 observed by the retry transport.
// It is delivered to RateLimitObserver callbacks so the caller can record
// per-run statistics (counts, longest pause, first-event-of-each-kind
// snapshots) without parsing logs.
type RateLimitEvent struct {
	Kind        RateLimitKind
	RetryAfter  time.Duration // 0 if no Retry-After header was present
	WaitChosen  time.Duration // duration the transport will sleep before retrying; 0 if no retry
	// WallClockAdded is the gate-deduplicated wall-clock pause this
	// event actually contributes to the migration. For SQC 429s that
	// share the rate-limit gate, only the first (or extending) event of
	// a window reports the full pause; piggy-back events report 0. For
	// non-gated 429s (Cloudflare/unknown) this equals WaitChosen. Use
	// this — not WaitChosen — when summing total pause time, otherwise
	// concurrent workers parking on the same gate window will inflate
	// the total by N×.
	WallClockAdded time.Duration
	BodySnippet    string // truncated to bodySnippetMax bytes
	Headers        map[string]string
	ObservedAt     time.Time
}

// RateLimitObserver is invoked once per observed 429. Implementations
// must be safe for concurrent calls — the retry transport invokes it
// directly from RoundTrip, which can run on many goroutines.
type RateLimitObserver func(event RateLimitEvent)

// bodySnippetMax is the maximum number of body bytes preserved in a
// RateLimitEvent. Matches the truncation used by APIError.Body so the
// report and the error message see comparable amounts of context.
const bodySnippetMax = 500

// rateLimitHeaders are the response headers worth capturing on a 429.
// Order is preserved for readable summary strings in the PDF report.
var rateLimitHeaders = []string{
	"Retry-After",
	"X-RateLimit-Limit",
	"X-RateLimit-Remaining",
	"X-RateLimit-Reset",
	"CF-Ray",
	"CF-Mitigated",
	"Server",
}

// classify429 inspects a 429 response's body and headers and returns
// the most likely source of the throttling decision.
//
// The detection is deliberately conservative: only responses that look
// like SonarQube's JSON error envelope are classified as SQC; only
// responses that carry Cloudflare-specific headers or HTML markers are
// classified as Cloudflare; anything else falls through to
// KindUnknown429 so the operator is shown the body in the PDF report.
func classify429(headers http.Header, body []byte) RateLimitKind {
	if isCloudflare429(headers, body) {
		return KindCloudflareRateLimit
	}
	if isSonarJSONError(body) {
		return KindSQCRateLimit
	}
	return KindUnknown429
}

// isCloudflare429 reports whether the response carries signals that
// strongly suggest Cloudflare authored the 429 (managed rate limit,
// WAF rule, or bot mitigation).
func isCloudflare429(headers http.Header, body []byte) bool {
	if headers.Get("CF-Ray") != "" {
		return true
	}
	if headers.Get("CF-Mitigated") != "" {
		return true
	}
	if strings.EqualFold(headers.Get("Server"), "cloudflare") {
		return true
	}
	// Cloudflare's branded Error 1015 page is HTML and mentions
	// "rate limited" or "Cloudflare" prominently. Catch the
	// case where the headers were stripped but the body survives.
	lower := strings.ToLower(string(trimForCheck(body)))
	if strings.HasPrefix(lower, "<") && strings.Contains(lower, "cloudflare") {
		return true
	}
	return false
}

// isSonarJSONError reports whether the body looks like SonarQube's
// canonical error response shape: {"errors":[{"msg":"..."}]}. The check
// is intentionally loose — it tolerates whitespace and additional fields
// — so a future format tweak by Sonar that leaves the envelope intact
// keeps working without code changes.
func isSonarJSONError(body []byte) bool {
	trimmed := trimForCheck(body)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return false
	}
	lower := strings.ToLower(string(trimmed))
	return strings.Contains(lower, `"errors"`)
}

// trimForCheck returns the body with leading whitespace stripped, capped
// at bodySnippetMax bytes so prefix checks remain bounded.
func trimForCheck(body []byte) []byte {
	if len(body) > bodySnippetMax {
		body = body[:bodySnippetMax]
	}
	return []byte(strings.TrimLeft(string(body), " \t\r\n"))
}

// parseRetryAfter parses the Retry-After response header in either of
// the two RFC 7231 forms: a non-negative integer count of seconds, or
// an HTTP-date. Returns 0 when the header is absent or unparseable, so
// callers can use a "max(parsed, defaultBackoff)" pattern without nil
// checks.
func parseRetryAfter(headers http.Header) time.Duration {
	raw := strings.TrimSpace(headers.Get("Retry-After"))
	if raw == "" {
		return 0
	}
	if secs, err := strconv.Atoi(raw); err == nil && secs >= 0 {
		return time.Duration(secs) * time.Second
	}
	if t, err := http.ParseTime(raw); err == nil {
		d := time.Until(t)
		if d < 0 {
			return 0
		}
		return d
	}
	return 0
}

// snapshotHeaders extracts the rate-limit-relevant headers into a flat
// map suitable for embedding in a RateLimitEvent or JSON-serialising
// into the run's rate_limit_events.json artefact.
func snapshotHeaders(headers http.Header) map[string]string {
	out := make(map[string]string, len(rateLimitHeaders))
	for _, name := range rateLimitHeaders {
		if v := headers.Get(name); v != "" {
			out[name] = v
		}
	}
	return out
}

// snapshotBody returns the first bodySnippetMax bytes of body as a string.
func snapshotBody(body []byte) string {
	if len(body) > bodySnippetMax {
		body = body[:bodySnippetMax]
	}
	return string(body)
}
