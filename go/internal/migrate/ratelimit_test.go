// Copyright (C) SonarSource Sàrl
// For more information, see https://sonarsource.com/legal/
// mailto:info AT sonarsource DOT com

package migrate_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sonar-solutions/sonar-migration-tool/internal/migrate"
	sqapi "github.com/sonar-solutions/sq-api-go"
)

func TestTrackerObserveFirstOfKind(t *testing.T) {
	tracker := migrate.NewRateLimitTracker()

	first := tracker.Observe(sqapi.RateLimitEvent{
		Kind:       sqapi.KindSQCRateLimit,
		WaitChosen: 5 * time.Second,
		ObservedAt: time.Now(),
	})
	assert.True(t, first, "first event of a kind must be flagged")

	second := tracker.Observe(sqapi.RateLimitEvent{
		Kind:       sqapi.KindSQCRateLimit,
		WaitChosen: 5 * time.Second,
		ObservedAt: time.Now(),
	})
	assert.False(t, second, "second event of the same kind must not be flagged")

	otherKind := tracker.Observe(sqapi.RateLimitEvent{
		Kind:       sqapi.KindCloudflareRateLimit,
		WaitChosen: 0,
		ObservedAt: time.Now(),
	})
	assert.True(t, otherKind, "first event of a new kind must be flagged")
}

func TestTrackerAccumulatesPause(t *testing.T) {
	tracker := migrate.NewRateLimitTracker()
	now := time.Now()
	// CumulativePauseSeconds sums WallClockAdded (gate-deduplicated),
	// not WaitChosen — three sequential SQC events each extending the
	// gate contribute the full delta each time.
	tracker.Observe(sqapi.RateLimitEvent{Kind: sqapi.KindSQCRateLimit, WaitChosen: 5 * time.Second, WallClockAdded: 5 * time.Second, ObservedAt: now})
	tracker.Observe(sqapi.RateLimitEvent{Kind: sqapi.KindSQCRateLimit, WaitChosen: 15 * time.Second, WallClockAdded: 15 * time.Second, ObservedAt: now.Add(10 * time.Second)})
	tracker.Observe(sqapi.RateLimitEvent{Kind: sqapi.KindSQCRateLimit, WaitChosen: 7 * time.Second, WallClockAdded: 7 * time.Second, ObservedAt: now.Add(20 * time.Second)})

	dir := t.TempDir()
	path := filepath.Join(dir, migrate.RateLimitEventsFile)
	require.NoError(t, tracker.WriteJSON(path))

	var state migrate.RateLimitState
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(data, &state))

	assert.Equal(t, 3, state.Total)
	assert.Equal(t, 3, state.Counts[sqapi.KindSQCRateLimit.String()])
	assert.InDelta(t, 27.0, state.CumulativePauseSeconds, 0.001)
	assert.InDelta(t, 15.0, state.LongestPauseSeconds, 0.001)
}

// TestTrackerCumulativePauseDeduped verifies that concurrent SQC events
// parking on a shared gate window do not inflate CumulativePauseSeconds.
// Each request still reports its full WaitChosen (used for
// LongestPauseSeconds), but only one event per window contributes the
// full wall-clock pause via WallClockAdded.
func TestTrackerCumulativePauseDeduped(t *testing.T) {
	tracker := migrate.NewRateLimitTracker()
	now := time.Now()
	// 20 concurrent workers each chose to wait 60s; only the leading
	// 429 actually added 60s of wall-clock pause to the migration.
	for i := 0; i < 20; i++ {
		wallClock := time.Duration(0)
		if i == 0 {
			wallClock = 60 * time.Second
		}
		tracker.Observe(sqapi.RateLimitEvent{
			Kind:           sqapi.KindSQCRateLimit,
			WaitChosen:     60 * time.Second,
			WallClockAdded: wallClock,
			ObservedAt:     now,
		})
	}

	dir := t.TempDir()
	path := filepath.Join(dir, migrate.RateLimitEventsFile)
	require.NoError(t, tracker.WriteJSON(path))

	var state migrate.RateLimitState
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(data, &state))

	assert.Equal(t, 20, state.Total, "all events counted")
	assert.InDelta(t, 60.0, state.CumulativePauseSeconds, 0.001,
		"CumulativePauseSeconds must reflect wall-clock pause (60s), not 20x60s")
	assert.InDelta(t, 60.0, state.LongestPauseSeconds, 0.001,
		"LongestPauseSeconds reflects the largest single WaitChosen")
}

func TestTrackerWriteJSONSkipsCleanRun(t *testing.T) {
	tracker := migrate.NewRateLimitTracker()
	dir := t.TempDir()
	path := filepath.Join(dir, migrate.RateLimitEventsFile)
	require.NoError(t, tracker.WriteJSON(path))

	_, err := os.Stat(path)
	assert.True(t, os.IsNotExist(err), "no JSON file should be written for a clean run")
}

func TestTrackerConcurrentObserve(t *testing.T) {
	tracker := migrate.NewRateLimitTracker()
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			tracker.Observe(sqapi.RateLimitEvent{
				Kind:       sqapi.KindSQCRateLimit,
				WaitChosen: time.Second,
				ObservedAt: time.Now(),
			})
		}()
	}
	wg.Wait()
	assert.True(t, tracker.HasHits())
}

func TestTrackerSnapshotsFirstBody(t *testing.T) {
	tracker := migrate.NewRateLimitTracker()
	tracker.Observe(sqapi.RateLimitEvent{
		Kind:        sqapi.KindCloudflareRateLimit,
		WaitChosen:  2 * time.Second,
		BodySnippet: "<html>1015</html>",
		Headers:     map[string]string{"CF-Ray": "abc"},
		ObservedAt:  time.Now(),
	})
	tracker.Observe(sqapi.RateLimitEvent{
		Kind:        sqapi.KindCloudflareRateLimit,
		WaitChosen:  2 * time.Second,
		BodySnippet: "<html>different body, ignored</html>",
		Headers:     map[string]string{"CF-Ray": "def"},
		ObservedAt:  time.Now(),
	})

	dir := t.TempDir()
	path := filepath.Join(dir, migrate.RateLimitEventsFile)
	require.NoError(t, tracker.WriteJSON(path))

	var state migrate.RateLimitState
	data, _ := os.ReadFile(path)
	require.NoError(t, json.Unmarshal(data, &state))

	snap, ok := state.FirstByKind[sqapi.KindCloudflareRateLimit.String()]
	require.True(t, ok)
	assert.Equal(t, "<html>1015</html>", snap.BodySnippet,
		"FirstByKind must hold the FIRST snapshot, not be overwritten")
}
