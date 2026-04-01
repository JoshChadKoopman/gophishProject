package api

import (
	"net/http"

	ctx "github.com/gophish/gophish/context"
	"github.com/gophish/gophish/models"
	"github.com/gorilla/mux"
)

// TrainingCertificateVerify verifies a certificate by its verification code.
// GET: public (no authentication required — verified via RequireAPIKey on the router,
// so this is accessible to any authenticated API user or via the web UI).
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

	// Load presentation and user details for the response
	presentation, _ := models.GetTrainingPresentation(cert.PresentationId, getOrgScope(r))
	user, _ := models.GetUser(cert.UserId)

	JSONResponse(w, map[string]interface{}{
		"valid":             true,
		"verification_code": cert.VerificationCode,
		"user_name":         user.FirstName + " " + user.LastName,
		"user_username":     user.Username,
		"course_name":       presentation.Name,
		"issued_date":       cert.IssuedDate,
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

	// Enrich with presentation names
	type certResponse struct {
		models.Certificate
		CourseName string `json:"course_name"`
	}
	resp := []certResponse{}
	for _, c := range certs {
		name := ""
		pres, err := models.GetTrainingPresentation(c.PresentationId, getOrgScope(r))
		if err == nil {
			name = pres.Name
		}
		resp = append(resp, certResponse{
			Certificate: c,
			CourseName:  name,
		})
	}
	JSONResponse(w, resp, http.StatusOK)
}
