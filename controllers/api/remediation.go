package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	ctx "github.com/gophish/gophish/context"
	log "github.com/gophish/gophish/logger"
	"github.com/gophish/gophish/models"
	"github.com/gorilla/mux"
)

// RemediationPaths handles GET (list) and POST (create) for /api/remediation/paths.
func (as *Server) RemediationPaths(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)

	switch r.Method {
	case http.MethodGet:
		paths, err := models.GetRemediationPaths(user.OrgId)
		if err != nil {
			log.Error(err)
			JSONResponse(w, models.Response{Success: false, Message: "Error fetching remediation paths"}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, paths, http.StatusOK)

	case http.MethodPost:
		var req struct {
			UserId    int64   `json:"user_id"`
			UserEmail string  `json:"user_email"`
			Name      string  `json:"name"`
			FailCount int     `json:"fail_count"`
			RiskLevel string  `json:"risk_level"`
			DueDate   string  `json:"due_date"`
			CourseIds []int64 `json:"course_ids"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: ErrInvalidJSON}, http.StatusBadRequest)
			return
		}
		if len(req.CourseIds) == 0 {
			JSONResponse(w, models.Response{Success: false, Message: "At least one course_id is required"}, http.StatusBadRequest)
			return
		}

		path := &models.RemediationPath{
			OrgId:     user.OrgId,
			UserId:    req.UserId,
			UserEmail: req.UserEmail,
			Name:      req.Name,
			FailCount: req.FailCount,
			RiskLevel: req.RiskLevel,
		}
		if req.DueDate != "" {
			if parsed, pErr := time.Parse(time.RFC3339, req.DueDate); pErr == nil {
				path.DueDate = parsed
			}
		}

		if err := models.PostRemediationPath(path, req.CourseIds); err != nil {
			log.Error(err)
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusBadRequest)
			return
		}
		// Reload with steps
		full, _ := models.GetRemediationPath(path.Id, user.OrgId)
		JSONResponse(w, full, http.StatusCreated)

	default:
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
	}
}

// RemediationPath handles GET and DELETE for /api/remediation/paths/{id}.
func (as *Server) RemediationPath(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)
	vars := mux.Vars(r)
	id, ok := parseIDParam(w, vars, "id")
	if !ok {
		return
	}

	switch r.Method {
	case http.MethodGet:
		path, err := models.GetRemediationPath(id, user.OrgId)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: "Path not found"}, http.StatusNotFound)
			return
		}
		JSONResponse(w, path, http.StatusOK)

	case http.MethodDelete:
		if err := models.CancelRemediationPath(id, user.OrgId); err != nil {
			log.Error(err)
			JSONResponse(w, models.Response{Success: false, Message: "Error cancelling path"}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, models.Response{Success: true, Message: "Remediation path cancelled"}, http.StatusOK)

	default:
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
	}
}

// RemediationMyPaths handles GET /api/remediation/my-paths — user's own remediation paths.
func (as *Server) RemediationMyPaths(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)

	paths, err := models.GetRemediationPathsForUser(user.Id, user.OrgId)
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Error fetching paths"}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, paths, http.StatusOK)
}

// RemediationCompleteStep handles POST /api/remediation/paths/{id}/complete-step.
func (as *Server) RemediationCompleteStep(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)
	vars := mux.Vars(r)
	pathId, _ := strconv.ParseInt(vars["id"], 10, 64)

	var req struct {
		PresentationId int64 `json:"presentation_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: ErrInvalidJSON}, http.StatusBadRequest)
		return
	}

	// Verify path belongs to user's org
	path, err := models.GetRemediationPath(pathId, user.OrgId)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "Path not found"}, http.StatusNotFound)
		return
	}

	if err := models.CompleteRemediationStep(path.Id, req.PresentationId); err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusBadRequest)
		return
	}

	updated, _ := models.GetRemediationPath(pathId, user.OrgId)
	JSONResponse(w, updated, http.StatusOK)
}

// RemediationEvaluate handles POST /api/remediation/evaluate — runs escalation + auto-creates paths.
func (as *Server) RemediationEvaluate(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)

	paths, err := models.EvaluateAndCreateRemediations(user.OrgId)
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Error running remediation evaluation"}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, map[string]interface{}{
		"success":       true,
		"paths_created": len(paths),
		"paths":         paths,
	}, http.StatusOK)
}

// RemediationSummary handles GET /api/remediation/summary.
func (as *Server) RemediationSummary(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)

	summary, err := models.GetRemediationSummary(user.OrgId)
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Error fetching summary"}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, summary, http.StatusOK)
}

// RemediationMarkExpired handles POST /api/remediation/mark-expired.
func (as *Server) RemediationMarkExpired(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)

	count, err := models.MarkExpiredRemediationPaths(user.OrgId)
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Error marking expired"}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, map[string]interface{}{
		"success":        true,
		"marked_expired": count,
	}, http.StatusOK)
}
