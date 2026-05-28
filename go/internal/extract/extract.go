package extract

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/sonar-solutions/sonar-migration-tool/internal/common"
	sqapi "github.com/sonar-solutions/sq-api-go"
	"github.com/sonar-solutions/sq-api-go/server"
	"golang.org/x/sync/errgroup"
)

// ExtractConfig holds all parameters for an extract run.
type ExtractConfig struct {
	URL             string
	Token           string
	ExportDirectory string
	ExtractType     string // "all" or report type
	PEMFilePath     string
	KeyFilePath     string
	CertPassword    string
	Concurrency     int
	Timeout         int
	ExtractID       string
	TargetTask         string
	IncludeScanHistory bool
	// ProjectKey is a comma-separated list of project keys to scope
	// the extract to (issue #98). Empty = extract every project the
	// token can see. Use cases: extracting a single representative
	// project before a full migration, or re-extracting only the
	// projects that failed in a prior run. Whitespace around each
	// key is trimmed.
	ProjectKey string
}

// Executor is the runtime context passed to every task function.
type Executor struct {
	Raw       *RawClient
	Store     *DataStore
	ServerURL string
	Edition   Edition
	Version   float64
	Sem       chan struct{}
	Logger    *slog.Logger

	mu              sync.Mutex
	skippedProjects map[string]bool

	// projectKeyFilter, when non-nil, scopes per-project tasks to
	// the named projects (issue #98). nil = no filter (today's
	// pre-filter behaviour). Read via IsSkipped — every per-project
	// task already consults that predicate, so the filter takes
	// effect everywhere downstream for free.
	projectKeyFilter map[string]bool
}

// SetProjectKeyFilter installs a project-key allow-list. Keys not in
// the set are treated as skipped by IsSkipped, which the per-project
// extract tasks already check. A nil or empty map disables filtering.
func (e *Executor) SetProjectKeyFilter(keys map[string]bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if len(keys) == 0 {
		e.projectKeyFilter = nil
		return
	}
	e.projectKeyFilter = keys
}

// RecordSkipped marks a project as skipped due to insufficient privileges.
func (e *Executor) RecordSkipped(projectKey string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.skippedProjects == nil {
		e.skippedProjects = make(map[string]bool)
	}
	e.skippedProjects[projectKey] = true
}

// IsSkipped returns true if the project has been marked as skipped
// for either of two reasons:
//
//   - the project failed an earlier per-project task with a
//     non-fatal HTTP error (insufficient privileges, missing
//     branch, etc.) — recorded via RecordSkipped.
//   - the executor has a project_key allow-list installed
//     (issue #98) and the project isn't in it.
//
// Per-project extract tasks consult this predicate before doing any
// network work, so installing the allow-list is sufficient to skip
// every downstream side-effect.
func (e *Executor) IsSkipped(projectKey string) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.skippedProjects[projectKey] {
		return true
	}
	if e.projectKeyFilter != nil && !e.projectKeyFilter[projectKey] {
		return true
	}
	return false
}

