package worker

import (
	"bytes"
	"fmt"
	"html/template"
	"time"

	"github.com/gophish/gomail"
	log "github.com/gophish/gophish/logger"
	"github.com/gophish/gophish/models"
)

// ReminderCheckInterval is how often the reminder worker runs.
const ReminderCheckInterval = 1 * time.Hour

// ReminderHoursBeforeDue is how many hours before due date to send the first reminder.
const ReminderHoursBeforeDue = 48

// StartReminderWorker launches a goroutine that periodically checks for
// assignments approaching their due date and sends automated reminder emails.
func StartReminderWorker() {
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("Reminder Worker: recovered from panic: %v", r)
		}
	}()
	log.Info("Reminder Worker Started - checking every hour for pending reminders")

	// Initial run 3 minutes after startup
	time.AfterFunc(3*time.Minute, func() {
		defer func() {
			if r := recover(); r != nil {
				log.Errorf("Reminder Worker: recovered from panic in initial run: %v", r)
			}
		}()
		log.Info("Reminder Worker: running initial reminder check")
		processReminders()
	})

	for range time.Tick(ReminderCheckInterval) {
		func() {
			defer func() {
				if r := recover(); r != nil {
					log.Errorf("Reminder Worker: recovered from panic in cycle: %v", r)
				}
			}()
			processReminders()
		}()
	}
}

func processReminders() {
	// 1. Mark overdue assignments
	overdue, err := models.MarkOverdueAssignments()
	if err != nil {
		log.Errorf("Reminder Worker: error marking overdue: %v", err)
	} else if overdue > 0 {
		log.Infof("Reminder Worker: marked %d assignments as overdue", overdue)
	}

	// 2. Find assignments approaching due date that need reminders
	pending, err := models.GetPendingReminderAssignments(ReminderHoursBeforeDue)
	if err != nil {
		log.Errorf("Reminder Worker: error fetching pending reminders: %v", err)
		return
	}

	if len(pending) == 0 {
		return
	}

	log.Infof("Reminder Worker: found %d assignments needing reminders", len(pending))

	sent := 0
	for _, a := range pending {
		if err := sendTrainingReminder(a); err != nil {
			log.Errorf("Reminder Worker: failed to send reminder for assignment %d: %v", a.Id, err)
			continue
		}
		// Mark reminder as sent
		if err := models.MarkReminderSent(a.Id); err != nil {
			log.Errorf("Reminder Worker: failed to mark reminder sent for assignment %d: %v", a.Id, err)
		}
		sent++
	}

	if sent > 0 {
		log.Infof("Reminder Worker: sent %d training reminders", sent)
	}

	// 3. Process escalations for overdue high-priority assignments
	processEscalations()
}

// sendTrainingReminder sends an actual reminder email to the user via the
// org's configured SMTP sending profile, and records the reminder in the database.
// Falls back to a DB-only record if no SMTP profile is configured.
func sendTrainingReminder(a models.CourseAssignment) error {
	user, err := models.GetUser(a.UserId)
	if err != nil {
		return fmt.Errorf("user %d not found: %w", a.UserId, err)
	}

	// Get the presentation name for a useful reminder
	scope := models.OrgScope{OrgId: user.OrgId, UserId: user.Id}
	tp, err := models.GetTrainingPresentation(a.PresentationId, scope)
	if err != nil {
		return fmt.Errorf("presentation %d not found: %w", a.PresentationId, err)
	}

	hoursUntilDue := time.Until(a.DueDate).Hours()
	reminderMsg := buildReminderMessage(user, tp.Name, a.DueDate, hoursUntilDue)
	rType := reminderType(hoursUntilDue)

	// Create notification record
	notification := &models.TrainingReminder{
		UserId:         user.Id,
		AssignmentId:   a.Id,
		PresentationId: a.PresentationId,
		CourseName:     tp.Name,
		DueDate:        a.DueDate,
		ReminderType:   rType,
		Message:        reminderMsg,
		SentDate:       time.Now().UTC(),
	}

	// Attempt to send the email via SMTP
	emailSent := false
	cfg := models.GetReminderConfig(user.OrgId)
	if cfg.Enabled && cfg.SendingProfileId > 0 {
		smtpProfile, smtpErr := models.GetSMTP(cfg.SendingProfileId, models.OrgScope{OrgId: user.OrgId, IsSuperAdmin: true})
		if smtpErr == nil {
			emailErr := sendReminderEmail(smtpProfile, user, tp.Name, a.DueDate, rType, hoursUntilDue, cfg.EmailTemplate)
			if emailErr != nil {
				log.Errorf("Reminder Worker: SMTP send failed for user %d: %v", user.Id, emailErr)
			} else {
				emailSent = true
			}
		} else {
			log.Warnf("Reminder Worker: sending profile %d not found for org %d: %v", cfg.SendingProfileId, user.OrgId, smtpErr)
		}
	}
	notification.EmailSent = emailSent

	return models.CreateTrainingReminder(notification)
}

