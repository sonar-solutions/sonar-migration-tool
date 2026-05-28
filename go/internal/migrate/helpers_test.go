package migrate

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	sqapi "github.com/sonar-solutions/sq-api-go"
	"github.com/sonar-solutions/sonar-migration-tool/internal/common"
	"github.com/sonar-solutions/sonar-migration-tool/internal/structure"
)

func TestLoadCSVToJSONL(t *testing.T) {
	dir := t.TempDir()

	// Create a test CSV.
	csvContent := "name,value\nfoo,bar\nbaz,qux\n"
	if err := os.WriteFile(filepath.Join(dir, "test.csv"), []byte(csvContent), 0o644); err != nil {
		t.Fatal(err)
	}

	runDir := filepath.Join(dir, "run-01")
	os.MkdirAll(runDir, 0o755)
	store := common.NewDataStore(runDir)

	e := &Executor{
		Store:     store,
		ExportDir: dir,
	}

	if err := loadCSVToJSONL(e, "testTask", "test.csv"); err != nil {
		t.Fatalf("loadCSVToJSONL: %v", err)
	}

	items, err := store.ReadAll("testTask")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}

	// Verify first item.
	var row map[string]any
	json.Unmarshal(items[0], &row)
	if row["name"] != "foo" {
		t.Errorf("expected name=foo, got %v", row["name"])
	}
}

func TestLoadCSVToJSONLMissingFile(t *testing.T) {
	dir := t.TempDir()
	runDir := filepath.Join(dir, "run-01")
	os.MkdirAll(runDir, 0o755)
	store := common.NewDataStore(runDir)

	e := &Executor{
		Store:     store,
		ExportDir: dir,
	}

	// Missing CSV should result in empty output, not error (LoadCSV returns nil for missing).
	err := loadCSVToJSONL(e, "testTask", "nonexistent.csv")
	if err != nil {
		t.Fatalf("expected no error for missing CSV, got %v", err)
	}
}

func TestForEachMigrateItem(t *testing.T) {
	dir := t.TempDir()
	store := common.NewDataStore(dir)

	// Write test dependency data.
	w, _ := store.Writer("dep")
	w.WriteChunk([]json.RawMessage{
		json.RawMessage(`{"key":"a"}`),
		json.RawMessage(`{"key":"b"}`),
	})

	e := &Executor{
		Store:  store,
		Sem:    make(chan struct{}, 5),
		Logger: slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError})),
	}

	var count atomic.Int32
	err := forEachMigrateItem(context.Background(), e, "test", "dep",
		func(_ context.Context, item json.RawMessage, w *common.ChunkWriter) error {
			count.Add(1)
			return nil
		})
	if err != nil {
		t.Fatal(err)
	}
	if count.Load() != 2 {
		t.Errorf("expected 2 iterations, got %d", count.Load())
	}
}

func TestForEachMigrateItemFiltered(t *testing.T) {
	dir := t.TempDir()
	store := common.NewDataStore(dir)

	w, _ := store.Writer("dep")
	w.WriteChunk([]json.RawMessage{
		json.RawMessage(`{"key":"a","skip":true}`),
		json.RawMessage(`{"key":"b","skip":false}`),
	})

	e := &Executor{
		Store:  store,
		Sem:    make(chan struct{}, 5),
		Logger: slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError})),
	}

	var keys []string
	err := forEachMigrateItemFiltered(context.Background(), e, "test", "dep",
		func(item json.RawMessage) bool {
			return !extractBool(item, "skip")
		},
		func(_ context.Context, item json.RawMessage, w *common.ChunkWriter) error {
			keys = append(keys, extractField(item, "key"))
			return nil
		})
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) != 1 || keys[0] != "b" {
		t.Errorf("expected [b], got %v", keys)
	}
}

