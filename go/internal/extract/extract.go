package extract

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

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
	TargetTask      string
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

// IsSkipped returns true if the project has been marked as skipped.
func (e *Executor) IsSkipped(projectKey string) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.skippedProjects[projectKey]
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

	// Create sqapi client with optional mTLS and timeout.
	var opts []sqapi.Option
	opts = append(opts, sqapi.WithTimeout(cfg.Timeout))
	if cfg.PEMFilePath != "" {
		opts = append(opts, sqapi.WithClientCert(cfg.PEMFilePath, cfg.KeyFilePath, cfg.CertPassword))
	}

	// Detect server version.
	version, err := detectVersion(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("detecting server version: %w", err)
	}

	client := sqapi.NewServerClient(cfg.URL, cfg.Token, version, opts...)
	if client.CertErr() != nil {
		return nil, fmt.Errorf("certificate error: %w", client.CertErr())
	}

	// Detect edition via /api/system/info.
	raw := NewRawClient(client.HTTPClient(), client.BaseURL())
	edition, err := detectEdition(ctx, raw)
	if err != nil {
		return nil, fmt.Errorf("detecting edition: %w", err)
	}

	// Generate extract ID.
	extractID := cfg.ExtractID
	if extractID == "" {
		extractID = generateRunID(cfg.ExportDirectory)
	}
	extractDir := filepath.Join(cfg.ExportDirectory, extractID)
	if err := os.MkdirAll(extractDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating extract dir: %w", err)
	}

	// Build task registry and plan.
	allDefs := RegisterAll()
	registry := BuildRegistry(allDefs)
	registry = FilterByEdition(registry, edition)

	targets := TargetTasks(registry, cfg.TargetTask, cfg.ExtractType)
	taskSet := ResolveDependencies(targets, registry)
	if taskSet == nil {
		return nil, fmt.Errorf("cannot resolve dependencies for target tasks")
	}

	plan, err := PlanPhases(taskSet, registry)
	if err != nil {
		return nil, err
	}

	// Write extract.json metadata.
	meta := extractMeta{
		Plan: plan, RunID: extractID, Version: version,
		Edition: edition, URL: cfg.URL, Targets: targets, Registry: registry,
	}
	if err := writeMetadataFile(extractDir, meta); err != nil {
		return nil, err
	}

	// Filter out completed tasks (resumability).
	store := NewDataStore(extractDir)
	plan = filterCompleted(plan, store)

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	executor := &Executor{
		Raw:       raw,
		Store:     store,
		ServerURL: client.BaseURL(),
		Edition:   edition,
		Version:   version,
		Sem:       make(chan struct{}, cfg.Concurrency),
		Logger:    logger,
	}

	// Execute phases.
	for i, phase := range plan {
		logger.Info("starting phase", "phase", i+1, "tasks", len(phase))
		if err := runPhase(ctx, executor, phase, registry); err != nil {
			return nil, fmt.Errorf("phase %d: %w", i+1, err)
		}
		for _, taskName := range phase {
			store.MarkComplete(taskName)
		}
	}

	fmt.Printf("Extract Complete: %s\n", extractID)
	return executor.SkippedProjectKeys(), nil
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
		return EditionCommunity, err
	}
	return ParseEdition(body), nil
}

func generateRunID(directory string) string {
	today := time.Now().UTC().Format("01-02-2006")
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
