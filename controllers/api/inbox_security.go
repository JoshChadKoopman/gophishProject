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

// ────────────────────────────────────────────────────────────────
// Inbox Monitor Configuration
// ────────────────────────────────────────────────────────────────

// InboxMonitorConfig handles GET/PUT /api/inbox-monitor/config
func (as *Server) InboxMonitorConfig(w http.ResponseWriter, r *http.Request) {
	scope := getOrgScope(r)
	switch r.Method {
	case http.MethodGet:
		config, err := models.GetInboxMonitorConfig(scope.OrgId)
		if err != nil {
			JSONResponse(w, models.InboxMonitorConfig{OrgId: scope.OrgId}, http.StatusOK)
			return
		}
		JSONResponse(w, config, http.StatusOK)
	case http.MethodPut:
		var config models.InboxMonitorConfig
		if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: ErrInvalidJSON}, http.StatusBadRequest)
			return
		}
		config.OrgId = scope.OrgId
		existing, err := models.GetInboxMonitorConfig(scope.OrgId)
		if err == nil {
			config.Id = existing.Id
			config.CreatedDate = existing.CreatedDate
		}
		if err := models.SaveInboxMonitorConfig(&config); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, config, http.StatusOK)
	default:
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
	}
}

// InboxScanResults handles GET /api/inbox-monitor/results
func (as *Server) InboxScanResults(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	scope := getOrgScope(r)
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 100
	}
	results, err := models.GetInboxScanResults(scope.OrgId, limit)
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Failed to retrieve scan results"}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, results, http.StatusOK)
}

// InboxScanResultDetail handles GET /api/inbox-monitor/results/{id}
func (as *Server) InboxScanResultDetail(w http.ResponseWriter, r *http.Request) {
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
	result, err := models.GetInboxScanResult(id, scope.OrgId)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "Scan result not found"}, http.StatusNotFound)
		return
	}
	JSONResponse(w, result, http.StatusOK)
}

// InboxScanSummary handles GET /api/inbox-monitor/summary
func (as *Server) InboxScanSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	scope := getOrgScope(r)
	summary, err := models.GetInboxScanSummary(scope.OrgId)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "Failed to retrieve summary"}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, summary, http.StatusOK)
}

// ────────────────────────────────────────────────────────────────
// BEC Detection
// ────────────────────────────────────────────────────────────────

// BECProfiles handles GET/POST /api/bec/profiles
func (as *Server) BECProfiles(w http.ResponseWriter, r *http.Request) {
	scope := getOrgScope(r)
	switch r.Method {
	case http.MethodGet:
		profiles, err := models.GetBECProfiles(scope.OrgId)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, profiles, http.StatusOK)
	case http.MethodPost:
		var profile models.BECProfile
		if err := json.NewDecoder(r.Body).Decode(&profile); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: ErrInvalidJSON}, http.StatusBadRequest)
			return
		}
		profile.OrgId = scope.OrgId
		if err := models.SaveBECProfile(&profile); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, profile, http.StatusCreated)
	default:
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
	}
}

// BECProfile handles GET/PUT/DELETE /api/bec/profiles/{id}
func (as *Server) BECProfile(w http.ResponseWriter, r *http.Request) {
	scope := getOrgScope(r)
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "Invalid ID"}, http.StatusBadRequest)
		return
	}
	switch r.Method {
	case http.MethodGet:
		profile, err := models.GetBECProfile(id, scope.OrgId)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: "Profile not found"}, http.StatusNotFound)
			return
		}
		JSONResponse(w, profile, http.StatusOK)
	case http.MethodPut:
		var profile models.BECProfile
		if err := json.NewDecoder(r.Body).Decode(&profile); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: ErrInvalidJSON}, http.StatusBadRequest)
			return
		}
		profile.Id = id
		profile.OrgId = scope.OrgId
		if err := models.SaveBECProfile(&profile); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, profile, http.StatusOK)
	case http.MethodDelete:
		if err := models.DeleteBECProfile(id, scope.OrgId); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, models.Response{Success: true, Message: "BEC profile deleted"}, http.StatusOK)
	default:
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
	}
}

