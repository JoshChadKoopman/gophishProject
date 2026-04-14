package api

import (
	"encoding/json"
	"net/http"

	ctx "github.com/gophish/gophish/context"
	"github.com/gophish/gophish/models"
)

// ContentUpdateConfig handles GET/PUT for /api/content-updates/config.
// GET returns the current auto-update configuration for the org.
// PUT updates the configuration.
func (as *Server) ContentUpdateConfig(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)
	switch r.Method {
	case http.MethodGet:
		cfg := models.GetContentUpdateConfig(user.OrgId)
		JSONResponse(w, cfg, http.StatusOK)
	case http.MethodPut:
		var cfg models.ContentUpdateConfig
		if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: "Invalid request body"}, http.StatusBadRequest)
			return
		}
		cfg.OrgId = user.OrgId
		if err := models.SaveContentUpdateConfig(&cfg); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: "Error saving configuration"}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, cfg, http.StatusOK)
	default:
		JSONResponse(w, models.Response{Success: false, Message: "Method not allowed"}, http.StatusMethodNotAllowed)
	}
}

// ContentUpdateSummary handles GET /api/content-updates/summary.
// Returns a summary of the content auto-update system for the org.
func (as *Server) ContentUpdateSummary(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)
	summary := models.GetContentUpdateSummary(user.OrgId)
	JSONResponse(w, summary, http.StatusOK)
}

// ContentUpdateHistory handles GET /api/content-updates/history.
// Returns the update history for the org.
func (as *Server) ContentUpdateHistory(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)
	logs, err := models.GetContentUpdateHistory(user.OrgId, 50)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "Error fetching history"}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, logs, http.StatusOK)
}
