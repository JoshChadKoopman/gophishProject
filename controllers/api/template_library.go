package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	log "github.com/gophish/gophish/logger"
	"github.com/gophish/gophish/models"
	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
)

// TemplateLibraryList handles GET /api/template-library/ — returns the built-in
// template library with optional ?category= and ?difficulty= filters.
func (as *Server) TemplateLibraryList(w http.ResponseWriter, r *http.Request) {
	category := r.URL.Query().Get("category")
	difficulty, _ := strconv.Atoi(r.URL.Query().Get("difficulty"))
	templates := models.GetTemplateLibrary(category, difficulty)
	JSONResponse(w, templates, http.StatusOK)
}

// TemplateLibraryCategories handles GET /api/template-library/categories.
func (as *Server) TemplateLibraryCategories(w http.ResponseWriter, r *http.Request) {
	JSONResponse(w, models.GetTemplateLibraryCategories(), http.StatusOK)
}

// TemplateLibraryImport handles POST /api/template-library/{slug}/import.
// It copies a library template into the user's own templates.
func (as *Server) TemplateLibraryImport(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	slug := vars["slug"]

	lt, ok := models.GetLibraryTemplate(slug)
	if !ok {
		JSONResponse(w, models.Response{Success: false, Message: "Library template not found"}, http.StatusNotFound)
		return
	}

	// Allow the caller to override the name via JSON body.
	type importReq struct {
		Name string `json:"name"`
	}
	var req importReq
	if r.Body != nil {
		json.NewDecoder(r.Body).Decode(&req)
	}
	name := lt.Name
	if req.Name != "" {
		name = req.Name
	}

	scope := getOrgScope(r)

	// Check for duplicate name.
	_, err := models.GetTemplateByName(name, scope)
	if err != gorm.ErrRecordNotFound {
		JSONResponse(w, models.Response{Success: false, Message: "Template name already in use"}, http.StatusConflict)
		return
	}

	t := models.Template{
		UserId:          scope.UserId,
		OrgId:           scope.OrgId,
		Name:            name,
		Subject:         lt.Subject,
		Text:            lt.Text,
		HTML:            lt.HTML,
		EnvelopeSender:  lt.EnvelopeSender,
		ModifiedDate:    time.Now().UTC(),
		DifficultyLevel: lt.DifficultyLevel,
		Language:        lt.Language,
		TargetRole:      lt.TargetRole,
		Category:        lt.Category,
	}

	err = models.PostTemplate(&t)
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Error importing template"}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, t, http.StatusCreated)
}
