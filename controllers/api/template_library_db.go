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
	id, ok := parseIDParam(w, vars, "id")
	if !ok {
		return
	}

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

// TemplateLibraryDBSeedAll handles POST /api/template-library-db/seed-all
// Seeds built-in + new categories + multilingual skeleton templates.
func (as *Server) TemplateLibraryDBSeedAll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	total, err := models.SeedAllTemplates()
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}
	type seedResult struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
		Total   int    `json:"total"`
	}
	JSONResponse(w, seedResult{Success: true, Message: "All templates seeded", Total: total}, http.StatusOK)
}

// TemplateLibraryDBSeedMultilingual handles POST /api/template-library-db/seed-multilingual
// Creates skeleton records for NL, DE, FR, ES translations of all 18 builtin templates.
func (as *Server) TemplateLibraryDBSeedMultilingual(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	created, skipped := models.GenerateMultilingualSeeds()
	type mlResult struct {
		Created int `json:"created"`
		Skipped int `json:"skipped"`
	}
	JSONResponse(w, mlResult{Created: created, Skipped: skipped}, http.StatusOK)
}

// ── Community Marketplace Endpoints ─────────────────────────────

// TemplateLibraryDBCommunitySubmit handles POST /api/template-library-db/community/submit
func (as *Server) TemplateLibraryDBCommunitySubmit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	user := ctx.Get(r, "user").(models.User)

	var req struct {
		TemplateId   int64 `json:"template_id"`
		AnonymizeOrg bool  `json:"anonymize_org"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: ErrInvalidJSON}, http.StatusBadRequest)
		return
	}

	sub, err := models.SubmitToCommunity(req.TemplateId, user.OrgId, user.Id, req.AnonymizeOrg)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusBadRequest)
		return
	}
	JSONResponse(w, sub, http.StatusCreated)
}

// TemplateLibraryDBCommunityList handles GET /api/template-library-db/community/submissions
func (as *Server) TemplateLibraryDBCommunityList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	user := ctx.Get(r, "user").(models.User)

	// Super admins see all pending; others see their org's submissions
	if user.Role.Slug == models.RoleSuperAdmin {
		subs, err := models.GetPendingCommunitySubmissions()
		if err != nil {
			log.Error(err)
			JSONResponse(w, models.Response{Success: false, Message: "Failed to list submissions"}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, subs, http.StatusOK)
	} else {
		subs, err := models.GetCommunitySubmissions(user.OrgId)
		if err != nil {
			log.Error(err)
			JSONResponse(w, models.Response{Success: false, Message: "Failed to list submissions"}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, subs, http.StatusOK)
	}
}

// TemplateLibraryDBCommunityReview handles POST /api/template-library-db/community/review
func (as *Server) TemplateLibraryDBCommunityReview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	user := ctx.Get(r, "user").(models.User)

	var req struct {
		SubmissionId int64  `json:"submission_id"`
		Approve      bool   `json:"approve"`
		Notes        string `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: ErrInvalidJSON}, http.StatusBadRequest)
		return
	}

	if err := models.ReviewCommunitySubmission(req.SubmissionId, req.Approve, user.Id, req.Notes); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusBadRequest)
		return
	}

	action := "rejected"
	if req.Approve {
		action = "approved"
	}
	JSONResponse(w, models.Response{Success: true, Message: "Submission " + action}, http.StatusOK)
}

// ── Full-Text Search ────────────────────────────────────────────

// TemplateLibraryDBSearch handles GET /api/template-library-db/search?q=...
// Uses FTS5/FULLTEXT index for instant search.
func (as *Server) TemplateLibraryDBSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	user := ctx.Get(r, "user").(models.User)
	query := r.URL.Query().Get("q")
	if query == "" {
		JSONResponse(w, models.Response{Success: false, Message: "query parameter 'q' is required"}, http.StatusBadRequest)
		return
	}
	page := 1
	pageSize := 25
	if p, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil {
		page = p
	}
	if ps, err := strconv.Atoi(r.URL.Query().Get("page_size")); err == nil {
		pageSize = ps
	}
	result, err := models.FTSSearchLibraryTemplates(query, user.OrgId, page, pageSize)
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Search failed"}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, result, http.StatusOK)
}

// ── TF-IDF Similarity ───────────────────────────────────────────

