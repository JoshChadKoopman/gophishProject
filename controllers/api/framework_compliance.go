package api

import (
	"net/http"
	"time"

	ctx "github.com/gophish/gophish/context"
	log "github.com/gophish/gophish/logger"
	"github.com/gophish/gophish/models"
	"github.com/gorilla/mux"
)

// ── Framework Compliance Certificates ──

// FrameworkCertDefinitions returns all pre-built framework compliance cert definitions.
// GET /api/compliance/framework-certs/definitions
func (as *Server) FrameworkCertDefinitions(w http.ResponseWriter, r *http.Request) {
	framework := r.URL.Query().Get("framework")
	if framework != "" {
		JSONResponse(w, models.GetCertsForFramework(framework), http.StatusOK)
		return
	}
	JSONResponse(w, models.GetFrameworkComplianceCerts(), http.StatusOK)
}

// FrameworkCertOrgCerts returns all earned framework compliance certs for the user's org.
// GET /api/compliance/framework-certs
func (as *Server) FrameworkCertOrgCerts(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)

	activeOnly := r.URL.Query().Get("active") == "true"
	var certs []models.OrgFrameworkCert
	var err error
	if activeOnly {
		certs, err = models.GetActiveOrgFrameworkCerts(user.OrgId)
	} else {
		certs, err = models.GetOrgFrameworkCerts(user.OrgId)
	}
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Error fetching framework certs"}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, certs, http.StatusOK)
}

// FrameworkCertEvaluate triggers evaluation and auto-issuance of framework certs for the org.
// POST /api/compliance/framework-certs/evaluate
func (as *Server) FrameworkCertEvaluate(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)

	issued, err := models.EvaluateAndIssueFrameworkCerts(user.OrgId)
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Error evaluating framework certs"}, http.StatusInternalServerError)
		return
	}

	JSONResponse(w, map[string]interface{}{
		"success":      true,
		"message":      "Framework certificates evaluated",
		"issued":       issued,
		"issued_count": len(issued),
	}, http.StatusOK)
}

// FrameworkCertSummary returns a summary of framework cert status for the org.
// GET /api/compliance/framework-certs/summary
func (as *Server) FrameworkCertSummary(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)
	summary := models.GetFrameworkCertSummary(user.OrgId)
	JSONResponse(w, summary, http.StatusOK)
}

// FrameworkCertVerify verifies a framework compliance cert by verification code.
// GET /api/compliance/framework-certs/verify/{code}
func (as *Server) FrameworkCertVerify(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	code := vars["code"]

	cert, err := models.VerifyOrgFrameworkCert(code)
	if err != nil {
		JSONResponse(w, map[string]interface{}{
			"valid":   false,
			"message": "Certificate not found",
		}, http.StatusNotFound)
		return
	}

	def := models.GetFrameworkComplianceCert(cert.CertSlug)

	JSONResponse(w, map[string]interface{}{
		"valid":             !cert.IsRevoked && cert.ExpiresDate.After(time.Now().UTC()),
		"verification_code": cert.VerificationCode,
		"cert_name":         cert.CertName,
		"framework_slug":    cert.FrameworkSlug,
		"framework_score":   cert.FrameworkScore,
		"controls_passed":   cert.ControlsPassed,
		"total_controls":    cert.TotalControls,
		"issued_date":       cert.IssuedDate,
		"expires_date":      cert.ExpiresDate,
		"is_revoked":        cert.IsRevoked,
		"issuing_authority": safeIssuer(def),
	}, http.StatusOK)
}

// safeIssuer extracts the issuing authority from a cert definition, or returns a default.
func safeIssuer(def *models.FrameworkComplianceCert) string {
	if def != nil {
		return def.IssuingAuthority
	}
	return "Nivoxis Security Platform"
}

// FrameworkCertRevoke revokes an org framework cert.
// POST /api/compliance/framework-certs/{id}/revoke
func (as *Server) FrameworkCertRevoke(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, ok := parseIDParam(w, vars, "id")
	if !ok {
		return
	}

	if err := models.RevokeOrgFrameworkCert(id); err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Error revoking framework cert"}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, models.Response{Success: true, Message: "Framework certificate revoked"}, http.StatusOK)
}

// ── Compliance Training Modules ──

// ComplianceTrainingModules returns all framework-specific training modules.
// GET /api/compliance/training-modules
func (as *Server) ComplianceTrainingModules(w http.ResponseWriter, r *http.Request) {
	framework := r.URL.Query().Get("framework")
	full := r.URL.Query().Get("full") == "true"

	if framework != "" {
		if full {
			JSONResponse(w, models.GetComplianceModulesForFramework(framework), http.StatusOK)
		} else {
			// Filter summaries by framework
			summaries := models.GetComplianceModuleSummaries()
			var filtered []models.ComplianceModuleSummary
			for _, s := range summaries {
				if s.FrameworkSlug == framework {
					filtered = append(filtered, s)
				}
			}
			JSONResponse(w, filtered, http.StatusOK)
		}
		return
	}

	if full {
		JSONResponse(w, models.GetComplianceTrainingModules(), http.StatusOK)
		return
	}
	JSONResponse(w, models.GetComplianceModuleSummaries(), http.StatusOK)
}

// ComplianceTrainingModule returns a single compliance training module by slug.
// GET /api/compliance/training-modules/{slug}
func (as *Server) ComplianceTrainingModule(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	slug := vars["slug"]

	module := models.GetComplianceTrainingModule(slug)
	if module == nil {
		JSONResponse(w, models.Response{Success: false, Message: "Module not found"}, http.StatusNotFound)
		return
	}
	JSONResponse(w, module, http.StatusOK)
}

// ── Platform Certifications ──

// PlatformCertifications returns all platform-level security certifications.
// GET /api/compliance/platform-certifications
func (as *Server) PlatformCertifications(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	if status != "" {
		JSONResponse(w, models.GetPlatformCertificationsByStatus(status), http.StatusOK)
		return
	}
	JSONResponse(w, models.GetPlatformCertifications(), http.StatusOK)
}

// PlatformCertification returns a single platform certification by slug.
// GET /api/compliance/platform-certifications/{slug}
func (as *Server) PlatformCertification(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	slug := vars["slug"]

	cert := models.GetPlatformCertification(slug)
	if cert == nil {
		JSONResponse(w, models.Response{Success: false, Message: "Certification not found"}, http.StatusNotFound)
		return
	}
	JSONResponse(w, cert, http.StatusOK)
}

// PlatformSecurityPosture returns the full platform security posture summary.
// GET /api/compliance/platform-security-posture
func (as *Server) PlatformSecurityPosture(w http.ResponseWriter, r *http.Request) {
	posture := models.GetPlatformSecurityPosture()
	JSONResponse(w, posture, http.StatusOK)
}

// PlatformComplianceSupport returns how the platform supports each framework.
// GET /api/compliance/platform-support
func (as *Server) PlatformComplianceSupport(w http.ResponseWriter, r *http.Request) {
	support := models.GetPlatformComplianceSupport()
	JSONResponse(w, support, http.StatusOK)
}
