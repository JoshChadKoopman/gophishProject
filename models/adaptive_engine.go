package models

import (
	"fmt"
	"math"
	"time"

	log "github.com/gophish/gophish/logger"
)

// ── Adaptive Difficulty Engine ──
// AI-driven per-user difficulty adjustment based on BRS, click rate trends,
// quiz performance, and training completion patterns.

// AdaptiveEngineConfig controls the behaviour of the adaptive engine per org.
type AdaptiveEngineConfig struct {
	Id                    int64     `json:"id" gorm:"primary_key"`
	OrgId                 int64     `json:"org_id" gorm:"unique_index"`
	Enabled               bool      `json:"enabled"`
	EvalIntervalDays      int       `json:"eval_interval_days"`      // How often to re-evaluate (default 7)
	BRSWeightPct          float64   `json:"brs_weight_pct"`          // Weight for BRS score (default 40)
	ClickRateWeightPct    float64   `json:"click_rate_weight_pct"`   // Weight for click rate (default 30)
	QuizScoreWeightPct    float64   `json:"quiz_score_weight_pct"`   // Weight for quiz performance (default 20)
	TrendWeightPct        float64   `json:"trend_weight_pct"`        // Weight for improvement trend (default 10)
	PromoteThreshold      float64   `json:"promote_threshold"`       // Composite score above which to promote (default 75)
	DemoteThreshold       float64   `json:"demote_threshold"`        // Composite score below which to demote (default 35)
	MinSimulationsPromote int       `json:"min_simulations_promote"` // Min sims before promotion (default 3)
	CooldownDays          int       `json:"cooldown_days"`           // Min days between adjustments (default 14)
	ModifiedDate          time.Time `json:"modified_date" gorm:"column:modified_date"`
}

func (AdaptiveEngineConfig) TableName() string { return "adaptive_engine_configs" }

// Default engine parameters
const (
	DefaultEvalIntervalDays = 7
	DefaultBRSWeight        = 40.0
	DefaultClickRateWeight  = 30.0
	DefaultQuizScoreWeight  = 20.0
	DefaultTrendWeight      = 10.0
	DefaultPromoteThreshold = 75.0
	DefaultDemoteThreshold  = 35.0
	DefaultMinSimsPromote   = 3
	DefaultCooldownDays     = 14
)

// GetAdaptiveEngineConfig retrieves the adaptive engine config for an org, or returns defaults.
func GetAdaptiveEngineConfig(orgId int64) AdaptiveEngineConfig {
	cfg := AdaptiveEngineConfig{}
	err := db.Where(queryWhereOrgID, orgId).First(&cfg).Error
	if err != nil {
		return AdaptiveEngineConfig{
			OrgId:                 orgId,
			Enabled:               true,
			EvalIntervalDays:      DefaultEvalIntervalDays,
			BRSWeightPct:          DefaultBRSWeight,
			ClickRateWeightPct:    DefaultClickRateWeight,
			QuizScoreWeightPct:    DefaultQuizScoreWeight,
			TrendWeightPct:        DefaultTrendWeight,
			PromoteThreshold:      DefaultPromoteThreshold,
			DemoteThreshold:       DefaultDemoteThreshold,
			MinSimulationsPromote: DefaultMinSimsPromote,
			CooldownDays:          DefaultCooldownDays,
		}
	}
	return cfg
}

// SaveAdaptiveEngineConfig creates or updates the adaptive engine config.
func SaveAdaptiveEngineConfig(cfg *AdaptiveEngineConfig) error {
	existing := AdaptiveEngineConfig{}
	err := db.Where(queryWhereOrgID, cfg.OrgId).First(&existing).Error
	if err == nil {
		cfg.Id = existing.Id
	}
	cfg.ModifiedDate = time.Now().UTC()
	return db.Save(cfg).Error
}

