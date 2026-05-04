package api

import (
	"encoding/json"
	"net/http"

	ctx "github.com/gophish/gophish/context"
	"github.com/gophish/gophish/models"
	"github.com/gorilla/mux"
)

// TrainingCertificateVerify verifies a certificate by its verification code.
// GET: public (any authenticated API user)
func (as *Server) TrainingCertificateVerify(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	code := vars["code"]

	cert, err := models.GetCertificate(code)
	if err != nil {
		JSONResponse(w, map[string]interface{}{
			"valid": false,
		}, http.StatusNotFound)
		return
	}

	valid := models.IsCertificateValid(cert)

	// Load presentation and user details for the response
	presentation, _ := models.GetTrainingPresentation(cert.PresentationId, getOrgScope(r))
	user, _ := models.GetUser(cert.UserId)

	enriched := models.EnrichCertificate(cert)

	JSONResponse(w, map[string]interface{}{
		"valid":             valid,
		"verification_code": enriched.FormattedCode,
		"user_name":         user.FirstName + " " + user.LastName,
		"user_username":     user.Username,
		"course_name":       presentation.Name,
		"issued_date":       cert.IssuedDate,
		"expires_date":      cert.ExpiresDate,
		"is_revoked":        cert.IsRevoked,
		"template":          enriched.Template,
	}, http.StatusOK)
}

// TrainingMyCertificates returns all certificates for the current user.
// GET: any authenticated user
func (as *Server) TrainingMyCertificates(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)
	certs, err := models.GetCertificatesForUser(user.Id)
	if err != nil {
		JSONResponse(w, []models.Certificate{}, http.StatusOK)
		return
	}

	resp := []models.EnrichedCertificate{}
	for _, c := range certs {
		ec := models.EnrichCertificate(c)
		pres, err := models.GetTrainingPresentation(c.PresentationId, getOrgScope(r))
		if err == nil {
			ec.CourseName = pres.Name
		}
		ec.UserName = user.FirstName + " " + user.LastName
		resp = append(resp, ec)
	}
	JSONResponse(w, resp, http.StatusOK)
}

// TrainingCertificateTemplates returns all available specialized certificate templates.
// GET: any authenticated user
func (as *Server) TrainingCertificateTemplates(w http.ResponseWriter, r *http.Request) {
	category := r.URL.Query().Get("category")
	if category != "" {
		JSONResponse(w, models.GetCertificateTemplatesByCategory(category), http.StatusOK)
		return
	}
	JSONResponse(w, models.GetCertificateTemplates(), http.StatusOK)
}

// TrainingCertificateIssue manually issues a certificate for a user on a course.
// POST: requires manage_training
func (as *Server) TrainingCertificateIssue(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)
	hasPermission, _ := user.HasPermission(models.PermissionManageTraining)
	if !hasPermission {
		JSONResponse(w, models.Response{Success: false, Message: ErrPermissionDenied}, http.StatusForbidden)
		return
	}

	var req struct {
		UserId         int64  `json:"user_id"`
		PresentationId int64  `json:"presentation_id"`
		TemplateSlug   string `json:"template_slug"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: ErrInvalidJSON}, http.StatusBadRequest)
		return
	}
	if req.UserId == 0 || req.PresentationId == 0 {
		JSONResponse(w, models.Response{Success: false, Message: "user_id and presentation_id are required"}, http.StatusBadRequest)
		return
	}
	if req.TemplateSlug == "" {
		req.TemplateSlug = "cybersecurity-awareness-foundation"
	}
	if models.GetCertificateTemplate(req.TemplateSlug) == nil {
		JSONResponse(w, models.Response{Success: false, Message: "Unknown certificate template slug"}, http.StatusBadRequest)
		return
	}

	cert, err := models.IssueCertificateWithTemplate(req.UserId, req.PresentationId, 0, req.TemplateSlug)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "An internal error occurred"}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, models.EnrichCertificate(*cert), http.StatusCreated)
}

// TrainingCertificateRevoke revokes a certificate by ID.
// POST: requires manage_training
func (as *Server) TrainingCertificateRevoke(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)
	hasPermission, _ := user.HasPermission(models.PermissionManageTraining)
	if !hasPermission {
		JSONResponse(w, models.Response{Success: false, Message: ErrPermissionDenied}, http.StatusForbidden)
		return
	}

	vars := mux.Vars(r)
	id, ok := parseIDParam(w, vars, "id")
	if !ok {
		return
	}

	if err := models.RevokeCertificate(id); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "An internal error occurred"}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, models.Response{Success: true, Message: "Certificate revoked"}, http.StatusOK)
}

// TrainingCertificateRenew renews an expiring/expired certificate.
// POST: requires manage_training
func (as *Server) TrainingCertificateRenew(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)
	hasPermission, _ := user.HasPermission(models.PermissionManageTraining)
	if !hasPermission {
		JSONResponse(w, models.Response{Success: false, Message: ErrPermissionDenied}, http.StatusForbidden)
		return
	}

	vars := mux.Vars(r)
	id, ok := parseIDParam(w, vars, "id")
	if !ok {
		return
	}

	cert, err := models.RenewCertificate(id)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "An internal error occurred"}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, models.EnrichCertificate(*cert), http.StatusCreated)
}

// TrainingCertificateSummary returns aggregate certificate statistics.
// GET: requires manage_training
func (as *Server) TrainingCertificateSummary(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)
	hasPermission, _ := user.HasPermission(models.PermissionManageTraining)
	if !hasPermission {
		JSONResponse(w, models.Response{Success: false, Message: ErrPermissionDenied}, http.StatusForbidden)
		return
	}
	JSONResponse(w, models.GetCertificateSummary(), http.StatusOK)
}