// sendReminderEmail sends the actual SMTP email for a training reminder.
func sendReminderEmail(smtp models.SMTP, user models.User, courseName string, dueDate time.Time, rType string, hoursUntilDue float64, customTemplate string) error {
	dialer, err := smtp.GetDialer()
	if err != nil {
		return fmt.Errorf("failed to create SMTP dialer: %w", err)
	}

	sender, err := dialer.Dial()
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP: %w", err)
	}
	defer sender.Close()

	subject := buildReminderSubject(courseName, rType)
	htmlBody := buildReminderHTML(user, courseName, dueDate, rType, hoursUntilDue, customTemplate)

	m := gomail.NewMessage()
	m.SetHeader("From", smtp.FromAddress)
	m.SetHeader("To", user.Email)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", htmlBody)
	m.SetBody("text/plain", buildReminderMessage(user, courseName, dueDate, hoursUntilDue))

	// Add custom SMTP headers
	for _, h := range smtp.Headers {
		m.SetHeader(h.Key, h.Value)
	}

	return gomail.Send(sender, m)
}

// buildReminderSubject creates the email subject based on urgency.
func buildReminderSubject(courseName, rType string) string {
	switch rType {
	case "urgent":
		return fmt.Sprintf("⚠️ URGENT: \"%s\" is due in less than 4 hours", courseName)
	case "final":
		return fmt.Sprintf("🔔 Final Reminder: \"%s\" is due tomorrow", courseName)
	default:
		return fmt.Sprintf("📋 Training Reminder: \"%s\" is approaching its deadline", courseName)
	}
}

