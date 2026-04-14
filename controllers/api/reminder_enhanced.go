package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	ctx "github.com/gophish/gophish/context"
	log "github.com/gophish/gophish/logger"
	"github.com/gophish/gophish/models"
	"github.com/gorilla/mux"
)

// ── Automated Training Reminders — Enhanced Endpoints ───────────
// Extends the existing reminder config/stats/history endpoints with
// manual nudge, template management, and assignment-level reminder control.

// ReminderNudge handles POST /api/reminders/nudge.
// Manually triggers a reminder for a specific assignment or user.
func (as *Server) ReminderNudge(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	user := ctx.Get(r, "user").(models.User)

	var req struct {
		AssignmentId int64 `json:"assignment_id"`
		UserId       int64 `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: ErrInvalidJSON}, http.StatusBadRequest)
		return
	}

	if req.AssignmentId > 0 {
		// Nudge for a specific assignment
		a, err := models.GetAssignmentById(req.AssignmentId)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: "Assignment not found"}, http.StatusNotFound)
			return
		}
		if err := models.RecordReminderSent(a.UserId, a.Id, "manual", user.Id); err != nil {
			log.Error(err)
			JSONResponse(w, models.Response{Success: false, Message: "Error sending nudge"}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, models.Response{Success: true, Message: "Nudge sent for assignment"}, http.StatusOK)
		return
	}

	if req.UserId > 0 {
		// Nudge all pending assignments for a user
		assignments, err := models.GetPendingAssignmentsForUser(req.UserId)
		if err != nil {
			log.Error(err)
			JSONResponse(w, models.Response{Success: false, Message: "Error fetching user assignments"}, http.StatusInternalServerError)
			return
		}
		sent := 0
		for _, a := range assignments {
			if err := models.RecordReminderSent(a.UserId, a.Id, "manual", user.Id); err != nil {
				log.Errorf("reminder nudge: failed for assignment %d: %v", a.Id, err)
				continue
			}
			sent++
		}
		JSONResponse(w, struct {
			Success bool   `json:"success"`
			Message string `json:"message"`
			Sent    int    `json:"sent"`
		}{true, "Nudges sent", sent}, http.StatusOK)
		return
	}

	JSONResponse(w, models.Response{Success: false, Message: "Provide assignment_id or user_id"}, http.StatusBadRequest)
}

// ReminderBulkNudge handles POST /api/reminders/bulk-nudge.
// Sends reminders to all users with overdue assignments.
func (as *Server) ReminderBulkNudge(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	user := ctx.Get(r, "user").(models.User)

	// Get all overdue assignments for the org
	overdue, err := models.GetOverdueAssignmentsForOrg(user.OrgId)
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Error fetching overdue assignments"}, http.StatusInternalServerError)
		return
	}

	sent := 0
	for _, a := range overdue {
		if err := models.RecordReminderSent(a.UserId, a.Id, "bulk_manual", user.Id); err != nil {
			log.Errorf("bulk nudge: failed for assignment %d: %v", a.Id, err)
			continue
		}
		sent++
	}

	JSONResponse(w, struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
		Total   int    `json:"total_overdue"`
		Sent    int    `json:"sent"`
	}{true, "Bulk nudge completed", len(overdue), sent}, http.StatusOK)
}

// ReminderTemplate handles GET/PUT /api/reminders/template.
// Manages the custom email template used for training reminders.
func (as *Server) ReminderTemplate(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)
	switch r.Method {
	case http.MethodGet:
		tpl := models.GetReminderTemplate(user.OrgId)
		JSONResponse(w, tpl, http.StatusOK)
	case http.MethodPut:
		var tpl models.ReminderTemplate
		if err := json.NewDecoder(r.Body).Decode(&tpl); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: ErrInvalidJSON}, http.StatusBadRequest)
			return
		}
		tpl.OrgId = user.OrgId
		if err := models.SaveReminderTemplate(&tpl); err != nil {
			log.Error(err)
			JSONResponse(w, models.Response{Success: false, Message: "Error saving template"}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, tpl, http.StatusOK)
	default:
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
	}
}

// ReminderAssignmentHistory handles GET /api/reminders/assignment/{id}/history.
// Returns reminder history for a specific assignment.
func (as *Server) ReminderAssignmentHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	vars := mux.Vars(r)
	assignmentId, _ := strconv.ParseInt(vars["id"], 10, 64)

	reminders, err := models.GetRemindersForAssignment(assignmentId)
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Error fetching reminders"}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, reminders, http.StatusOK)
}
