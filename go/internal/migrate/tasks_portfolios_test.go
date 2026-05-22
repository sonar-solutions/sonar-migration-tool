package migrate

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestTransformPortfolioRegex(t *testing.T) {
	cases := []struct {
		name    string
		regex   string
		orgKeys []string
		want    string
	}{
		{"anchored simple",
			"^backend-", []string{"org1"}, "^org1_backend-"},
		{"anchored char class",
			"^[A-Z].*", []string{"org1"}, "^org1_[A-Z].*"},
		{"unanchored",
			"backend", []string{"org1"}, "org1_backend"},
		{"two orgs alternation",
			"^foo", []string{"org1", "org2"},
			"^(?:org1_|org2_)foo"},
		{"empty regex stays empty",
			"", []string{"org1"}, ""},
		{"no orgs returns original",
			"^foo", nil, "^foo"},
		{"org key with regex metachars is quoted",
			"^bar", []string{"org-1.x"}, `^org-1\.x_bar`},
		{"duplicate org keys deduplicated",
			"^foo", []string{"org1", "org1"}, "^org1_foo"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := transformPortfolioRegex(tc.regex, tc.orgKeys)
			if got != tc.want {
				t.Errorf("transformPortfolioRegex(%q, %v): got %q, want %q",
					tc.regex, tc.orgKeys, got, tc.want)
			}
		})
	}
}

// TestRunConfigurePortfoliosRegex drives runConfigurePortfolios with pre-built
// JSONL inputs and asserts the PATCH /enterprises/portfolios/<id> body
// carries the rewritten regex + resolved organization UUID.
func TestRunConfigurePortfoliosRegex(t *testing.T) {
	cloudSrv := newMockCloudServer()
	defer cloudSrv.Close()

	var (
		mu        sync.Mutex
		patchBody map[string]any
		patchPath string
	)
	apiMux := http.NewServeMux()
	apiMux.HandleFunc("GET /enterprises/enterprises", func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode([]map[string]any{
			{"id": "ent-1", "key": "test-enterprise"},
		})
	})
	apiMux.HandleFunc("PATCH /enterprises/portfolios/", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		patchPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&patchBody)
		w.WriteHeader(http.StatusNoContent)
	})
	apiSrv := httptest.NewServer(apiMux)
	defer apiSrv.Close()

	dir := t.TempDir()
	e := newTestExecutor(cloudSrv, apiSrv, dir)

	// Stub the extract data the task reads. newTestExecutor wires Mapping to
	// the "extract-01" directory it already created.
	extractTaskDir := filepath.Join(dir, "extract-01", "getPortfolioProjects")
	if err := os.MkdirAll(extractTaskDir, 0o755); err != nil {
		t.Fatalf("mkdir extract task: %v", err)
	}
	writeJSONLLine(t, filepath.Join(extractTaskDir, "results.1.jsonl"), map[string]any{
		"portfolioKey":  "src-portfolio-1",
		"portfolioName": "Backend Portfolio",
		"refKey":        "proj-backend",
		"serverUrl":     testServerURL,
	})

	// createProjects: one project that lives in SQC org "myorg".
	pw, _ := e.Store.Writer("createProjects")
	pw.WriteOne(json.RawMessage(`{"key":"proj-backend","server_url":"` + testServerURL + `","sonarcloud_org_key":"myorg","cloud_project_key":"myorg_proj-backend"}`))

	// createPortfolios: one REGEXP portfolio.
	cpw, _ := e.Store.Writer("createPortfolios")
	cpw.WriteOne(json.RawMessage(`{"source_portfolio_key":"src-portfolio-1","name":"Backend Portfolio","selection_mode":"REGEXP","regexp":"^backend-","cloud_portfolio_id":"cloud-portfolio-1"}`))

	if err := runConfigurePortfolios(context.Background(), e); err != nil {
		t.Fatalf("runConfigurePortfolios: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if patchPath != "/enterprises/portfolios/cloud-portfolio-1" {
		t.Errorf("expected PATCH on cloud-portfolio-1, got %q", patchPath)
	}
	if patchBody["selection"] != "regex" {
		t.Errorf("expected selection=regex, got %v", patchBody["selection"])
	}
	if patchBody["regularExpression"] != "^myorg_backend-" {
		t.Errorf("expected regex rewrite, got %q", patchBody["regularExpression"])
	}
	// branchKey must always be present alongside selection (SQC requirement).
	if branchKey, ok := patchBody["branchKey"]; !ok || branchKey != "" {
		t.Errorf("expected branchKey=\"\", got %v (present=%v)", branchKey, ok)
	}
	orgs, _ := patchBody["organizationIds"].([]any)
	if len(orgs) != 1 || orgs[0] != "myorg-uuid" {
		t.Errorf("expected organizationIds=[myorg-uuid], got %v", orgs)
	}
}

