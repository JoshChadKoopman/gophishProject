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

// AdminAssistantChat handles POST /api/admin-assistant/chat — sends a
// question to the AI admin assistant and returns the assistant's reply.
func (as *Server) AdminAssistantChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	if !as.aiConfig.Enabled {
		JSONResponse(w, models.Response{Success: false, Message: "AI provider is not configured. Configure it in Settings → AI."}, http.StatusServiceUnavailable)
		return
	}

	scope := getOrgScope(r)
	var req struct {
		ConversationId int64  `json:"conversation_id"`
		Question       string `json:"question"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: ErrInvalidJSON}, http.StatusBadRequest)
		return
	}
	if req.Question == "" {
		JSONResponse(w, models.Response{Success: false, Message: "question is required"}, http.StatusBadRequest)
		return
	}
	req.Question = ai.SanitizePromptField(req.Question)

	if err := ai.CheckBudget(scope.OrgId, as.aiConfig.MonthlyTokenBudget, 1000); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "Monthly AI token budget exceeded"}, http.StatusTooManyRequests)
		return
	}

	client, err := ai.NewClient(as.aiConfig.Provider, as.aiConfig.APIKey, as.aiConfig.Model)
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Failed to create AI client"}, http.StatusInternalServerError)
		return
	}

	reply, err := models.AskAdminAssistant(scope.OrgId, scope.UserId, req.ConversationId, req.Question, client)
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "AI assistant request failed"}, http.StatusInternalServerError)
		return
	}
	ai.RecordTokenUsage(scope.OrgId, reply.TokensUsed)
	JSONResponse(w, reply, http.StatusOK)
}

// AdminAssistantConversations handles GET /api/admin-assistant/conversations
// — lists all conversations for the current admin.
func (as *Server) AdminAssistantConversations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	scope := getOrgScope(r)
	convs, err := models.ListAssistantConversations(scope.OrgId, scope.UserId)
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Failed to list conversations"}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, convs, http.StatusOK)
}

// AdminAssistantConversation handles GET /api/admin-assistant/conversations/{id}
// — returns a single conversation with its messages.
func (as *Server) AdminAssistantConversation(w http.ResponseWriter, r *http.Request) {
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
	conv, err := models.GetAssistantConversation(id, scope.OrgId, scope.UserId)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "Not found"}, http.StatusNotFound)
		return
	}
	JSONResponse(w, conv, http.StatusOK)
}

// AdminAssistantOnboarding handles GET /api/admin-assistant/onboarding —
// returns the onboarding completion status for the current admin.
func (as *Server) AdminAssistantOnboarding(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	scope := getOrgScope(r)
	status, err := models.GetAdminOnboardingStatus(scope.OrgId, scope.UserId)
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Failed to retrieve onboarding status"}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, status, http.StatusOK)
}

// AdminAssistantOnboardingComplete handles POST /api/admin-assistant/onboarding/{step}/complete
// — marks an onboarding step as completed for the current admin.
func (as *Server) AdminAssistantOnboardingComplete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	scope := getOrgScope(r)
	vars := mux.Vars(r)
	step := vars["step"]
	if step == "" {
		JSONResponse(w, models.Response{Success: false, Message: "step is required"}, http.StatusBadRequest)
		return
	}
	if err := models.CompleteOnboardingStep(scope.OrgId, scope.UserId, step); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusBadRequest)
		return
	}
	status, err := models.GetAdminOnboardingStatus(scope.OrgId, scope.UserId)
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Failed to retrieve onboarding status"}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, status, http.StatusOK)
}
