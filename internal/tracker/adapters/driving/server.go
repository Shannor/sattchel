package driving

import (
	"context"
	"embed"
	"io/fs"
	"net"
	"net/http"
	"sattchel/internal/tracker/core"
)

//go:embed static
var staticFS embed.FS

// HTTPServer drives the core logic through HTTP.
type HTTPServer struct {
	service *core.Service
	server  *http.Server
}

// NewHTTPServer creates a new instance of HTTPServer.
func NewHTTPServer(service *core.Service) *HTTPServer {
	return &HTTPServer{service: service}
}

// Start launches the server on the provided address and returns the bound address & shutdown function.
func (s *HTTPServer) Start(ctx context.Context, listenAddr string) (string, func() error, error) {
	mux := http.NewServeMux()

	// 1. Mount Generic API Endpoints (from api.go)
	mux.HandleFunc("/api/goals", s.handleGetGoals)
	mux.HandleFunc("/api/goals/move", s.handleMoveGoal)

	// 2. Mount UI / Visualizer Page (served from embedded static assets)
	subFS, err := fs.Sub(staticFS, "static")
	if err != nil {
		return "", nil, err
	}
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(subFS))))

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		content, err := fs.ReadFile(subFS, "index.html")
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(content)
	})

	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return "", nil, err
	}

	s.server = &http.Server{Handler: mux}
	go func() {
		_ = s.server.Serve(listener)
	}()

	return listener.Addr().String(), func() error {
		return s.server.Shutdown(context.Background())
	}, nil
}
