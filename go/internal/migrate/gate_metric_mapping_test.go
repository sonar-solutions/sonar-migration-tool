package migrate

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"sync"
	"testing"

	"github.com/sonar-solutions/sq-api-go/types"
)

func TestLookupMetricReplacement(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		wantOK  bool
		wantOut []string
	}{
		{"unmapped passes through", "coverage", false, nil},
		{"aica suffix dropped", "new_security_rating_with_aica", true, []string{"new_security_rating"}},
		{"software_quality_* rating", "software_quality_security_rating", true, []string{"security_rating"}},
		{"debt ratio rename", "software_quality_maintainability_debt_ratio", true, []string{"sqale_debt_ratio"}},
		{"composite — software_quality_blocker_issues",
			"software_quality_blocker_issues",
			true, []string{"security_review_rating", "reliability_rating"}},
		{"no SQC equivalent → drop",
			"contains_ai_code", true, []string{}},
		{"new_software_quality_maintainability_issues → new_issues",
			"new_software_quality_maintainability_issues", true, []string{"new_issues"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := lookupMetricReplacement(tc.input)
			if ok != tc.wantOK {
				t.Fatalf("ok: got %v, want %v", ok, tc.wantOK)
			}
			if len(got) != len(tc.wantOut) {
				t.Fatalf("targets length: got %v, want %v", got, tc.wantOut)
			}
			for i := range got {
				if got[i] != tc.wantOut[i] {
					t.Errorf("targets[%d]: got %q, want %q", i, got[i], tc.wantOut[i])
				}
			}
		})
	}
}

// TestAddGateConditionsAppliesMetricMapping exercises the end-to-end
// path: source conditions on mapped/unmapped/dropped metrics, asserting
// the CreateCondition payloads the migrator actually sends to SQC.
func TestAddGateConditionsAppliesMetricMapping(t *testing.T) {
	var (
		mu       sync.Mutex
		recorded []map[string]string
	)
	cloudMux := http.NewServeMux()
	cloudMux.HandleFunc("POST /api/qualitygates/create_condition", func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		mu.Lock()
		recorded = append(recorded, map[string]string{
			"metric": r.FormValue("metric"),
			"op":     r.FormValue("op"),
			"error":  r.FormValue("error"),
		})
		mu.Unlock()
		_ = json.NewEncoder(w).Encode(types.QualityGateCondition{ID: 1, Metric: r.FormValue("metric")})
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

	// Mix of source metrics:
	//  - coverage (unmapped, pass-through)
	//  - new_security_rating_with_aica (1:1 rename)
	//  - software_quality_blocker_issues (composite: 2 conditions)
	//  - contains_ai_code (no SQC equivalent → dropped, NO HTTP call)
	w, _ := e.Store.Writer("getGateConditions")
	payload := map[string]any{
		"gate_name":          "Custom",
		"sonarcloud_org_key": "cloud-org1",
		"cloud_gate_id":      "42",
		"was_preexisting":    false,
		"conditions": []map[string]any{
			{"metric": "coverage", "op": "LT", "error": "80"},
			{"metric": "new_security_rating_with_aica", "op": "GT", "error": "1"},
			{"metric": "software_quality_blocker_issues", "op": "GT", "error": "0"},
			{"metric": "contains_ai_code", "op": "GT", "error": "0"},
		},
	}
	b, _ := json.Marshal(payload)
	w.WriteOne(b)

	if err := runAddGateConditions(context.Background(), e); err != nil {
		t.Fatalf("runAddGateConditions: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	// Expected: 1 (coverage) + 1 (new_security_rating) + 2 (composite) = 4
	if len(recorded) != 4 {
		t.Fatalf("expected 4 create_condition calls (1 + 1 + 2, with contains_ai_code dropped), got %d: %v",
			len(recorded), recorded)
	}
	metrics := make([]string, 0, len(recorded))
	for _, r := range recorded {
		metrics = append(metrics, r["metric"])
	}
	sort.Strings(metrics)
	want := []string{"coverage", "new_security_rating", "reliability_rating", "security_review_rating"}
	for i := range want {
		if metrics[i] != want[i] {
			t.Errorf("metrics[%d]: got %q, want %q (full=%v)", i, metrics[i], want[i], metrics)
		}
	}
	// contains_ai_code should never reach create_condition.
	for _, r := range recorded {
		if r["metric"] == "contains_ai_code" {
			t.Errorf("contains_ai_code should have been dropped, but a CreateCondition call was made: %v", r)
		}
	}
}
