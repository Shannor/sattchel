package driving

import (
	"encoding/json"
	"net/http"
	"sattchel/internal/tracker/adapters/driving/templates"
)

func (s *HTTPServer) handleVisualizerUI(w http.ResponseWriter, r *http.Request) {
	projectID := r.URL.Query().Get("projectId")
	goals, err := s.service.GetGoals(r.Context(), projectID)
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
}
