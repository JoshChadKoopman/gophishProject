package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	ctx "github.com/gophish/gophish/context"
	"github.com/gophish/gophish/models"
	"github.com/gorilla/mux"
)

const (
	errInvalidPresentationID = "Invalid presentation ID"
	errInvalidRequestBody    = "Invalid request body"
)

// AntiSkipPolicy handles GET/PUT /api/training/{id}/anti-skip-policy
// GET returns the policy for a presentation (defaults if none custom set).
// PUT creates or updates a custom policy (admin only).
func (as *Server) AntiSkipPolicy(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	presId, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: errInvalidPresentationID}, http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		policy := models.GetAntiSkipPolicy(presId)
		JSONResponse(w, policy, http.StatusOK)

	case http.MethodPut:
		user := ctx.Get(r, "user").(models.User)
		hasAdmin, _ := user.HasPermission(models.PermissionManageTraining)
		if !hasAdmin {
			JSONResponse(w, models.Response{Success: false, Message: "Admin access required"}, http.StatusForbidden)
			return
		}
		var policy models.AntiSkipPolicy
		if err := json.NewDecoder(r.Body).Decode(&policy); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: errInvalidRequestBody}, http.StatusBadRequest)
			return
		}
		policy.PresentationId = presId
		if err := models.SaveAntiSkipPolicy(&policy); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, policy, http.StatusOK)

	case http.MethodDelete:
		user := ctx.Get(r, "user").(models.User)
		hasAdmin, _ := user.HasPermission(models.PermissionManageTraining)
		if !hasAdmin {
			JSONResponse(w, models.Response{Success: false, Message: "Admin access required"}, http.StatusForbidden)
			return
		}
		models.DeleteAntiSkipPolicy(presId)
		JSONResponse(w, models.Response{Success: true, Message: "Policy reset to defaults"}, http.StatusOK)

	default:
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
	}
}

// AntiSkipEngage handles PUT /api/training/{id}/engage
// Records per-page engagement evidence (dwell time, scroll depth, acknowledgment).
func (as *Server) AntiSkipEngage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}

	user := ctx.Get(r, "user").(models.User)
	vars := mux.Vars(r)
	presId, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: errInvalidPresentationID}, http.StatusBadRequest)
		return
	}

	var update models.PageEngagementUpdate
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: errInvalidRequestBody}, http.StatusBadRequest)
		return
	}

	// Clamp values
	if update.DwellSeconds < 0 {
		update.DwellSeconds = 0
	}
	if update.ScrollDepthPct < 0 {
		update.ScrollDepthPct = 0
	}
	if update.ScrollDepthPct > 100 {
		update.ScrollDepthPct = 100
	}

	if err := models.RecordPageEngagement(user.Id, presId, update); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}

	JSONResponse(w, models.Response{Success: true, Message: "Engagement recorded"}, http.StatusOK)
}

// AntiSkipValidateAdvance handles POST /api/training/{id}/validate-advance
// Server-side check: can the user move from current_page to next_page?
func (as *Server) AntiSkipValidateAdvance(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}

	user := ctx.Get(r, "user").(models.User)
	vars := mux.Vars(r)
	presId, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: errInvalidPresentationID}, http.StatusBadRequest)
		return
	}

	var req struct {
		CurrentPage int `json:"current_page"`
		NextPage    int `json:"next_page"`
		TotalPages  int `json:"total_pages"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: errInvalidRequestBody}, http.StatusBadRequest)
		return
	}

	result := models.ValidatePageAdvance(user.Id, presId, req.CurrentPage, req.NextPage, req.TotalPages)
	JSONResponse(w, result, http.StatusOK)
}

// AntiSkipValidateComplete handles POST /api/training/{id}/validate-complete
// Server-side gate: can the user mark this course as complete?
func (as *Server) AntiSkipValidateComplete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}

	user := ctx.Get(r, "user").(models.User)
	vars := mux.Vars(r)
	presId, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: errInvalidPresentationID}, http.StatusBadRequest)
		return
	}

	var req struct {
		TotalPages int `json:"total_pages"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: errInvalidRequestBody}, http.StatusBadRequest)
		return
	}

	result := models.ValidateCourseCompletion(user.Id, presId, req.TotalPages)
	JSONResponse(w, result, http.StatusOK)
}

// AntiSkipEngagementSummary handles GET /api/training/{id}/engagement-summary
// Admin-only report of user engagement for a course.
func (as *Server) AntiSkipEngagementSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}

	vars := mux.Vars(r)
	presId, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: errInvalidPresentationID}, http.StatusBadRequest)
		return
	}

	rows, err := models.GetEngagementSummary(presId)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, rows, http.StatusOK)
}
