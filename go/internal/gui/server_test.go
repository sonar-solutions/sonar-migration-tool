package gui

import (
	"context"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
	"time"
)

func TestSPAHandlerServesStaticFile(t *testing.T) {
	frontend := fstest.MapFS{
		"index.html":      {Data: []byte("<html>app</html>")},
		"assets/style.css": {Data: []byte("body{}")},
	}

	handler := spaHandler(frontend)
	req := httptest.NewRequest("GET", "/assets/style.css", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: %d", w.Code)
	}
	if w.Body.String() != "body{}" {
		t.Errorf("body: %q", w.Body.String())
	}
}

func TestSPAHandlerFallsBackToIndex(t *testing.T) {
	frontend := fstest.MapFS{
		"index.html": {Data: []byte("<html>app</html>")},
	}

	handler := spaHandler(frontend)
	req := httptest.NewRequest("GET", "/history/04-20-2026-01", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: %d", w.Code)
	}
	if w.Body.String() != "<html>app</html>" {
		t.Errorf("body: %q", w.Body.String())
	}
}

func TestSPAHandlerServesRoot(t *testing.T) {
	frontend := fstest.MapFS{
		"index.html": {Data: []byte("<html>root</html>")},
	}

	handler := spaHandler(frontend)
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: %d", w.Code)
	}
}

func TestServerStartAndShutdown(t *testing.T) {
	wp := NewWebPrompter(context.Background(), func(ServerMessage) {})
	hub := NewHub(wp)

	frontend := fstest.MapFS{
		"index.html": {Data: []byte("<html>test</html>")},
	}

	srv := NewServer("localhost:0", t.TempDir(), hub, frontend)

	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Start(ctx)
	}()

	// Wait for ready.
	select {
	case <-srv.Ready():
	case <-time.After(2 * time.Second):
		t.Fatal("server did not become ready")
	}

	url := srv.URL()
	if url == "" {
		t.Fatal("URL should not be empty after ready")
	}

	// Verify it serves requests.
	resp, err := http.Get(url + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET / status: %d", resp.StatusCode)
	}

	// Shutdown.
	cancel()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("Start returned error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("server did not shut down")
	}
}

func TestServerAPIRoutes(t *testing.T) {
	wp := NewWebPrompter(context.Background(), func(ServerMessage) {})
	hub := NewHub(wp)

	exportDir := t.TempDir()

	// Create a run directory for the API to find.
	runDir := filepath.Join(exportDir, "04-20-2026-01")
	os.MkdirAll(runDir, 0o755)
	os.WriteFile(filepath.Join(runDir, "extract.json"),
		[]byte(`{"url":"https://test.com/"}`), 0o644)

	frontend := fstest.MapFS{
		"index.html": {Data: []byte("<html>test</html>")},
	}

	srv := NewServer("localhost:0", exportDir, hub, frontend)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go srv.Start(ctx)
	<-srv.Ready()
	base := srv.URL()

	// Test /api/runs.
	resp, err := http.Get(base + "/api/runs")
	if err != nil {
		t.Fatalf("GET /api/runs: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("/api/runs status: %d", resp.StatusCode)
	}

	// Test /api/state.
	resp, err = http.Get(base + "/api/state")
	if err != nil {
		t.Fatalf("GET /api/state: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("/api/state status: %d", resp.StatusCode)
	}
}

func TestServerURLBeforeReady(t *testing.T) {
	frontend, _ := fs.Sub(fstest.MapFS{"index.html": {Data: []byte("")}}, ".")
	srv := NewServer("localhost:0", t.TempDir(), nil, frontend)
	if url := srv.URL(); url != "" {
		t.Errorf("URL before start should be empty, got %q", url)
	}
}
