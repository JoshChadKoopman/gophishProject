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

// ── DB-Backed Template Library (Admin) ──────────────────────────
// These endpoints manage the persistent, DB-backed template library
// with search, CRUD, import/export, and stats.

// TemplateLibraryDB handles GET (search) and POST (create) for /api/template-library-db/.
func (as *Server) TemplateLibraryDB(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)
	switch r.Method {
	case http.MethodGet:
		params := models.LibrarySearchParams{
			Query:    r.URL.Query().Get("q"),
			Category: r.URL.Query().Get("category"),
			Language: r.URL.Query().Get("language"),
			Tag:      r.URL.Query().Get("tag"),
			Source:   r.URL.Query().Get("source"),
			OrgId:    user.OrgId,
		}
		if d, err := strconv.Atoi(r.URL.Query().Get("difficulty")); err == nil {
			params.Difficulty = d
		}
		if p, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil {
			params.Page = p
		}
		if ps, err := strconv.Atoi(r.URL.Query().Get("page_size")); err == nil {
			params.PageSize = ps
		}
		result, err := models.SearchLibraryTemplates(params)
		if err != nil {
			log.Error(err)
			JSONResponse(w, models.Response{Success: false, Message: "Search failed"}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, result, http.StatusOK)

	case http.MethodPost:
		var t models.DBLibraryTemplate
		if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: ErrInvalidJSON}, http.StatusBadRequest)
			return
		}
		t.OrgId = user.OrgId
		t.CreatedBy = user.Id
		if t.Source == "" {
			t.Source = "user"
		}
		if err := models.CreateDBLibraryTemplate(&t); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusBadRequest)
			return
		}
		JSONResponse(w, t, http.StatusCreated)

	default:
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
	}
}

// TemplateLibraryDBItem handles GET, PUT, DELETE for /api/template-library-db/{id}.
func (as *Server) TemplateLibraryDBItem(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.ParseInt(vars["id"], 10, 64)

	switch r.Method {
	case http.MethodGet:
		t, err := models.GetDBLibraryTemplate(id)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: "Template not found"}, http.StatusNotFound)
			return
		}
		JSONResponse(w, t, http.StatusOK)

	case http.MethodPut:
		t, err := models.GetDBLibraryTemplate(id)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: "Template not found"}, http.StatusNotFound)
			return
		}
		var update models.DBLibraryTemplate
		if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: ErrInvalidJSON}, http.StatusBadRequest)
			return
		}
		update.Id = t.Id
		update.CreatedDate = t.CreatedDate
		update.CreatedBy = t.CreatedBy
		if err := models.UpdateDBLibraryTemplate(&update); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, update, http.StatusOK)

	case http.MethodDelete:
		if err := models.DeleteDBLibraryTemplate(id); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, models.Response{Success: true, Message: "Template deleted"}, http.StatusOK)

	default:
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
	}
}

// TemplateLibraryDBStats handles GET /api/template-library-db/stats.
func (as *Server) TemplateLibraryDBStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	user := ctx.Get(r, "user").(models.User)
	stats := models.GetDBTemplateLibraryStats(user.OrgId)
	JSONResponse(w, stats, http.StatusOK)
}

// TemplateLibraryDBCategories handles GET /api/template-library-db/categories.
func (as *Server) TemplateLibraryDBCategories(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	user := ctx.Get(r, "user").(models.User)
	cats := models.GetDBLibraryCategories(user.OrgId)
	JSONResponse(w, cats, http.StatusOK)
}

// TemplateLibraryDBImport handles POST /api/template-library-db/import
// Bulk-imports templates from a JSON payload.
func (as *Server) TemplateLibraryDBImport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	user := ctx.Get(r, "user").(models.User)

	var payload models.TemplateImportPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: ErrInvalidJSON}, http.StatusBadRequest)
		return
	}

	imported, skipped, err := models.BulkImportLibraryTemplates(payload.Templates, user.OrgId, user.Id)
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Import failed"}, http.StatusInternalServerError)
		return
	}

	type importResult struct {
		Imported int `json:"imported"`
		Skipped  int `json:"skipped"`
	}
	JSONResponse(w, importResult{Imported: imported, Skipped: skipped}, http.StatusOK)
}

// TemplateLibraryDBExport handles GET /api/template-library-db/export
// Exports all templates for the org as JSON.
func (as *Server) TemplateLibraryDBExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	user := ctx.Get(r, "user").(models.User)
	templates, err := models.ExportLibraryTemplates(user.OrgId)
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Export failed"}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, models.TemplateImportPayload{Templates: templates}, http.StatusOK)
}

// TemplateLibraryDBSeed handles POST /api/template-library-db/seed
// Seeds built-in templates into the DB library.
func (as *Server) TemplateLibraryDBSeed(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	if err := models.SeedBuiltinTemplates(); err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, models.Response{Success: true, Message: "Built-in templates seeded successfully"}, http.StatusOK)
}
