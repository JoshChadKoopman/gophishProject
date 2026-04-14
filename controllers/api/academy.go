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

// AcademyTiers handles GET /api/academy/tiers — list tiers with user progress.
func (as *Server) AcademyTiers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	scope := getOrgScope(r)
	user := ctx.Get(r, "user").(models.User)

	tiers, err := models.GetAcademyTiersWithProgress(scope.OrgId, user.Id)
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Failed to load tiers"}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, tiers, http.StatusOK)
}

// AcademyTierSessions handles GET /api/academy/tiers/{slug}/sessions — sessions in a tier.
func (as *Server) AcademyTierSessions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	scope := getOrgScope(r)
	user := ctx.Get(r, "user").(models.User)
	slug := mux.Vars(r)["slug"]

	tier, err := models.GetAcademyTierBySlug(scope.OrgId, slug)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "Tier not found"}, http.StatusNotFound)
		return
	}

	sessions, err := models.GetAcademySessionsWithUserProgress(tier.Id, user.Id)
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Failed to load sessions"}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, sessions, http.StatusOK)
}

// AcademyTierComplete handles POST /api/academy/tiers/{slug}/complete — attempt tier completion.
func (as *Server) AcademyTierComplete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	scope := getOrgScope(r)
	user := ctx.Get(r, "user").(models.User)
	slug := mux.Vars(r)["slug"]

	tier, err := models.GetAcademyTierBySlug(scope.OrgId, slug)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "Tier not found"}, http.StatusNotFound)
		return
	}

	if err := models.UpdateAcademyProgress(user.Id, tier.Id); err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Failed to update progress"}, http.StatusInternalServerError)
		return
	}

	// Check for newly earned badges
	newBadges := models.CheckAndAwardBadges(user.Id)
	// Record training activity for streak
	models.RecordTrainingActivity(user.Id)

	progress, err := models.GetAcademyUserProgress(user.Id, tier.Id)
	if err != nil {
		log.Error(err)
	}

	// Include praise message for tier completion
	praiseMsg := models.GetPraiseMessageByEvent(scope.OrgId, models.PraiseEventTierComplete)

	type completeResponse struct {
		Progress      models.AcademyUserProgress `json:"progress"`
		NewBadges     []models.UserBadge         `json:"new_badges"`
		PraiseMessage models.PraiseMessage       `json:"praise_message"`
	}
	JSONResponse(w, completeResponse{Progress: progress, NewBadges: newBadges, PraiseMessage: praiseMsg}, http.StatusOK)
}

// AcademyMyProgress handles GET /api/academy/my-progress — overall academy progress.
func (as *Server) AcademyMyProgress(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	user := ctx.Get(r, "user").(models.User)
	progress, err := models.GetUserAcademyOverview(user.Id)
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Failed to load progress"}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, progress, http.StatusOK)
}

// AcademySessionManage handles POST/PUT/DELETE /api/academy/sessions — manage sessions (admin).
func (as *Server) AcademySessionManage(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		s := models.AcademySession{}
		if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: ErrInvalidJSON}, http.StatusBadRequest)
			return
		}
		if err := models.CreateAcademySession(&s); err != nil {
			log.Error(err)
			JSONResponse(w, models.Response{Success: false, Message: "Failed to create session"}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, s, http.StatusCreated)

	case http.MethodPut:
		s := models.AcademySession{}
		if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: ErrInvalidJSON}, http.StatusBadRequest)
			return
		}
		if err := models.UpdateAcademySession(&s); err != nil {
			log.Error(err)
			JSONResponse(w, models.Response{Success: false, Message: "Failed to update session"}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, s, http.StatusOK)

	default:
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
	}
}

// AcademySessionDelete handles DELETE /api/academy/sessions/{id}.
func (as *Server) AcademySessionDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	id, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "Invalid ID"}, http.StatusBadRequest)
		return
	}
	if err := models.DeleteAcademySession(id); err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Failed to delete session"}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, models.Response{Success: true, Message: "Session deleted"}, http.StatusOK)
}

// ComplianceCertifications handles GET /api/academy/compliance — list certifications with progress.
func (as *Server) ComplianceCertifications(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	scope := getOrgScope(r)
	user := ctx.Get(r, "user").(models.User)

	certs, err := models.GetComplianceCertificationsWithProgress(scope.OrgId, user.Id)
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Failed to load certifications"}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, certs, http.StatusOK)
}

// ComplianceCertComplete handles POST /api/academy/compliance/{id}/complete — attempt certification.
func (as *Server) ComplianceCertComplete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	user := ctx.Get(r, "user").(models.User)
	id, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "Invalid ID"}, http.StatusBadRequest)
		return
	}

	cert, err := models.GetComplianceCertification(id)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "Certification not found"}, http.StatusNotFound)
		return
	}

	uc, issued := models.CheckAndIssueComplianceCert(user.Id, cert)
	if !issued {
		JSONResponse(w, models.Response{Success: false, Message: "Not all required sessions are completed"}, http.StatusBadRequest)
		return
	}

	// Check badges
	models.CheckAndAwardBadges(user.Id)

	JSONResponse(w, uc, http.StatusOK)
}

// ComplianceMyCerts handles GET /api/academy/compliance/my-certs.
func (as *Server) ComplianceMyCerts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	user := ctx.Get(r, "user").(models.User)
	certs, err := models.GetUserComplianceCerts(user.Id)
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Failed to load certificates"}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, certs, http.StatusOK)
}

// ComplianceCertVerify handles GET /api/academy/compliance/verify/{code} — public verification.
func (as *Server) ComplianceCertVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	code := mux.Vars(r)["code"]
	uc, err := models.VerifyComplianceCert(code)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "Certificate not found"}, http.StatusNotFound)
		return
	}
	JSONResponse(w, uc, http.StatusOK)
}
