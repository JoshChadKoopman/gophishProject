package models

import (
	"errors"
	"time"

	"github.com/gophish/gophish/ai"
	log "github.com/gophish/gophish/logger"
	"github.com/jinzhu/gorm"
)

// Admin assistant role constants.
const (
	AssistantRoleAdmin     = "admin"
	AssistantRoleAssistant = "assistant"
)

// OnboardingStep slugs — the canonical ordered list for admin onboarding.
var OnboardingSteps = []string{
	"org_profile",
	"sending_profile",
	"first_group",
	"first_template",
	"first_landing_page",
	"first_campaign",
	"report_button",
	"training_assignment",
	"review_dashboard",
}

// ErrAssistantConversationNotFound is returned when a lookup fails.
var ErrAssistantConversationNotFound = errors.New("assistant conversation not found")

// AssistantConversation groups a sequence of admin/assistant messages for a
// single admin user within an organization.
type AssistantConversation struct {
	Id           int64              `json:"id" gorm:"primary_key;auto_increment"`
	OrgId        int64              `json:"org_id"`
	UserId       int64              `json:"user_id"`
	Title        string             `json:"title"`
	CreatedDate  time.Time          `json:"created_date"`
	ModifiedDate time.Time          `json:"modified_date"`
	Messages     []AssistantMessage `json:"messages" gorm:"-"`
}

// TableName overrides the default table name.
func (AssistantConversation) TableName() string {
	return "assistant_conversations"
}

// AssistantMessage is a single admin or assistant turn within a conversation.
type AssistantMessage struct {
	Id             int64     `json:"id" gorm:"primary_key;auto_increment"`
	ConversationId int64     `json:"conversation_id"`
	Role           string    `json:"role"`
	Content        string    `json:"content" gorm:"type:text"`
	TokensUsed     int       `json:"tokens_used"`
	CreatedDate    time.Time `json:"created_date"`
}

// TableName overrides the default table name.
func (AssistantMessage) TableName() string {
	return "assistant_messages"
}

// AdminOnboardingProgress tracks which onboarding steps an admin has
// completed. There is one row per (org, user, step).
type AdminOnboardingProgress struct {
	Id            int64     `json:"id" gorm:"primary_key;auto_increment"`
	OrgId         int64     `json:"org_id"`
	UserId        int64     `json:"user_id"`
	Step          string    `json:"step"`
	CompletedDate time.Time `json:"completed_date"`
}

// TableName overrides the default table name.
func (AdminOnboardingProgress) TableName() string {
	return "admin_onboarding_progress"
}

// OnboardingStatus is the response shape for GET /admin-assistant/onboarding.
type OnboardingStatus struct {
	Steps          []OnboardingStepStatus `json:"steps"`
	CompletedCount int                    `json:"completed_count"`
	TotalCount     int                    `json:"total_count"`
	NextStep       string                 `json:"next_step"`
}

// OnboardingStepStatus is the completion state of a single step.
type OnboardingStepStatus struct {
	Step          string    `json:"step"`
	Completed     bool      `json:"completed"`
	CompletedDate time.Time `json:"completed_date,omitempty"`
}

// GetAdminOnboardingStatus returns the onboarding completion status for an
// admin, ordered by the canonical step list.
func GetAdminOnboardingStatus(orgId, userId int64) (OnboardingStatus, error) {
	var rows []AdminOnboardingProgress
	if err := db.Where("org_id = ? AND user_id = ?", orgId, userId).Find(&rows).Error; err != nil {
		return OnboardingStatus{}, err
	}
	completed := map[string]time.Time{}
	for _, r := range rows {
		completed[r.Step] = r.CompletedDate
	}
	status := OnboardingStatus{TotalCount: len(OnboardingSteps)}
	for _, step := range OnboardingSteps {
		s := OnboardingStepStatus{Step: step}
		if ts, ok := completed[step]; ok {
			s.Completed = true
			s.CompletedDate = ts
			status.CompletedCount++
		} else if status.NextStep == "" {
			status.NextStep = step
		}
		status.Steps = append(status.Steps, s)
	}
	return status, nil
}

// CompleteOnboardingStep records that an admin has completed a step.
// Returns an error if the step slug is not recognized.
func CompleteOnboardingStep(orgId, userId int64, step string) error {
	known := false
	for _, s := range OnboardingSteps {
		if s == step {
			known = true
			break
		}
	}
	if !known {
		return errors.New("unknown onboarding step")
	}
	var existing AdminOnboardingProgress
	err := db.Where("org_id = ? AND user_id = ? AND step = ?", orgId, userId, step).First(&existing).Error
	if err == nil {
		return nil
	}
	if err != gorm.ErrRecordNotFound {
		return err
	}
	return db.Create(&AdminOnboardingProgress{
		OrgId:         orgId,
		UserId:        userId,
		Step:          step,
		CompletedDate: time.Now().UTC(),
	}).Error
}

