package api

import (
	"net/http"

	ctx "github.com/gophish/gophish/context"
	log "github.com/gophish/gophish/logger"
	"github.com/gophish/gophish/models"
)

// GamificationLeaderboard handles GET /api/gamification/leaderboard.
func (as *Server) GamificationLeaderboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	scope := getOrgScope(r)
	period := r.URL.Query().Get("period")
	if period == "" {
		period = "all_time"
	}
	department := r.URL.Query().Get("department")
	entries, err := models.GetLeaderboard(scope.OrgId, period, department, 50)
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Failed to load leaderboard"}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, entries, http.StatusOK)
}

// GamificationMyPosition handles GET /api/gamification/my-position.
func (as *Server) GamificationMyPosition(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	scope := getOrgScope(r)
	user := ctx.Get(r, "user").(models.User)
	period := r.URL.Query().Get("period")
	if period == "" {
		period = "all_time"
	}
	entry, err := models.GetUserLeaderboardPosition(user.Id, scope.OrgId, period)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "Not yet ranked"}, http.StatusNotFound)
		return
	}
	JSONResponse(w, entry, http.StatusOK)
}

// GamificationBadges handles GET /api/gamification/badges — all available badges.
func (as *Server) GamificationBadges(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	badges, err := models.GetAllBadges()
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Failed to load badges"}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, badges, http.StatusOK)
}

// GamificationMyBadges handles GET /api/gamification/my-badges — user's earned badges.
func (as *Server) GamificationMyBadges(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	user := ctx.Get(r, "user").(models.User)
	badges, err := models.GetUserBadges(user.Id)
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Failed to load badges"}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, badges, http.StatusOK)
}

// GamificationMyStreak handles GET /api/gamification/my-streak.
func (as *Server) GamificationMyStreak(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	user := ctx.Get(r, "user").(models.User)
	streaks, err := models.GetUserStreaks(user.Id)
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Failed to load streaks"}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, streaks, http.StatusOK)
}
