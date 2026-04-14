package worker

import (
	"strings"
	"testing"
	"time"

	"github.com/gophish/gophish/models"
)

// ─── reminderType tests ───

func TestReminderTypeUrgent(t *testing.T) {
	if got := reminderType(3.5); got != "urgent" {
		t.Fatalf("expected 'urgent', got %q", got)
	}
}

func TestReminderTypeFinal(t *testing.T) {
	if got := reminderType(12.0); got != "final" {
		t.Fatalf("expected 'final', got %q", got)
	}
}

func TestReminderTypeStandard(t *testing.T) {
	if got := reminderType(36.0); got != "standard" {
		t.Fatalf("expected 'standard', got %q", got)
	}
}

func TestReminderTypeBoundary4h(t *testing.T) {
	if got := reminderType(4.0); got != "urgent" {
		t.Fatalf("expected 'urgent' at boundary 4h, got %q", got)
	}
}

func TestReminderTypeBoundary24h(t *testing.T) {
	if got := reminderType(24.0); got != "final" {
		t.Fatalf("expected 'final' at boundary 24h, got %q", got)
	}
}

func TestReminderTypeBoundary25h(t *testing.T) {
	if got := reminderType(25.0); got != "standard" {
		t.Fatalf("expected 'standard' at 25h, got %q", got)
	}
}

// ─── formatTimeLeft tests ───

func TestFormatTimeLeftLessThanOneHour(t *testing.T) {
	if got := formatTimeLeft(0.5); got != "less than 1 hour" {
		t.Fatalf("expected 'less than 1 hour', got %q", got)
	}
}

func TestFormatTimeLeftHours(t *testing.T) {
	if got := formatTimeLeft(10.0); got != "10 hours" {
		t.Fatalf("expected '10 hours', got %q", got)
	}
}

func TestFormatTimeLeftOneDay(t *testing.T) {
	if got := formatTimeLeft(30.0); got != "1 day" {
		t.Fatalf("expected '1 day', got %q", got)
	}
}

func TestFormatTimeLeftMultipleDays(t *testing.T) {
	if got := formatTimeLeft(96.0); got != "4 days" {
		t.Fatalf("expected '4 days', got %q", got)
	}
}

func TestFormatTimeLeftZero(t *testing.T) {
	if got := formatTimeLeft(0); got != "less than 1 hour" {
		t.Fatalf("expected 'less than 1 hour', got %q", got)
	}
}

// ─── buildReminderSubject tests ───

func TestBuildReminderSubjectUrgent(t *testing.T) {
	s := buildReminderSubject("Phishing 101", "urgent")
	if !strings.Contains(s, "URGENT") || !strings.Contains(s, "Phishing 101") {
		t.Fatalf("unexpected subject: %q", s)
	}
}

func TestBuildReminderSubjectFinal(t *testing.T) {
	s := buildReminderSubject("Safety", "final")
	if !strings.Contains(s, "Final Reminder") {
		t.Fatalf("expected 'Final Reminder', got %q", s)
	}
}

func TestBuildReminderSubjectStandard(t *testing.T) {
	s := buildReminderSubject("Basics", "standard")
	if !strings.Contains(s, "Training Reminder") {
		t.Fatalf("expected 'Training Reminder', got %q", s)
	}
}

// ─── buildReminderMessage tests ───

func TestBuildReminderMessageWithName(t *testing.T) {
	user := models.User{FirstName: "Alice"}
	due := time.Now().Add(24 * time.Hour)
	msg := buildReminderMessage(user, "Course A", due, 24)
	if !strings.Contains(msg, "Hi Alice") || !strings.Contains(msg, "Course A") {
		t.Fatalf("unexpected message: %q", msg)
	}
}

func TestBuildReminderMessageWithoutName(t *testing.T) {
	user := models.User{FirstName: ""}
	due := time.Now().Add(24 * time.Hour)
	msg := buildReminderMessage(user, "Course B", due, 24)
	if !strings.HasPrefix(msg, "Hi,") {
		t.Fatalf("expected 'Hi,' prefix for unnamed user, got %q", msg)
	}
}

// ─── buildReminderHTML tests ───

func TestBuildReminderHTMLDefault(t *testing.T) {
	user := models.User{FirstName: "Bob"}
	due := time.Now().Add(48 * time.Hour)
	html := buildReminderHTML(user, "Safety 101", due, "standard", 48, "")
	if !strings.Contains(html, "Bob") || !strings.Contains(html, "Safety 101") || !strings.Contains(html, "#4A90D9") {
		t.Fatal("unexpected default HTML output")
	}
}

func TestBuildReminderHTMLUrgent(t *testing.T) {
	user := models.User{FirstName: "Charlie"}
	due := time.Now().Add(2 * time.Hour)
	html := buildReminderHTML(user, "Urgent", due, "urgent", 2, "")
	if !strings.Contains(html, "#E74C3C") || !strings.Contains(html, "Urgent Reminder") {
		t.Fatal("unexpected urgent HTML output")
	}
}

func TestBuildReminderHTMLFinal(t *testing.T) {
	user := models.User{FirstName: "Diana"}
	due := time.Now().Add(20 * time.Hour)
	html := buildReminderHTML(user, "Final", due, "final", 20, "")
	if !strings.Contains(html, "#F39C12") || !strings.Contains(html, "Final Reminder") {
		t.Fatal("unexpected final HTML output")
	}
}

func TestBuildReminderHTMLEmptyName(t *testing.T) {
	user := models.User{FirstName: ""}
	due := time.Now().Add(48 * time.Hour)
	html := buildReminderHTML(user, "X", due, "standard", 48, "")
	if !strings.Contains(html, "Team Member") {
		t.Fatal("expected 'Team Member' fallback")
	}
}

// ─── renderCustomTemplate tests ───

func TestRenderCustomTemplateValid(t *testing.T) {
	user := models.User{FirstName: "Eve", LastName: "Smith", Email: "eve@example.com"}
	due := time.Now().Add(48 * time.Hour)
	tmpl := "Hello {{.FirstName}} {{.LastName}}, complete {{.CourseName}}"
	result := renderCustomTemplate(tmpl, user, "Custom", due, 48)
	if !strings.Contains(result, "Eve Smith") || !strings.Contains(result, "Custom") {
		t.Fatalf("unexpected result: %q", result)
	}
}

func TestRenderCustomTemplateInvalid(t *testing.T) {
	user := models.User{FirstName: "Eve"}
	due := time.Now().Add(48 * time.Hour)
	tmpl := "Hello {{.Invalid"
	result := renderCustomTemplate(tmpl, user, "Course", due, 48)
	if result != tmpl {
		t.Fatalf("expected raw template on parse error, got %q", result)
	}
}

func TestRenderCustomTemplateTimeLeft(t *testing.T) {
	user := models.User{FirstName: "Eve"}
	due := time.Now().Add(48 * time.Hour)
	result := renderCustomTemplate("{{.TimeLeft}} remaining", user, "Course", due, 48)
	if !strings.Contains(result, "2 days") {
		t.Fatalf("expected '2 days remaining', got %q", result)
	}
}

// ─── Constants tests ───

func TestReminderCheckInterval(t *testing.T) {
	if ReminderCheckInterval != 1*time.Hour {
		t.Fatalf("expected 1h, got %v", ReminderCheckInterval)
	}
}

func TestReminderHoursBeforeDue(t *testing.T) {
	if ReminderHoursBeforeDue != 48 {
		t.Fatalf("expected 48, got %d", ReminderHoursBeforeDue)
	}
}