func TestForEachExtractItem(t *testing.T) {
	dir := t.TempDir()

	// Set up extract data with rule details.
	setupExtractData(dir)

	store := common.NewDataStore(filepath.Join(dir, "run-test"))
	os.MkdirAll(filepath.Join(dir, "run-test"), 0o755)

	e := &Executor{
		Store:     store,
		ExportDir: dir,
		Mapping:   structure.ExtractMapping{testServerURL: "extract-01"},
		Sem:       make(chan struct{}, 5),
		Logger:    slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError})),
	}

	var count int
	err := forEachExtractItem(context.Background(), e, "testExtract", "getRuleDetails",
		func(_ context.Context, item structure.ExtractItem, w *common.ChunkWriter) error {
			count++
			key := extractField(item.Data, "key")
			if key == "" {
				t.Error("expected non-empty key")
			}
			return nil
		})
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Errorf("expected 1 extract item, got %d", count)
	}
}

func TestBuildServerOrgLookup(t *testing.T) {
	dir := t.TempDir()
	store := common.NewDataStore(dir)

	// Write org mapping data.
	w, _ := store.Writer("generateOrganizationMappings")
	w.WriteChunk([]json.RawMessage{
		json.RawMessage(`{"server_url":"https://sq.test/","sonarcloud_org_key":"cloud-org1"}`),
	})

	e := &Executor{Store: store}
	lookup := buildServerOrgLookup(e)

	if lookup["https://sq.test/"] != "cloud-org1" {
		t.Errorf("expected cloud-org1, got %v", lookup["https://sq.test/"])
	}
	if lookup["unknown"] != "" {
		t.Errorf("expected empty for unknown, got %v", lookup["unknown"])
	}
}

func TestUnsupportedLanguages(t *testing.T) {
	if !unsupportedLanguages["c++"] {
		t.Error("expected c++ to be unsupported")
	}
	if unsupportedLanguages["java"] {
		t.Error("expected java to be supported")
	}
}

func TestValidPermissions(t *testing.T) {
	if !validPermissions["scan"] {
		t.Error("expected scan to be valid")
	}
	if validPermissions["delete"] {
		t.Error("expected delete to be invalid")
	}
}

func TestLogAPIWarnWithAPIError(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn}))

	apiErr := &sqapi.APIError{
		StatusCode: 403,
		Method:     "POST",
		URL:        "https://sonarcloud.io/api/permissions/add_group",
		Body:       `{"errors":[{"msg":"Insufficient privileges"}]}`,
	}
	logAPIWarn(logger, "operation failed", apiErr, "project", "proj1")

	output := buf.String()
	if !strings.Contains(output, "Insufficient privileges") {
		t.Errorf("expected parsed error message, got: %s", output)
	}
	if !strings.Contains(output, "status=403") {
		t.Errorf("expected status=403, got: %s", output)
	}
	if !strings.Contains(output, "endpoint=/api/permissions/add_group") {
		t.Errorf("expected endpoint, got: %s", output)
	}
	if !strings.Contains(output, "project=proj1") {
		t.Errorf("expected project attr, got: %s", output)
	}
}

func TestLogAPIWarnWithPlainError(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn}))

	logAPIWarn(logger, "something failed", errors.New("connection refused"), "task", "test")

	output := buf.String()
	if !strings.Contains(output, "connection refused") {
		t.Errorf("expected plain error, got: %s", output)
	}
	if strings.Contains(output, "status=") {
		t.Errorf("should not have status for plain error, got: %s", output)
	}
}

func TestTaskCounterFailAndSummary(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	c := NewTaskCounter("testTask")
	c.Success()
	c.Success()
	c.Fail()
	c.LogSummary(logger)

	output := buf.String()
	if !strings.Contains(output, "succeeded=2") {
		t.Errorf("expected succeeded=2, got: %s", output)
	}
	if !strings.Contains(output, "failed=1") {
		t.Errorf("expected failed=1, got: %s", output)
	}
	if !strings.Contains(output, "total=3") {
		t.Errorf("expected total=3, got: %s", output)
	}
}

func TestTaskCounterEmptySummary(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	c := NewTaskCounter("empty")
	c.LogSummary(logger)

	if buf.Len() > 0 {
		t.Errorf("expected no output for empty counter, got: %s", buf.String())
	}
}

