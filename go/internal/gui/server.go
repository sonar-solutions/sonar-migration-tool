package gui

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"time"
)

const shutdownTimeout = 5 * time.Second

// Server is the browser-based GUI HTTP server.
type Server struct {
	addr      string
	exportDir string
	hub       *Hub
	frontend  fs.FS
	listener  net.Listener
	ready     chan struct{} // closed once the listener is bound
}

// NewServer creates a Server that serves the given frontend filesystem
// and routes API/WebSocket requests.
func NewServer(addr, exportDir string, hub *Hub, frontend fs.FS) *Server {
	return &Server{
		addr:      addr,
		exportDir: exportDir,
		hub:       hub,
		frontend:  frontend,
		ready:     make(chan struct{}),
	}
}

// Ready returns a channel that is closed when the listener is bound.
func (s *Server) Ready() <-chan struct{} { return s.ready }

// URL returns the base URL where the server is listening.
// Must be called after Ready is closed.
func (s *Server) URL() string {
	if s.listener == nil {
		return ""
	}
	return fmt.Sprintf("http://%s", s.listener.Addr().String())
}

// Start binds the listener, builds the mux, and serves until ctx is cancelled.
func (s *Server) Start(ctx context.Context) error {
	ln, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("gui: listen: %w", err)
	}
	s.listener = ln
	close(s.ready)

	mux := http.NewServeMux()

	// WebSocket.
	mux.HandleFunc("GET /ws", s.hub.ServeWS)

	// REST API.
	mux.HandleFunc("GET /api/runs", RunsListHandler(s.exportDir))
	mux.HandleFunc("GET /api/runs/{runId}", RunDetailHandler(s.exportDir))
	mux.HandleFunc("GET /api/runs/{runId}/analysis", RunAnalysisHandler(s.exportDir))
	mux.HandleFunc("GET /api/reports/{type}", GenerateReportHandler(s.exportDir))
	mux.HandleFunc("GET /api/state", WizardStateHandler(s.exportDir))

	// SPA static files — fallback to index.html for client-side routing.
	mux.Handle("/", spaHandler(s.frontend))

	srv := &http.Server{Handler: mux}

	// Graceful shutdown goroutine.
	go func() {
		<-ctx.Done()
		shutCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := srv.Shutdown(shutCtx); err != nil {
			log.Printf("gui: shutdown error: %v", err)
		}
	}()

	log.Printf("gui: serving on %s", ln.Addr())
	if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("gui: serve: %w", err)
	}
	return nil
}

// spaHandler serves static files, falling back to index.html for unknown paths.
func spaHandler(frontend fs.FS) http.Handler {
	fileServer := http.FileServer(http.FS(frontend))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try serving the exact file.
		path := r.URL.Path
		if path == "/" {
			path = "index.html"
		} else if len(path) > 0 && path[0] == '/' {
			path = path[1:]
		}

		if _, err := fs.Stat(frontend, path); err == nil {
			fileServer.ServeHTTP(w, r)
			return
		}

		// Fallback to index.html for SPA client-side routing.
		r.URL.Path = "/"
		fileServer.ServeHTTP(w, r)
	})
}
