package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	ctx "github.com/gophish/gophish/context"
	"github.com/gophish/gophish/models"
	"github.com/gorilla/mux"
)

const errInvalidUserID = "Invalid user ID"

// difficultySetRequest is the JSON body for setting manual difficulty.
type difficultySetRequest struct {
	Level int `json:"level"` // 1-4
}

// DifficultyProfile handles GET /api/difficulty/profile/{id}
// Returns the user's difficulty profile including effective level,
// adaptive recommendation, manual override, and recent adjustments.
func (as *Server) DifficultyProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	vars := mux.Vars(r)
	uid, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: errInvalidUserID}, http.StatusBadRequest)
		return
	}
	profile, err := models.GetUserDifficultyProfile(uid)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "Could not build difficulty profile"}, http.StatusNotFound)
		return
	}
	JSONResponse(w, profile, http.StatusOK)
}

// DifficultySet handles PUT /api/difficulty/set/{id}
// Sets a manual difficulty level (1-4) for a user, overriding adaptive mode.
func (as *Server) DifficultySet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	vars := mux.Vars(r)
	uid, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: errInvalidUserID}, http.StatusBadRequest)
		return
	}

	var req difficultySetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "Invalid request body"}, http.StatusBadRequest)
		return
	}

	if err := models.ValidateDifficultyLevel(req.Level); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusBadRequest)
		return
	}

	// Determine who is making the change
	user := ctx.Get(r, "user").(models.User)
	changedBy := user.Username

	if err := models.SetManualDifficulty(uid, req.Level, changedBy); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}

	// Return the updated profile
	profile, _ := models.GetUserDifficultyProfile(uid)
	JSONResponse(w, profile, http.StatusOK)
}

// DifficultyClear handles PUT /api/difficulty/clear/{id}
// Removes manual difficulty override and switches back to adaptive mode.
func (as *Server) DifficultyClear(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	vars := mux.Vars(r)
	uid, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: errInvalidUserID}, http.StatusBadRequest)
		return
	}

	user := ctx.Get(r, "user").(models.User)
	changedBy := user.Username

	if err := models.ClearManualDifficulty(uid, changedBy); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}

	profile, _ := models.GetUserDifficultyProfile(uid)
	JSONResponse(w, profile, http.StatusOK)
}

// DifficultyHistory handles GET /api/difficulty/history/{id}
// Returns the full adjustment history for a user.
func (as *Server) DifficultyHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	vars := mux.Vars(r)
	uid, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: errInvalidUserID}, http.StatusBadRequest)
		return
	}
	logs, err := models.GetDifficultyAdjustmentHistory(uid)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "Could not fetch history"}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, logs, http.StatusOK)
}

// DifficultyOrgStats handles GET /api/difficulty/org-stats
// Returns difficulty distribution stats for the authenticated user's org.
func (as *Server) DifficultyOrgStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	user := ctx.Get(r, "user").(models.User)
	stats, err := models.GetOrgDifficultyStats(user.OrgId)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "Could not fetch stats"}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, stats, http.StatusOK)
}

// DifficultyMyProfile handles GET /api/difficulty/my-profile
// Returns the authenticated user's own difficulty profile.
func (as *Server) DifficultyMyProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	user := ctx.Get(r, "user").(models.User)
	profile, err := models.GetUserDifficultyProfile(user.Id)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "Could not build difficulty profile"}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, profile, http.StatusOK)
}

// DifficultyMySet handles PUT /api/difficulty/my-set
// Allows the authenticated user to set their own manual difficulty.
func (as *Server) DifficultyMySet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}

	var req difficultySetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "Invalid request body"}, http.StatusBadRequest)
		return
	}

	if err := models.ValidateDifficultyLevel(req.Level); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusBadRequest)
		return
	}

	user := ctx.Get(r, "user").(models.User)
	if err := models.SetManualDifficulty(user.Id, req.Level, user.Username+" (self)"); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}

	profile, _ := models.GetUserDifficultyProfile(user.Id)
	JSONResponse(w, profile, http.StatusOK)
}

// DifficultyMyClear handles PUT /api/difficulty/my-clear
// Allows the authenticated user to switch back to adaptive mode.
func (as *Server) DifficultyMyClear(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}

	user := ctx.Get(r, "user").(models.User)
	if err := models.ClearManualDifficulty(user.Id, user.Username+" (self)"); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}

	profile, _ := models.GetUserDifficultyProfile(user.Id)
	JSONResponse(w, profile, http.StatusOK)
}

// DifficultyRunAdaptive handles POST /api/difficulty/run-adaptive
// Triggers the adaptive adjustment algorithm for the authenticated user's org.
func (as *Server) DifficultyRunAdaptive(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}

	user := ctx.Get(r, "user").(models.User)
	adjusted, err := models.RunAdaptiveAdjustment(user.OrgId)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}

	JSONResponse(w, map[string]interface{}{
		"success":        true,
		"users_adjusted": adjusted,
	}, http.StatusOK)
}
