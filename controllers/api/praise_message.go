package api

import (
	"encoding/json"
	"net/http"

	ctx "github.com/gophish/gophish/context"
	"github.com/gophish/gophish/models"
)

// TrainingPraiseMessages handles GET and PUT for configurable praise/feedback
// messages shown on training completion, quiz pass, cert award, etc.
//
// GET /api/training/praise-messages — returns all active praise messages for
// the current user's org (or defaults).
//
// PUT /api/training/praise-messages — updates/creates praise messages for the
// org. Requires manage_training permission.
func (as *Server) TrainingPraiseMessages(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)

	switch r.Method {
	case http.MethodGet:
		msgs, err := models.GetPraiseMessages(user.OrgId)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, msgs, http.StatusOK)

	case http.MethodPut:
		hasPermission, _ := user.HasPermission(models.PermissionManageTraining)
		if !hasPermission {
			JSONResponse(w, models.Response{Success: false, Message: ErrPermissionDenied}, http.StatusForbidden)
			return
		}
		var msgs []models.PraiseMessage
		if err := json.NewDecoder(r.Body).Decode(&msgs); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: "Invalid JSON: " + err.Error()}, http.StatusBadRequest)
			return
		}
		if err := models.SavePraiseMessages(user.OrgId, msgs); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
			return
		}
		// Return the updated set
		updated, _ := models.GetPraiseMessages(user.OrgId)
		JSONResponse(w, updated, http.StatusOK)

	default:
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
	}
}

// TrainingPraiseMessagesReset resets praise messages to defaults for the org.
// DELETE /api/training/praise-messages/reset
func (as *Server) TrainingPraiseMessagesReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}

	user := ctx.Get(r, "user").(models.User)
	hasPermission, _ := user.HasPermission(models.PermissionManageTraining)
	if !hasPermission {
		JSONResponse(w, models.Response{Success: false, Message: ErrPermissionDenied}, http.StatusForbidden)
		return
	}

	if err := models.ResetPraiseMessages(user.OrgId); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}

	defaults := models.DefaultPraiseMessages()
	JSONResponse(w, defaults, http.StatusOK)
}