// buildReminderHTML constructs the HTML email body for the reminder.
func buildReminderHTML(user models.User, courseName string, dueDate time.Time, rType string, hoursUntilDue float64, customTemplate string) string {
	if customTemplate != "" {
		return renderCustomTemplate(customTemplate, user, courseName, dueDate, hoursUntilDue)
	}

	name := user.FirstName
	if name == "" {
		name = "Team Member"
	}
	timeLeft := formatTimeLeft(hoursUntilDue)
	dueDateStr := dueDate.Format("Monday, January 2, 2006 at 15:04 UTC")

	urgencyColor := "#4A90D9" // standard blue
	urgencyLabel := "Reminder"
	switch rType {
	case "urgent":
		urgencyColor = "#E74C3C"
		urgencyLabel = "Urgent Reminder"
	case "final":
		urgencyColor = "#F39C12"
		urgencyLabel = "Final Reminder"
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="utf-8"></head>
<body style="margin:0;padding:0;font-family:Arial,Helvetica,sans-serif;background:#f4f4f4;">
<table width="100%%" cellpadding="0" cellspacing="0" style="max-width:600px;margin:0 auto;">
<tr><td style="background:%s;padding:20px;text-align:center;color:#fff;">
<h2 style="margin:0;font-size:20px;">🎓 %s</h2>
</td></tr>
<tr><td style="background:#fff;padding:30px;border:1px solid #e0e0e0;">
<p style="font-size:15px;">Hi %s,</p>
<p style="font-size:14px;color:#333;">Your training course <strong>"%s"</strong> is due in <strong>%s</strong>.</p>
<table style="width:100%%;background:#f8f9fa;border-left:4px solid %s;margin:20px 0;" cellpadding="12">
<tr><td style="font-size:13px;color:#555;">
<strong>Course:</strong> %s<br>
<strong>Deadline:</strong> %s<br>
<strong>Time Remaining:</strong> %s
</td></tr>
</table>
<p style="font-size:14px;color:#333;">Please complete this training before the deadline to stay compliant with your organization's security policies.</p>
<p style="text-align:center;margin:25px 0;">
<a href="#" style="background:%s;color:#fff;padding:12px 30px;text-decoration:none;border-radius:4px;font-weight:bold;font-size:14px;">Complete Training Now</a>
</p>
<p style="font-size:12px;color:#999;">If you have already completed this training, please disregard this message.</p>
</td></tr>
<tr><td style="padding:15px;text-align:center;font-size:11px;color:#aaa;">
This is an automated reminder from your organization's security awareness platform.
</td></tr>
</table>
</body>
</html>`, urgencyColor, urgencyLabel, name, courseName, timeLeft,
		urgencyColor, courseName, dueDateStr, timeLeft, urgencyColor)
}

// renderCustomTemplate processes a custom email template with user variables.
func renderCustomTemplate(tmplStr string, user models.User, courseName string, dueDate time.Time, hoursUntilDue float64) string {
	data := map[string]interface{}{
		"FirstName":  user.FirstName,
		"LastName":   user.LastName,
		"Email":      user.Email,
		"CourseName": courseName,
		"DueDate":    dueDate.Format("January 2, 2006 15:04 UTC"),
		"TimeLeft":   formatTimeLeft(hoursUntilDue),
	}
	tmpl, err := template.New("reminder").Parse(tmplStr)
	if err != nil {
		log.Errorf("Reminder Worker: custom template parse error: %v", err)
		return tmplStr
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		log.Errorf("Reminder Worker: custom template execute error: %v", err)
		return tmplStr
	}
	return buf.String()
}

// reminderType determines the reminder urgency based on hours until due.
func reminderType(hoursUntilDue float64) string {
	switch {
	case hoursUntilDue <= 4:
		return "urgent"
	case hoursUntilDue <= 24:
		return "final"
	default:
		return "standard"
	}
}

// buildReminderMessage constructs the reminder message text.
func buildReminderMessage(user models.User, courseName string, dueDate time.Time, hoursUntilDue float64) string {
	greeting := fmt.Sprintf("Hi %s", user.FirstName)
	if user.FirstName == "" {
		greeting = "Hi"
	}

	timeLeft := formatTimeLeft(hoursUntilDue)

	return fmt.Sprintf(
		"%s, your training course \"%s\" is due in %s (deadline: %s). "+
			"Please complete it as soon as possible to stay compliant.",
		greeting, courseName, timeLeft, dueDate.Format("Jan 2, 2006 15:04 UTC"),
	)
}

// formatTimeLeft formats hours into a human-readable duration string.
func formatTimeLeft(hours float64) string {
	if hours < 1 {
		return "less than 1 hour"
	}
	if hours < 24 {
		return fmt.Sprintf("%.0f hours", hours)
	}
	days := hours / 24
	if days < 2 {
		return "1 day"
	}
	return fmt.Sprintf("%.0f days", days)
}

// processEscalations handles overdue high/critical assignments that need escalation.
func processEscalations() {
	overdueAssignments, err := models.GetOverdueAssignments()
	if err != nil {
		log.Errorf("Reminder Worker: error fetching overdue assignments: %v", err)
		return
	}

	escalated := 0
	for _, a := range overdueAssignments {
		// Only escalate high/critical priority that haven't been escalated yet
		if a.EscalatedTo > 0 {
			continue
		}
		if a.Priority != models.AssignmentPriorityHigh && a.Priority != models.AssignmentPriorityCritical {
			continue
		}

		// Escalate to the user who assigned the training (the manager)
		if a.AssignedBy > 0 {
			if err := models.EscalateAssignment(a.Id, a.AssignedBy); err != nil {
				log.Errorf("Reminder Worker: failed to escalate assignment %d: %v", a.Id, err)
				continue
			}
			escalated++
		}
	}

	if escalated > 0 {
		log.Infof("Reminder Worker: escalated %d overdue high-priority assignments", escalated)
	}
}
