package models

import (
	"encoding/json"
	"time"

	log "github.com/gophish/gophish/logger"
	"github.com/jinzhu/gorm"
)

// PraiseMessage holds configurable praise/feedback messages that are displayed
// when a user completes a training course, passes a quiz, or earns a cert.
// Each organization can customise these messages per event type.
type PraiseMessage struct {
	Id           int64     `json:"id" gorm:"column:id; primary_key:yes"`
	OrgId        int64     `json:"org_id" gorm:"column:org_id"`
	EventType    string    `json:"event_type" gorm:"column:event_type"`     // "course_complete", "quiz_passed", "cert_earned", "tier_complete"
	Heading      string    `json:"heading" gorm:"column:heading"`           // e.g. "Course Complete!"
	Body         string    `json:"body" gorm:"column:body"`                 // e.g. "Congratulations! You finished {{.CourseName}}"
	ButtonText   string    `json:"button_text" gorm:"column:button_text"`   // e.g. "Awesome!"
	Icon         string    `json:"icon" gorm:"column:icon"`                 // emoji or icon class, e.g. "⭐" or "fa-trophy"
	ColorScheme  string    `json:"color_scheme" gorm:"column:color_scheme"` // hex colour for accent, e.g. "#27ae60"
	IsActive     bool      `json:"is_active" gorm:"column:is_active"`
	ModifiedDate time.Time `json:"modified_date" gorm:"column:modified_date"`
}

// PraiseEvent type constants.
const (
	PraiseEventCourseComplete = "course_complete"
	PraiseEventQuizPassed     = "quiz_passed"
	PraiseEventCertEarned     = "cert_earned"
	PraiseEventTierComplete   = "tier_complete"
)

// DefaultPraiseMessages returns the built-in praise messages used when an org
// has not configured custom ones.
func DefaultPraiseMessages() []PraiseMessage {
	return []PraiseMessage{
		{
			EventType:   PraiseEventCourseComplete,
			Heading:     "Course Complete!",
			Body:        "Congratulations! You finished <strong>{{.CourseName}}</strong>",
			ButtonText:  "Awesome!",
			Icon:        "⭐",
			ColorScheme: "#27ae60",
			IsActive:    true,
		},
		{
			EventType:   PraiseEventQuizPassed,
			Heading:     "Quiz Passed!",
			Body:        "Great work! You scored {{.Score}}/{{.Total}} on <strong>{{.CourseName}}</strong>",
			ButtonText:  "Well Done!",
			Icon:        "🏆",
			ColorScheme: "#f39c12",
			IsActive:    true,
		},
		{
			EventType:   PraiseEventCertEarned,
			Heading:     "Certificate Earned!",
			Body:        "You've earned the <strong>{{.CertName}}</strong> certificate. Keep up the great work!",
			ButtonText:  "View Certificate",
			Icon:        "🎓",
			ColorScheme: "#2c3e50",
			IsActive:    true,
		},
		{
			EventType:   PraiseEventTierComplete,
			Heading:     "Tier Completed!",
			Body:        "Outstanding! You've completed the <strong>{{.TierName}}</strong> tier.",
			ButtonText:  "Continue Learning",
			Icon:        "🏅",
			ColorScheme: "#8e44ad",
			IsActive:    true,
		},
	}
}

// GetPraiseMessages returns all praise messages for an org.
// Falls back to system defaults (org_id = 0) if the org has none configured.
func GetPraiseMessages(orgId int64) ([]PraiseMessage, error) {
	msgs := []PraiseMessage{}
	err := db.Where("org_id = ? AND is_active = 1", orgId).Find(&msgs).Error
	if err != nil || len(msgs) == 0 {
		// Try system defaults
		err = db.Where("org_id = 0 AND is_active = 1").Find(&msgs).Error
	}
	if err != nil || len(msgs) == 0 {
		return DefaultPraiseMessages(), nil
	}
	return msgs, nil
}

// GetPraiseMessageByEvent returns the active praise message for a specific
// event type within an org. Falls back to system default, then hardcoded default.
func GetPraiseMessageByEvent(orgId int64, eventType string) PraiseMessage {
	msg := PraiseMessage{}
	err := db.Where("org_id = ? AND event_type = ? AND is_active = 1", orgId, eventType).First(&msg).Error
	if err != nil {
		// Try system-level default
		err = db.Where("org_id = 0 AND event_type = ? AND is_active = 1", eventType).First(&msg).Error
	}
	if err != nil {
		// Return hardcoded default
		for _, d := range DefaultPraiseMessages() {
			if d.EventType == eventType {
				return d
			}
		}
		return PraiseMessage{
			EventType:  eventType,
			Heading:    "Well Done!",
			Body:       "You've completed this activity.",
			ButtonText: "OK",
			Icon:       "⭐",
			IsActive:   true,
		}
	}
	return msg
}