// AdaptiveEvaluation is the result of evaluating a single user's difficulty.
type AdaptiveEvaluation struct {
	UserId           int64   `json:"user_id"`
	Email            string  `json:"email"`
	CurrentLevel     int     `json:"current_level"`
	RecommendedLevel int     `json:"recommended_level"`
	CompositeScore   float64 `json:"composite_score"`
	BRSComponent     float64 `json:"brs_component"`
	ClickComponent   float64 `json:"click_component"`
	QuizComponent    float64 `json:"quiz_component"`
	TrendComponent   float64 `json:"trend_component"`
	Action           string  `json:"action"` // "promote", "demote", "maintain"
	Reason           string  `json:"reason"`
	Applied          bool    `json:"applied"`
}

// RunAdaptiveEngine evaluates all users in an org and adjusts difficulty levels.
// Returns the list of evaluations performed.
func RunAdaptiveEngine(orgId int64) ([]AdaptiveEvaluation, error) {
	cfg := GetAdaptiveEngineConfig(orgId)
	if !cfg.Enabled {
		return nil, nil
	}

	users, err := getOrgUsers(orgId)
	if err != nil {
		return nil, fmt.Errorf("adaptive engine: failed to load users: %w", err)
	}

	var evaluations []AdaptiveEvaluation
	for _, u := range users {
		// Skip users in manual mode
		if u.TrainingDifficultyMode == DifficultyModeManual {
			continue
		}

		eval := evaluateUser(u, cfg)
		evaluations = append(evaluations, eval)
	}

	return evaluations, nil
}

// evaluateUser computes the adaptive score for a single user and applies changes.
func evaluateUser(u User, cfg AdaptiveEngineConfig) AdaptiveEvaluation {
	eval := AdaptiveEvaluation{
		UserId:       u.Id,
		Email:        u.Email,
		CurrentLevel: GetEffectiveDifficulty(u.Id),
	}

	// Check cooldown — don't adjust too frequently
	lastAdjustment := getLastAdjustmentDate(u.Id)
	if !lastAdjustment.IsZero() {
		daysSince := time.Since(lastAdjustment).Hours() / 24
		if daysSince < float64(cfg.CooldownDays) {
			eval.RecommendedLevel = eval.CurrentLevel
			eval.Action = "maintain"
			eval.Reason = fmt.Sprintf("Cooldown: only %.0f of %d days since last adjustment", daysSince, cfg.CooldownDays)
			return eval
		}
	}

	// ── Compute component scores (0-100 scale) ──

	// BRS Component: directly use the composite score
	brsRecord := UserRiskScoreRecord{}
	if err := db.Where(queryWhereUserID, u.Id).First(&brsRecord).Error; err == nil {
		eval.BRSComponent = brsRecord.CompositeScore
	} else {
		eval.BRSComponent = 50.0 // neutral default
	}

	// Click Rate Component: lower click rate = higher score
	profile, profileErr := GetUserTargetingProfile(u.Id)
	if profileErr == nil {
		// Invert: 0% click rate = 100 score, 100% click rate = 0 score
		eval.ClickComponent = math.Max(0, 100-profile.OverallClickRate*2)
	} else {
		eval.ClickComponent = 50.0
	}

	// Quiz Component: average quiz score directly
	var avgQuiz float64
	row := db.Table("quiz_attempts").
		Select("COALESCE(AVG(score_percentage), 50)").
		Where("user_id = ?", u.Id).Row()
	row.Scan(&avgQuiz)
	eval.QuizComponent = avgQuiz

	// Trend Component: improvement over last 3 BRS history points
	eval.TrendComponent = computeTrendScore(u.Id)

	// ── Weighted composite ──
	totalWeight := cfg.BRSWeightPct + cfg.ClickRateWeightPct + cfg.QuizScoreWeightPct + cfg.TrendWeightPct
	if totalWeight <= 0 {
		totalWeight = 100
	}
	eval.CompositeScore = math.Round((eval.BRSComponent*cfg.BRSWeightPct+
		eval.ClickComponent*cfg.ClickRateWeightPct+
		eval.QuizComponent*cfg.QuizScoreWeightPct+
		eval.TrendComponent*cfg.TrendWeightPct)/totalWeight*100) / 100

	// ── Determine action ──
	eval.RecommendedLevel = eval.CurrentLevel

	if eval.CompositeScore >= cfg.PromoteThreshold && eval.CurrentLevel < DifficultySophisticated {
		// Check minimum simulations
		simCount := int64(0)
		if profileErr == nil {
			simCount = profile.TotalSimulations
		}
		if simCount >= int64(cfg.MinSimulationsPromote) {
			eval.RecommendedLevel = eval.CurrentLevel + 1
			eval.Action = "promote"
			eval.Reason = fmt.Sprintf("Composite score %.1f >= promote threshold %.1f (BRS=%.0f, Click=%.0f, Quiz=%.0f, Trend=%.0f)",
				eval.CompositeScore, cfg.PromoteThreshold,
				eval.BRSComponent, eval.ClickComponent, eval.QuizComponent, eval.TrendComponent)
		} else {
			eval.Action = "maintain"
			eval.Reason = fmt.Sprintf("Score qualifies for promotion but only %d of %d required simulations completed",
				simCount, cfg.MinSimulationsPromote)
		}
	} else if eval.CompositeScore <= cfg.DemoteThreshold && eval.CurrentLevel > DifficultyEasy {
		eval.RecommendedLevel = eval.CurrentLevel - 1
		eval.Action = "demote"
		eval.Reason = fmt.Sprintf("Composite score %.1f <= demote threshold %.1f — reducing difficulty to support learning",
			eval.CompositeScore, cfg.DemoteThreshold)
	} else {
		eval.Action = "maintain"
		eval.Reason = fmt.Sprintf("Composite score %.1f within maintenance range (%.1f–%.1f)",
			eval.CompositeScore, cfg.DemoteThreshold, cfg.PromoteThreshold)
	}

	// ── Apply change if needed ──
	if eval.RecommendedLevel != eval.CurrentLevel {
		err := applyAdaptiveAdjustment(u.Id, eval)
		if err != nil {
			log.Errorf("adaptive engine: failed to apply adjustment for user %d: %v", u.Id, err)
			eval.Applied = false
		} else {
			eval.Applied = true
		}
	}

	return eval
}

