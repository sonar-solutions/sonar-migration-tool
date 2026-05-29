package pipeline

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	sqapi "github.com/sonar-solutions/sq-api-go"
)

// newTestClient creates a sqapi.Client pointing at a test HTTP server.
func newTestClient(t *testing.T, baseURL string) *sqapi.Client {
	t.Helper()
	return sqapi.NewServerClient(baseURL, "test-token", 10.0)
}

func TestPaginateAll(t *testing.T) {
	data := [][]string{{"a", "b"}, {"c", "d"}, {"e", "f"}}
	calls := 0
	items, err := paginateAll(context.Background(), func(_ context.Context, page int) ([]string, int, error) {
		calls++
		if page <= len(data) {
			return data[page-1], 6, nil
		}
		return nil, 6, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 6 {
		t.Fatalf("got %d items, want 6", len(items))
	}
	// Pages 1-3 are fetched; on page 3 len(all)==total so the loop breaks.
	if calls != 3 {
		t.Errorf("expected 3 fetches, got %d", calls)
	}
}

func TestPaginateAllEmpty(t *testing.T) {
	calls := 0
	items, err := paginateAll(context.Background(), func(_ context.Context, _ int) ([]string, int, error) {
		calls++
		return nil, 0, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("got %d items, want 0", len(items))
	}
	if calls != 1 {
		t.Errorf("expected 1 fetch (empty first page stops immediately), got %d", calls)
	}
}

// metricsHandler returns an httptest handler that serves
// /api/measures/component_tree. The componentsByMetric map maps a metric key
// to the components (and their single measure) to return.
func metricsHandler(t *testing.T, componentsByMetric map[string]string, statusByMetric map[string]int) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		metricKey := r.URL.Query().Get("metricKeys")
		if code, ok := statusByMetric[metricKey]; ok {
			http.Error(w, "error", code)
			return
		}
		compKey := componentsByMetric[metricKey]
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"paging": map[string]any{"pageIndex": 1, "pageSize": 500, "total": 1},
			"components": []map[string]any{
				{"key": compKey, "measures": []map[string]any{{"metric": metricKey, "value": "42"}}},
			},
		})
	}
}

func TestFetchAllMetricsBatchMergesSameComponent(t *testing.T) {
	srv := httptest.NewServer(metricsHandler(t,
		map[string]string{"lines": "my-comp", "coverage": "my-comp"},
		nil,
	))
	defer srv.Close()

	result, err := fetchAllMetrics(context.Background(), newTestClient(t, srv.URL), "proj", []string{"lines", "coverage"}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 {
		t.Fatalf("got %d components, want 1 (same component in both batches)", len(result))
	}
	if len(result[0].Measures) != 2 {
		t.Fatalf("got %d measures, want 2 (merged from 2 batches)", len(result[0].Measures))
	}
}

func TestFetchAllMetricsBatchComponentOnlyInSecondBatch(t *testing.T) {
	srv := httptest.NewServer(metricsHandler(t,
		map[string]string{"lines": "comp-a", "coverage": "comp-b"},
		nil,
	))
	defer srv.Close()

	result, err := fetchAllMetrics(context.Background(), newTestClient(t, srv.URL), "proj", []string{"lines", "coverage"}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 2 {
		t.Fatalf("got %d components, want 2 (different component per batch)", len(result))
	}
}

func TestFetchAllMetricsBatchErrorPropagates(t *testing.T) {
	srv := httptest.NewServer(metricsHandler(t,
		map[string]string{"lines": "comp-a"},
		map[string]int{"coverage": http.StatusInternalServerError},
	))
	defer srv.Close()

	_, err := fetchAllMetrics(context.Background(), newTestClient(t, srv.URL), "proj", []string{"lines", "coverage"}, 1)
	if err == nil {
		t.Fatal("expected error from failed batch, got nil")
	}
	if !strings.Contains(err.Error(), "metrics batch") {
		t.Errorf("error should mention 'metrics batch', got: %v", err)
	}
}
