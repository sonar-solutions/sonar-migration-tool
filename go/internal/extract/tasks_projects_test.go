package extract

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/sonar-solutions/sonar-migration-tool/internal/common"
)

// TestProjectTagsTaskFetchesAndFiltersEmpty verifies that getProjectTags
// iterates over each project in getProjects, calls
// /api/components/show?component=<key>, emits one record per project
// whose tags array is non-empty, and skips projects with no tags.
func TestProjectTagsTaskFetchesAndFiltersEmpty(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/components/show", func(w http.ResponseWriter, r *http.Request) {
		comp := r.URL.Query().Get("component")
		switch comp {
		case "proj-with-tags":
			json.NewEncoder(w).Encode(map[string]any{
				"component": map[string]any{"key": comp, "name": "P1", "tags": []string{"java", "backend"}},
			})
		case "proj-empty":
			json.NewEncoder(w).Encode(map[string]any{
				"component": map[string]any{"key": comp, "name": "P2", "tags": []string{}},
			})
		case "proj-no-tags":
			json.NewEncoder(w).Encode(map[string]any{
				"component": map[string]any{"key": comp, "name": "P3"},
			})
		default:
			http.NotFound(w, r)
		}
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	dir := t.TempDir()
	store := NewDataStore(dir)
	raw := common.NewRawClient(srv.Client(), srv.URL+"/")

	e := &Executor{
		Raw:       raw,
		Store:     store,
		ServerURL: "https://test.example.com/",
		Logger:    slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn})),
		Sem:       make(chan struct{}, 1),
	}

	// Pre-populate getProjects (the dependency this task iterates over).
	pw, _ := store.Writer("getProjects")
	for _, key := range []string{"proj-with-tags", "proj-empty", "proj-no-tags"} {
		b, _ := json.Marshal(map[string]any{"key": key})
		pw.WriteOne(b)
	}

	if err := projectTagsTask()(context.Background(), e); err != nil {
		t.Fatalf("projectTagsTask: %v", err)
	}

	items, err := store.ReadAll("getProjectTags")
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 record (only proj-with-tags qualifies), got %d", len(items))
	}
	var rec map[string]any
	if err := json.Unmarshal(items[0], &rec); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if rec["projectKey"] != "proj-with-tags" {
		t.Errorf("expected projectKey=proj-with-tags, got %v", rec["projectKey"])
	}
	tags, _ := rec["tags"].([]any)
	if len(tags) != 2 {
		t.Fatalf("expected 2 tags, got %v", tags)
	}
}

func TestExtractTagsArray(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want []string
	}{
		{"no tags field", `{"key":"a"}`, nil},
		{"null tags", `{"key":"a","tags":null}`, nil},
		{"empty tags array", `{"key":"a","tags":[]}`, []string{}},
		{"non-empty tags", `{"key":"a","tags":["x","y"]}`, []string{"x", "y"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := extractTagsArray(json.RawMessage(tc.in))
			if !equalStringSlices(got, tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

// equalStringSlices treats nil and empty slices as different (matching test
// expectations) but otherwise compares element-by-element.
func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

