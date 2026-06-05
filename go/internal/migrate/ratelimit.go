// Copyright (C) SonarSource Sàrl
// For more information, see https://sonarsource.com/legal/
// mailto:info AT sonarsource DOT com

package migrate

import (
	"encoding/json"
	"os"
	"sync"
	"time"

	sqapi "github.com/sonar-solutions/sq-api-go"
)

// RateLimitEventsFile is the filename written under the run directory
// when one or more 429 responses were observed during the run. The PDF
// report collector reads this artefact to decide whether to render the
// amber rate-limit warning.
const RateLimitEventsFile = "rate_limit_events.json"

// FirstEventSnapshot is the JSON shape of the first observed event of
// a given RateLimitKind. The report uses BodySnippet and Headers to
// surface non-standard 429s (Cloudflare/unknown) for operator review.
type FirstEventSnapshot struct {
	ObservedAt        time.Time         `json:"observedAt"`
	RetryAfterSeconds float64           `json:"retryAfterSeconds"`
	WaitChosenSeconds float64           `json:"waitChosenSeconds"`
	BodySnippet       string            `json:"bodySnippet"`
	Headers           map[string]string `json:"headers"`
}

// RateLimitState is the JSON shape persisted to RateLimitEventsFile.
// Counts and FirstByKind are keyed by sqapi.RateLimitKind.String() so
// the report can group hits by classification without an enum mapping.
type RateLimitState struct {
	Total                  int                           `json:"total"`
	Counts                 map[string]int                `json:"counts"`
	FirstHitAt             time.Time                     `json:"firstHitAt"`
	LastHitAt              time.Time                     `json:"lastHitAt"`
	CumulativePauseSeconds float64                       `json:"cumulativePauseSeconds"`
	LongestPauseSeconds    float64                       `json:"longestPauseSeconds"`
	FirstByKind            map[string]FirstEventSnapshot `json:"firstByKind"`
}

// RateLimitTracker accumulates RateLimitEvent observations from the
// HTTP retry transport and renders them as a single JSON artefact at
// the end of the run. It is safe for concurrent use — Observe is
// called directly from arbitrary task goroutines.
type RateLimitTracker struct {
	mu    sync.Mutex
	state RateLimitState
}

// NewRateLimitTracker returns an empty tracker ready to receive events.
func NewRateLimitTracker() *RateLimitTracker {
	return &RateLimitTracker{
		state: RateLimitState{
			Counts:      make(map[string]int),
			FirstByKind: make(map[string]FirstEventSnapshot),
		},
	}
}

// Observe records a single 429 event. Returns true when this is the
// first event of its RateLimitKind seen in this run — callers use that
// to emit a one-time "rate limiting detected" warn log per kind rather
// than spamming the log on every retry.
func (t *RateLimitTracker) Observe(event sqapi.RateLimitEvent) (firstOfKind bool) {
	t.mu.Lock()
	defer t.mu.Unlock()

	kind := event.Kind.String()
	_, seen := t.state.FirstByKind[kind]
	firstOfKind = !seen

	t.state.Total++
	t.state.Counts[kind]++

	if t.state.FirstHitAt.IsZero() || event.ObservedAt.Before(t.state.FirstHitAt) {
		t.state.FirstHitAt = event.ObservedAt
	}
	if event.ObservedAt.After(t.state.LastHitAt) {
		t.state.LastHitAt = event.ObservedAt
	}

	waitSec := event.WaitChosen.Seconds()
	// CumulativePauseSeconds is gate-deduplicated wall-clock pause —
	// summing per-request WaitChosen would inflate by N when N concurrent
	// workers park on the same gate window. WallClockAdded carries the
	// already-deduplicated contribution from the transport.
	t.state.CumulativePauseSeconds += event.WallClockAdded.Seconds()
	if waitSec > t.state.LongestPauseSeconds {
		t.state.LongestPauseSeconds = waitSec
	}

	if firstOfKind {
		t.state.FirstByKind[kind] = FirstEventSnapshot{
			ObservedAt:        event.ObservedAt,
			RetryAfterSeconds: event.RetryAfter.Seconds(),
			WaitChosenSeconds: waitSec,
			BodySnippet:       event.BodySnippet,
			Headers:           event.Headers,
		}
	}

	return firstOfKind
}

// HasHits reports whether any 429 has been observed.
func (t *RateLimitTracker) HasHits() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.state.Total > 0
}

// WriteJSON serialises the tracker's accumulated state to path. Returns
// nil without writing when no hits have been recorded — clean runs do
// not produce the artefact, keeping run dirs free of zero-information
// noise.
func (t *RateLimitTracker) WriteJSON(path string) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.state.Total == 0 {
		return nil
	}
	data, err := json.MarshalIndent(t.state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}
