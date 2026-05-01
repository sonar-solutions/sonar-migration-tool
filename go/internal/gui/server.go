package gui

import (
	"context"
	"fmt"
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
	tmpl      *Templates
	listener  net.Listener
	ready     chan struct{} // closed once the listener is bound
}

// NewServer creates a Server that serves Go templates and routes
// API/WebSocket requests.
func NewServer(addr, exportDir string, hub *Hub, tmpl *Templates) *Server {
	return &Server{
		addr:      addr,
		exportDir: exportDir,
		hub:       hub,
		tmpl:      tmpl,
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

	// JSON API (kept for potential HTMX partial fetches).
	mux.HandleFunc("GET /api/runs", RunsListHandler(s.exportDir))
	mux.HandleFunc("GET /api/runs/{runId}", RunDetailHandler(s.exportDir))
	mux.HandleFunc("GET /api/runs/{runId}/analysis", RunAnalysisHandler(s.exportDir))
	mux.HandleFunc("GET /api/reports/{type}", GenerateReportHandler(s.exportDir))
	mux.HandleFunc("GET /api/state", WizardStateHandler(s.exportDir))
	mux.HandleFunc("GET /api/report/pdf", ReportPDFHandler(s.exportDir))

	// Static files (CSS, JS).
	mux.Handle("GET /static/", StaticHandler())

	// Template-rendered pages.
	mux.HandleFunc("GET /history/{runId}", RunDetailPageHandler(s.tmpl, s.exportDir))
	mux.HandleFunc("GET /history", HistoryPageHandler(s.tmpl, s.exportDir))
	mux.HandleFunc("GET /", WizardPageHandler(s.tmpl))

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
