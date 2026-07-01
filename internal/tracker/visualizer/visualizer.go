package visualizer

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"sattchel/internal/tracker/core"
	"sattchel/internal/tracker/visualizer/templates"
)

// MoveRequest represents the payload to change a goal's parent
type MoveRequest struct {
	ChildID     string `json:"childId"`
	NewParentID string `json:"newParentId"`
}

// StartServer starts an ephemeral HTTP server on a random port and returns the URL and a shutdown function
func StartServer(ctx context.Context, goals []core.Goal, service *core.Service, projectID string) (string, func() error, error) {
	mux := http.NewServeMux()

	// Render the mind map page dynamically on each request
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		goals, err := service.GetGoals(r.Context(), projectID)
		if err != nil {
			http.Error(w, "failed to get goals: "+err.Error(), http.StatusInternalServerError)
			return
		}

		goalsJSON, err := json.Marshal(goals)
		if err != nil {
			http.Error(w, "failed to serialize goals: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		component := templates.MindmapPage(string(goalsJSON))
		_ = component.Render(r.Context(), w)
	})

	// API to support moving goals from the UI in the future
	mux.HandleFunc("/api/goals/move", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req MoveRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request: "+err.Error(), http.StatusBadRequest)
			return
		}

		fmt.Printf("[DEBUG] Move request received: ChildID=%s, NewParentID=%s\n", req.ChildID, req.NewParentID)

		// Change parent in core service
		_, err := service.ChangeParent(r.Context(), projectID, req.ChildID, req.NewParentID, core.GoalOptions{})
		if err != nil {
			fmt.Printf("[DEBUG] Move request failed: %v\n", err)
			http.Error(w, "failed to move goal: "+err.Error(), http.StatusInternalServerError)
			return
		}

		fmt.Println("[DEBUG] Move request completed successfully")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	// Bind to port 0 (automatically allocates a free local port)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", nil, fmt.Errorf("listen on free port: %w", err)
	}

	addr := listener.Addr().String()
	url := fmt.Sprintf("http://%s", addr)

	server := &http.Server{
		Handler: mux,
	}

	go func() {
		_ = server.Serve(listener)
	}()

	shutdownFunc := func() error {
		return server.Shutdown(context.Background())
	}

	return url, shutdownFunc, nil
}
