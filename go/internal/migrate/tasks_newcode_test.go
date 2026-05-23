package migrate

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
)

// Helper: build a cloud httptest server with capture for both
// /api/settings/set and /api/new_code_periods/set. globalSettingsSetCalls
// records (org, key, value) for each settings POST so the
// runSetGlobalNewCodePeriod tests can assert the two-key write.
type ncdGlobalCall struct {
	org   string
	key   string
	value string
}

// runSetGlobalNCDTest wires the extract + migrate fixtures and runs
// runSetGlobalNewCodePeriod. ncd is the SQS-side global NCD; orgs is
// the generateOrganizationMappings content; logLevel selects the slog
// level for buf.
func runSetGlobalNCDTest(t *testing.T, ncd map[string]any, orgs []map[string]any) (hits []ncdGlobalCall, logs string) {
	t.Helper()
	var (
		mu       sync.Mutex
		recorded []ncdGlobalCall
	)
	cloudMux := http.NewServeMux()
	cloudMux.HandleFunc("POST /api/settings/set", func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		mu.Lock()
		recorded = append(recorded, ncdGlobalCall{
			org:   r.FormValue("organization"),
			key:   r.FormValue("key"),
			value: r.FormValue("value"),
		})
		mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	})
	cloudMux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{})
	})
	cloudSrv := httptest.NewServer(cloudMux)
	t.Cleanup(cloudSrv.Close)

	apiSrv := newMockAPIServer()
	t.Cleanup(apiSrv.Close)

	dir := t.TempDir()
	e := newTestExecutor(cloudSrv, apiSrv, dir)
	var buf bytes.Buffer
	e.Logger = slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	// SQS global NCD extract — written into the extract directory
	// (not the migrate store) because readExtractItems goes through
	// e.Mapping.
	extractDir := filepath.Join(dir, "extract-01", "getGlobalNewCodePeriod")
	if err := os.MkdirAll(extractDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	f, _ := os.Create(filepath.Join(extractDir, "results.1.jsonl"))
	if ncd != nil {
		b, _ := json.Marshal(ncd)
		f.Write(b)
		f.Write([]byte("\n"))
	}
	f.Close()
	// extract.json so structure.ReadExtractData can resolve
	// server URL → extract dir.
	b, _ := json.Marshal(map[string]any{"url": testServerURL})
	os.WriteFile(filepath.Join(dir, "extract-01", "extract.json"), b, 0o644)

	// generateOrganizationMappings lives in the migrate store.
	pw, _ := e.Store.Writer("generateOrganizationMappings")
	for _, o := range orgs {
		bb, _ := json.Marshal(o)
		pw.WriteOne(bb)
	}

	if err := runSetGlobalNewCodePeriod(context.Background(), e); err != nil {
		t.Fatalf("runSetGlobalNewCodePeriod: %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	hits = append(hits, recorded...)
	return hits, buf.String()
}

// SQS NUMBER_OF_DAYS = 30 → each SQC org receives two settings POSTs:
// sonar.leak.period.type=NUMBER_OF_DAYS and sonar.leak.period=30.
// Matches sonar-tools' settings.py::set_new_code_period behaviour.
func TestRunSetGlobalNewCodePeriodFansOutDaysToEveryOrg(t *testing.T) {
	hits, _ := runSetGlobalNCDTest(t,
		map[string]any{"type": "NUMBER_OF_DAYS", "value": "30", "serverUrl": testServerURL},
		[]map[string]any{
			{"sonarcloud_org_key": "orgA"},
			{"sonarcloud_org_key": "orgB"},
		},
	)
	// Two orgs × two keys each = 4 calls.
	if len(hits) != 4 {
		t.Fatalf("expected 4 settings POSTs (2 orgs × 2 keys), got %d: %+v", len(hits), hits)
	}
	sort.Slice(hits, func(i, j int) bool {
		if hits[i].org != hits[j].org {
			return hits[i].org < hits[j].org
		}
		return hits[i].key < hits[j].key
	})
	want := []ncdGlobalCall{
		{org: "orgA", key: "sonar.leak.period", value: "30"},
		{org: "orgA", key: "sonar.leak.period.type", value: "NUMBER_OF_DAYS"},
		{org: "orgB", key: "sonar.leak.period", value: "30"},
		{org: "orgB", key: "sonar.leak.period.type", value: "NUMBER_OF_DAYS"},
	}
	for i, w := range want {
		if hits[i] != w {
			t.Errorf("call %d: got %+v, want %+v", i, hits[i], w)
		}
	}
}

// PREVIOUS_VERSION is SQC's own default — task must not POST anything
// (issue #196 principle: don't migrate settings equal to default).
func TestRunSetGlobalNewCodePeriodSkipsPreviousVersion(t *testing.T) {
	hits, logs := runSetGlobalNCDTest(t,
		map[string]any{"type": "PREVIOUS_VERSION", "serverUrl": testServerURL},
		[]map[string]any{{"sonarcloud_org_key": "orgA"}},
	)
	if len(hits) != 0 {
		t.Errorf("PREVIOUS_VERSION must NOT trigger any settings POST, got %d", len(hits))
	}
	if !strings.Contains(logs, "PREVIOUS_VERSION") || !strings.Contains(logs, "skipping") {
		t.Errorf("expected Info log noting the skip, got:\n%s", logs)
	}
}

// REFERENCE_BRANCH with no value — only the type POST fires, the value
// POST is omitted (we don't send an empty value).
func TestRunSetGlobalNewCodePeriodReferenceBranchWithoutValue(t *testing.T) {
	hits, _ := runSetGlobalNCDTest(t,
		map[string]any{"type": "REFERENCE_BRANCH", "serverUrl": testServerURL},
		[]map[string]any{{"sonarcloud_org_key": "orgA"}},
	)
	if len(hits) != 1 {
		t.Fatalf("expected exactly 1 settings POST (type only, no value), got %d: %+v", len(hits), hits)
	}
	if hits[0].key != "sonar.leak.period.type" || hits[0].value != "REFERENCE_BRANCH" {
		t.Errorf("expected type=REFERENCE_BRANCH, got %+v", hits[0])
	}
}

// SQS sometimes exports the legacy alias DAYS instead of NUMBER_OF_DAYS.
// The migrate task must normalize it before forwarding — same trick as
// sonar-tools' settings.py::set_new_code_period.
func TestRunSetGlobalNewCodePeriodNormalizesLegacyDaysAlias(t *testing.T) {
	hits, _ := runSetGlobalNCDTest(t,
		map[string]any{"type": "DAYS", "value": "7", "serverUrl": testServerURL},
		[]map[string]any{{"sonarcloud_org_key": "orgA"}},
	)
	for _, h := range hits {
		if h.key == "sonar.leak.period.type" && h.value != "NUMBER_OF_DAYS" {
			t.Errorf("DAYS must be normalized to NUMBER_OF_DAYS, got %q", h.value)
		}
	}
}

// TestRunSetNewCodePeriodsTranslatesAndSets verifies that runSetNewCodePeriods
// translates SQS NCD types to their SQC equivalents, omits the value for
// previous_version, and resolves projectKey + branch to the right cloud
// project + organization.
func TestRunSetNewCodePeriodsTranslatesAndSets(t *testing.T) {
	type call struct {
		project, branch, ncdType, value, org string
	}
	var (
		mu       sync.Mutex
		recorded []call
	)
	cloudMux := http.NewServeMux()
	cloudMux.HandleFunc("POST /api/new_code_periods/set", func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		mu.Lock()
		recorded = append(recorded, call{
			project: r.FormValue("project"),
			branch:  r.FormValue("branch"),
			ncdType: r.FormValue("type"),
			value:   r.FormValue("value"),
			org:     r.FormValue("organization"),
		})
		mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	})
	cloudMux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{})
	})
	cloudSrv := httptest.NewServer(cloudMux)
	defer cloudSrv.Close()

	apiSrv := newMockAPIServer()
	defer apiSrv.Close()

	dir := t.TempDir()
	e := newTestExecutor(cloudSrv, apiSrv, dir)

	// Three extract records covering each translated NCD type plus an
	// unmapped one (UNKNOWN) which the task should skip with a warning.
	extractDir := filepath.Join(dir, "extract-01", "getNewCodePeriods")
	if err := os.MkdirAll(extractDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	f, _ := os.Create(filepath.Join(extractDir, "results.1.jsonl"))
	for _, rec := range []map[string]any{
		{"projectKey": "proj-days", "branchKey": "main", "type": "NUMBER_OF_DAYS", "value": "14"},
		{"projectKey": "proj-prev", "branchKey": "main", "type": "PREVIOUS_VERSION", "value": nil},
		{"projectKey": "proj-ref", "branchKey": "main", "type": "REFERENCE_BRANCH", "value": "develop"},
		{"projectKey": "proj-unknown", "branchKey": "main", "type": "UNKNOWN_MODE"},
	} {
		b, _ := json.Marshal(rec)
		f.Write(b)
		f.Write([]byte("\n"))
	}
	f.Close()

	pw, _ := e.Store.Writer("createProjects")
	for _, src := range []map[string]any{
		{"key": "proj-days", "server_url": testServerURL, "sonarcloud_org_key": "org1", "cloud_project_key": "org1_proj-days"},
		{"key": "proj-prev", "server_url": testServerURL, "sonarcloud_org_key": "org1", "cloud_project_key": "org1_proj-prev"},
		{"key": "proj-ref", "server_url": testServerURL, "sonarcloud_org_key": "org1", "cloud_project_key": "org1_proj-ref"},
		{"key": "proj-unknown", "server_url": testServerURL, "sonarcloud_org_key": "org1", "cloud_project_key": "org1_proj-unknown"},
	} {
		b, _ := json.Marshal(src)
		pw.WriteOne(b)
	}

	if err := runSetNewCodePeriods(context.Background(), e); err != nil {
		t.Fatalf("runSetNewCodePeriods: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	// Expect 3 calls — the UNKNOWN_MODE record should be skipped.
	if len(recorded) != 3 {
		t.Fatalf("expected 3 calls, got %d: %+v", len(recorded), recorded)
	}
	sort.Slice(recorded, func(i, j int) bool { return recorded[i].project < recorded[j].project })

	want := []call{
		{project: "org1_proj-days", branch: "main", ncdType: "days", value: "14", org: "org1"},
		{project: "org1_proj-prev", branch: "main", ncdType: "previous_version", value: "", org: "org1"},
		{project: "org1_proj-ref", branch: "main", ncdType: "reference_branch", value: "develop", org: "org1"},
	}
	for i, w := range want {
		if recorded[i] != w {
			t.Errorf("call %d: got %+v, want %+v", i, recorded[i], w)
		}
	}
}
