package api

import (
	"encoding/json"
	"net/http"

	ctx "github.com/gophish/gophish/context"
	log "github.com/gophish/gophish/logger"
	"github.com/gophish/gophish/models"
	"github.com/gorilla/mux"
)

// ── Pre-Built Compliance Training Module Progress ───────────────
// Endpoints for tracking user progress through compliance training modules,
// managing assignments, and viewing org-level stats.

// ComplianceModuleProgress handles GET/POST /api/compliance/module-progress.
// GET: returns the current user's compliance module progress.
// POST: updates progress for a module.
func (as *Server) ComplianceModuleProgressHandler(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)

	switch r.Method {
	case http.MethodGet:
		progress, err := models.GetUserComplianceProgress(user.Id)
		if err != nil {
			log.Error(err)
			JSONResponse(w, models.Response{Success: false, Message: "Error fetching progress"}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, progress, http.StatusOK)

	case http.MethodPost:
		var p models.ComplianceModuleProgress
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: ErrInvalidJSON}, http.StatusBadRequest)
			return
		}
		p.UserId = user.Id
		p.OrgId = user.OrgId
		if err := models.SaveComplianceModuleProgress(&p); err != nil {
			log.Error(err)
			JSONResponse(w, models.Response{Success: false, Message: "Error saving progress"}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, p, http.StatusOK)

	default:
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
	}
}

// ComplianceModuleAssign handles POST /api/compliance/module-assign.
// Assigns a compliance module to a user or group.
func (as *Server) ComplianceModuleAssign(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	user := ctx.Get(r, "user").(models.User)

	var a models.ComplianceModuleAssignment
	if err := json.NewDecoder(r.Body).Decode(&a); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: ErrInvalidJSON}, http.StatusBadRequest)
		return
	}
	a.OrgId = user.OrgId
	a.AssignedBy = user.Id

	// Validate the module exists
	m := models.GetComplianceTrainingModule(a.ModuleSlug)
	if m == nil {
		JSONResponse(w, models.Response{Success: false, Message: "Module not found"}, http.StatusNotFound)
		return
	}

	if err := models.AssignComplianceModule(&a); err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Error assigning module"}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, a, http.StatusCreated)
}

// ComplianceModuleOrgStats handles GET /api/compliance/module-stats.
// Returns org-level compliance training statistics.
func (as *Server) ComplianceModuleOrgStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	user := ctx.Get(r, "user").(models.User)
	stats, err := models.GetComplianceOrgStats(user.OrgId)
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Error fetching stats"}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, stats, http.StatusOK)
}

// ComplianceModuleOrgAssignments handles GET /api/compliance/module-assignments.
func (as *Server) ComplianceModuleOrgAssignments(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	user := ctx.Get(r, "user").(models.User)
	assignments, err := models.GetOrgComplianceModuleAssignments(user.OrgId)
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Error fetching assignments"}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, assignments, http.StatusOK)
}

// ComplianceModuleSeed handles POST /api/compliance/module-seed.
// Seeds compliance module assignments based on enabled frameworks.
func (as *Server) ComplianceModuleSeed(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	user := ctx.Get(r, "user").(models.User)

	var req struct {
		DueDate string `json:"due_date"` // ISO 8601 date
	}
	json.NewDecoder(r.Body).Decode(&req)

	dueDate := parseDate(req.DueDate, 30) // Default 30 days from now

	seeded, err := models.SeedComplianceModuleAssignmentsForOrg(user.OrgId, user.Id, dueDate)
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Error seeding assignments"}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
		Seeded  int    `json:"seeded"`
	}{true, "Compliance module assignments seeded", seeded}, http.StatusOK)
}

// ComplianceModuleDetail handles GET /api/compliance/training-modules/{slug}/progress.
// Returns the current user's progress on a specific module.
func (as *Server) ComplianceModuleDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	user := ctx.Get(r, "user").(models.User)
	vars := mux.Vars(r)
	slug := vars["slug"]

	progress, err := models.GetComplianceModuleProgressForUser(user.Id, slug)
	if err != nil {
		// No progress yet
		JSONResponse(w, models.ComplianceModuleProgress{
			UserId:     user.Id,
			ModuleSlug: slug,
			Status:     models.CompModStatusPending,
		}, http.StatusOK)
		return
	}
	JSONResponse(w, progress, http.StatusOK)
}
