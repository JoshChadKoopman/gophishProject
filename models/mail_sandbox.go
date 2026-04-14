package models

import (
	"errors"
	"time"
)

// ZIM (Zero Incident Mail) Sandbox — lets admins test phishing templates
// in a safe, sandboxed environment before launching live campaigns.
// The sandbox sends a real email to a designated test address and records
// the full rendered HTML so admins can review it before approving.

const (
	SandboxStatusPending   = "pending"
	SandboxStatusSending   = "sending"
	SandboxStatusDelivered = "delivered"
	SandboxStatusFailed    = "failed"
	SandboxStatusApproved  = "approved"
	SandboxStatusRejected  = "rejected"
)

var ErrSandboxNotFound = errors.New("sandbox test not found")

// SandboxTest represents a single sandbox test run for a phishing template.
type SandboxTest struct {
	Id           int64     `json:"id" gorm:"column:id; primary_key:yes"`
	OrgId        int64     `json:"org_id" gorm:"column:org_id"`
	CreatedBy    int64     `json:"created_by" gorm:"column:created_by"`
	TemplateId   int64     `json:"template_id" gorm:"column:template_id"`
	TemplateName string    `json:"template_name" gorm:"-"`
	SmtpId       int64     `json:"smtp_id" gorm:"column:smtp_id"`
	SmtpName     string    `json:"smtp_name" gorm:"-"`
	ToEmail      string    `json:"to_email" gorm:"column:to_email"`
	Subject      string    `json:"subject" gorm:"column:subject"`
	RenderedHTML string    `json:"rendered_html" gorm:"column:rendered_html"`
	Status       string    `json:"status" gorm:"column:status"`
	ErrorMsg     string    `json:"error_msg,omitempty" gorm:"column:error_msg"`
	Notes        string    `json:"notes,omitempty" gorm:"column:notes"`
	SentAt       time.Time `json:"sent_at,omitempty" gorm:"column:sent_at"`
	ReviewedAt   time.Time `json:"reviewed_at,omitempty" gorm:"column:reviewed_at"`
	ReviewedBy   int64     `json:"reviewed_by,omitempty" gorm:"column:reviewed_by"`
	CreatedDate  time.Time `json:"created_date" gorm:"column:created_date"`
	ModifiedDate time.Time `json:"modified_date" gorm:"column:modified_date"`
}

// TableName sets the table name for GORM.
func (s *SandboxTest) TableName() string {
	return "sandbox_tests"
}

// sandboxTemplateContext is a minimal TemplateContext for sandbox preview rendering.
// It lives in the models package so it can satisfy the unexported TemplateContext interface.
type sandboxTemplateContext struct {
	fromAddr string
}

func (s *sandboxTemplateContext) getFromAddress() string { return s.fromAddr }
func (s *sandboxTemplateContext) getBaseURL() string     { return "https://sandbox.preview" }
func (s *sandboxTemplateContext) getOrgId() int64        { return 0 }

// RenderSandboxHTML renders a template's HTML using a placeholder recipient,
// suitable for the in-app preview in the ZIM sandbox.
func RenderSandboxHTML(tmpl Template, fromAddr string) string {
	if tmpl.HTML == "" {
		return ""
	}
	placeholder := BaseRecipient{
		Email:     "sandbox@preview.local",
		FirstName: "Sandbox",
		LastName:  "User",
	}
	ptc, err := NewPhishingTemplateContext(&sandboxTemplateContext{fromAddr: fromAddr}, placeholder, "sandbox-preview")
	if err != nil {
		return tmpl.HTML
	}
	rendered, err := ExecuteTemplate(tmpl.HTML, ptc)
	if err != nil {
		return tmpl.HTML
	}
	return rendered
}

// Validate checks required fields before creating a sandbox test.
func (s *SandboxTest) Validate() error {
	if s.TemplateId == 0 {
		return errors.New("template_id is required")
	}
	if s.SmtpId == 0 {
		return errors.New("smtp_id is required")
	}
	if s.ToEmail == "" {
		return errors.New("to_email is required")
	}
	return nil
}

// GetSandboxTests returns all sandbox tests for an org, newest first.
func GetSandboxTests(orgId int64) ([]SandboxTest, error) {
	tests := []SandboxTest{}
	err := db.Where("org_id = ?", orgId).
		Order("created_date desc").
		Find(&tests).Error
	if err != nil {
		return tests, err
	}
	// Hydrate template/smtp names
	scope := OrgScope{OrgId: orgId}
	for i := range tests {
		if t, err := GetTemplate(tests[i].TemplateId, scope); err == nil {
			tests[i].TemplateName = t.Name
		}
		if s, err := GetSMTP(tests[i].SmtpId, scope); err == nil {
			tests[i].SmtpName = s.Name
		}
	}
	return tests, nil
}

// GetSandboxTest returns a single sandbox test by ID, enforcing org ownership.
func GetSandboxTest(id, orgId int64) (SandboxTest, error) {
	test := SandboxTest{}
	err := db.Where("id = ? AND org_id = ?", id, orgId).First(&test).Error
	if err != nil {
		return test, ErrSandboxNotFound
	}
	scope := OrgScope{OrgId: orgId}
	if t, err := GetTemplate(test.TemplateId, scope); err == nil {
		test.TemplateName = t.Name
	}
	if s, err := GetSMTP(test.SmtpId, scope); err == nil {
		test.SmtpName = s.Name
	}
	return test, nil
}

// PostSandboxTest creates a new sandbox test record (status = pending).
func PostSandboxTest(s *SandboxTest) error {
	s.Status = SandboxStatusPending
	s.CreatedDate = time.Now().UTC()
	s.ModifiedDate = time.Now().UTC()
	return db.Save(s).Error
}

// UpdateSandboxTestStatus updates the status, rendered HTML, and error of a test.
func UpdateSandboxTestStatus(id int64, status, renderedHTML, errorMsg string, sentAt time.Time) error {
	updates := map[string]interface{}{
		"status":        status,
		"rendered_html": renderedHTML,
		"error_msg":     errorMsg,
		"modified_date": time.Now().UTC(),
	}
	if !sentAt.IsZero() {
		updates["sent_at"] = sentAt
	}
	return db.Model(&SandboxTest{}).Where("id = ?", id).Updates(updates).Error
}

// ReviewSandboxTest records an admin's approve/reject decision.
func ReviewSandboxTest(id, reviewerId int64, status, notes string) error {
	return db.Model(&SandboxTest{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":        status,
		"notes":         notes,
		"reviewed_by":   reviewerId,
		"reviewed_at":   time.Now().UTC(),
		"modified_date": time.Now().UTC(),
	}).Error
}

// DeleteSandboxTest removes a sandbox test (org-scoped).
func DeleteSandboxTest(id, orgId int64) error {
	return db.Where("id = ? AND org_id = ?", id, orgId).Delete(&SandboxTest{}).Error
}
