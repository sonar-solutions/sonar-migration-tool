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
	tracker.Observe(sqapi.RateLimitEvent{Kind: sqapi.KindSQCRateLimit, WaitChosen: 5 * time.Second, ObservedAt: now})
	tracker.Observe(sqapi.RateLimitEvent{Kind: sqapi.KindSQCRateLimit, WaitChosen: 15 * time.Second, ObservedAt: now.Add(10 * time.Second)})
	tracker.Observe(sqapi.RateLimitEvent{Kind: sqapi.KindSQCRateLimit, WaitChosen: 7 * time.Second, ObservedAt: now.Add(20 * time.Second)})

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