// BECDetections handles GET /api/bec/detections
func (as *Server) BECDetections(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	scope := getOrgScope(r)
	includeResolved := r.URL.Query().Get("include_resolved") == "true"
	detections, err := models.GetBECDetections(scope.OrgId, includeResolved)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, detections, http.StatusOK)
}

// BECDetectionResolve handles POST /api/bec/detections/{id}/resolve
func (as *Server) BECDetectionResolve(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
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
	var req struct {
		Action string `json:"action"` // flagged, quarantined, deleted, etc.
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: ErrInvalidJSON}, http.StatusBadRequest)
		return
	}
	if err := models.ResolveBECDetection(id, scope.OrgId, scope.UserId, req.Action); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, models.Response{Success: true, Message: "BEC detection resolved"}, http.StatusOK)
}

// BECDetectionSummary handles GET /api/bec/summary
func (as *Server) BECDetectionSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	scope := getOrgScope(r)
	summary, err := models.GetBECDetectionSummary(scope.OrgId)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, summary, http.StatusOK)
}

// BECAnalyzeEmail handles POST /api/bec/analyze — runs BEC-specific analysis.
func (as *Server) BECAnalyzeEmail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	if !as.aiConfig.Enabled {
		JSONResponse(w, models.Response{Success: false, Message: "AI analysis is not enabled"}, http.StatusServiceUnavailable)
		return
	}
	scope := getOrgScope(r)
	var req struct {
		EmailHeaders string `json:"email_headers"`
		EmailBody    string `json:"email_body"`
		SenderEmail  string `json:"sender_email"`
		Subject      string `json:"subject"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: ErrInvalidJSON}, http.StatusBadRequest)
		return
	}

	client, err := ai.NewClient(as.aiConfig.Provider, as.aiConfig.APIKey, as.aiConfig.Model)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "Failed to create AI client"}, http.StatusInternalServerError)
		return
	}

	// Build executive context from BEC profiles
	profiles, _ := models.GetBECProfiles(scope.OrgId)
	executives := make([]ai.BECExecutiveContext, 0, len(profiles))
	for _, p := range profiles {
		executives = append(executives, ai.BECExecutiveContext{
			Name:       p.ExecutiveName,
			Email:      p.ExecutiveEmail,
			Title:      p.Title,
			Department: p.Department,
		})
	}

	prompt := ai.BuildBECDetectionPrompt(req.EmailHeaders, req.EmailBody, req.SenderEmail, req.Subject, executives)
	resp, err := client.Generate(ai.BECDetectionSystemPrompt, prompt)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "BEC analysis failed: " + err.Error()}, http.StatusInternalServerError)
		return
	}

	result, err := ai.ParseBECDetectionResponse(resp.Content)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "Failed to parse BEC result: " + err.Error()}, http.StatusInternalServerError)
		return
	}

	// If BEC detected, create a detection record
	if result.IsBEC {
		detection := &models.BECDetection{
			OrgId:                 scope.OrgId,
			ImpersonatedEmail:     result.ImpersonatedEmail,
			ImpersonatedName:      result.ImpersonatedName,
			ActualSender:          result.ActualSender,
			AttackType:            result.AttackType,
			UrgencyLevel:          result.UrgencyLevel,
			FinancialRequest:      result.FinancialRequest,
			WireTransferMentioned: result.WireTransferMentioned,
			GiftCardMentioned:     result.GiftCardMentioned,
			ConfidenceScore:       result.Confidence,
			Summary:               result.Summary,
		}
		models.CreateBECDetection(detection)
	}

	JSONResponse(w, result, http.StatusOK)
}

// ────────────────────────────────────────────────────────────────
// Graymail Classification
// ────────────────────────────────────────────────────────────────

// GraymailRules handles GET/POST /api/graymail/rules
func (as *Server) GraymailRules(w http.ResponseWriter, r *http.Request) {
	scope := getOrgScope(r)
	switch r.Method {
	case http.MethodGet:
		rules, err := models.GetGraymailRules(scope.OrgId)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, rules, http.StatusOK)
	case http.MethodPost:
		var rule models.GraymailRule
		if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: ErrInvalidJSON}, http.StatusBadRequest)
			return
		}
		rule.OrgId = scope.OrgId
		if err := models.SaveGraymailRule(&rule); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, rule, http.StatusCreated)
	default:
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
	}
}

// GraymailRule handles PUT/DELETE /api/graymail/rules/{id}
func (as *Server) GraymailRule(w http.ResponseWriter, r *http.Request) {
	scope := getOrgScope(r)
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "Invalid ID"}, http.StatusBadRequest)
		return
	}
	switch r.Method {
	case http.MethodPut:
		var rule models.GraymailRule
		if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: ErrInvalidJSON}, http.StatusBadRequest)
			return
		}
		rule.Id = id
		rule.OrgId = scope.OrgId
		if err := models.SaveGraymailRule(&rule); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, rule, http.StatusOK)
	case http.MethodDelete:
		if err := models.DeleteGraymailRule(id, scope.OrgId); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, models.Response{Success: true, Message: "Rule deleted"}, http.StatusOK)
	default:
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
	}
}

// GraymailClassifications handles GET /api/graymail/classifications
func (as *Server) GraymailClassifications(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	scope := getOrgScope(r)
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 100
	}
	results, err := models.GetGraymailClassifications(scope.OrgId, limit)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, results, http.StatusOK)
}

// GraymailSummary handles GET /api/graymail/summary
func (as *Server) GraymailSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	scope := getOrgScope(r)
	summary, err := models.GetGraymailSummary(scope.OrgId)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, summary, http.StatusOK)
}

// GraymailAnalyze handles POST /api/graymail/analyze — runs graymail classification on an email.
func (as *Server) GraymailAnalyze(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	if !as.aiConfig.Enabled {
		JSONResponse(w, models.Response{Success: false, Message: "AI analysis is not enabled"}, http.StatusServiceUnavailable)
		return
	}
	scope := getOrgScope(r)
	var req struct {
		EmailHeaders string `json:"email_headers"`
		EmailBody    string `json:"email_body"`
		SenderEmail  string `json:"sender_email"`
		Subject      string `json:"subject"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: ErrInvalidJSON}, http.StatusBadRequest)
		return
	}

	client, err := ai.NewClient(as.aiConfig.Provider, as.aiConfig.APIKey, as.aiConfig.Model)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "Failed to create AI client"}, http.StatusInternalServerError)
		return
	}

	prompt := ai.BuildGraymailClassificationPrompt(req.EmailHeaders, req.EmailBody, req.SenderEmail, req.Subject)
	resp, err := client.Generate(ai.GraymailClassificationSystemPrompt, prompt)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "Graymail analysis failed: " + err.Error()}, http.StatusInternalServerError)
		return
	}

	result, err := ai.ParseGraymailClassificationResponse(resp.Content)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "Failed to parse graymail result"}, http.StatusInternalServerError)
		return
	}

	if result.IsGraymail {
		classification := &models.GraymailClassification{
			OrgId:           scope.OrgId,
			EmailSubject:    req.Subject,
			SenderEmail:     req.SenderEmail,
			Category:        result.Category,
			Subcategory:     result.Subcategory,
			ConfidenceScore: result.Confidence,
			ActionTaken:     result.SuggestedAction,
		}
		models.CreateGraymailClassification(classification)
	}

	JSONResponse(w, result, http.StatusOK)
}

