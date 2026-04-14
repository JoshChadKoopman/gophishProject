package models

import (
	"testing"
)

func TestTemplateValidateMissingName(t *testing.T) {
	tmpl := Template{
		Text: "Some text content",
	}
	err := tmpl.Validate()
	if err != ErrTemplateNameNotSpecified {
		t.Errorf("expected ErrTemplateNameNotSpecified, got %v", err)
	}
}

func TestTemplateValidateMissingContent(t *testing.T) {
	tmpl := Template{
		Name: "Test Template",
	}
	err := tmpl.Validate()
	if err != ErrTemplateMissingParameter {
		t.Errorf("expected ErrTemplateMissingParameter, got %v", err)
	}
}

func TestTemplateValidateValidText(t *testing.T) {
	tmpl := Template{
		Name: "Test Template",
		Text: "Hello {{.FirstName}}",
	}
	err := tmpl.Validate()
	if err != nil {
		t.Errorf("expected no error for valid text template, got %v", err)
	}
}

func TestTemplateValidateValidHTML(t *testing.T) {
	tmpl := Template{
		Name: "Test Template",
		HTML: "<html><body>{{.FirstName}}</body></html>",
	}
	err := tmpl.Validate()
	if err != nil {
		t.Errorf("expected no error for valid HTML template, got %v", err)
	}
}

func TestTemplateValidateBothTextAndHTML(t *testing.T) {
	tmpl := Template{
		Name: "Test Template",
		Text: "Plain text version",
		HTML: "<html><body>HTML version</body></html>",
	}
	err := tmpl.Validate()
	if err != nil {
		t.Errorf("expected no error when both text and HTML provided, got %v", err)
	}
}

func TestTemplateValidateInvalidEnvelopeSender(t *testing.T) {
	tmpl := Template{
		Name:           "Test Template",
		Text:           "Some content",
		EnvelopeSender: "not-a-valid-email",
	}
	err := tmpl.Validate()
	if err == nil {
		t.Error("expected error for invalid envelope sender")
	}
}

func TestTemplateValidateValidEnvelopeSender(t *testing.T) {
	tmpl := Template{
		Name:           "Test Template",
		Text:           "Some content",
		EnvelopeSender: "sender@example.com",
	}
	err := tmpl.Validate()
	if err != nil {
		t.Errorf("expected no error for valid envelope sender, got %v", err)
	}
}

func TestTemplateErrorMessages(t *testing.T) {
	if ErrTemplateNameNotSpecified.Error() != "Template name not specified" {
		t.Errorf("unexpected error message: %s", ErrTemplateNameNotSpecified.Error())
	}
	if ErrTemplateMissingParameter.Error() != "Need to specify at least plaintext or HTML content" {
		t.Errorf("unexpected error message: %s", ErrTemplateMissingParameter.Error())
	}
}

func TestTemplateDefaults(t *testing.T) {
	tmpl := Template{}
	if tmpl.Id != 0 || tmpl.UserId != 0 || tmpl.OrgId != 0 {
		t.Error("default template should have zero IDs")
	}
	if tmpl.AIGenerated != false {
		t.Error("default template should not be AI generated")
	}
	if tmpl.DifficultyLevel != 0 {
		t.Error("default template should have zero difficulty")
	}
	if tmpl.Attachments != nil {
		t.Error("default template should have nil attachments")
	}
}
