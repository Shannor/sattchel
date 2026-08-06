package driving

import (
	"encoding/json"
	"errors"
	"net/http"
	"sattchel/internal/tracker/core"
)

// MoveRequest represents the payload to change a goal's parent.
type MoveRequest struct {
	ProjectID   string `json:"projectId"`
	ChildID     string `json:"childId"`
	NewParentID string `json:"newParentId"`
}

// UpdateGoalRequest represents the payload to update a goal's details.
type UpdateGoalRequest struct {
	GoalID      string          `json:"goalId"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Status      core.GoalStatus `json:"status"`
	Impact      core.Impact     `json:"impact"`
	Effort      core.Effort     `json:"effort"`
	MemberID    string          `json:"memberId"`
}

func (s *HTTPServer) handleUpdateGoal(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req UpdateGoalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request: "+err.Error(), http.StatusBadRequest)
		return
	}

	opts := core.GoalOptions{
		Description: req.Description,
		Status:      req.Status,
		Impact:      req.Impact,
		Effort:      req.Effort,
		MemberID:    req.MemberID,
	}

	_, err := s.service.UpdateGoal(r.Context(), req.GoalID, req.Name, opts)
	if err != nil {
		http.Error(w, "failed to update goal: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

func (s *HTTPServer) handleGetMembers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	members, err := s.service.GetMembers(r.Context())
	if err != nil {
		http.Error(w, "failed to get members: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(members); err != nil {
		http.Error(w, "failed to encode response: "+err.Error(), http.StatusInternalServerError)
		return
	}
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
		if errors.Is(err, core.ErrCannotMoveRoot) {
			http.Error(w, "failed to move goal: "+err.Error(), http.StatusBadRequest)
			return
		}
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