// ────────────────────────────────────────────────────────────────
// One-Click Remediation
// ────────────────────────────────────────────────────────────────

// RemediationActions handles GET /api/remediation-actions/
func (as *Server) InboxRemediationActions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	scope := getOrgScope(r)
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 100
	}
	actions, err := models.GetRemediationActions(scope.OrgId, limit)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, actions, http.StatusOK)
}

// RemediationActionCreate handles POST /api/remediation-actions/ — one-click remediation
func (as *Server) InboxRemediationActionCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	scope := getOrgScope(r)
	var req struct {
		ActionType  string `json:"action_type"`
		TargetType  string `json:"target_type"`
		TargetId    int64  `json:"target_id"`
		TargetEmail string `json:"target_email"`
		MessageId   string `json:"message_id"`
		Subject     string `json:"subject"`
		SenderEmail string `json:"sender_email"`
		Scope       string `json:"scope"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: ErrInvalidJSON}, http.StatusBadRequest)
		return
	}
	if req.ActionType == "" {
		JSONResponse(w, models.Response{Success: false, Message: "action_type is required"}, http.StatusBadRequest)
		return
	}

	action := &models.RemediationAction{
		OrgId:       scope.OrgId,
		ActionType:  req.ActionType,
		TargetType:  req.TargetType,
		TargetId:    req.TargetId,
		TargetEmail: req.TargetEmail,
		MessageId:   req.MessageId,
		Subject:     req.Subject,
		SenderEmail: req.SenderEmail,
		Scope:       req.Scope,
		InitiatedBy: scope.UserId,
	}
	if action.Scope == "" {
		action.Scope = models.RemediationScopeSingle
	}

	// Org-wide purge requires approval
	if action.Scope == models.RemediationScopeOrgWide {
		action.RequiresApproval = true
	}

	if err := models.CreateRemediationAction(action); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}

	// If no approval needed, mark as executing (the worker will pick it up)
	if !action.RequiresApproval {
		models.UpdateRemediationStatus(action.Id, models.InboxRemStatusExecuting, "", 0)
		action.Status = models.InboxRemStatusExecuting
	}

	JSONResponse(w, action, http.StatusCreated)
}

// RemediationActionApprove handles POST /api/remediation-actions/{id}/approve
func (as *Server) InboxRemediationActionApprove(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	scope := getOrgScope(r)
	vars := mux.Vars(r)
	id, _ := strconv.ParseInt(vars["id"], 10, 64)
	if err := models.ApproveRemediation(id, scope.OrgId, scope.UserId); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, models.Response{Success: true, Message: "Remediation approved"}, http.StatusOK)
}

// RemediationActionReject handles POST /api/remediation-actions/{id}/reject
func (as *Server) InboxRemediationActionReject(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	scope := getOrgScope(r)
	vars := mux.Vars(r)
	id, _ := strconv.ParseInt(vars["id"], 10, 64)
	var req struct {
		Reason string `json:"reason"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if err := models.RejectRemediation(id, scope.OrgId, req.Reason); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, models.Response{Success: true, Message: "Remediation rejected"}, http.StatusOK)
}

