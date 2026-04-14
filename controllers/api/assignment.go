package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	ctx "github.com/gophish/gophish/context"
	"github.com/gophish/gophish/models"
	"github.com/gorilla/mux"
)

// TrainingAssignments handles listing and creating individual course assignments.
// GET/POST: requires manage_training
func (as *Server) TrainingAssignments(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)
	hasPermission, _ := user.HasPermission(models.PermissionManageTraining)
	if !hasPermission {
		JSONResponse(w, models.Response{Success: false, Message: ErrPermissionDenied}, http.StatusForbidden)
		return
	}

	switch r.Method {
	case "GET":
		as.listAssignments(w)
	case "POST":
		as.createAssignment(w, r, user.Id)
	}
}

func (as *Server) listAssignments(w http.ResponseWriter) {
	assignments, err := models.GetAllAssignments()
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, assignments, http.StatusOK)
}

func (as *Server) createAssignment(w http.ResponseWriter, r *http.Request, assignedBy int64) {
	var req struct {
		UserId         int64  `json:"user_id"`
		PresentationId int64  `json:"presentation_id"`
		DueDate        string `json:"due_date"`
		Priority       string `json:"priority"`
		Notes          string `json:"notes"`
	}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusBadRequest)
		return
	}
	if req.UserId == 0 || req.PresentationId == 0 {
		JSONResponse(w, models.Response{Success: false, Message: "user_id and presentation_id are required"}, http.StatusBadRequest)
		return
	}

	dueDate, err := parseDueDate(req.DueDate)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusBadRequest)
		return
	}

	if req.Priority != "" && !models.ValidAssignmentPriorities[req.Priority] {
		JSONResponse(w, models.Response{Success: false, Message: "Invalid priority. Use: low, normal, high, critical"}, http.StatusBadRequest)
		return
	}

	assignment := &models.CourseAssignment{
		UserId:         req.UserId,
		PresentationId: req.PresentationId,
		AssignedBy:     assignedBy,
		DueDate:        dueDate,
		Priority:       req.Priority,
		Notes:          req.Notes,
	}
	err = models.PostAssignment(assignment)
	if err != nil {
		code := http.StatusInternalServerError
		if err == models.ErrAssignmentExists {
			code = http.StatusConflict
		}
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, code)
		return
	}
	JSONResponse(w, assignment, http.StatusCreated)
}

func parseDueDate(raw string) (time.Time, error) {
	if raw == "" {
		return time.Time{}, nil
	}
	return time.Parse(time.RFC3339, raw)
}

// TrainingAssignment handles operations on a single assignment by ID.
// DELETE, PUT: requires manage_training
func (as *Server) TrainingAssignment(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)
	hasPermission, _ := user.HasPermission(models.PermissionManageTraining)
	if !hasPermission {
		JSONResponse(w, models.Response{Success: false, Message: ErrPermissionDenied}, http.StatusForbidden)
		return
	}

	vars := mux.Vars(r)
	id, _ := strconv.ParseInt(vars["id"], 0, 64)

	switch r.Method {
	case "DELETE":
		as.deleteAssignment(w, id)
	case "PUT":
		as.updateAssignment(w, r, id)
	}
}

func (as *Server) deleteAssignment(w http.ResponseWriter, id int64) {
	err := models.DeleteAssignment(id)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, models.Response{Success: true, Message: "Assignment deleted"}, http.StatusOK)
}