// TestRunConfigurePortfoliosSkipsManual ensures portfolios that come through
// with selection_mode empty / MANUAL are left untouched here (handled by
// setPortfolioProjects instead).
func TestRunConfigurePortfoliosManualBuildsRegex(t *testing.T) {
	cloudSrv := newMockCloudServer()
	defer cloudSrv.Close()

	var (
		mu        sync.Mutex
		patchBody map[string]any
	)
	apiMux := http.NewServeMux()
	apiMux.HandleFunc("GET /enterprises/enterprises", func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode([]map[string]any{{"id": "ent-1", "key": "test-enterprise"}})
	})
	apiMux.HandleFunc("PATCH /enterprises/portfolios/", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		_ = json.NewDecoder(r.Body).Decode(&patchBody)
		w.WriteHeader(http.StatusNoContent)
	})
	apiSrv := httptest.NewServer(apiMux)
	defer apiSrv.Close()

	dir := t.TempDir()
	e := newTestExecutor(cloudSrv, apiSrv, dir)

	// Two SQS projects, both migrated to org "myorg".
	extractTaskDir := filepath.Join(dir, "extract-01", "getPortfolioProjects")
	if err := os.MkdirAll(extractTaskDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	f, _ := os.Create(filepath.Join(extractTaskDir, "results.1.jsonl"))
	for _, ref := range []string{"sqs-proj-a", "sqs-proj-b"} {
		b, _ := json.Marshal(map[string]any{
			"portfolioKey":  "manual-portfolio",
			"portfolioName": "Manual Portfolio",
			"refKey":        ref,
			"serverUrl":     testServerURL,
		})
		f.Write(b)
		f.Write([]byte("\n"))
	}
	f.Close()

	pw, _ := e.Store.Writer("createProjects")
	pw.WriteOne(json.RawMessage(`{"key":"sqs-proj-a","server_url":"` + testServerURL + `","sonarcloud_org_key":"myorg","cloud_project_key":"myorg_sqs-proj-a"}`))
	pw.WriteOne(json.RawMessage(`{"key":"sqs-proj-b","server_url":"` + testServerURL + `","sonarcloud_org_key":"myorg","cloud_project_key":"myorg_sqs-proj-b"}`))

	cpw, _ := e.Store.Writer("createPortfolios")
	cpw.WriteOne(json.RawMessage(`{"source_portfolio_key":"manual-portfolio","name":"Manual Portfolio","selection_mode":"MANUAL","cloud_portfolio_id":"cloud-manual-1"}`))

	if err := runConfigurePortfolios(context.Background(), e); err != nil {
		t.Fatalf("runConfigurePortfolios: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if patchBody["selection"] != "regex" {
		t.Errorf("expected selection=regex for MANUAL→regex translation, got %v", patchBody["selection"])
	}
	got := patchBody["regularExpression"].(string)
	// Both projects must be present in the alternation; ordering is not
	// guaranteed because the input set comes from a map.
	for _, want := range []string{"myorg_sqs-proj-a", "myorg_sqs-proj-b"} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in regex, got %q", want, got)
		}
	}
	if !strings.HasPrefix(got, "^(?:") || !strings.HasSuffix(got, ")$") {
		t.Errorf("expected anchored alternation, got %q", got)
	}
}

func TestRunConfigurePortfoliosSkipsNoneAndRest(t *testing.T) {
	cloudSrv := newMockCloudServer()
	defer cloudSrv.Close()

	patchCalled := false
	apiMux := http.NewServeMux()
	apiMux.HandleFunc("GET /enterprises/enterprises", func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode([]map[string]any{{"id": "ent-1", "key": "test-enterprise"}})
	})
	apiMux.HandleFunc("PATCH /enterprises/portfolios/", func(w http.ResponseWriter, _ *http.Request) {
		patchCalled = true
		w.WriteHeader(http.StatusNoContent)
	})
	apiSrv := httptest.NewServer(apiMux)
	defer apiSrv.Close()

	dir := t.TempDir()
	e := newTestExecutor(cloudSrv, apiSrv, dir)

	cpw, _ := e.Store.Writer("createPortfolios")
	cpw.WriteOne(json.RawMessage(`{"source_portfolio_key":"none-portfolio","name":"None","selection_mode":"NONE","cloud_portfolio_id":"cloud-none"}`))
	cpw.WriteOne(json.RawMessage(`{"source_portfolio_key":"rest-portfolio","name":"Rest","selection_mode":"REST","cloud_portfolio_id":"cloud-rest"}`))

	if err := runConfigurePortfolios(context.Background(), e); err != nil {
		t.Fatalf("runConfigurePortfolios: %v", err)
	}
	if patchCalled {
		t.Error("PATCH must not be called for NONE/REST portfolios")
	}
}

func TestManualPortfolioRegex(t *testing.T) {
	cases := []struct {
		name string
		keys []string
		want string
	}{
		{"single key", []string{"org_proj1"}, "^org_proj1$"},
		{"two keys", []string{"org_a", "org_b"}, "^(?:org_a|org_b)$"},
		{"escapes metachars", []string{"org_proj.1", "org-2_x"}, `^(?:org_proj\.1|org-2_x)$`},
		{"dedups", []string{"a", "a", "b"}, "^(?:a|b)$"},
		{"empty input", nil, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := manualPortfolioRegex(tc.keys); got != tc.want {
				t.Errorf("manualPortfolioRegex(%v): got %q, want %q", tc.keys, got, tc.want)
			}
		})
	}
}

func writeJSONLLine(t *testing.T, path string, item map[string]any) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create %s: %v", path, err)
	}
	defer f.Close()
	b, _ := json.Marshal(item)
	f.Write(b)
	f.Write([]byte("\n"))
}