// RemediationSummary handles GET /api/remediation-actions/summary
func (as *Server) InboxRemediationSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	scope := getOrgScope(r)
	summary, err := models.GetRemediationSummaryStats(scope.OrgId)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, summary, http.StatusOK)
}

// ────────────────────────────────────────────────────────────────
// Phishing Ticket Management
// ────────────────────────────────────────────────────────────────

// PhishingTickets handles GET /api/phishing-tickets/
func (as *Server) PhishingTickets(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	scope := getOrgScope(r)
	statusFilter := r.URL.Query().Get("status")
	tickets, err := models.GetPhishingTickets(scope.OrgId, statusFilter)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, tickets, http.StatusOK)
}

// PhishingTicket handles GET/PUT /api/phishing-tickets/{id}
func (as *Server) PhishingTicketDetail(w http.ResponseWriter, r *http.Request) {
	scope := getOrgScope(r)
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "Invalid ID"}, http.StatusBadRequest)
		return
	}
	switch r.Method {
	case http.MethodGet:
		ticket, err := models.GetPhishingTicket(id, scope.OrgId)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusNotFound)
			return
		}
		JSONResponse(w, ticket, http.StatusOK)
	case http.MethodPut:
		ticket, err := models.GetPhishingTicket(id, scope.OrgId)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusNotFound)
			return
		}
		var updates struct {
			Status          string `json:"status"`
			AssignedTo      int64  `json:"assigned_to"`
			ResolutionNotes string `json:"resolution_notes"`
		}
		if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: ErrInvalidJSON}, http.StatusBadRequest)
			return
		}
		if updates.Status != "" {
			ticket.Status = updates.Status
		}
		if updates.AssignedTo > 0 {
			ticket.AssignedTo = updates.AssignedTo
		}
		if updates.ResolutionNotes != "" {
			ticket.ResolutionNotes = updates.ResolutionNotes
		}
		if err := models.UpdatePhishingTicket(&ticket); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, ticket, http.StatusOK)
	default:
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
	}
}

