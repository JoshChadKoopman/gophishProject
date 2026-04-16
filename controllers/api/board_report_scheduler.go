package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	ctx "github.com/gophish/gophish/context"
	log "github.com/gophish/gophish/logger"
	"github.com/gophish/gophish/models"
	"github.com/gorilla/mux"
)

// ── Board Report Scheduler Endpoints ────────────────────────────

// BoardReportSchedules handles GET (list) and POST (create/update) for schedules.
func (as *Server) BoardReportSchedules(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)
	switch r.Method {
	case http.MethodGet:
		schedules, err := models.GetBoardReportSchedules(user.OrgId)
		if err != nil {
			log.Error(err)
			JSONResponse(w, models.Response{Success: false, Message: "Error fetching schedules"}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, schedules, http.StatusOK)

	case http.MethodPost:
		var s models.BoardReportSchedule
		if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: ErrInvalidJSON}, http.StatusBadRequest)
			return
		}
		s.OrgId = user.OrgId
		s.CreatedBy = user.Id
		if err := models.SaveBoardReportSchedule(&s); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusBadRequest)
			return
		}
		JSONResponse(w, s, http.StatusCreated)

	default:
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
	}
}

// BoardReportSchedule handles GET, PUT, DELETE for a single schedule.
func (as *Server) BoardReportScheduleItem(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)
	vars := mux.Vars(r)
	id, _ := strconv.ParseInt(vars["id"], 10, 64)

	switch r.Method {
	case http.MethodGet:
		s, err := models.GetBoardReportSchedule(id, user.OrgId)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: "Schedule not found"}, http.StatusNotFound)
			return
		}
		JSONResponse(w, s, http.StatusOK)

	case http.MethodPut:
		s, err := models.GetBoardReportSchedule(id, user.OrgId)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: "Schedule not found"}, http.StatusNotFound)
			return
		}
		var update models.BoardReportSchedule
		if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: ErrInvalidJSON}, http.StatusBadRequest)
			return
		}
		s.Frequency = update.Frequency
		s.DayOfMonth = update.DayOfMonth
		s.Enabled = update.Enabled
		s.AutoPublish = update.AutoPublish
		s.NotifyEmails = update.NotifyEmails
		if err := models.SaveBoardReportSchedule(&s); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusBadRequest)
			return
		}
		JSONResponse(w, s, http.StatusOK)

	case http.MethodDelete:
		if err := models.DeleteBoardReportSchedule(id, user.OrgId); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, models.Response{Success: true, Message: "Schedule deleted"}, http.StatusOK)

	default:
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
	}
}

// BoardReportScheduleRun handles POST to manually trigger scheduled report generation.
func (as *Server) BoardReportScheduleRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	count, err := models.RunScheduledBoardReports()
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, map[string]interface{}{
		"success":   true,
		"generated": count,
	}, http.StatusOK)
}

// ── Branded PDF Export ──────────────────────────────────────────

// BoardReportExportBranded handles GET /api/board-reports/{id}/export-branded
// Generates a professional branded PDF with narrative, ROI, deltas, and org branding.
func (as *Server) BoardReportExportBranded(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	user := ctx.Get(r, "user").(models.User)
	vars := mux.Vars(r)
	id, _ := strconv.ParseInt(vars["id"], 10, 64)

	br, err := models.GetBoardReport(id, user.OrgId)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusNotFound)
		return
	}

	payload, err := models.BuildFullBoardReportPayload(br, user.OrgId)
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Error generating report data"}, http.StatusInternalServerError)
		return
	}

	// Use org name from query param or default
	branding := models.DefaultBranding()
	if orgName := r.URL.Query().Get("org_name"); orgName != "" {
		branding.OrgName = orgName
	}

	filename := fmt.Sprintf("board-report-%s-%s.pdf",
		branding.OrgName, time.Now().Format("2006-01-02"))
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))

	if err := models.GenerateBrandedBoardPDF(w, payload, branding); err != nil {
		log.Error(err)
	}
}
