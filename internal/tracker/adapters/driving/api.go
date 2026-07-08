package driving

import (
	"encoding/json"
	"net/http"
	"sattchel/internal/tracker/core"
)

// MoveRequest represents the payload to change a goal's parent.
type MoveRequest struct {
	ProjectID   string `json:"projectId"`
	ChildID     string `json:"childId"`
	NewParentID string `json:"newParentId"`
}

func (s *HTTPServer) handleMoveGoal(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req MoveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request: "+err.Error(), http.StatusBadRequest)
		return
	}

	_, err := s.service.ChangeParent(r.Context(), req.ProjectID, req.ChildID, req.NewParentID, core.GoalOptions{})
	if err != nil {
		http.Error(w, "failed to move goal: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

func (s *HTTPServer) handleGetGoals(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	projectID := r.URL.Query().Get("projectId")
	goals, err := s.service.GetGoals(r.Context(), projectID)
	if err != nil {
		http.Error(w, "failed to get goals: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(goals); err != nil {
		http.Error(w, "failed to encode response: "+err.Error(), http.StatusInternalServerError)
		return
	}
}