// GetOrCreateAssistantConversation returns the most recent conversation for
// the admin, or creates a new one if none exists.
func GetOrCreateAssistantConversation(orgId, userId int64) (AssistantConversation, error) {
	var conv AssistantConversation
	err := db.Where("org_id = ? AND user_id = ?", orgId, userId).Order("modified_date desc").First(&conv).Error
	if err == nil {
		return conv, nil
	}
	if err != gorm.ErrRecordNotFound {
		return conv, err
	}
	conv = AssistantConversation{
		OrgId:        orgId,
		UserId:       userId,
		Title:        "Onboarding chat",
		CreatedDate:  time.Now().UTC(),
		ModifiedDate: time.Now().UTC(),
	}
	if err := db.Create(&conv).Error; err != nil {
		return conv, err
	}
	return conv, nil
}

// GetAssistantConversation fetches a conversation by ID, scoped to the org
// and user, and hydrates its messages ordered oldest first.
func GetAssistantConversation(id, orgId, userId int64) (AssistantConversation, error) {
	var conv AssistantConversation
	err := db.Where("id = ? AND org_id = ? AND user_id = ?", id, orgId, userId).First(&conv).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return conv, ErrAssistantConversationNotFound
		}
		return conv, err
	}
	var msgs []AssistantMessage
	if err := db.Where("conversation_id = ?", conv.Id).Order("created_date asc").Find(&msgs).Error; err != nil {
		return conv, err
	}
	conv.Messages = msgs
	return conv, nil
}

// ListAssistantConversations returns all conversations for an admin, newest
// first. Messages are not hydrated.
func ListAssistantConversations(orgId, userId int64) ([]AssistantConversation, error) {
	var convs []AssistantConversation
	err := db.Where("org_id = ? AND user_id = ?", orgId, userId).Order("modified_date desc").Find(&convs).Error
	return convs, err
}

// AskAdminAssistant runs a single turn of the admin-assistant conversation:
// appends the admin question, calls the AI provider with recent history and
// onboarding progress, and appends the assistant reply. Returns the saved
// assistant message.
func AskAdminAssistant(orgId, userId, conversationId int64, question string, aiClient ai.Client) (AssistantMessage, error) {
	var conv AssistantConversation
	if conversationId > 0 {
		c, err := GetAssistantConversation(conversationId, orgId, userId)
		if err != nil {
			return AssistantMessage{}, err
		}
		conv = c
	} else {
		c, err := GetOrCreateAssistantConversation(orgId, userId)
		if err != nil {
			return AssistantMessage{}, err
		}
		conv = c
		var msgs []AssistantMessage
		if err := db.Where("conversation_id = ?", conv.Id).Order("created_date asc").Find(&msgs).Error; err != nil {
			return AssistantMessage{}, err
		}
		conv.Messages = msgs
	}

	// Persist the admin's question.
	adminMsg := AssistantMessage{
		ConversationId: conv.Id,
		Role:           AssistantRoleAdmin,
		Content:        question,
		CreatedDate:    time.Now().UTC(),
	}
	if err := db.Create(&adminMsg).Error; err != nil {
		return AssistantMessage{}, err
	}

	// Build history for the prompt (last 8 turns).
	history := conv.Messages
	if len(history) > 8 {
		history = history[len(history)-8:]
	}
	turns := make([]ai.AssistantTurn, 0, len(history))
	for _, m := range history {
		turns = append(turns, ai.AssistantTurn{Role: m.Role, Content: m.Content})
	}

	// Gather completed onboarding steps.
	status, err := GetAdminOnboardingStatus(orgId, userId)
	if err != nil {
		return AssistantMessage{}, err
	}
	var completed []string
	for _, s := range status.Steps {
		if s.Completed {
			completed = append(completed, s.Step)
		}
	}

	userPrompt := ai.BuildAdminAssistantPrompt(turns, completed, question)
	resp, err := aiClient.Generate(ai.AdminAssistantSystemPrompt, userPrompt)
	if err != nil {
		log.Errorf("admin assistant AI call failed: %v", err)
		return AssistantMessage{}, err
	}

	reply := AssistantMessage{
		ConversationId: conv.Id,
		Role:           AssistantRoleAssistant,
		Content:        resp.Content,
		TokensUsed:     resp.InputTokens + resp.OutputTokens,
		CreatedDate:    time.Now().UTC(),
	}
	if err := db.Create(&reply).Error; err != nil {
		return AssistantMessage{}, err
	}

	conv.ModifiedDate = time.Now().UTC()
	db.Save(&conv)

	return reply, nil
}