// TemplateLibraryDBSimilarity handles POST /api/template-library-db/similarity
// Accepts {subject, text, threshold, max_results} and returns similar templates.
func (as *Server) TemplateLibraryDBSimilarity(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Subject    string  `json:"subject"`
		Text       string  `json:"text"`
		Threshold  float64 `json:"threshold"`
		MaxResults int     `json:"max_results"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: ErrInvalidJSON}, http.StatusBadRequest)
		return
	}
	results, err := models.FindSimilarTemplates(req.Subject, req.Text, req.Threshold, req.MaxResults)
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Similarity search failed"}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, results, http.StatusOK)
}

// ── Human Review Workflow ───────────────────────────────────────

// TemplateLibraryDBReviews handles GET /api/template-library-db/reviews
// Returns pending reviews, optionally filtered by ?type=translation|ai_generated|community
func (as *Server) TemplateLibraryDBReviews(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	reviewType := r.URL.Query().Get("type")
	results, err := models.GetPendingReviewsWithTemplates(reviewType)
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Failed to list reviews"}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, results, http.StatusOK)
}

// TemplateLibraryDBReviewStats handles GET /api/template-library-db/reviews/stats
func (as *Server) TemplateLibraryDBReviewStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	stats := models.GetReviewStats()
	JSONResponse(w, stats, http.StatusOK)
}

// TemplateLibraryDBReviewComplete handles POST /api/template-library-db/reviews/complete
func (as *Server) TemplateLibraryDBReviewComplete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	user := ctx.Get(r, "user").(models.User)
	var req struct {
		ReviewId     int64  `json:"review_id"`
		Approved     bool   `json:"approved"`
		Notes        string `json:"notes"`
		QualityScore int    `json:"quality_score"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: ErrInvalidJSON}, http.StatusBadRequest)
		return
	}
	if err := models.CompleteTemplateReview(req.ReviewId, req.Approved, user.Id, req.Notes, req.QualityScore); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusBadRequest)
		return
	}
	action := "rejected"
	if req.Approved {
		action = "approved"
	}
	JSONResponse(w, models.Response{Success: true, Message: "Review " + action}, http.StatusOK)
}

// TemplateLibraryDBReviewRevision handles POST /api/template-library-db/reviews/revision
func (as *Server) TemplateLibraryDBReviewRevision(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	user := ctx.Get(r, "user").(models.User)
	var req struct {
		ReviewId int64  `json:"review_id"`
		Notes    string `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: ErrInvalidJSON}, http.StatusBadRequest)
		return
	}
	if err := models.RequestRevision(req.ReviewId, user.Id, req.Notes); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusBadRequest)
		return
	}
	JSONResponse(w, models.Response{Success: true, Message: "Revision requested"}, http.StatusOK)
}

// TemplateLibraryDBCreateReviews handles POST /api/template-library-db/reviews/create-pending
// Auto-creates review records for all unpublished templates without reviews.
func (as *Server) TemplateLibraryDBCreateReviews(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	created, err := models.CreateReviewsForUnreviewedTemplates()
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}
	type result struct {
		Created int `json:"created"`
	}
	JSONResponse(w, result{Created: created}, http.StatusOK)
}

// ── CSV Import/Export ───────────────────────────────────────────

// TemplateLibraryDBExportCSV handles GET /api/template-library-db/export-csv
func (as *Server) TemplateLibraryDBExportCSV(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	user := ctx.Get(r, "user").(models.User)
	data, err := models.ExportLibraryTemplatesCSV(user.OrgId)
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "CSV export failed"}, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=template_library.csv")
	w.Write(data)
}

// TemplateLibraryDBImportCSV handles POST /api/template-library-db/import-csv
func (as *Server) TemplateLibraryDBImportCSV(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	user := ctx.Get(r, "user").(models.User)

	// Read body as CSV data (max 10MB)
	r.Body = http.MaxBytesReader(w, r.Body, 10<<20)
	data := make([]byte, 0, 1024)
	buf := make([]byte, 4096)
	for {
		n, err := r.Body.Read(buf)
		if n > 0 {
			data = append(data, buf[:n]...)
		}
		if err != nil {
			break
		}
	}

	imported, skipped, err := models.ImportLibraryTemplatesCSV(data, user.OrgId, user.Id)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusBadRequest)
		return
	}
	type importResult struct {
		Imported int `json:"imported"`
		Skipped  int `json:"skipped"`
	}
	JSONResponse(w, importResult{Imported: imported, Skipped: skipped}, http.StatusOK)
}

// ── Bulk Moderation ─────────────────────────────────────────────

// TemplateLibraryDBBulkPublish handles POST /api/template-library-db/bulk-publish
func (as *Server) TemplateLibraryDBBulkPublish(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Ids []int64 `json:"ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: ErrInvalidJSON}, http.StatusBadRequest)
		return
	}
	count, err := models.BulkPublishTemplates(req.Ids)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, map[string]interface{}{"published": count}, http.StatusOK)
}

// TemplateLibraryDBBulkUnpublish handles POST /api/template-library-db/bulk-unpublish
func (as *Server) TemplateLibraryDBBulkUnpublish(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Ids []int64 `json:"ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: ErrInvalidJSON}, http.StatusBadRequest)
		return
	}
	count, err := models.BulkUnpublishTemplates(req.Ids)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, map[string]interface{}{"unpublished": count}, http.StatusOK)
}

// TemplateLibraryDBBulkDelete handles POST /api/template-library-db/bulk-delete
func (as *Server) TemplateLibraryDBBulkDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Ids []int64 `json:"ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: ErrInvalidJSON}, http.StatusBadRequest)
		return
	}
	count, err := models.BulkDeleteTemplates(req.Ids)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, map[string]interface{}{"deleted": count}, http.StatusOK)
}

// TemplateLibraryDBBulkTag handles POST /api/template-library-db/bulk-tag
func (as *Server) TemplateLibraryDBBulkTag(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Ids  []int64  `json:"ids"`
		Tags []string `json:"tags"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: ErrInvalidJSON}, http.StatusBadRequest)
		return
	}
	count, err := models.BulkTagTemplates(req.Ids, req.Tags)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, map[string]interface{}{"tagged": count}, http.StatusOK)
}
