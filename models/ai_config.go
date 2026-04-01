package models

import (
	"time"

	log "github.com/gophish/gophish/logger"
)

// AIGenerationLog records each AI template generation for auditing and
// token usage tracking.
type AIGenerationLog struct {
	Id           int64     `json:"id" gorm:"primary_key"`
	OrgId        int64     `json:"org_id"`
	UserId       int64     `json:"user_id"`
	Provider     string    `json:"provider"`
	ModelUsed    string    `json:"model_used"`
	InputTokens  int       `json:"input_tokens"`
	OutputTokens int       `json:"output_tokens"`
	TemplateId   int64     `json:"template_id"`
	CreatedDate  time.Time `json:"created_date"`
}

// CreateAIGenerationLog inserts a new AI generation log entry.
func CreateAIGenerationLog(entry *AIGenerationLog) error {
	entry.CreatedDate = time.Now().UTC()
	err := db.Save(entry).Error
	if err != nil {
		log.Error(err)
	}
	return err
}

// GetAIGenerationLogs returns generation logs for an org, most recent first.
func GetAIGenerationLogs(orgId int64, limit int) ([]AIGenerationLog, error) {
	logs := []AIGenerationLog{}
	if limit <= 0 {
		limit = 50
	}
	err := db.Where("org_id = ?", orgId).
		Order("created_date desc").
		Limit(limit).
		Find(&logs).Error
	if err != nil {
		log.Error(err)
	}
	return logs, err
}

// AIUsageSummary aggregates token usage for an org within a time range.
type AIUsageSummary struct {
	TotalGenerations int `json:"total_generations"`
	TotalInputTokens int `json:"total_input_tokens"`
	TotalOutputTokens int `json:"total_output_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// GetAIUsageSummary returns aggregate token usage for the given org since the
// provided start date.
func GetAIUsageSummary(orgId int64, since time.Time) (*AIUsageSummary, error) {
	summary := &AIUsageSummary{}
	row := db.Table("ai_generation_logs").
		Select("COUNT(*) as total_generations, COALESCE(SUM(input_tokens),0) as total_input_tokens, COALESCE(SUM(output_tokens),0) as total_output_tokens").
		Where("org_id = ? AND created_date >= ?", orgId, since).
		Row()
	err := row.Scan(&summary.TotalGenerations, &summary.TotalInputTokens, &summary.TotalOutputTokens)
	if err != nil {
		log.Error(err)
		return summary, err
	}
	summary.TotalTokens = summary.TotalInputTokens + summary.TotalOutputTokens
	return summary, nil
}