func (as *Server) updateAssignment(w http.ResponseWriter, r *http.Request, id int64) {
	var req struct {
		Status   string `json:"status"`
		Priority string `json:"priority"`
		Notes    string `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusBadRequest)
		return
	}
	if req.Status != "" {
		a, err := models.GetAssignmentById(id)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusNotFound)
			return
		}
		if err := models.UpdateAssignmentStatus(a.UserId, a.PresentationId, req.Status); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusBadRequest)
			return
		}
	}
	if req.Priority != "" {
		if err := models.UpdateAssignmentPriority(id, req.Priority); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusBadRequest)
			return
		}
	}
	if req.Notes != "" {
		if err := models.UpdateAssignmentNotes(id, req.Notes); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
			return
		}
	}
	JSONResponse(w, models.Response{Success: true, Message: "Assignment updated"}, http.StatusOK)
}

// TrainingAssignGroup assigns a course to all platform users in a group.
// POST: requires manage_training
func (as *Server) TrainingAssignGroup(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)
	hasPermission, _ := user.HasPermission(models.PermissionManageTraining)
	if !hasPermission {
		JSONResponse(w, models.Response{Success: false, Message: ErrPermissionDenied}, http.StatusForbidden)
		return
	}

	if r.Method != "POST" {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		GroupId        int64  `json:"group_id"`
		PresentationId int64  `json:"presentation_id"`
		DueDate        string `json:"due_date"`
		Priority       string `json:"priority"`
	}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusBadRequest)
		return
	}
	if req.GroupId == 0 || req.PresentationId == 0 {
		JSONResponse(w, models.Response{Success: false, Message: "group_id and presentation_id are required"}, http.StatusBadRequest)
		return
	}

	var dueDate time.Time
	if req.DueDate != "" {
		dueDate, err = time.Parse(time.RFC3339, req.DueDate)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: "Invalid due_date format (use RFC3339)"}, http.StatusBadRequest)
			return
		}
	}

	priority := req.Priority
	if priority == "" {
		priority = models.AssignmentPriorityNormal
	}

	result, err := models.AssignCourseToGroupWithPriority(req.PresentationId, req.GroupId, user.Id, dueDate, priority)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}

	JSONResponse(w, result, http.StatusOK)
}

// TrainingMyAssignments returns the current user's course assignments.
// GET: any authenticated user
func (as *Server) TrainingMyAssignments(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)
	assignments, err := models.GetAssignmentsForUser(user.Id)
	if err != nil {
		JSONResponse(w, []models.CourseAssignment{}, http.StatusOK)
		return
	}
	// Compute overdue status at read time
	now := time.Now().UTC()
	type assignmentResponse struct {
		models.CourseAssignment
		IsOverdue     bool   `json:"is_overdue"`
		DaysRemaining int    `json:"days_remaining"`
		CourseName    string `json:"course_name"`
	}
	resp := []assignmentResponse{}
	for _, a := range assignments {
		overdue := !a.DueDate.IsZero() && a.Status != models.AssignmentStatusCompleted &&
			a.Status != models.AssignmentStatusCancelled && a.DueDate.Before(now)
		daysRemaining := 0
		if !a.DueDate.IsZero() && a.DueDate.After(now) {
			daysRemaining = int(a.DueDate.Sub(now).Hours() / 24)
		}
		courseName := ""
		pres, err := models.GetTrainingPresentation(a.PresentationId, getOrgScope(r))
		if err == nil {
			courseName = pres.Name
		}
		resp = append(resp, assignmentResponse{
			CourseAssignment: a,
			IsOverdue:        overdue,
			DaysRemaining:    daysRemaining,
			CourseName:       courseName,
		})
	}
	JSONResponse(w, resp, http.StatusOK)
}

// TrainingAssignmentSummary returns aggregate assignment statistics.
// GET: requires manage_training
func (as *Server) TrainingAssignmentSummary(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)
	hasPermission, _ := user.HasPermission(models.PermissionManageTraining)
	if !hasPermission {
		JSONResponse(w, models.Response{Success: false, Message: ErrPermissionDenied}, http.StatusForbidden)
		return
	}

	summary, err := models.GetAssignmentSummary()
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, summary, http.StatusOK)
}

// TrainingBulkAssignmentUpdate handles bulk status updates for assignments.
// POST: requires manage_training
func (as *Server) TrainingBulkAssignmentUpdate(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)
	hasPermission, _ := user.HasPermission(models.PermissionManageTraining)
	if !hasPermission {
		JSONResponse(w, models.Response{Success: false, Message: ErrPermissionDenied}, http.StatusForbidden)
		return
	}

	var req struct {
		Ids    []int64 `json:"ids"`
		Status string  `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusBadRequest)
		return
	}
	if len(req.Ids) == 0 || req.Status == "" {
		JSONResponse(w, models.Response{Success: false, Message: "ids and status are required"}, http.StatusBadRequest)
		return
	}

	count, err := models.BulkUpdateAssignmentStatus(req.Ids, req.Status)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusBadRequest)
		return
	}
	JSONResponse(w, map[string]interface{}{
		"updated": count,
		"status":  req.Status,
	}, http.StatusOK)
}

// TrainingMarkOverdue runs the overdue checker and returns the count of newly-overdue assignments.
// POST: requires manage_training
func (as *Server) TrainingMarkOverdue(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)
	hasPermission, _ := user.HasPermission(models.PermissionManageTraining)
	if !hasPermission {
		JSONResponse(w, models.Response{Success: false, Message: ErrPermissionDenied}, http.StatusForbidden)
		return
	}

	count, err := models.MarkOverdueAssignments()
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, map[string]interface{}{
		"marked_overdue": count,
	}, http.StatusOK)
}
