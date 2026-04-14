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

// TrainingSatisfactionRate handles POST /api/training/{id}/rate — submit a satisfaction rating.
func (as *Server) TrainingSatisfactionRate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}

	user := ctx.Get(r, "user").(models.User)
	vars := mux.Vars(r)
	presId, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "Invalid ID"}, http.StatusBadRequest)
		return
	}

	var req struct {
		Rating   int    `json:"rating"`
		Feedback string `json:"feedback"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "Invalid request body"}, http.StatusBadRequest)
		return
	}

	if req.Rating < 1 || req.Rating > 5 {
		JSONResponse(w, models.Response{Success: false, Message: "Rating must be between 1 and 5"}, http.StatusBadRequest)
		return
	}

	rating := &models.TrainingSatisfactionRating{
		UserId:         user.Id,
		PresentationId: presId,
		Rating:         req.Rating,
		Feedback:       req.Feedback,
	}

	if err := models.PostTrainingSatisfactionRating(rating); err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Failed to save rating"}, http.StatusInternalServerError)
		return
	}

	JSONResponse(w, models.Response{Success: true, Message: "Rating saved"}, http.StatusOK)
}

// TrainingSatisfactionStats handles GET /api/training/satisfaction — org-wide satisfaction stats.
func (as *Server) TrainingSatisfactionStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}

	scope := getOrgScope(r)
	stats := models.GetOrgSatisfactionStats(scope.OrgId)
	JSONResponse(w, stats, http.StatusOK)
}

// TrainingAnalytics handles GET /api/training/analytics — comprehensive training analytics.
func (as *Server) TrainingAnalytics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}

	user := ctx.Get(r, "user").(models.User)
	hasPermission, _ := user.HasPermission(models.PermissionViewReports)
	if !hasPermission {
		JSONResponse(w, models.Response{Success: false, Message: ErrPermissionDenied}, http.StatusForbidden)
		return
	}

	scope := getOrgScope(r)
	analytics := models.GetTrainingAnalytics(scope.OrgId)
	JSONResponse(w, analytics, http.StatusOK)
}
