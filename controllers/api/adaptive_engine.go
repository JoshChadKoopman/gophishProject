package api

import (
	"encoding/json"
	"net/http"

	ctx "github.com/gophish/gophish/context"
	"github.com/gophish/gophish/models"
)

// AdaptiveEngineConfig handles GET/PUT for /api/adaptive-engine/config.
// GET returns the current adaptive engine configuration for the org.
// PUT updates the configuration.
func (as *Server) AdaptiveEngineConfig(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)
	switch r.Method {
	case http.MethodGet:
		cfg := models.GetAdaptiveEngineConfig(user.OrgId)
		JSONResponse(w, cfg, http.StatusOK)
	case http.MethodPut:
		var cfg models.AdaptiveEngineConfig
		if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: "Invalid request body"}, http.StatusBadRequest)
			return
		}
		cfg.OrgId = user.OrgId
		if err := models.SaveAdaptiveEngineConfig(&cfg); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: "Error saving configuration"}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, cfg, http.StatusOK)
	default:
		JSONResponse(w, models.Response{Success: false, Message: "Method not allowed"}, http.StatusMethodNotAllowed)
	}
}

// AdaptiveEngineSummary handles GET /api/adaptive-engine/summary.
// Returns a high-level overview of the adaptive engine for the current org.
func (as *Server) AdaptiveEngineSummary(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)
	summary := models.GetAdaptiveEngineSummary(user.OrgId)
	JSONResponse(w, summary, http.StatusOK)
}

// AdaptiveEngineHistory handles GET /api/adaptive-engine/history.
// Returns the run history of the adaptive engine for audit purposes.
func (as *Server) AdaptiveEngineHistory(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)
	runs, err := models.GetAdaptiveEngineRunHistory(user.OrgId, 50)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "Error fetching history"}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, runs, http.StatusOK)
}

// AdaptiveEngineRun handles POST /api/adaptive-engine/run.
// Triggers an immediate adaptive engine evaluation for the current org.
func (as *Server) AdaptiveEngineRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		JSONResponse(w, models.Response{Success: false, Message: "Method not allowed"}, http.StatusMethodNotAllowed)
		return
	}
	user := ctx.Get(r, "user").(models.User)
	evaluations, err := models.RunAdaptiveEngine(user.OrgId)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}
	models.RecordAdaptiveEngineRun(user.OrgId, evaluations, 0)
	JSONResponse(w, evaluations, http.StatusOK)
}

// NanolearningStats handles GET /api/nanolearning/stats.
// Returns aggregate nanolearning metrics for the current org.
func (as *Server) NanolearningStats(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)
	stats := models.GetNanolearningStatsForOrg(user.OrgId)
	JSONResponse(w, stats, http.StatusOK)
}

// NanolearningEvents handles GET /api/nanolearning/events.
// Returns nanolearning events for the current user.
func (as *Server) NanolearningEvents(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)
	events, err := models.GetNanolearningEventsForUser(user.Id)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "Error fetching events"}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, events, http.StatusOK)
}
