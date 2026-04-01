package api

import (
	"encoding/json"
	"net/http"

	ctx "github.com/gophish/gophish/context"
	"github.com/gophish/gophish/i18n"
	"github.com/gophish/gophish/models"
	"github.com/gorilla/mux"
)

// I18nTranslations returns the full translation map for a given locale.
// GET /api/i18n/{locale}
func (as *Server) I18nTranslations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	vars := mux.Vars(r)
	locale := vars["locale"]
	if !i18n.IsSupported(locale) {
		locale = i18n.DefaultLocale
	}
	translations := i18n.GetTranslations(locale)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(translations)
}

// I18nLanguages returns the list of supported languages.
// GET /api/i18n/languages
func (as *Server) I18nLanguages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(i18n.GetLanguages())
}

// UserLanguage allows a user to update their preferred language.
// PUT /api/user/language  body: {"preferred_language": "nl"}
func (as *Server) UserLanguage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	user := ctx.Get(r, "user").(models.User)
	var req struct {
		PreferredLanguage string `json:"preferred_language"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "Invalid request"}, http.StatusBadRequest)
		return
	}
	if !i18n.IsSupported(req.PreferredLanguage) {
		JSONResponse(w, models.Response{Success: false, Message: "Unsupported language"}, http.StatusBadRequest)
		return
	}
	user.PreferredLanguage = req.PreferredLanguage
	if err := models.PutUser(&user); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, models.Response{Success: true, Message: "Language updated"}, http.StatusOK)
}
