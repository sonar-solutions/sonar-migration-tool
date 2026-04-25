package migrate

import (
	"context"
	"encoding/json"
	"fmt"

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

	w, err := e.Store.Writer(taskName)
	if err != nil {
		return err
	}

	g, ctx := errgroup.WithContext(ctx)
	for _, item := range items {
		if filterFn != nil && !filterFn(item) {
			continue
		}
		g.Go(func() error {
			if err := common.AcquireSem(ctx, e.Sem); err != nil {
				return err
			}
			defer func() { <-e.Sem }()
			return fn(ctx, item, w)
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

// extractField is a convenience alias.
var extractField = common.ExtractField

// extractBool is a convenience alias.
var extractBool = common.ExtractBool
