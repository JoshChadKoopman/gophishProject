package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	ctx "github.com/gophish/gophish/context"
	"github.com/gophish/gophish/models"
	"github.com/gorilla/mux"
)

// ThreatAlerts handles GET /api/threat-alerts — returns alerts.
// For users with modify_objects or view_reports: all org alerts (admin view).
// For regular users: only published alerts targeted at them.
func (as *Server) ThreatAlerts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: "Method not allowed"}, http.StatusMethodNotAllowed)
		return
	}
	scope := getOrgScope(r)
	user := ctx.Get(r, "user").(models.User)
	canAdmin, _ := user.HasPermission(models.PermissionModifyObjects)
	if canAdmin {
		alerts, err := models.GetThreatAlerts(scope.OrgId)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, alerts, http.StatusOK)
		return
	}
	alerts, err := models.GetPublishedThreatAlerts(scope.OrgId, user.Id)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, alerts, http.StatusOK)
}

// ThreatAlertCreate handles POST /api/threat-alerts — creates a new alert.
func (as *Server) ThreatAlertCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		JSONResponse(w, models.Response{Success: false, Message: "Method not allowed"}, http.StatusMethodNotAllowed)
		return
	}
	scope := getOrgScope(r)
	alert := models.ThreatAlert{}
	if err := json.NewDecoder(r.Body).Decode(&alert); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "Invalid request body"}, http.StatusBadRequest)
		return
	}
	if alert.Title == "" || alert.Body == "" {
		JSONResponse(w, models.Response{Success: false, Message: "title and body are required"}, http.StatusBadRequest)
		return
	}
	alert.OrgId = scope.OrgId
	alert.CreatedBy = scope.UserId
	if err := models.CreateThreatAlert(&alert); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, alert, http.StatusCreated)
}

// ThreatAlert handles GET/PUT/DELETE /api/threat-alerts/{id}.
func (as *Server) ThreatAlert(w http.ResponseWriter, r *http.Request) {
	scope := getOrgScope(r)
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "Invalid ID"}, http.StatusBadRequest)
		return
	}
	switch r.Method {
	case http.MethodGet:
		alert, err := models.GetThreatAlert(id, scope.OrgId)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: "Alert not found"}, http.StatusNotFound)
			return
		}
		// Mark as read for the requesting user
		models.MarkThreatAlertRead(alert.Id, scope.UserId)
		JSONResponse(w, alert, http.StatusOK)
	case http.MethodPut:
		alert := models.ThreatAlert{}
		if err := json.NewDecoder(r.Body).Decode(&alert); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: "Invalid request body"}, http.StatusBadRequest)
			return
		}
		existing, err := models.GetThreatAlert(id, scope.OrgId)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: "Alert not found"}, http.StatusNotFound)
			return
		}
		existing.Title = alert.Title
		existing.Body = alert.Body
		existing.Severity = alert.Severity
		existing.TargetRoles = alert.TargetRoles
		existing.TargetDepartments = alert.TargetDepartments
		existing.Published = alert.Published
		if err := models.UpdateThreatAlert(&existing); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, existing, http.StatusOK)
	case http.MethodDelete:
		if err := models.DeleteThreatAlert(id, scope.OrgId); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, models.Response{Success: true, Message: "Alert deleted"}, http.StatusOK)
	default:
		JSONResponse(w, models.Response{Success: false, Message: "Method not allowed"}, http.StatusMethodNotAllowed)
	}
}

// ThreatAlertUnreadCount handles GET /api/threat-alerts/unread-count —
// returns the number of unread published alerts for the current user.
func (as *Server) ThreatAlertUnreadCount(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: "Method not allowed"}, http.StatusMethodNotAllowed)
		return
	}
	scope := getOrgScope(r)
	count := models.GetUnreadThreatAlertCount(scope.OrgId, scope.UserId)
	JSONResponse(w, struct {
		Count int64 `json:"count"`
	}{Count: count}, http.StatusOK)
}
