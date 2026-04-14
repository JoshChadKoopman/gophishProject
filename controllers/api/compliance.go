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

// ComplianceFrameworks handles GET /api/compliance/frameworks — list all available frameworks.
func (as *Server) ComplianceFrameworks(w http.ResponseWriter, r *http.Request) {
	frameworks, err := models.GetComplianceFrameworks()
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Error fetching frameworks"}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, frameworks, http.StatusOK)
}

// ComplianceOrgFrameworks handles GET/POST /api/compliance/org-frameworks.
// GET returns frameworks enabled for the user's org.
// POST enables a framework for the org.
func (as *Server) ComplianceOrgFrameworks(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)
	orgId := user.OrgId

	switch r.Method {
	case http.MethodGet:
		frameworks, err := models.GetOrgFrameworks(orgId)
		if err != nil {
			log.Error(err)
			JSONResponse(w, models.Response{Success: false, Message: "Error fetching org frameworks"}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, frameworks, http.StatusOK)

	case http.MethodPost:
		var req struct {
			FrameworkId int64 `json:"framework_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: "Invalid request"}, http.StatusBadRequest)
			return
		}
		if err := models.EnableOrgFramework(orgId, req.FrameworkId); err != nil {
			log.Error(err)
			JSONResponse(w, models.Response{Success: false, Message: "Error enabling framework"}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, models.Response{Success: true, Message: "Framework enabled"}, http.StatusOK)
	}
}

// ComplianceOrgFrameworkDisable handles POST /api/compliance/org-frameworks/{id}/disable.
func (as *Server) ComplianceOrgFrameworkDisable(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)
	vars := mux.Vars(r)
	frameworkId, _ := strconv.ParseInt(vars["id"], 0, 64)

	if err := models.DisableOrgFramework(user.OrgId, frameworkId); err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Error disabling framework"}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, models.Response{Success: true, Message: "Framework disabled"}, http.StatusOK)
}

// ComplianceDashboard handles GET /api/compliance/dashboard — overall compliance posture.
func (as *Server) ComplianceDashboard(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)
	dashboard, err := models.GetComplianceDashboard(user.OrgId)
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Error fetching compliance dashboard"}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, dashboard, http.StatusOK)
}

// ComplianceFrameworkDetail handles GET /api/compliance/frameworks/{id}/detail.
func (as *Server) ComplianceFrameworkDetail(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)
	vars := mux.Vars(r)
	frameworkId, _ := strconv.ParseInt(vars["id"], 0, 64)

	summary, err := models.GetFrameworkSummary(user.OrgId, frameworkId, true)
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Error fetching framework detail"}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, summary, http.StatusOK)
}

// ComplianceAssess handles POST /api/compliance/frameworks/{id}/assess — runs auto-assessment.
func (as *Server) ComplianceAssess(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)
	vars := mux.Vars(r)
	frameworkId, _ := strconv.ParseInt(vars["id"], 0, 64)

	summary, err := models.AutoAssessFramework(user.OrgId, frameworkId)
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Error running assessment"}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, summary, http.StatusOK)
}

// ComplianceManualAssess handles POST /api/compliance/controls/{id}/assess — manual assessment.
func (as *Server) ComplianceManualAssess(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)
	vars := mux.Vars(r)
	controlId, _ := strconv.ParseInt(vars["id"], 0, 64)

	var req struct {
		Status string `json:"status"`
		Score  float64 `json:"score"`
		Notes  string `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "Invalid request"}, http.StatusBadRequest)
		return
	}

	// Look up the control to get framework_id
	control := models.ComplianceControl{}
	controls, _ := models.GetComplianceFrameworks()
	_ = controls // We need to find the control differently
	// Query the control directly
	var found bool
	frameworks, _ := models.GetComplianceFrameworks()
	for _, f := range frameworks {
		ctrls, _ := models.GetFrameworkControls(f.Id)
		for _, c := range ctrls {
			if c.Id == controlId {
				control = c
				found = true
				break
			}
		}
		if found {
			break
		}
	}

	if !found {
		JSONResponse(w, models.Response{Success: false, Message: "Control not found"}, http.StatusNotFound)
		return
	}

	assessment := &models.ComplianceAssessment{
		OrgId:       user.OrgId,
		FrameworkId: control.FrameworkId,
		ControlId:   controlId,
		Status:      req.Status,
		Score:       req.Score,
		Evidence:    "Manual assessment",
		AssessedBy:  user.Id,
		Notes:       req.Notes,
	}

	if err := models.SaveComplianceAssessment(assessment); err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Error saving assessment"}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, assessment, http.StatusOK)
}
