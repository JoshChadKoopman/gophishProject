package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gophish/gophish/ai"
	ctx "github.com/gophish/gophish/context"
	log "github.com/gophish/gophish/logger"
	"github.com/gophish/gophish/models"
	"github.com/gorilla/mux"
)

// ── Board Report Enhanced Endpoints ─────────────────────────────
// AI narrative generation, period deltas, department heatmap,
// approval workflow with audit trail.

// BoardReportFull handles GET /api/board-reports/{id}/full
// Returns the complete enhanced payload with snapshot, narrative,
// deltas, heatmap, and audit trail.
func (as *Server) BoardReportFull(w http.ResponseWriter, r *http.Request) {
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
		JSONResponse(w, models.Response{Success: false, Message: "Error generating enhanced report"}, http.StatusInternalServerError)
		return
	}

	JSONResponse(w, payload, http.StatusOK)
}

// BoardReportGenerateNarrative handles POST /api/board-reports/{id}/generate-narrative
// Generates the AI narrative (or deterministic fallback) and stores it.
// Admins can then review and edit before publishing.
func (as *Server) BoardReportGenerateNarrative(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
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

	// Generate current snapshot
	snap, err := models.GenerateBoardReportSnapshot(user.OrgId, br.PeriodStart, br.PeriodEnd)
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Error generating snapshot"}, http.StatusInternalServerError)
		return
	}

	// Prior period for deltas
	duration := br.PeriodEnd.Sub(br.PeriodStart)
	priorStart := br.PeriodStart.Add(-duration)
	priorSnap, _ := models.GenerateBoardReportSnapshot(user.OrgId, priorStart, br.PeriodStart)
	deltas := models.ComputePeriodDeltas(snap, priorSnap)

	// Heatmap
	heatmap, _ := models.GenerateDeptHeatmap(user.OrgId)

	// Try AI generation first, fall back to deterministic
	var narrative *models.BoardReportNarrativeContent

	if as.aiConfig.Enabled && as.aiConfig.APIKey != "" {
		narrative, err = generateAINarrative(as, snap, deltas, heatmap, user)
		if err != nil {
			log.Errorf("AI narrative generation failed, using deterministic fallback: %v", err)
			narrative = models.BuildDeterministicNarrative(snap, deltas, heatmap)
		}
	} else {
		narrative = models.BuildDeterministicNarrative(snap, deltas, heatmap)
	}

	// Persist the narrative
	narrative.OrgId = user.OrgId
	narrative.ReportId = br.Id
	narrative.EditedBy = user.Id
	if err := models.SaveBoardReportNarrative(narrative); err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Error saving narrative"}, http.StatusInternalServerError)
		return
	}

	JSONResponse(w, narrative, http.StatusOK)
}

// generateAINarrative calls the AI provider to generate the 3-paragraph narrative.
func generateAINarrative(as *Server, snap *models.BoardReportSnapshot, deltas []models.PeriodDelta, heatmap *models.DeptHeatmap, user models.User) (*models.BoardReportNarrativeContent, error) {
	client, err := ai.NewClient(as.aiConfig.Provider, as.aiConfig.APIKey, as.aiConfig.Model)
	if err != nil {
		return nil, fmt.Errorf("create AI client: %w", err)
	}

	systemPrompt, userPrompt := models.BuildBoardNarrativePrompt(snap, deltas, heatmap)
	resp, err := client.Generate(systemPrompt, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("AI generation: %w", err)
	}

	// Parse the 3 paragraphs from the response
	paragraphs := splitParagraphs(resp.Content)
	narrative := &models.BoardReportNarrativeContent{
		AIGenerated:  true,
		CreatedDate:  time.Now().UTC(),
		ModifiedDate: time.Now().UTC(),
	}

	if len(paragraphs) >= 1 {
		narrative.Paragraph1 = paragraphs[0]
	}
	if len(paragraphs) >= 2 {
		narrative.Paragraph2 = paragraphs[1]
	}
	if len(paragraphs) >= 3 {
		narrative.Paragraph3 = paragraphs[2]
	}
	narrative.FullNarrative = resp.Content

	// Log AI usage
	models.CreateAIGenerationLog(&models.AIGenerationLog{
		OrgId:        user.OrgId,
		UserId:       user.Id,
		Provider:     client.Provider(),
		ModelUsed:    as.aiConfig.Model,
		InputTokens:  resp.InputTokens,
		OutputTokens: resp.OutputTokens,
	})

	return narrative, nil
}

