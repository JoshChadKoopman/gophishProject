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

// ── Scheduled Reports API ───────────────────────────────────────
// CRUD endpoints for admin-configurable recurring report delivery.
// Admins can configure "Send me a weekly PDF summary every Monday at 8am".

// ScheduledReports handles GET (list) and POST (create) on /api/scheduled-reports/.
func (as *Server) ScheduledReports(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)

	switch r.Method {
	case http.MethodGet:
		reports, err := models.GetScheduledReports(user.OrgId)
		if err != nil {
			log.Error(err)
			JSONResponse(w, models.Response{Success: false, Message: "Error fetching scheduled reports"}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, reports, http.StatusOK)

	case http.MethodPost:
		var sr models.ScheduledReport
		if err := json.NewDecoder(r.Body).Decode(&sr); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: ErrInvalidJSON}, http.StatusBadRequest)
			return
		}
		sr.OrgId = user.OrgId
		sr.UserId = user.Id

		if msg := sr.Validate(); msg != "" {
			JSONResponse(w, models.Response{Success: false, Message: msg}, http.StatusBadRequest)
			return
		}

		if err := models.CreateScheduledReport(&sr); err != nil {
			log.Error(err)
			JSONResponse(w, models.Response{Success: false, Message: "Error creating scheduled report"}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, sr, http.StatusCreated)

	default:
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
	}
}

// ScheduledReport handles GET (single), PUT (update), DELETE on /api/scheduled-reports/{id}.
func (as *Server) ScheduledReport(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)
	vars := mux.Vars(r)
	id, _ := strconv.ParseInt(vars["id"], 10, 64)

	switch r.Method {
	case http.MethodGet:
		sr, err := models.GetScheduledReport(id, user.OrgId)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: "Scheduled report not found"}, http.StatusNotFound)
			return
		}
		JSONResponse(w, sr, http.StatusOK)

	case http.MethodPut:
		existing, err := models.GetScheduledReport(id, user.OrgId)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: "Scheduled report not found"}, http.StatusNotFound)
			return
		}

		var sr models.ScheduledReport
		if err := json.NewDecoder(r.Body).Decode(&sr); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: ErrInvalidJSON}, http.StatusBadRequest)
			return
		}
		sr.Id = existing.Id
		sr.OrgId = user.OrgId
		sr.UserId = existing.UserId
		sr.RunCount = existing.RunCount
		sr.LastRunAt = existing.LastRunAt
		sr.CreatedDate = existing.CreatedDate

		if msg := sr.Validate(); msg != "" {
			JSONResponse(w, models.Response{Success: false, Message: msg}, http.StatusBadRequest)
			return
		}

		if err := models.UpdateScheduledReport(&sr); err != nil {
			log.Error(err)
			JSONResponse(w, models.Response{Success: false, Message: "Error updating scheduled report"}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, sr, http.StatusOK)

	case http.MethodDelete:
		if err := models.DeleteScheduledReport(id, user.OrgId); err != nil {
			log.Error(err)
			JSONResponse(w, models.Response{Success: false, Message: "Error deleting scheduled report"}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, models.Response{Success: true, Message: "Scheduled report deleted"}, http.StatusOK)

	default:
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
	}
}

// ScheduledReportToggle handles POST on /api/scheduled-reports/{id}/toggle.
func (as *Server) ScheduledReportToggle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	user := ctx.Get(r, "user").(models.User)
	vars := mux.Vars(r)
	id, _ := strconv.ParseInt(vars["id"], 10, 64)

	var req struct {
		IsActive bool `json:"is_active"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	if err := models.ToggleScheduledReport(id, user.OrgId, req.IsActive); err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Error toggling scheduled report"}, http.StatusInternalServerError)
		return
	}

	JSONResponse(w, models.Response{Success: true, Message: "Scheduled report updated"}, http.StatusOK)
}

// ScheduledReportSummary handles GET on /api/scheduled-reports/summary.
func (as *Server) ScheduledReportSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	user := ctx.Get(r, "user").(models.User)
	summary := models.GetScheduledReportSummary(user.OrgId)
	JSONResponse(w, summary, http.StatusOK)
}

// ScheduledReportTypes handles GET on /api/scheduled-reports/types — returns
// the available report types, frequencies, and formats for the UI.
func (as *Server) ScheduledReportTypes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}

	type typeInfo struct {
		Slug  string `json:"slug"`
		Label string `json:"label"`
	}

	types := []typeInfo{
		{models.ReportTypeExecutiveSummary, "Executive Summary"},
		{models.ReportTypeCampaigns, "Campaigns"},
		{models.ReportTypeTraining, "Training"},
		{models.ReportTypePhishingTickets, "Phishing Tickets"},
		{models.ReportTypeEmailSecurity, "Email Security"},
		{models.ReportTypeNetworkEvents, "Network Events"},
		{models.ReportTypeROI, "ROI"},
		{models.ReportTypeCompliance, "Compliance"},
		{models.ReportTypeHygiene, "Cyber Hygiene"},
		{models.ReportTypeRiskScores, "Risk Scores"},
	}

	frequencies := []typeInfo{
		{models.FrequencyDaily, "Daily"},
		{models.FrequencyWeekly, "Weekly"},
		{models.FrequencyBiweekly, "Bi-Weekly"},
		{models.FrequencyMonthly, "Monthly"},
		{models.FrequencyQuarterly, "Quarterly"},
	}

	formats := []typeInfo{
		{"pdf", "PDF"},
		{"xlsx", "Excel (XLSX)"},
		{"csv", "CSV"},
	}

	JSONResponse(w, map[string]interface{}{
		"report_types": types,
		"frequencies":  frequencies,
		"formats":      formats,
	}, http.StatusOK)
}
