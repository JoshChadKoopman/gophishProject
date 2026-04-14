package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gophish/gophish/models"
	"github.com/gorilla/mux"
)

// ReportButtonConfig handles GET/PUT /api/report-button/config — manages
// the org's report button plugin configuration.
func (as *Server) ReportButtonConfig(w http.ResponseWriter, r *http.Request) {
	scope := getOrgScope(r)
	switch r.Method {
	case http.MethodGet:
		config, err := models.GetReportButtonConfig(scope.OrgId)
		if err != nil {
			// Return empty config if none exists
			JSONResponse(w, models.ReportButtonConfig{OrgId: scope.OrgId}, http.StatusOK)
			return
		}
		JSONResponse(w, config, http.StatusOK)
	case http.MethodPut:
		config := models.ReportButtonConfig{}
		if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: "Invalid request body"}, http.StatusBadRequest)
			return
		}
		config.OrgId = scope.OrgId
		// Check if config already exists
		existing, err := models.GetReportButtonConfig(scope.OrgId)
		if err == nil {
			config.Id = existing.Id
			config.PluginApiKey = existing.PluginApiKey
			config.CreatedDate = existing.CreatedDate
		}
		if err := models.CreateReportButtonConfig(&config); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, config, http.StatusOK)
	default:
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
	}
}

// ReportButtonRegenerateKey handles POST /api/report-button/regenerate-key —
// generates a new plugin API key for the org.
func (as *Server) ReportButtonRegenerateKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	scope := getOrgScope(r)
	config, err := models.RegeneratePluginAPIKey(scope.OrgId)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, config, http.StatusOK)
}

// ReportedEmails handles GET /api/reported-emails — returns all reported
// emails for the admin's org.
func (as *Server) ReportedEmails(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	scope := getOrgScope(r)
	emails, err := models.GetReportedEmails(scope.OrgId)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, emails, http.StatusOK)
}

// ReportedEmailClassify handles PUT /api/reported-emails/{id}/classify —
// allows an admin to classify a reported email.
func (as *Server) ReportedEmailClassify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	scope := getOrgScope(r)
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "Invalid ID"}, http.StatusBadRequest)
		return
	}
	var req struct {
		Classification string `json:"classification"`
		AdminNotes     string `json:"admin_notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "Invalid request body"}, http.StatusBadRequest)
		return
	}
	if req.Classification == "" {
		JSONResponse(w, models.Response{Success: false, Message: "classification is required"}, http.StatusBadRequest)
		return
	}
	if err := models.ClassifyReportedEmail(id, scope.OrgId, req.Classification, req.AdminNotes); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, models.Response{Success: true, Message: "Email classified successfully"}, http.StatusOK)
}
