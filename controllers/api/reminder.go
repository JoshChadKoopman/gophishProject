package api

import (
	"encoding/json"
	"net/http"

	ctx "github.com/gophish/gophish/context"
	"github.com/gophish/gophish/models"
)

// ReminderConfig handles GET/PUT for /api/reminders/config.
// GET returns the current reminder configuration for the org.
// PUT updates the configuration.
func (as *Server) ReminderConfig(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)
	switch r.Method {
	case http.MethodGet:
		cfg := models.GetReminderConfig(user.OrgId)
		JSONResponse(w, cfg, http.StatusOK)
	case http.MethodPut:
		var cfg models.ReminderConfig
		if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: "Invalid request body"}, http.StatusBadRequest)
			return
		}
		cfg.OrgId = user.OrgId
		if err := models.SaveReminderConfig(&cfg); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: "Error saving configuration"}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, cfg, http.StatusOK)
	default:
		JSONResponse(w, models.Response{Success: false, Message: "Method not allowed"}, http.StatusMethodNotAllowed)
	}
}

// ReminderStats handles GET /api/reminders/stats.
// Returns aggregate reminder statistics for the current org.
func (as *Server) ReminderStats(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)
	stats := models.GetReminderStatsForOrg(user.OrgId)
	JSONResponse(w, stats, http.StatusOK)
}

// ReminderHistory handles GET /api/reminders/history.
// Returns recent reminders sent for the current user.
func (as *Server) ReminderHistory(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)
	reminders, err := models.GetUserReminders(user.Id, 50)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "Error fetching reminders"}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, reminders, http.StatusOK)
}
