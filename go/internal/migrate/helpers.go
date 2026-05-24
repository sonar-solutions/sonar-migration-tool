package migrate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync/atomic"

	sqapi "github.com/sonar-solutions/sq-api-go"
	"github.com/sonar-solutions/sonar-migration-tool/internal/common"
	"github.com/sonar-solutions/sonar-migration-tool/internal/structure"
	"golang.org/x/sync/errgroup"
)

// readExtractItems reads JSONL items from an extract task across all extract runs.
func readExtractItems(e *Executor, taskKey string) ([]structure.ExtractItem, error) {
	return structure.ReadExtractData(e.ExportDir, e.Mapping, taskKey)
}

// forEachMigrateItem reads items from a completed migrate task and calls fn
// for each, concurrently bounded by semaphore.
func forEachMigrateItem(ctx context.Context, e *Executor, taskName, depTask string,
	fn func(ctx context.Context, item json.RawMessage, w *common.ChunkWriter) error) error {

	return forEachMigrateItemFiltered(ctx, e, taskName, depTask, nil, fn)
}

// forEachMigrateItemFiltered is like forEachMigrateItem with an optional filter.
func forEachMigrateItemFiltered(ctx context.Context, e *Executor, taskName, depTask string,
	filterFn func(json.RawMessage) bool,
	fn func(ctx context.Context, item json.RawMessage, w *common.ChunkWriter) error) error {

	items, err := e.Store.ReadAll(depTask)
	if err != nil {
		return fmt.Errorf("%s: reading %s: %w", taskName, depTask, err)
	}

	// Pre-filter to get accurate count for progress logging.
	var filtered []json.RawMessage
	for _, item := range items {
		if filterFn == nil || filterFn(item) {
			filtered = append(filtered, item)
		}
	}

	e.Logger.Info("starting task", "task", taskName, "items", len(filtered))
	prog := newProgressLogger(e.Logger, taskName, len(filtered))

	w, err := e.Store.Writer(taskName)
	if err != nil {
		return err
	}

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(cap(e.Sem))
	for _, item := range filtered {
		g.Go(func() error {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			err := fn(ctx, item, w)
			prog.Increment()
			return err
		})
	}
	return g.Wait()
}

// forEachExtractItem reads items from an extract task and calls fn for each,
// concurrently bounded by semaphore. Unlike forEachMigrateItem which reads from
// the migrate store, this reads from extract data across all extract runs.
func forEachExtractItem(ctx context.Context, e *Executor, taskName, extractKey string,
	fn func(ctx context.Context, item structure.ExtractItem, w *common.ChunkWriter) error) error {

	items, err := readExtractItems(e, extractKey)
	if err != nil {
		return fmt.Errorf("%s: reading %s: %w", taskName, extractKey, err)
	}

	e.Logger.Info("starting task", "task", taskName, "items", len(items))
	prog := newProgressLogger(e.Logger, taskName, len(items))

	w, err := e.Store.Writer(taskName)
	if err != nil {
		return err
	}

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(cap(e.Sem))
	for _, item := range items {
		g.Go(func() error {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			err := fn(ctx, item, w)
			prog.Increment()
			return err
		})
	}
	return g.Wait()
}

// buildOrgKeyLookup loads organizations.csv and returns a map from
// sonarqube_org_key to sonarcloud_org_key.
func buildOrgKeyLookup(exportDir string) (map[string]string, error) {
	rows, err := structure.LoadCSV(exportDir, "organizations.csv")
	if err != nil {
		return nil, err
	}
	lookup := make(map[string]string, len(rows))
	for _, row := range rows {
		sqKey, _ := row["sonarqube_org_key"].(string)
		scKey, _ := row["sonarcloud_org_key"].(string)
		if sqKey != "" {
			lookup[sqKey] = scKey
		}
	}
	return lookup, nil
}

// loadCSVToJSONL reads a CSV file and writes each row as a JSONL object
// to the task output. Used by generate*Mappings tasks.
// It enriches each row with sonarcloud_org_key by joining on sonarqube_org_key
// from organizations.csv.
func loadCSVToJSONL(e *Executor, taskName, csvFilename string) error {
	rows, err := structure.LoadCSV(e.ExportDir, csvFilename)
	if err != nil {
		return fmt.Errorf("%s: loading %s: %w", taskName, csvFilename, err)
	}

	orgLookup, err := buildOrgKeyLookup(e.ExportDir)
	if err != nil {
		return fmt.Errorf("%s: loading organizations.csv for join: %w", taskName, err)
	}

	w, err := e.Store.Writer(taskName)
	if err != nil {
		return err
	}

	items := make([]json.RawMessage, 0, len(rows))
	for _, row := range rows {
		// Enrich with sonarcloud_org_key from org lookup.
		if sqKey, ok := row["sonarqube_org_key"].(string); ok && sqKey != "" {
			if scKey, found := orgLookup[sqKey]; found {
				row["sonarcloud_org_key"] = scKey
			}
		}
		b, err := json.Marshal(row)
		if err != nil {
			continue
		}
		items = append(items, b)
	}
	return w.WriteChunk(items)
}

