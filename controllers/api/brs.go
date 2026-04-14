package api

import (
	"net/http"
	"strconv"

	ctx "github.com/gophish/gophish/context"
	"github.com/gophish/gophish/models"
	"github.com/gorilla/mux"
)

// BRSUserDetail handles GET /api/reports/brs/user/{id}
// Returns the 5-factor BRS breakdown for a single user.
func (as *Server) BRSUserDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	vars := mux.Vars(r)
	uid, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "Invalid user ID"}, http.StatusBadRequest)
		return
	}
	detail, err := models.GetUserBRS(uid)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "BRS not yet calculated for this user"}, http.StatusNotFound)
		return
	}
	JSONResponse(w, detail, http.StatusOK)
}

// BRSDepartment handles GET /api/reports/brs/department
// Returns department-level aggregated BRS.
func (as *Server) BRSDepartment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	scores, err := models.GetDepartmentBRS(getOrgScope(r))
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, scores, http.StatusOK)
}

// BRSBenchmark handles GET /api/reports/brs/benchmark
// Returns org and global benchmark comparison.
func (as *Server) BRSBenchmark(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	user := ctx.Get(r, "user").(models.User)
	bench, err := models.GetBRSBenchmark(user.OrgId)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, bench, http.StatusOK)
}

// BRSTrend handles GET /api/reports/brs/trend?user_id=X&days=90
// Returns historical BRS data points for trend charts.
func (as *Server) BRSTrend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	userID, err := strconv.ParseInt(r.URL.Query().Get("user_id"), 10, 64)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "user_id is required"}, http.StatusBadRequest)
		return
	}
	days := 90
	if d := r.URL.Query().Get("days"); d != "" {
		if parsed, parseErr := strconv.Atoi(d); parseErr == nil && parsed > 0 && parsed <= 365 {
			days = parsed
		}
	}
	trend, err := models.GetBRSTrend(userID, days)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, trend, http.StatusOK)
}

// BRSLeaderboard handles GET /api/reports/brs/leaderboard?limit=25
// Returns the top users by composite BRS (lowest risk first).
func (as *Server) BRSLeaderboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	limit := 25
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, parseErr := strconv.Atoi(l); parseErr == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}
	board, err := models.GetBRSLeaderboard(getOrgScope(r), limit)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, board, http.StatusOK)
}

// BRSRecalculate handles POST /api/reports/brs/recalculate
// Triggers an on-demand BRS recalculation for the org.
func (as *Server) BRSRecalculate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	user := ctx.Get(r, "user").(models.User)
	go func() {
		_ = models.RecalculateOrgBRS(user.OrgId)
	}()
	JSONResponse(w, models.Response{Success: true, Message: "BRS recalculation started"}, http.StatusOK)
}
