package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	ctx "github.com/gophish/gophish/context"
	log "github.com/gophish/gophish/logger"
	"github.com/gophish/gophish/models"
	"github.com/gorilla/mux"
)

const msgMethodNotAllowed = ErrMethodNotAllowed

// SandboxTests handles GET (list) and POST (create + send) for /api/sandbox/.
func (as *Server) SandboxTests(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)

	switch r.Method {
	case http.MethodGet:
		tests, err := models.GetSandboxTests(user.OrgId)
		if err != nil {
			log.Error(err)
			JSONResponse(w, models.Response{Success: false, Message: "Error fetching sandbox tests"}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, tests, http.StatusOK)

	case http.MethodPost:
		var req struct {
			TemplateId int64  `json:"template_id"`
			SmtpId     int64  `json:"smtp_id"`
			ToEmail    string `json:"to_email"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: ErrInvalidJSON}, http.StatusBadRequest)
			return
		}

		test := &models.SandboxTest{
			OrgId:      user.OrgId,
			CreatedBy:  user.Id,
			TemplateId: req.TemplateId,
			SmtpId:     req.SmtpId,
			ToEmail:    req.ToEmail,
		}
		if err := test.Validate(); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusBadRequest)
			return
		}

		scope := models.OrgScope{OrgId: user.OrgId, UserId: user.Id}

		// Load template and SMTP
		tmpl, err := models.GetTemplate(req.TemplateId, scope)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: "Template not found"}, http.StatusNotFound)
			return
		}
		smtp, err := models.GetSMTP(req.SmtpId, scope)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: "Sending profile not found"}, http.StatusNotFound)
			return
		}

		test.Subject = tmpl.Subject
		test.SmtpName = smtp.Name
		test.TemplateName = tmpl.Name

		// Render HTML for preview (best-effort; no tracking URL in sandbox)
		fromAddr := tmpl.EnvelopeSender
		if fromAddr == "" {
			fromAddr = smtp.FromAddress
		}
		test.RenderedHTML = models.RenderSandboxHTML(tmpl, fromAddr)

		// Save the record before sending
		if err := models.PostSandboxTest(test); err != nil {
			log.Error(err)
			JSONResponse(w, models.Response{Success: false, Message: "Error saving sandbox test"}, http.StatusInternalServerError)
			return
		}

		// Send the test email asynchronously so the HTTP response returns quickly.
		go as.dispatchSandboxEmail(test.Id, test.OrgId, &tmpl, &smtp, req.ToEmail, fromAddr)

		// Reload to return with hydrated names
		created, _ := models.GetSandboxTest(test.Id, user.OrgId)
		JSONResponse(w, created, http.StatusCreated)

	default:
		JSONResponse(w, models.Response{Success: false, Message: msgMethodNotAllowed}, http.StatusMethodNotAllowed)
	}
}

// SandboxTest handles GET, DELETE for /api/sandbox/{id}.
func (as *Server) SandboxTest(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)
	vars := mux.Vars(r)
	id, _ := strconv.ParseInt(vars["id"], 10, 64)

	test, err := models.GetSandboxTest(id, user.OrgId)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "Sandbox test not found"}, http.StatusNotFound)
		return
	}

	switch r.Method {
	case http.MethodGet:
		JSONResponse(w, test, http.StatusOK)

	case http.MethodDelete:
		if err := models.DeleteSandboxTest(id, user.OrgId); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: "Error deleting sandbox test"}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, models.Response{Success: true, Message: "Sandbox test deleted"}, http.StatusOK)

	default:
		JSONResponse(w, models.Response{Success: false, Message: msgMethodNotAllowed}, http.StatusMethodNotAllowed)
	}
}

// SandboxTestReview handles POST /api/sandbox/{id}/review — approve or reject.
func (as *Server) SandboxTestReview(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)
	vars := mux.Vars(r)
	id, _ := strconv.ParseInt(vars["id"], 10, 64)

	if r.Method != http.MethodPost {
		JSONResponse(w, models.Response{Success: false, Message: msgMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}

	// Confirm the sandbox test belongs to the org
	if _, err := models.GetSandboxTest(id, user.OrgId); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "Sandbox test not found"}, http.StatusNotFound)
		return
	}

	var req struct {
		Status string `json:"status"` // "approved" or "rejected"
		Notes  string `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: ErrInvalidJSON}, http.StatusBadRequest)
		return
	}
	if req.Status != models.SandboxStatusApproved && req.Status != models.SandboxStatusRejected {
		JSONResponse(w, models.Response{Success: false, Message: "status must be 'approved' or 'rejected'"}, http.StatusBadRequest)
		return
	}

	if err := models.ReviewSandboxTest(id, user.Id, req.Status, req.Notes); err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Error saving review"}, http.StatusInternalServerError)
		return
	}

	updated, _ := models.GetSandboxTest(id, user.OrgId)
	JSONResponse(w, updated, http.StatusOK)
}

// dispatchSandboxEmail sends the test email and updates the sandbox record.
// Runs in a goroutine so the HTTP handler can return immediately.
func (as *Server) dispatchSandboxEmail(testId, orgId int64, tmpl *models.Template, smtp *models.SMTP, toEmail, fromAddr string) {
	errChan := make(chan error, 1)
	req := &models.EmailRequest{
		Template:  *tmpl,
		SMTP:      *smtp,
		ErrorChan: errChan,
		FromAddress: fromAddr,
		BaseRecipient: models.BaseRecipient{
			Email:     toEmail,
			FirstName: "Sandbox",
			LastName:  "Test",
		},
		RId: "sandbox-" + strconv.FormatInt(testId, 10),
	}

	// Mark as sending
	models.UpdateSandboxTestStatus(testId, models.SandboxStatusSending, "", "", time.Time{})

	sendErr := as.worker.SendTestEmail(req)
	if sendErr != nil {
		log.Errorf("ZIM sandbox send error (test %d): %v", testId, sendErr)
		models.UpdateSandboxTestStatus(testId, models.SandboxStatusFailed, "", sendErr.Error(), time.Time{})
		return
	}
	models.UpdateSandboxTestStatus(testId, models.SandboxStatusDelivered, "", "", time.Now().UTC())
}