// applyAdaptiveAdjustment updates the user's difficulty and logs the change.
func applyAdaptiveAdjustment(userId int64, eval AdaptiveEvaluation) error {
	err := db.Model(&User{}).Where("id=?", userId).Updates(map[string]interface{}{
		"training_difficulty_manual": eval.RecommendedLevel,
	}).Error
	if err != nil {
		return err
	}

	// Log the adjustment
	logDifficultyChange(userId, eval.CurrentLevel, eval.RecommendedLevel, "adaptive_engine", eval.Reason, 0)

	// Create a user-facing notification about the change
	recordDifficultyNotification(userId, eval)

	return nil
}

// recordDifficultyNotification creates a user-visible notification when the
// adaptive engine adjusts their difficulty level.
func recordDifficultyNotification(userId int64, eval AdaptiveEvaluation) {
	direction := "increased"
	encouragement := "Keep up the great work! You're ready for more advanced challenges."
	if eval.RecommendedLevel < eval.CurrentLevel {
		direction = "decreased"
		encouragement = "We're adjusting your training to reinforce key concepts before moving forward."
	}

	message := fmt.Sprintf(
		"Your training difficulty has been %s from %s to %s based on your recent performance. %s",
		direction,
		DifficultyLevelLabels[eval.CurrentLevel],
		DifficultyLevelLabels[eval.RecommendedLevel],
		encouragement,
	)

	notification := &TrainingReminder{
		UserId:       userId,
		ReminderType: "difficulty_change",
		Message:      message,
	}
	if err := db.Save(notification).Error; err != nil {
		log.Errorf("adaptive engine: failed to save difficulty notification for user %d: %v", userId, err)
	}
}

// computeTrendScore calculates an improvement score from BRS history.
// Rising trend = high score, flat = 50, declining = low score.
func computeTrendScore(userId int64) float64 {
	var points []BRSHistoryPoint
	db.Where(queryWhereUserID, userId).
		Order("calculated_date desc").
		Limit(5).
		Find(&points)

	if len(points) < 2 {
		return 50.0 // neutral
	}

	// Calculate average change between consecutive points
	totalChange := 0.0
	for i := 0; i < len(points)-1; i++ {
		totalChange += points[i].CompositeScore - points[i+1].CompositeScore
	}
	avgChange := totalChange / float64(len(points)-1)

	// Map to 0-100: -10 or less = 0, +10 or more = 100, 0 = 50
	trendScore := 50 + avgChange*5
	return math.Max(0, math.Min(100, trendScore))
}

