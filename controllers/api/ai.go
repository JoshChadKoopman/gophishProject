package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gophish/gophish/ai"
	ctx "github.com/gophish/gophish/context"
	log "github.com/gophish/gophish/logger"
	"github.com/gophish/gophish/models"
)

// AIGenerateTemplate handles POST /api/ai/generate-template.
// It calls the configured AI provider to generate a phishing email template
// and optionally saves it directly to the templates table.
func (as *Server) AIGenerateTemplate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}

	if as.aiConfig.APIKey == "" || as.aiConfig.Provider == "" {
		JSONResponse(w, models.Response{Success: false, Message: "AI is not configured. Set the AI provider and API key in config or environment variables."}, http.StatusServiceUnavailable)
		return
	}

	var req struct {
		ai.GenerateRequest
		SaveAsTemplate bool   `json:"save_as_template"`
		TemplateName   string `json:"template_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: ErrInvalidJSON}, http.StatusBadRequest)
		return
	}

	if req.Prompt == "" {
		JSONResponse(w, models.Response{Success: false, Message: "Prompt is required"}, http.StatusBadRequest)
		return
	}

	client, err := ai.NewClient(as.aiConfig.Provider, as.aiConfig.APIKey, as.aiConfig.Model)
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Failed to create AI client"}, http.StatusInternalServerError)
		return
	}

	result, err := ai.GenerateTemplate(client, req.GenerateRequest)
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "AI generation failed: " + err.Error()}, http.StatusInternalServerError)
		return
	}

	scope := getOrgScope(r)
	user := ctx.Get(r, "user").(models.User)

	// Log the generation for auditing / token tracking
	genLog := &models.AIGenerationLog{
		OrgId:        scope.OrgId,
		UserId:       user.Id,
		Provider:     result.Provider,
		ModelUsed:    result.Model,
		InputTokens:  result.InputTokens,
		OutputTokens: result.OutputTokens,
	}

	// Optionally save as a template directly
	var savedTemplate *models.Template
	if req.SaveAsTemplate {
		name := req.TemplateName
		if name == "" {
			name = "AI Generated - " + time.Now().Format("2006-01-02 15:04")
		}
		t := models.Template{
			UserId:          user.Id,
			OrgId:           scope.OrgId,
			Name:            name,
			Subject:         result.Subject,
			HTML:            result.HTML,
			Text:            result.Text,
			ModifiedDate:    time.Now().UTC(),
			AIGenerated:     true,
			DifficultyLevel: req.DifficultyLevel,
			Language:        req.Language,
			TargetRole:      req.TargetRole,
		}
		if err := models.PostTemplate(&t); err != nil {
			log.Error(err)
			JSONResponse(w, models.Response{Success: false, Message: "Template generated but failed to save: " + err.Error()}, http.StatusInternalServerError)
			return
		}
		genLog.TemplateId = t.Id
		savedTemplate = &t
	}

	if err := models.CreateAIGenerationLog(genLog); err != nil {
		log.Error(err)
	}

	type generateResponse struct {
		Subject      string           `json:"subject"`
		HTML         string           `json:"html"`
		Text         string           `json:"text"`
		InputTokens  int              `json:"input_tokens"`
		OutputTokens int              `json:"output_tokens"`
		Provider     string           `json:"provider"`
		Model        string           `json:"model"`
		Template     *models.Template `json:"template,omitempty"`
	}

	JSONResponse(w, generateResponse{
		Subject:      result.Subject,
		HTML:         result.HTML,
		Text:         result.Text,
		InputTokens:  result.InputTokens,
		OutputTokens: result.OutputTokens,
		Provider:     result.Provider,
		Model:        result.Model,
		Template:     savedTemplate,
	}, http.StatusOK)
}

// AIUsage handles GET /api/ai/usage.
// Returns token usage summary for the current org in the current month.
func (as *Server) AIUsage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}

	scope := getOrgScope(r)

	// Start of current month
	now := time.Now().UTC()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	summary, err := models.GetAIUsageSummary(scope.OrgId, monthStart)
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Failed to load usage data"}, http.StatusInternalServerError)
		return
	}

	type usageResponse struct {
		models.AIUsageSummary
		MonthlyTokenBudget int    `json:"monthly_token_budget"`
		Period             string `json:"period"`
	}

	JSONResponse(w, usageResponse{
		AIUsageSummary:     *summary,
		MonthlyTokenBudget: as.aiConfig.MonthlyTokenBudget,
		Period:             monthStart.Format("2006-01"),
	}, http.StatusOK)
}
