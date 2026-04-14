package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	ctx "github.com/gophish/gophish/context"
	log "github.com/gophish/gophish/logger"
	"github.com/gophish/gophish/models"
)

// ── ROI Reporting Dashboard ─────────────────────────────────────
// Executive-level endpoints that demonstrate training value to leadership
// with period-over-period comparison, trend data, and leadership briefs.

// ROIDashboard handles GET /api/roi/dashboard — generates the full executive dashboard.
func (as *Server) ROIDashboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	user := ctx.Get(r, "user").(models.User)

	periodMonths := 12
	if m, err := strconv.Atoi(r.URL.Query().Get("months")); err == nil && m > 0 && m <= 60 {
		periodMonths = m
	}
	var periodEnd time.Time
	if endStr := r.URL.Query().Get("end"); endStr != "" {
		if t, err := time.Parse("2006-01-02", endStr); err == nil {
			periodEnd = t
		}
	}

	dashboard, err := models.GenerateROIDashboard(user.OrgId, periodEnd, periodMonths)
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Error generating ROI dashboard"}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, dashboard, http.StatusOK)
}

// ROIInvestmentConfig handles GET/PUT /api/roi/investment-config.
func (as *Server) ROIInvestmentConfig(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)
	switch r.Method {
	case http.MethodGet:
		cfg := models.GetROIInvestmentConfig(user.OrgId)
		JSONResponse(w, cfg, http.StatusOK)
	case http.MethodPut:
		var cfg models.ROIInvestmentConfig
		if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: ErrInvalidJSON}, http.StatusBadRequest)
			return
		}
		cfg.OrgId = user.OrgId
		if err := models.SaveROIInvestmentConfig(&cfg); err != nil {
			log.Error(err)
			JSONResponse(w, models.Response{Success: false, Message: "Error saving investment config"}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, cfg, http.StatusOK)
	default:
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
	}
}