// PhishingTicketResolve handles POST /api/phishing-tickets/{id}/resolve
func (as *Server) PhishingTicketResolve(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	scope := getOrgScope(r)
	vars := mux.Vars(r)
	id, _ := strconv.ParseInt(vars["id"], 10, 64)
	var req struct {
		Notes string `json:"notes"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if err := models.ResolvePhishingTicket(id, scope.OrgId, req.Notes, false, ""); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, models.Response{Success: true, Message: "Ticket resolved"}, http.StatusOK)
}

// PhishingTicketEscalate handles POST /api/phishing-tickets/{id}/escalate
func (as *Server) PhishingTicketEscalate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	scope := getOrgScope(r)
	vars := mux.Vars(r)
	id, _ := strconv.ParseInt(vars["id"], 10, 64)
	var req struct {
		EscalateTo int64 `json:"escalate_to"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if err := models.EscalatePhishingTicket(id, scope.OrgId, req.EscalateTo); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, models.Response{Success: true, Message: "Ticket escalated"}, http.StatusOK)
}

// PhishingTicketAutoRules handles GET/POST /api/phishing-tickets/auto-rules
func (as *Server) PhishingTicketAutoRules(w http.ResponseWriter, r *http.Request) {
	scope := getOrgScope(r)
	switch r.Method {
	case http.MethodGet:
		rules, err := models.GetPhishingTicketAutoRules(scope.OrgId)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, rules, http.StatusOK)
	case http.MethodPost:
		var rule models.PhishingTicketAutoRule
		if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: ErrInvalidJSON}, http.StatusBadRequest)
			return
		}
		rule.OrgId = scope.OrgId
		if err := models.SavePhishingTicketAutoRule(&rule); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, rule, http.StatusCreated)
	default:
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
	}
}

// PhishingTicketAutoRuleDelete handles DELETE /api/phishing-tickets/auto-rules/{id}
func (as *Server) PhishingTicketAutoRuleDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	scope := getOrgScope(r)
	vars := mux.Vars(r)
	id, _ := strconv.ParseInt(vars["id"], 10, 64)
	if err := models.DeletePhishingTicketAutoRule(id, scope.OrgId); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, models.Response{Success: true, Message: "Auto-rule deleted"}, http.StatusOK)
}

// PhishingTicketSummary handles GET /api/phishing-tickets/summary
func (as *Server) PhishingTicketSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	scope := getOrgScope(r)
	summary, err := models.GetPhishingTicketSummary(scope.OrgId)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, summary, http.StatusOK)
}

// ────────────────────────────────────────────────────────────────
// Email Security Dashboard (unified view)
// ────────────────────────────────────────────────────────────────

// EmailSecurityDashboard handles GET /api/email-security/dashboard
func (as *Server) EmailSecurityDashboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	scope := getOrgScope(r)

	type Dashboard struct {
		InboxScan     models.InboxScanSummary        `json:"inbox_scan"`
		BEC           models.BECDetectionSummary     `json:"bec"`
		Graymail      models.GraymailSummary         `json:"graymail"`
		Remediation   models.RemediationSummaryStats `json:"remediation"`
		Tickets       models.PhishingTicketSummary   `json:"tickets"`
		EmailAnalysis models.EmailAnalysisSummary    `json:"email_analysis"`
	}

	var dashboard Dashboard
	dashboard.InboxScan, _ = models.GetInboxScanSummary(scope.OrgId)
	dashboard.BEC, _ = models.GetBECDetectionSummary(scope.OrgId)
	dashboard.Graymail, _ = models.GetGraymailSummary(scope.OrgId)
	dashboard.Remediation, _ = models.GetRemediationSummaryStats(scope.OrgId)
	dashboard.Tickets, _ = models.GetPhishingTicketSummary(scope.OrgId)
	dashboard.EmailAnalysis, _ = models.GetEmailAnalysisSummary(scope.OrgId)

	JSONResponse(w, dashboard, http.StatusOK)
}
