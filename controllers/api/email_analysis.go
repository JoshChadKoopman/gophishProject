package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gophish/gophish/ai"
	log "github.com/gophish/gophish/logger"
	"github.com/gophish/gophish/models"
	"github.com/gorilla/mux"
)

// EmailAnalyses handles GET /api/email-analysis/ — returns all email
// analyses for the authenticated user's organization.
func (as *Server) EmailAnalyses(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	scope := getOrgScope(r)
	analyses, err := models.GetEmailAnalyses(scope.OrgId)
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Failed to retrieve email analyses"}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, analyses, http.StatusOK)
}

// EmailAnalysis handles GET /api/email-analysis/{id} — returns a single
// email analysis with its indicators, scoped to the user's organization.
func (as *Server) EmailAnalysis(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	scope := getOrgScope(r)
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "Invalid ID"}, http.StatusBadRequest)
		return
	}
	analysis, err := models.GetEmailAnalysis(id, scope.OrgId)
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusNotFound)
		return
	}
	JSONResponse(w, analysis, http.StatusOK)
}

// EmailAnalyzeReported handles POST /api/email-analysis/analyze — triggers
// an AI-powered NLP analysis on a reported email. The request must include
// the reported email ID and the raw email headers/body.
func (as *Server) EmailAnalyzeReported(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}

	if !as.aiConfig.Enabled {
		JSONResponse(w, models.Response{Success: false, Message: "AI analysis is not enabled. Configure the AI provider in settings."}, http.StatusServiceUnavailable)
		return
	}

	scope := getOrgScope(r)

	var req struct {
		ReportedEmailId int64  `json:"reported_email_id"`
		EmailHeaders    string `json:"email_headers"`
		EmailBody       string `json:"email_body"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: ErrInvalidJSON}, http.StatusBadRequest)
		return
	}

	if req.ReportedEmailId == 0 {
		JSONResponse(w, models.Response{Success: false, Message: "reported_email_id is required"}, http.StatusBadRequest)
		return
	}

	// Look up the reported email to get sender and subject
	reportedEmail, err := models.GetReportedEmail(req.ReportedEmailId, scope.OrgId)
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Reported email not found"}, http.StatusNotFound)
		return
	}

	// Create the AI client
	client, err := ai.NewClient(as.aiConfig.Provider, as.aiConfig.APIKey, as.aiConfig.Model)
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Failed to create AI client"}, http.StatusInternalServerError)
		return
	}

	// Run the analysis
	analysis, err := models.AnalyzeReportedEmail(
		scope.OrgId,
		req.ReportedEmailId,
		req.EmailHeaders,
		req.EmailBody,
		reportedEmail.SenderEmail,
		reportedEmail.Subject,
		client,
	)
	if err != nil {
		log.Error(err)
		// Return the analysis record even on failure so the caller can see the
		// status (e.g. "failed" or "analyzing").
		if analysis != nil {
			JSONResponse(w, analysis, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}

	JSONResponse(w, analysis, http.StatusOK)
}

// EmailAnalysisByReported handles GET /api/email-analysis/by-reported/{id}
// — returns the analysis associated with a specific reported email ID.
func (as *Server) EmailAnalysisByReported(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	scope := getOrgScope(r)
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "Invalid ID"}, http.StatusBadRequest)
		return
	}
	analysis, err := models.GetEmailAnalysisByReportedEmail(id, scope.OrgId)
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusNotFound)
		return
	}
	JSONResponse(w, analysis, http.StatusOK)
}

// EmailAnalysisSummary handles GET /api/email-analysis/summary — returns
// aggregate statistics for email analyses in the user's organization.
func (as *Server) EmailAnalysisSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	scope := getOrgScope(r)
	summary, err := models.GetEmailAnalysisSummary(scope.OrgId)
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Failed to retrieve analysis summary"}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, summary, http.StatusOK)
}