// getLastAdjustmentDate returns the most recent difficulty adjustment for a user.
func getLastAdjustmentDate(userId int64) time.Time {
	var log DifficultyAdjustmentLog
	err := db.Where(queryWhereUserID+" AND source = ?", userId, "adaptive_engine").
		Order("created_date desc").
		First(&log).Error
	if err != nil {
		return time.Time{}
	}
	return log.CreatedDate
}

// getOrgUsers returns all users for an organization (internal to adaptive engine).
func getOrgUsers(orgId int64) ([]User, error) {
	var users []User
	err := db.Preload("Role").Where(queryWhereOrgID, orgId).Find(&users).Error
	return users, err
}

// ── Adaptive Engine Audit Trail ──

// AdaptiveEngineRunLog records the outcome of each adaptive engine evaluation cycle.
type AdaptiveEngineRunLog struct {
	Id             int64     `json:"id" gorm:"primary_key"`
	OrgId          int64     `json:"org_id"`
	UsersEvaluated int       `json:"users_evaluated"`
	Promoted       int       `json:"promoted"`
	Demoted        int       `json:"demoted"`
	Maintained     int       `json:"maintained"`
	Skipped        int       `json:"skipped"`
	RunDate        time.Time `json:"run_date"`
}

func (AdaptiveEngineRunLog) TableName() string { return "adaptive_engine_run_logs" }

// RecordAdaptiveEngineRun saves the result of an engine cycle for audit purposes.
func RecordAdaptiveEngineRun(orgId int64, evaluations []AdaptiveEvaluation, skipped int) {
	entry := AdaptiveEngineRunLog{
		OrgId:          orgId,
		UsersEvaluated: len(evaluations),
		Skipped:        skipped,
		RunDate:        time.Now().UTC(),
	}
	for _, e := range evaluations {
		switch e.Action {
		case "promote":
			entry.Promoted++
		case "demote":
			entry.Demoted++
		default:
			entry.Maintained++
		}
	}
	if err := db.Save(&entry).Error; err != nil {
		log.Errorf("adaptive engine: failed to save run log: %v", err)
	}
}

// GetAdaptiveEngineRunHistory returns the last N engine run logs for an org.
func GetAdaptiveEngineRunHistory(orgId int64, limit int) ([]AdaptiveEngineRunLog, error) {
	var logs []AdaptiveEngineRunLog
	if limit <= 0 {
		limit = 20
	}
	err := db.Where(queryWhereOrgID, orgId).
		Order("run_date desc").
		Limit(limit).
		Find(&logs).Error
	return logs, err
}

// AdaptiveEngineSummary provides a high-level overview of the engine state for an org.
type AdaptiveEngineSummary struct {
	Config         AdaptiveEngineConfig   `json:"config"`
	LastRunDate    *time.Time             `json:"last_run_date"`
	TotalEvaluated int                    `json:"total_evaluated"`
	TotalPromoted  int                    `json:"total_promoted"`
	TotalDemoted   int                    `json:"total_demoted"`
	RecentRuns     []AdaptiveEngineRunLog `json:"recent_runs"`
}

// GetAdaptiveEngineSummary returns a summary of the adaptive engine state for an org.
func GetAdaptiveEngineSummary(orgId int64) AdaptiveEngineSummary {
	summary := AdaptiveEngineSummary{
		Config: GetAdaptiveEngineConfig(orgId),
	}

	runs, _ := GetAdaptiveEngineRunHistory(orgId, 10)
	summary.RecentRuns = runs

	if len(runs) > 0 {
		summary.LastRunDate = &runs[0].RunDate
	}

	for _, r := range runs {
		summary.TotalEvaluated += r.UsersEvaluated
		summary.TotalPromoted += r.Promoted
		summary.TotalDemoted += r.Demoted
	}

	return summary
}