func TestProgressLoggerZeroInterval(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	// total=0 → interval=0 → Increment should be a no-op
	prog := newProgressLogger(logger, "test", 0)
	prog.Increment()

	if buf.Len() > 0 {
		t.Errorf("expected no output for zero-interval progress, got: %s", buf.String())
	}
}

func TestProgressLoggerLogsAtInterval(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	// total=50 → interval=50 (< 100 branch)
	prog := newProgressLogger(logger, "test", 50)
	for i := range 49 {
		prog.Increment()
		if buf.Len() > 0 {
			t.Fatalf("unexpected log at iteration %d", i)
		}
	}
	prog.Increment() // 50th item
	// Issue #202: message now reads "task N/M - X%" — a single
	// readable line operators can scan when tailing the log.
	if !strings.Contains(buf.String(), "test 50/50 - 100%") {
		t.Errorf("expected progress message \"test 50/50 - 100%%\", got: %s", buf.String())
	}
}

// The final-item log MUST fire even when total isn't a clean
// multiple of the interval — e.g. createProjects with 975 items at
// every-10 logs at 970 then jumps to 975 with a 100% line. Issue
// #202 spec calls this out explicitly: operators need an explicit
// "task complete" marker.
func TestProgressLoggerFiresFinalHundredPercent(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	prog := newProgressLogger(logger, "createProjects", 975) // interval=10

	for i := 0; i < 975; i++ {
		prog.Increment()
	}
	out := buf.String()
	// Last regularly-scheduled line lands at 970 (interval × 97).
	if !strings.Contains(out, "createProjects 970/975 - 99%") {
		t.Errorf("expected interval-aligned line at 970, got:\n%s", out)
	}
	// Final line at 975 — fires because (n == total) even though
	// 975 isn't a multiple of 10.
	if !strings.Contains(out, "createProjects 975/975 - 100%") {
		t.Errorf("expected final 100%% line at 975, got:\n%s", out)
	}
}

// Per-task interval overrides take precedence over the size-based
// default. createProjects ships at every-10, setProjectGroupPermissions
// at every-100 (issue #202).
func TestProgressLoggerHonoursPerTaskInterval(t *testing.T) {
	cases := []struct {
		task          string
		total         int
		wantInterval  int64
		wantFirstAt   int // iteration count when the first log should fire
		wantFirstLine string
	}{
		{
			task:          "createProjects",
			total:         975,
			wantInterval:  10,
			wantFirstAt:   10,
			wantFirstLine: "createProjects 10/975 - 1%",
		},
		{
			task:          "configurePortfolios",
			total:         87,
			wantInterval:  10,
			wantFirstAt:   10,
			wantFirstLine: "configurePortfolios 10/87 - 11%",
		},
		{
			task:          "setProjectSettings",
			total:         1234,
			wantInterval:  50,
			wantFirstAt:   50,
			wantFirstLine: "setProjectSettings 50/1234 - 4%",
		},
		{
			task:          "setProjectGroupPermissions",
			total:         19778,
			wantInterval:  100,
			wantFirstAt:   100,
			wantFirstLine: "setProjectGroupPermissions 100/19778 - 0%",
		},
	}
	for _, c := range cases {
		t.Run(c.task, func(t *testing.T) {
			var buf bytes.Buffer
			logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))
			prog := newProgressLogger(logger, c.task, c.total)
			if prog.interval != c.wantInterval {
				t.Errorf("interval: want %d, got %d", c.wantInterval, prog.interval)
			}
			for i := 0; i < c.wantFirstAt-1; i++ {
				prog.Increment()
				if buf.Len() > 0 {
					t.Fatalf("unexpected log at iteration %d for %s", i, c.task)
				}
			}
			prog.Increment() // first interval hit
			if !strings.Contains(buf.String(), c.wantFirstLine) {
				t.Errorf("want first log to contain %q, got: %s", c.wantFirstLine, buf.String())
			}
		})
	}
}
