package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	log "github.com/gophish/gophish/logger"
	"github.com/gophish/gophish/models"
	"github.com/gorilla/mux"
)

// AutopilotConfig handles GET /api/autopilot/config and PUT /api/autopilot/config.
func (as *Server) AutopilotConfig(w http.ResponseWriter, r *http.Request) {
	scope := getOrgScope(r)

	switch r.Method {
	case http.MethodGet:
		ac, err := models.GetAutopilotConfig(scope.OrgId)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: "Autopilot not configured"}, http.StatusNotFound)
			return
		}
		JSONResponse(w, ac, http.StatusOK)

	case http.MethodPut:
		ac := models.AutopilotConfig{}
		if err := json.NewDecoder(r.Body).Decode(&ac); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: "Invalid JSON"}, http.StatusBadRequest)
			return
		}
		ac.OrgId = scope.OrgId
		if err := models.SaveAutopilotConfig(&ac); err != nil {
			log.Error(err)
			JSONResponse(w, models.Response{Success: false, Message: "Failed to save config"}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, ac, http.StatusOK)

	default:
		JSONResponse(w, models.Response{Success: false, Message: "Method not allowed"}, http.StatusMethodNotAllowed)
	}
}

// AutopilotEnable handles POST /api/autopilot/enable.
func (as *Server) AutopilotEnable(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		JSONResponse(w, models.Response{Success: false, Message: "Method not allowed"}, http.StatusMethodNotAllowed)
		return
	}
	scope := getOrgScope(r)
	if err := models.EnableAutopilot(scope.OrgId); err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusBadRequest)
		return
	}
	JSONResponse(w, models.Response{Success: true, Message: "Autopilot enabled"}, http.StatusOK)
}

// AutopilotDisable handles POST /api/autopilot/disable.
func (as *Server) AutopilotDisable(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		JSONResponse(w, models.Response{Success: false, Message: "Method not allowed"}, http.StatusMethodNotAllowed)
		return
	}
	scope := getOrgScope(r)
	if err := models.DisableAutopilot(scope.OrgId); err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Failed to disable autopilot"}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, models.Response{Success: true, Message: "Autopilot disabled"}, http.StatusOK)
}

// AutopilotSchedule handles GET /api/autopilot/schedule.
func (as *Server) AutopilotSchedule(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: "Method not allowed"}, http.StatusMethodNotAllowed)
		return
	}
	scope := getOrgScope(r)
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	entries, err := models.GetAutopilotSchedule(scope.OrgId, limit)
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Failed to load schedule"}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, entries, http.StatusOK)
}

// AutopilotBlackoutDates handles GET/POST /api/autopilot/blackout.
func (as *Server) AutopilotBlackoutDates(w http.ResponseWriter, r *http.Request) {
	scope := getOrgScope(r)

	switch r.Method {
	case http.MethodGet:
		dates, err := models.GetAutopilotBlackoutDates(scope.OrgId)
		if err != nil {
			log.Error(err)
			JSONResponse(w, models.Response{Success: false, Message: "Failed to load blackout dates"}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, dates, http.StatusOK)

	case http.MethodPost:
		d := models.AutopilotBlackoutDate{}
		if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: "Invalid JSON"}, http.StatusBadRequest)
			return
		}
		d.OrgId = scope.OrgId
		if d.Date == "" {
			JSONResponse(w, models.Response{Success: false, Message: "Date is required (YYYY-MM-DD)"}, http.StatusBadRequest)
			return
		}
		if err := models.CreateAutopilotBlackoutDate(&d); err != nil {
			log.Error(err)
			JSONResponse(w, models.Response{Success: false, Message: "Failed to create blackout date"}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, d, http.StatusCreated)

	default:
		JSONResponse(w, models.Response{Success: false, Message: "Method not allowed"}, http.StatusMethodNotAllowed)
	}
}

// AutopilotBlackoutDate handles DELETE /api/autopilot/blackout/{id}.
func (as *Server) AutopilotBlackoutDate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		JSONResponse(w, models.Response{Success: false, Message: "Method not allowed"}, http.StatusMethodNotAllowed)
		return
	}
	scope := getOrgScope(r)
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "Invalid ID"}, http.StatusBadRequest)
		return
	}
	if err := models.DeleteAutopilotBlackoutDate(id, scope.OrgId); err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Failed to delete blackout date"}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, models.Response{Success: true, Message: "Blackout date deleted"}, http.StatusOK)
}