// splitParagraphs splits text by double newlines into paragraphs.
func splitParagraphs(text string) []string {
	text = strings.TrimSpace(text)
	parts := strings.Split(text, "\n\n")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// BoardReportEditNarrative handles PUT /api/board-reports/{id}/narrative-edit
// Allows admins to review and edit the generated narrative before publishing.
func (as *Server) BoardReportEditNarrative(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	user := ctx.Get(r, "user").(models.User)
	vars := mux.Vars(r)
	id, _ := strconv.ParseInt(vars["id"], 10, 64)

	// Verify the report exists and belongs to this org
	_, err := models.GetBoardReport(id, user.OrgId)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusNotFound)
		return
	}

	var update models.BoardReportNarrativeContent
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: ErrInvalidJSON}, http.StatusBadRequest)
		return
	}

	update.ReportId = id
	update.OrgId = user.OrgId
	update.EditedBy = user.Id
	update.FullNarrative = update.Paragraph1 + "\n\n" + update.Paragraph2 + "\n\n" + update.Paragraph3
	if err := models.SaveBoardReportNarrative(&update); err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Error saving narrative"}, http.StatusInternalServerError)
		return
	}

	JSONResponse(w, update, http.StatusOK)
}

// BoardReportTransition handles POST /api/board-reports/{id}/transition
// Applies a status transition (draft → review → approved → published)
// and records it in the audit trail.
func (as *Server) BoardReportTransition(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	user := ctx.Get(r, "user").(models.User)
	vars := mux.Vars(r)
	id, _ := strconv.ParseInt(vars["id"], 10, 64)

	type transitionReq struct {
		Status  string `json:"status"`
		Comment string `json:"comment"`
	}
	var req transitionReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: ErrInvalidJSON}, http.StatusBadRequest)
		return
	}

	if req.Status == "" {
		JSONResponse(w, models.Response{Success: false, Message: "status is required"}, http.StatusBadRequest)
		return
	}

	if err := models.TransitionBoardReportStatus(id, user.OrgId, req.Status, user, req.Comment); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusBadRequest)
		return
	}

	// Return updated report
	br, _ := models.GetBoardReport(id, user.OrgId)
	approvals, _ := models.GetBoardReportApprovals(id, user.OrgId)

	JSONResponse(w, map[string]interface{}{
		"report":    br,
		"approvals": approvals,
	}, http.StatusOK)
}

// BoardReportApprovals handles GET /api/board-reports/{id}/approvals
// Returns the approval audit trail for a board report.
func (as *Server) BoardReportApprovals(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	user := ctx.Get(r, "user").(models.User)
	vars := mux.Vars(r)
	id, _ := strconv.ParseInt(vars["id"], 10, 64)

	approvals, err := models.GetBoardReportApprovals(id, user.OrgId)
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Error fetching approvals"}, http.StatusInternalServerError)
		return
	}

	JSONResponse(w, approvals, http.StatusOK)
}

// BoardReportHeatmap handles GET /api/board-reports/heatmap
// Returns the department-level risk heatmap.
func (as *Server) BoardReportHeatmap(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	user := ctx.Get(r, "user").(models.User)

	heatmap, err := models.GenerateDeptHeatmap(user.OrgId)
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Error generating heatmap"}, http.StatusInternalServerError)
		return
	}

	JSONResponse(w, heatmap, http.StatusOK)
}

// BoardReportDeltas handles POST /api/board-reports/deltas
// Returns period-over-period deltas for any date range.
func (as *Server) BoardReportDeltas(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	user := ctx.Get(r, "user").(models.User)

	type deltaReq struct {
		PeriodStart string `json:"period_start"`
		PeriodEnd   string `json:"period_end"`
	}
	var req deltaReq
	json.NewDecoder(r.Body).Decode(&req)

	start, _ := time.Parse("2006-01-02", req.PeriodStart)
	end, _ := time.Parse("2006-01-02", req.PeriodEnd)
	if start.IsZero() {
		start = time.Now().AddDate(0, -3, 0)
	}
	if end.IsZero() {
		end = time.Now()
	}

	current, err := models.GenerateBoardReportSnapshot(user.OrgId, start, end)
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Error generating snapshot"}, http.StatusInternalServerError)
		return
	}

	duration := end.Sub(start)
	priorStart := start.Add(-duration)
	prior, _ := models.GenerateBoardReportSnapshot(user.OrgId, priorStart, start)

	deltas := models.ComputePeriodDeltas(current, prior)
	JSONResponse(w, deltas, http.StatusOK)
}