// SkippedProjectKeys returns a sorted list of project keys that were skipped.
func (e *Executor) SkippedProjectKeys() []string {
	e.mu.Lock()
	defer e.mu.Unlock()
	keys := make([]string, 0, len(e.skippedProjects))
	for k := range e.skippedProjects {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// RunExtract is the main entry point for the extract command.
// It returns the list of project keys that were skipped due to insufficient privileges.
func RunExtract(ctx context.Context, cfg ExtractConfig) ([]string, error) {
	cfg.applyDefaults()

	client, raw, version, edition, err := initClient(ctx, cfg)
	if err != nil {
		return nil, err
	}

	extractID, extractDir, err := prepareExtractDir(cfg)
	if err != nil {
		return nil, err
	}

	registry, plan, targets, err := buildPlan(cfg, edition)
	if err != nil {
		return nil, err
	}

	meta := extractMeta{
		Plan: plan, RunID: extractID, Version: version,
		Edition: edition, URL: cfg.URL, Targets: targets, Registry: registry,
	}
	if err := writeMetadataFile(extractDir, meta); err != nil {
		return nil, err
	}

	store := NewDataStore(extractDir)
	plan = filterCompleted(plan, store)

	executor := newExecutor(raw, store, client.BaseURL(), edition, version, cfg.Concurrency)
	if filter := parseProjectKeyFilter(cfg.ProjectKey); filter != nil {
		executor.SetProjectKeyFilter(filter)
	}
	if err := executePhases(ctx, executor, plan, registry, store); err != nil {
		return nil, err
	}

	fmt.Printf("Extract Complete: %s\n", extractID)
	return executor.SkippedProjectKeys(), nil
}

func initClient(ctx context.Context, cfg ExtractConfig) (*sqapi.Client, *RawClient, float64, Edition, error) {
	var opts []sqapi.Option
	opts = append(opts, sqapi.WithTimeout(cfg.Timeout))
	if cfg.PEMFilePath != "" {
		opts = append(opts, sqapi.WithClientCert(cfg.PEMFilePath, cfg.KeyFilePath, cfg.CertPassword))
	}

	version, err := detectVersion(ctx, cfg)
	if err != nil {
		return nil, nil, 0, "", fmt.Errorf("detecting server version: %w", err)
	}

	client := sqapi.NewServerClient(cfg.URL, cfg.Token, version, opts...)
	if client.CertErr() != nil {
		return nil, nil, 0, "", fmt.Errorf("certificate error: %w", client.CertErr())
	}

	raw := NewRawClient(client.HTTPClient(), client.BaseURL())
	edition, err := detectEdition(ctx, raw)
	if err != nil {
		return nil, nil, 0, "", fmt.Errorf("detecting edition: %w", err)
	}

	return client, raw, version, edition, nil
}

func prepareExtractDir(cfg ExtractConfig) (string, string, error) {
	extractID := cfg.ExtractID
	if extractID == "" {
		extractID = generateRunID(cfg.ExportDirectory)
	}
	extractDir := filepath.Join(cfg.ExportDirectory, extractID)
	if err := os.MkdirAll(extractDir, 0o755); err != nil {
		return "", "", fmt.Errorf("creating extract dir: %w", err)
	}
	return extractID, extractDir, nil
}

func buildPlan(cfg ExtractConfig, edition Edition) (map[string]*TaskDef, [][]string, []string, error) {
	allDefs := RegisterAll()
	registry := BuildRegistry(allDefs)
	registry = FilterByEdition(registry, edition)

	var targets []string
	if cfg.IncludeScanHistory {
		targets = TargetTasksWithScanHistory(registry, cfg.TargetTask, cfg.ExtractType)
	} else {
		targets = TargetTasks(registry, cfg.TargetTask, cfg.ExtractType)
	}
	taskSet := ResolveDependencies(targets, registry)
	if taskSet == nil {
		return nil, nil, nil, fmt.Errorf("cannot resolve dependencies for target tasks")
	}

	plan, err := PlanPhases(taskSet, registry)
	if err != nil {
		return nil, nil, nil, err
	}
	return registry, plan, targets, nil
}

func newExecutor(raw *RawClient, store *DataStore, baseURL string, edition Edition, version float64, concurrency int) *Executor {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	return &Executor{
		Raw:       raw,
		Store:     store,
		ServerURL: baseURL,
		Edition:   edition,
		Version:   version,
		Sem:       make(chan struct{}, concurrency),
		Logger:    logger,
	}
}

func executePhases(ctx context.Context, executor *Executor, plan [][]string, registry map[string]*TaskDef, store *DataStore) error {
	for i, phase := range plan {
		executor.Logger.Info("starting phase", "phase", i+1, "tasks", len(phase))
		if err := runPhase(ctx, executor, phase, registry); err != nil {
			return fmt.Errorf("phase %d: %w", i+1, err)
		}
		for _, taskName := range phase {
			store.MarkComplete(taskName)
		}
	}
	return nil
}

func runPhase(ctx context.Context, e *Executor, taskNames []string, registry map[string]*TaskDef) error {
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(cap(e.Sem))
	for _, name := range taskNames {
		def := registry[name]
		e.Logger.Info("running task", "task", name)
		g.Go(func() error {
			if err := def.Run(ctx, e); err != nil {
				return fmt.Errorf("task %s: %w", name, err)
			}
			return nil
		})
	}
	return g.Wait()
}

// parseProjectKeyFilter splits the comma-separated --project_key CLI
// value into a lookup set. Empty input → nil (no filter). Whitespace
// around each key is trimmed; duplicates collapse naturally.
func parseProjectKeyFilter(csv string) map[string]bool {
	csv = strings.TrimSpace(csv)
	if csv == "" {
		return nil
	}
	out := make(map[string]bool)
	for _, k := range strings.Split(csv, ",") {
		k = strings.TrimSpace(k)
		if k != "" {
			out[k] = true
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func (cfg *ExtractConfig) applyDefaults() {
	if cfg.Concurrency <= 0 {
		cfg.Concurrency = 25
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 60
	}
	if cfg.ExportDirectory == "" {
		cfg.ExportDirectory = "/app/files/"
	}
	if cfg.ExtractType == "" {
		cfg.ExtractType = "all"
	}
	// Ensure trailing slash on URL.
	if cfg.URL != "" && cfg.URL[len(cfg.URL)-1] != '/' {
		cfg.URL += "/"
	}
}

func detectVersion(ctx context.Context, cfg ExtractConfig) (float64, error) {
	// Make a temporary client with version 10 (bearer auth) to fetch version.
	var opts []sqapi.Option
	opts = append(opts, sqapi.WithTimeout(cfg.Timeout))
	if cfg.PEMFilePath != "" {
		opts = append(opts, sqapi.WithClientCert(cfg.PEMFilePath, cfg.KeyFilePath, cfg.CertPassword))
	}
	tmpClient := sqapi.NewServerClient(cfg.URL, cfg.Token, 10.0, opts...)
	sc := server.New(tmpClient)
	return sc.System.Version(ctx)
}

func detectEdition(ctx context.Context, raw *RawClient) (Edition, error) {
	body, err := raw.Get(ctx, "api/system/info", nil)
	if err != nil {
		// /api/system/info requires admin; fall back to /api/navigation/global
		// which returns the same "edition" field without elevated privileges.
		if common.IsHTTPError(err, 403) {
			body, err = raw.Get(ctx, "api/navigation/global", nil)
			if err != nil {
				return EditionCommunity, err
			}
			return ParseEdition(body), nil
		}
		return EditionCommunity, err
	}
	return ParseEdition(body), nil
}

// generateRunID returns an ISO-date-prefixed extract ID (issue
// #108). Format: "YYYY-MM-DD-NN". See migrate.generateRunID for the
// rationale — the two helpers are deliberately kept in sync.
func generateRunID(directory string) string {
	today := time.Now().UTC().Format("2006-01-02")
	entries, _ := os.ReadDir(directory)
	count := 0
	for _, e := range entries {
		if e.IsDir() && len(e.Name()) > len(today) && e.Name()[:len(today)] == today {
			count++
		}
	}
	return fmt.Sprintf("%s-%02d", today, count+1)
}

// extractMeta groups the parameters for writeMetadata.
type extractMeta struct {
	Plan     [][]string
	RunID    string
	Version  float64
	Edition  Edition
	URL      string
	Targets  []string
	Registry map[string]*TaskDef
}

func writeMetadataFile(dir string, m extractMeta) error {
	configs := make([]string, 0, len(m.Registry))
	for name := range m.Registry {
		configs = append(configs, name)
	}

	meta := map[string]any{
		"plan":              m.Plan,
		"version":           m.Version,
		"edition":           string(m.Edition),
		"url":               m.URL,
		"target_tasks":      m.Targets,
		"available_configs": configs,
		"run_id":            m.RunID,
	}
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "extract.json"), data, 0o644)
}

func filterCompleted(plan [][]string, store *DataStore) [][]string {
	var filtered [][]string
	for _, phase := range plan {
		var fp []string
		for _, task := range phase {
			if !store.TaskDirExists(task) {
				fp = append(fp, task)
			}
		}
		if len(fp) > 0 {
			filtered = append(filtered, fp)
		}
	}
	return filtered
}
