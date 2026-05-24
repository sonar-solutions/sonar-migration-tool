package migrate

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

// TestRunResetGlobalSettings verifies the new task posts /api/settings/reset
// with the keys reported as customized (Inherited=false) by SQC, and
// omits keys that are still at their inherited default.
func TestRunResetGlobalSettings(t *testing.T) {
	var (
		mu          sync.Mutex
		resetKeys   []string
		resetOrg    string
		valuesCalls int
	)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/settings/values", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		valuesCalls++
		mu.Unlock()
		json.NewEncoder(w).Encode(map[string]any{
			"settings": []map[string]any{
				{"key": "sonar.exclusions", "values": []string{"**/*.gen.java"}, "inherited": false},
				{"key": "sonar.coverage.exclusions", "value": "**/*.test.java", "inherited": false},
				{"key": "sonar.inherited.example", "value": "default-value", "inherited": true},
			},
		})
	})
	mux.HandleFunc("POST /api/settings/reset", func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		mu.Lock()
		resetKeys = append(resetKeys, r.FormValue("keys"))
		resetOrg = r.FormValue("organization")
		mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	})
	cloudSrv := httptest.NewServer(mux)
	defer cloudSrv.Close()

	apiSrv := newMockAPIServer()
	defer apiSrv.Close()

	dir := t.TempDir()
	setupExtractData(dir)
	e := newTestExecutor(cloudSrv, apiSrv, dir)

	// Seed the org mapping so the task has one org to iterate.
	w, _ := e.Store.Writer("generateOrganizationMappings")
	b, _ := json.Marshal(map[string]any{
		"sonarqube_org_key":  "org1",
		"sonarcloud_org_key": testCloudOrg,
	})
	w.WriteOne(b)

	if err := runResetGlobalSettings(context.Background(), e); err != nil {
		t.Fatalf("runResetGlobalSettings: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if valuesCalls != 1 {
		t.Errorf("Settings.Values expected once, got %d", valuesCalls)
	}
	if len(resetKeys) != 1 {
		t.Fatalf("expected 1 Reset call, got %d", len(resetKeys))
	}
	// Inherited setting must not appear; customized ones must be
	// joined with a comma (SonarQube Web API expects a single "keys"
	// parameter, not repeated form values).
	wantKeys := "sonar.exclusions,sonar.coverage.exclusions"
	if resetKeys[0] != wantKeys {
		t.Errorf("Reset keys: got %q want %q", resetKeys[0], wantKeys)
	}
	if resetOrg != testCloudOrg {
		t.Errorf("Reset organization: got %q want %q", resetOrg, testCloudOrg)
	}
}

// TestRunResetGlobalSettingsAllInherited verifies that when an org has
// no customized settings (everything inherited), the task still
// succeeds and does NOT call POST /api/settings/reset (which would 400
// with an empty keys list).
func TestRunResetGlobalSettingsAllInherited(t *testing.T) {
	resetCalls := 0
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/settings/values", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"settings": []map[string]any{
				{"key": "sonar.foo", "value": "default", "inherited": true},
			},
		})
	})
	mux.HandleFunc("POST /api/settings/reset", func(w http.ResponseWriter, r *http.Request) {
		resetCalls++
		w.WriteHeader(http.StatusNoContent)
	})
	cloudSrv := httptest.NewServer(mux)
	defer cloudSrv.Close()

	apiSrv := newMockAPIServer()
	defer apiSrv.Close()

	dir := t.TempDir()
	setupExtractData(dir)
	e := newTestExecutor(cloudSrv, apiSrv, dir)

	w, _ := e.Store.Writer("generateOrganizationMappings")
	b, _ := json.Marshal(map[string]any{
		"sonarcloud_org_key": testCloudOrg,
	})
	w.WriteOne(b)

	if err := runResetGlobalSettings(context.Background(), e); err != nil {
		t.Fatalf("runResetGlobalSettings: %v", err)
	}
	if resetCalls != 0 {
		t.Errorf("Reset should not be called when no customized settings; got %d calls", resetCalls)
	}
}
