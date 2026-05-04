package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	ctx "github.com/gophish/gophish/context"
	log "github.com/gophish/gophish/logger"
	"github.com/gophish/gophish/models"
	"github.com/gorilla/mux"
)

// ── AI-Powered Content Translation ──────────────────────────────
// Dynamic translation of training content, templates, and pages using
// AI. Goes beyond static locale files to support any language on demand.

// AITranslationConfig handles GET/PUT /api/ai-translation/config.
func (as *Server) AITranslationConfig(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)
	switch r.Method {
	case http.MethodGet:
		cfg := models.GetTranslationConfig(user.OrgId)
		JSONResponse(w, cfg, http.StatusOK)
	case http.MethodPut:
		var cfg models.TranslationConfig
		if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: ErrInvalidJSON}, http.StatusBadRequest)
			return
		}
		cfg.OrgId = user.OrgId
		if err := models.SaveTranslationConfig(&cfg); err != nil {
			log.Error(err)
			JSONResponse(w, models.Response{Success: false, Message: "Error saving translation config"}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, cfg, http.StatusOK)
	default:
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
	}
}

// AITranslationLanguages handles GET /api/ai-translation/languages.
// Returns the list of supported AI translation languages.
func (as *Server) AITranslationLanguages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	type langEntry struct {
		Code string `json:"code"`
		Name string `json:"name"`
	}
	langs := make([]langEntry, 0, len(models.AITranslationLanguages))
	for code, name := range models.AITranslationLanguages {
		langs = append(langs, langEntry{Code: code, Name: name})
	}
	JSONResponse(w, langs, http.StatusOK)
}

// AITranslationTranslate handles POST /api/ai-translation/translate.
// Initiates an AI translation of content.
func (as *Server) AITranslationTranslate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	user := ctx.Get(r, "user").(models.User)

	var req struct {
		ContentType string `json:"content_type"`
		ContentId   int64  `json:"content_id"`
		SourceLang  string `json:"source_lang"`
		TargetLang  string `json:"target_lang"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: ErrInvalidJSON}, http.StatusBadRequest)
		return
	}

	if !models.IsValidTranslationLang(req.SourceLang) || !models.IsValidTranslationLang(req.TargetLang) {
		JSONResponse(w, models.Response{Success: false, Message: "Unsupported language code"}, http.StatusBadRequest)
		return
	}
	if req.SourceLang == req.TargetLang {
		JSONResponse(w, models.Response{Success: false, Message: "Source and target languages must differ"}, http.StatusBadRequest)
		return
	}

	// Check for cached translation
	existing, err := models.GetTranslatedContent(req.ContentType, req.ContentId, req.TargetLang)
	if err == nil && existing != nil {
		JSONResponse(w, existing, http.StatusOK)
		return
	}

	// Create translation request
	tReq := &models.TranslationRequest{
		OrgId:       user.OrgId,
		UserId:      user.Id,
		ContentType: req.ContentType,
		ContentId:   req.ContentId,
		SourceLang:  req.SourceLang,
		TargetLang:  req.TargetLang,
		Status:      models.TranslationStatusPending,
	}
	if err := models.CreateTranslationRequest(tReq); err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Error creating translation request"}, http.StatusInternalServerError)
		return
	}

	JSONResponse(w, tReq, http.StatusAccepted)
}

// AITranslationHistory handles GET /api/ai-translation/history.
func (as *Server) AITranslationHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	user := ctx.Get(r, "user").(models.User)
	limit := 50
	if l, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && l > 0 {
		limit = l
	}
	requests, err := models.GetTranslationRequests(user.OrgId, limit)
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Error fetching history"}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, requests, http.StatusOK)
}

// AITranslationUsage handles GET /api/ai-translation/usage.
func (as *Server) AITranslationUsage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	user := ctx.Get(r, "user").(models.User)
	since := time.Now().AddDate(0, -1, 0)
	if s := r.URL.Query().Get("since"); s != "" {
		if t, err := time.Parse("2006-01-02", s); err == nil {
			since = t
		}
	}
	summary, err := models.GetTranslationUsageSummary(user.OrgId, since)
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Error fetching usage"}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, summary, http.StatusOK)
}

// AITranslationContent handles GET /api/ai-translation/content/{id}.
// Returns all translations for a specific piece of content.
func (as *Server) AITranslationContent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	vars := mux.Vars(r)
	contentId, _ := strconv.ParseInt(vars["id"], 10, 64)
	contentType := r.URL.Query().Get("type")
	if contentType == "" {
		contentType = models.TranslationContentTemplate
	}

	translations, err := models.GetTranslationsForContent(contentType, contentId)
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Error fetching translations"}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, translations, http.StatusOK)
}

// AITranslationApprove handles POST /api/ai-translation/{id}/approve.
func (as *Server) AITranslationApprove(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	user := ctx.Get(r, "user").(models.User)
	vars := mux.Vars(r)
	id, ok := parseIDParam(w, vars, "id")
	if !ok {
		return
	}

	if err := models.ApproveTranslation(id, user.Id); err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Error approving translation"}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, models.Response{Success: true, Message: "Translation approved"}, http.StatusOK)
}