// GetPraiseMessagesRaw returns all praise messages for an org including inactive ones.
// Used for admin management.
func GetPraiseMessagesRaw(orgId int64) ([]PraiseMessage, error) {
	msgs := []PraiseMessage{}
	err := db.Where("org_id = ?", orgId).Order("event_type asc").Find(&msgs).Error
	return msgs, err
}

// SavePraiseMessage creates or updates a praise message.
func SavePraiseMessage(msg *PraiseMessage) error {
	msg.ModifiedDate = time.Now().UTC()
	return db.Save(msg).Error
}

// SavePraiseMessages bulk-saves a set of praise messages for an org.
// It replaces all existing messages for the org's event types that are provided.
func SavePraiseMessages(orgId int64, msgs []PraiseMessage) error {
	tx := db.Begin()
	for _, msg := range msgs {
		msg.OrgId = orgId
		msg.ModifiedDate = time.Now().UTC()
		if err := upsertPraiseMessage(tx, orgId, &msg); err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit().Error
}

// upsertPraiseMessage inserts or updates a single praise message within a transaction.
func upsertPraiseMessage(tx *gorm.DB, orgId int64, msg *PraiseMessage) error {
	if msg.Id > 0 {
		return tx.Save(msg).Error
	}
	existing := PraiseMessage{}
	if err := tx.Where("org_id = ? AND event_type = ?", orgId, msg.EventType).First(&existing).Error; err == nil {
		existing.Heading = msg.Heading
		existing.Body = msg.Body
		existing.ButtonText = msg.ButtonText
		existing.Icon = msg.Icon
		existing.ColorScheme = msg.ColorScheme
		existing.IsActive = msg.IsActive
		existing.ModifiedDate = msg.ModifiedDate
		return tx.Save(&existing).Error
	}
	return tx.Save(msg).Error
}

// DeletePraiseMessage deletes a praise message by ID.
func DeletePraiseMessage(id int64, orgId int64) error {
	return db.Where("id = ? AND org_id = ?", id, orgId).Delete(&PraiseMessage{}).Error
}

// ResetPraiseMessages removes all custom praise messages for an org, reverting to defaults.
func ResetPraiseMessages(orgId int64) error {
	return db.Where("org_id = ?", orgId).Delete(&PraiseMessage{}).Error
}

// PraiseMessageMap returns praise messages as a map keyed by event_type, for
// easy consumption by the frontend.
func PraiseMessageMap(orgId int64) map[string]PraiseMessage {
	msgs, _ := GetPraiseMessages(orgId)
	m := make(map[string]PraiseMessage, len(msgs))
	for _, msg := range msgs {
		m[msg.EventType] = msg
	}
	return m
}

// SeedDefaultPraiseMessages inserts the built-in defaults as org_id=0 rows
// if they don't already exist. Called during DB setup.
func SeedDefaultPraiseMessages() {
	for _, d := range DefaultPraiseMessages() {
		existing := PraiseMessage{}
		if err := db.Where("org_id = 0 AND event_type = ?", d.EventType).First(&existing).Error; err != nil {
			d.OrgId = 0
			d.IsActive = true
			d.ModifiedDate = time.Now().UTC()
			if err := db.Save(&d).Error; err != nil {
				log.Errorf("Failed to seed praise message '%s': %v", d.EventType, err)
			}
		}
	}
}

// ParsePraiseBody replaces template variables in a praise message body.
func ParsePraiseBody(body string, vars map[string]string) string {
	for k, v := range vars {
		body = replaceTemplateVar(body, k, v)
	}
	return body
}

// replaceTemplateVar replaces {{.VarName}} with the value.
func replaceTemplateVar(s, key, val string) string {
	placeholder := "{{." + key + "}}"
	result := s
	for i := 0; i < 10; i++ {
		idx := indexOf(result, placeholder)
		if idx < 0 {
			break
		}
		result = result[:idx] + val + result[idx+len(placeholder):]
	}
	return result
}

func indexOf(s, sub string) int {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

// PraiseMessageJSON returns all praise messages for an org as a JSON string,
// keyed by event type. Intended for embedding in page templates.
func PraiseMessageJSON(orgId int64) string {
	m := PraiseMessageMap(orgId)
	b, err := json.Marshal(m)
	if err != nil {
		return "{}"
	}
	return string(b)
}