// buildServerOrgLookup returns a map from server URL to SonarCloud org key
// using the generateOrganizationMappings migrate output.
func buildServerOrgLookup(e *Executor) map[string]string {
	orgItems, _ := e.Store.ReadAll("generateOrganizationMappings")
	orgKeys := make(map[string]string, len(orgItems))
	for _, o := range orgItems {
		serverURL := extractField(o, "server_url")
		cloudKey := extractField(o, "sonarcloud_org_key")
		orgKeys[serverURL] = cloudKey
	}
	return orgKeys
}

// Unsupported languages that are filtered during migration.
var unsupportedLanguages = map[string]bool{
	"c++": true, "grvy": true, "ps": true,
}

// validPermissions for project group permissions.
var validPermissions = map[string]bool{
	"admin": true, "codeviewer": true, "issueadmin": true,
	"securityhotspotadmin": true, "scan": true, "user": true,
}

// skippedOrgSentinel is the marker value for organizations that should be
// excluded from migration (user chose to skip them during the wizard).
const skippedOrgSentinel = "SKIPPED"

// shouldSkipOrg returns true if the org key is empty or marked SKIPPED.
func shouldSkipOrg(orgKey string) bool {
	return orgKey == "" || orgKey == skippedOrgSentinel
}

// logAPIWarn logs an API error with structured fields. If the error is an
// APIError, it extracts the human-readable message, status code, and endpoint.
func logAPIWarn(logger *slog.Logger, msg string, err error, attrs ...any) {
	var apiErr *sqapi.APIError
	if errors.As(err, &apiErr) {
		attrs = append(attrs,
			"err", apiErr.Message(),
			"status", apiErr.StatusCode,
			"endpoint", apiErr.Endpoint(),
		)
	} else {
		attrs = append(attrs, "err", err)
	}
	logger.Warn(msg, attrs...)
}

// TaskCounter tracks success/failure counts for a task. Safe for concurrent use.
type TaskCounter struct {
	succeeded atomic.Int64
	failed    atomic.Int64
	task      string
}

// NewTaskCounter creates a counter for tracking task operation results.
func NewTaskCounter(task string) *TaskCounter {
	return &TaskCounter{task: task}
}

// Success increments the success count.
func (c *TaskCounter) Success() { c.succeeded.Add(1) }

// Fail increments the failure count.
func (c *TaskCounter) Fail() { c.failed.Add(1) }

// LogSummary logs the final counts. Only logs if there were any operations.
func (c *TaskCounter) LogSummary(logger *slog.Logger) {
	s, f := c.succeeded.Load(), c.failed.Load()
	total := s + f
	if total == 0 {
		return
	}
	logger.Info("task summary",
		"task", c.task,
		"succeeded", s,
		"failed", f,
		"total", total,
	)
}

// progressLogger logs progress at regular intervals. Safe for concurrent use.
type progressLogger struct {
	task     string
	total    int
	done     atomic.Int64
	logger   *slog.Logger
	interval int64
}

// progressLogInterval names how often (in items) the progress logger
// should emit an INFO line for a given task. Per-task entries take
// precedence over the size-based fallback in newProgressLogger.
//
// Tuned for operator-visible cadence (issue #202):
//   - createProjects is one API call per project and the slowest
//     per-item task on large platforms; surface progress every 10.
//   - configurePortfolios issues multiple Enterprise-API calls per
//     portfolio (create + update + project membership); every 10
//     keeps progress visible at the user's expected cadence.
//   - setProjectSettings does multiple HTTP calls per record on
//     average (definition-driven dispatch, fan-out fallback);
//     every 50 strikes a balance between visibility and noise.
//   - setProjectGroupPermissions can run into the tens of thousands
//     of items (projects × groups × permissions); every 100 keeps
//     the log readable while still ticking visibly.
var progressLogInterval = map[string]int64{
	"createProjects":             10,
	"configurePortfolios":        10,
	"setProjectSettings":         50,
	"setProjectGroupPermissions": 100,
}

func newProgressLogger(logger *slog.Logger, task string, total int) *progressLogger {
	interval := int64(1000)
	if total < 1000 {
		interval = 100
	}
	if total < 100 {
		interval = int64(total)
	}
	// Per-task override beats the size-based default — operator
	// cadence trumps "log volume." Capped at total so the very last
	// item still emits a line when override > total.
	if explicit, ok := progressLogInterval[task]; ok && explicit > 0 {
		interval = explicit
		if int64(total) < interval {
			interval = int64(total)
		}
	}
	return &progressLogger{task: task, total: total, logger: logger, interval: interval}
}

func (p *progressLogger) Increment() {
	if p.interval <= 0 {
		return
	}
	n := p.done.Add(1)
	if n%p.interval == 0 || int(n) == p.total {
		percent := 0
		if p.total > 0 {
			percent = int(n * 100 / int64(p.total))
		}
		// One-line human-readable message per the issue #202 spec
		// — "task N/M - X%" — so operators tailing the log can read
		// progress at a glance.
		p.logger.Info(fmt.Sprintf("%s %d/%d - %d%%", p.task, n, p.total, percent))
	}
}

// extractField is a convenience alias.
var extractField = common.ExtractField

// extractBool is a convenience alias.
var extractBool = common.ExtractBool
