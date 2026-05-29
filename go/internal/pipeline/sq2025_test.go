package pipeline

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestExtractGroupsV2Success(t *testing.T) {
	var standardCalled bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/v2/authorizations/groups") {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"groups": []map[string]any{{"name": "devs", "description": "Developers"}},
				"page":   map[string]any{"pageIndex": 1, "pageSize": 500, "total": 1},
			})
			return
		}
		standardCalled = true
		http.Error(w, "unexpected call to standard API", http.StatusInternalServerError)
	}))
	defer srv.Close()

	p := newSQ2025(newTestClient(t, srv.URL))
	groups, err := p.ExtractGroups(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if standardCalled {
		t.Error("standard groups API should not be called when V2 succeeds")
	}
	if len(groups) != 1 || groups[0].Name != "devs" {
		t.Errorf("got %+v, want [{Name:devs}]", groups)
	}
}

func TestExtractGroupsV2FallbackOn404(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/v2/authorizations/groups") {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		// Standard API fallback.
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"groups": []map[string]any{{"id": 1, "name": "fallback-group", "description": ""}},
			"paging": map[string]any{"pageIndex": 1, "pageSize": 500, "total": 1},
		})
	}))
	defer srv.Close()

	p := newSQ2025(newTestClient(t, srv.URL))
	groups, err := p.ExtractGroups(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 1 || groups[0].Name != "fallback-group" {
		t.Errorf("got %+v, want [{Name:fallback-group}]", groups)
	}
}

// TestExtractGroupsV2FallbackOnServerError documents the design decision that
// any V2 error (not just 404) triggers the standard-API fallback. This avoids
// leaving callers with an empty group list if the V2 endpoint is temporarily
// unavailable.
func TestExtractGroupsV2FallbackOnServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/v2/authorizations/groups") {
			http.Error(w, "service unavailable", http.StatusServiceUnavailable)
			return
		}
		// Standard API fallback.
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"groups": []map[string]any{{"id": 2, "name": "fallback-group-2", "description": ""}},
			"paging": map[string]any{"pageIndex": 1, "pageSize": 500, "total": 1},
		})
	}))
	defer srv.Close()

	p := newSQ2025(newTestClient(t, srv.URL))
	groups, err := p.ExtractGroups(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 1 || groups[0].Name != "fallback-group-2" {
		t.Errorf("got %+v, want [{Name:fallback-group-2}]", groups)
	}
}

func TestExtractIssuesFiltersINSANDBOX(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"paging": map[string]any{"pageIndex": 1, "pageSize": 500, "total": 2},
			"issues": []map[string]any{
				{
					"key": "open-1", "status": "OPEN", "rule": "rule:1",
					"component": "comp", "type": "BUG",
					"creationDate": "2024-01-01", "updateDate": "2024-01-01",
				},
				{
					"key": "sandbox-1", "status": "IN_SANDBOX", "rule": "rule:2",
					"component": "comp", "type": "BUG",
					"creationDate": "2024-01-01", "updateDate": "2024-01-01",
				},
			},
		})
	}))
	defer srv.Close()

	p := newSQ2025(newTestClient(t, srv.URL))
	issues, err := p.ExtractIssues(context.Background(), "proj")
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) != 1 {
		t.Fatalf("got %d issues, want 1 (IN_SANDBOX should be filtered out)", len(issues))
	}
	if issues[0].Key != "open-1" {
		t.Errorf("unexpected surviving issue key: %s", issues[0].Key)
	}
}
