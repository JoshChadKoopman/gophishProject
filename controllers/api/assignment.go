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
		JSONResponse(w, models.Response{Success: false, Message: "Permission denied"}, http.StatusForbidden)
		return
	}

	switch {
	case r.Method == "GET":
		assignments, err := models.GetAllAssignments()
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, assignments, http.StatusOK)

	case r.Method == "POST":
		var req struct {
			UserId         int64  `json:"user_id"`
			PresentationId int64  `json:"presentation_id"`
			DueDate        string `json:"due_date"`
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

		var dueDate time.Time
		if req.DueDate != "" {
			dueDate, err = time.Parse(time.RFC3339, req.DueDate)
			if err != nil {
				JSONResponse(w, models.Response{Success: false, Message: "Invalid due_date format (use RFC3339)"}, http.StatusBadRequest)
				return
			}
		}

		assignment := &models.CourseAssignment{
			UserId:         req.UserId,
			PresentationId: req.PresentationId,
			AssignedBy:     user.Id,
			DueDate:        dueDate,
		}
		err = models.PostAssignment(assignment)
		if err != nil {
			if err == models.ErrAssignmentExists {
				JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusConflict)
			} else {
				JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
			}
			return
		}
		JSONResponse(w, assignment, http.StatusCreated)
	}
}

// TrainingAssignment handles deleting a single assignment by ID.
// DELETE: requires manage_training
func (as *Server) TrainingAssignment(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)
	hasPermission, _ := user.HasPermission(models.PermissionManageTraining)
	if !hasPermission {
		JSONResponse(w, models.Response{Success: false, Message: "Permission denied"}, http.StatusForbidden)
		return
	}

	vars := mux.Vars(r)
	id, _ := strconv.ParseInt(vars["id"], 0, 64)

	switch {
	case r.Method == "DELETE":
		err := models.DeleteAssignment(id)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, models.Response{Success: true, Message: "Assignment deleted"}, http.StatusOK)
	}
}

// TrainingAssignGroup assigns a course to all platform users in a group.
// POST: requires manage_training
func (as *Server) TrainingAssignGroup(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)
	hasPermission, _ := user.HasPermission(models.PermissionManageTraining)
	if !hasPermission {
		JSONResponse(w, models.Response{Success: false, Message: "Permission denied"}, http.StatusForbidden)
		return
	}

	if r.Method != "POST" {
		JSONResponse(w, models.Response{Success: false, Message: "Method not allowed"}, http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		GroupId        int64  `json:"group_id"`
		PresentationId int64  `json:"presentation_id"`
		DueDate        string `json:"due_date"`
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

	result, err := models.AssignCourseToGroup(req.PresentationId, req.GroupId, user.Id, dueDate)
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
		IsOverdue bool `json:"is_overdue"`
	}
	resp := []assignmentResponse{}
	for _, a := range assignments {
		overdue := !a.DueDate.IsZero() && a.Status != models.AssignmentStatusCompleted && a.DueDate.Before(now)
		resp = append(resp, assignmentResponse{
			CourseAssignment: a,
			IsOverdue:        overdue,
		})
	}
	JSONResponse(w, resp, http.StatusOK)
}
