package models

import (
	"errors"
	"fmt"
	"time"

	log "github.com/gophish/gophish/logger"
)

// Training difficulty modes
const (
	DifficultyModeAdaptive = "adaptive" // AI adjusts per user over time
	DifficultyModeManual   = "manual"   // User/admin selected fixed level
)

// Training difficulty levels (1-4)
const (
	DifficultyEasy          = 1 // Bronze / Fundamentals
	DifficultyMedium        = 2 // Silver / Intermediate
	DifficultyHard          = 3 // Gold / Advanced
	DifficultySophisticated = 4 // Platinum / Expert
)

// DifficultyLevelLabels maps numeric levels to human-readable labels.
var DifficultyLevelLabels = map[int]string{
	DifficultyEasy:          "Easy (Fundamentals)",
	DifficultyMedium:        "Medium (Intermediate)",
	DifficultyHard:          "Hard (Advanced)",
	DifficultySophisticated: "Sophisticated (Expert)",
}

// DifficultyAdjustmentLog records every change to a user's training difficulty.
type DifficultyAdjustmentLog struct {
	Id                int64     `json:"id" gorm:"primary_key"`
	UserId            int64     `json:"user_id" gorm:"column:user_id"`
	PreviousLevel     int       `json:"previous_level" gorm:"column:previous_level"`
	NewLevel          int       `json:"new_level" gorm:"column:new_level"`
	Source            string    `json:"source" gorm:"column:source"` // "adaptive", "manual", "admin"
	Reason            string    `json:"reason" gorm:"column:reason"` // human-readable explanation
	BRSAtChange       float64   `json:"brs_at_change" gorm:"column:brs_at_change"`
	ClickRateAtChange float64   `json:"click_rate_at_change" gorm:"column:click_rate_at_change"`
	CreatedDate       time.Time `json:"created_date" gorm:"column:created_date"`
}

// TableName overrides the GORM table name.
func (DifficultyAdjustmentLog) TableName() string {
	return "difficulty_adjustment_log"
}

// UserDifficultyProfile is the API response showing a user's current training
// difficulty status, including both adaptive and manual information.
type UserDifficultyProfile struct {
	UserId            int64                     `json:"user_id"`
	Mode              string                    `json:"mode"`            // "adaptive" or "manual"
	EffectiveLevel    int                       `json:"effective_level"` // the level actually in use (1-4)
	EffectiveLabel    string                    `json:"effective_label"` // human-readable
	AdaptiveLevel     int                       `json:"adaptive_level"`  // what AI recommends (always computed)
	AdaptiveLabel     string                    `json:"adaptive_label"`
	ManualLevel       int                       `json:"manual_level"` // manual override if set (0 = not set)
	BRSComposite      float64                   `json:"brs_composite"`
	OverallClickRate  float64                   `json:"overall_click_rate"`
	TrendDirection    string                    `json:"trend_direction"`
	TotalSimulations  int64                     `json:"total_simulations"`
	RecentAdjustments []DifficultyAdjustmentLog `json:"recent_adjustments"` // last 10 changes
}

// Errors
var (
	ErrInvalidDifficultyLevel = errors.New("Difficulty level must be between 1 and 4")
	ErrInvalidDifficultyMode  = errors.New("Difficulty mode must be 'adaptive' or 'manual'")
)

// GetUserDifficultyProfile builds the complete difficulty profile for a user,
// combining adaptive AI recommendations with any manual override.
func GetUserDifficultyProfile(userId int64) (*UserDifficultyProfile, error) {
	user, err := GetUser(userId)
	if err != nil {
		return nil, err
	}

	profile := &UserDifficultyProfile{
		UserId:      userId,
		Mode:        user.TrainingDifficultyMode,
		ManualLevel: user.TrainingDifficultyManual,
	}

	// Default mode if empty
	if profile.Mode == "" {
		profile.Mode = DifficultyModeAdaptive
	}

	// Always compute the adaptive recommendation (even in manual mode)
	targetingProfile, err := GetUserTargetingProfile(userId)
	if err != nil {
		// If targeting profile can't be built, default to easy
		profile.AdaptiveLevel = DifficultyEasy
		log.Warnf("adaptive_difficulty: could not build targeting profile for user %d: %v", userId, err)
	} else {
		profile.AdaptiveLevel = targetingProfile.RecommendedDifficulty
		profile.BRSComposite = targetingProfile.BRSComposite
		profile.OverallClickRate = targetingProfile.OverallClickRate
		profile.TrendDirection = targetingProfile.TrendDirection
		profile.TotalSimulations = targetingProfile.TotalSimulations
	}
	profile.AdaptiveLabel = DifficultyLevelLabels[profile.AdaptiveLevel]

	// Determine effective level based on mode
	switch profile.Mode {
	case DifficultyModeManual:
		if profile.ManualLevel >= DifficultyEasy && profile.ManualLevel <= DifficultySophisticated {
			profile.EffectiveLevel = profile.ManualLevel
		} else {
			// Invalid manual level, fall back to adaptive
			profile.EffectiveLevel = profile.AdaptiveLevel
		}
	default: // adaptive
		profile.EffectiveLevel = profile.AdaptiveLevel
	}
	profile.EffectiveLabel = DifficultyLevelLabels[profile.EffectiveLevel]

	// Load recent adjustment history
	profile.RecentAdjustments = getRecentAdjustments(userId, 10)

	return profile, nil
}

// GetEffectiveDifficulty returns the difficulty level to use for a user's training.
// This is the main entry point for assignment/content-selection logic.
func GetEffectiveDifficulty(userId int64) int {
	user, err := GetUser(userId)
	if err != nil {
		return DifficultyEasy
	}

	if user.TrainingDifficultyMode == DifficultyModeManual &&
		user.TrainingDifficultyManual >= DifficultyEasy &&
		user.TrainingDifficultyManual <= DifficultySophisticated {
		return user.TrainingDifficultyManual
	}

	// Adaptive mode: compute from targeting profile
	profile, err := GetUserTargetingProfile(userId)
	if err != nil {
		return DifficultyEasy
	}
	return profile.RecommendedDifficulty
}

// SetManualDifficulty sets a manual difficulty level for a user and logs the change.
func SetManualDifficulty(userId int64, level int, changedBy string) error {
	if level < DifficultyEasy || level > DifficultySophisticated {
		return ErrInvalidDifficultyLevel
	}

	user, err := GetUser(userId)
	if err != nil {
		return err
	}

	previousLevel := GetEffectiveDifficulty(userId)

	// Update user preferences
	err = db.Model(&User{}).Where("id=?", userId).Updates(map[string]interface{}{
		"training_difficulty_mode":   DifficultyModeManual,
		"training_difficulty_manual": level,
	}).Error
	if err != nil {
		return err
	}

	// Log the change
	reason := fmt.Sprintf("Manual override set to level %d (%s) by %s",
		level, DifficultyLevelLabels[level], changedBy)
	logDifficultyChange(userId, previousLevel, level, "manual", reason, user.OrgId)

	return nil
}

// ClearManualDifficulty removes the manual override and switches back to adaptive mode.
func ClearManualDifficulty(userId int64, changedBy string) error {
	user, err := GetUser(userId)
	if err != nil {
		return err
	}

	previousLevel := GetEffectiveDifficulty(userId)

	err = db.Model(&User{}).Where("id=?", userId).Updates(map[string]interface{}{
		"training_difficulty_mode":   DifficultyModeAdaptive,
		"training_difficulty_manual": 0,
	}).Error
	if err != nil {
		return err
	}

	// Compute new adaptive level
	newLevel := GetEffectiveDifficulty(userId)

	reason := fmt.Sprintf("Switched to adaptive mode by %s; AI recommends level %d (%s)",
		changedBy, newLevel, DifficultyLevelLabels[newLevel])
	logDifficultyChange(userId, previousLevel, newLevel, "adaptive", reason, user.OrgId)

	return nil
}

// RunAdaptiveAdjustment evaluates all adaptive-mode users in an org and
// adjusts their difficulty if their performance warrants a change.
// This should be called periodically (e.g., after campaign completion or daily cron).
func RunAdaptiveAdjustment(orgId int64) (int, error) {
	var users []User
	err := db.Where("org_id = ? AND (training_difficulty_mode = ? OR training_difficulty_mode = '' OR training_difficulty_mode IS NULL)",
		orgId, DifficultyModeAdaptive).Find(&users).Error
	if err != nil {
		return 0, err
	}

	adjusted := 0
	for _, u := range users {
		profile, err := GetUserTargetingProfile(u.Id)
		if err != nil {
			continue
		}

		newLevel := profile.RecommendedDifficulty
		currentLevel := GetEffectiveDifficulty(u.Id)

		if newLevel != currentLevel {
			// Auto-adjust
			err = db.Model(&User{}).Where("id=?", u.Id).Updates(map[string]interface{}{
				"training_difficulty_mode": DifficultyModeAdaptive,
			}).Error
			if err != nil {
				log.Errorf("adaptive_difficulty: failed to update user %d: %v", u.Id, err)
				continue
			}

			reason := buildAdaptiveReason(profile, currentLevel, newLevel)
			logDifficultyChange(u.Id, currentLevel, newLevel, "adaptive", reason, orgId)
			adjusted++
		}
	}

	log.Infof("adaptive_difficulty: adjusted %d users in org %d", adjusted, orgId)
	return adjusted, nil
}

// GetDifficultyAdjustmentHistory returns the full adjustment history for a user.
func GetDifficultyAdjustmentHistory(userId int64) ([]DifficultyAdjustmentLog, error) {
	var logs []DifficultyAdjustmentLog
	err := db.Where("user_id = ?", userId).Order("created_date desc").Find(&logs).Error
	return logs, err
}

// GetOrgDifficultyStats returns a summary of difficulty distribution across an org.
func GetOrgDifficultyStats(orgId int64) (*OrgDifficultyStats, error) {
	stats := &OrgDifficultyStats{OrgId: orgId}

	// Count by mode
	db.Model(&User{}).Where("org_id = ? AND (training_difficulty_mode = ? OR training_difficulty_mode = '' OR training_difficulty_mode IS NULL)",
		orgId, DifficultyModeAdaptive).Count(&stats.AdaptiveUsers)
	db.Model(&User{}).Where("org_id = ? AND training_difficulty_mode = ?",
		orgId, DifficultyModeManual).Count(&stats.ManualUsers)

	// Count by effective difficulty level (compute for each user)
	var users []User
	err := db.Where("org_id = ?", orgId).Find(&users).Error
	if err != nil {
		return stats, err
	}

	stats.LevelDistribution = map[int]int{1: 0, 2: 0, 3: 0, 4: 0}
	for _, u := range users {
		level := GetEffectiveDifficulty(u.Id)
		stats.LevelDistribution[level]++
	}

	return stats, nil
}

// OrgDifficultyStats summarizes difficulty settings across an organization.
type OrgDifficultyStats struct {
	OrgId             int64       `json:"org_id"`
	AdaptiveUsers     int         `json:"adaptive_users"`
	ManualUsers       int         `json:"manual_users"`
	LevelDistribution map[int]int `json:"level_distribution"` // level -> user count
}

// -- Internal helpers --------------------------------------------------------

func logDifficultyChange(userId int64, previousLevel, newLevel int, source, reason string, orgId int64) {
	brs := 0.0
	clickRate := 0.0
	if b, err := GetUserBRS(userId); err == nil {
		brs = b.CompositeScore
	}
	if tp, err := GetUserTargetingProfile(userId); err == nil {
		clickRate = tp.OverallClickRate
	}

	entry := &DifficultyAdjustmentLog{
		UserId:            userId,
		PreviousLevel:     previousLevel,
		NewLevel:          newLevel,
		Source:            source,
		Reason:            reason,
		BRSAtChange:       brs,
		ClickRateAtChange: clickRate,
		CreatedDate:       time.Now().UTC(),
	}
	if err := db.Save(entry).Error; err != nil {
		log.Errorf("adaptive_difficulty: failed to log change for user %d: %v", userId, err)
	}
}

func getRecentAdjustments(userId int64, limit int) []DifficultyAdjustmentLog {
	var logs []DifficultyAdjustmentLog
	db.Where("user_id = ?", userId).Order("created_date desc").Limit(limit).Find(&logs)
	return logs
}

func buildAdaptiveReason(profile *UserTargetingProfile, oldLevel, newLevel int) string {
	direction := "increased"
	if newLevel < oldLevel {
		direction = "decreased"
	}

	reason := fmt.Sprintf("AI auto-adjustment: difficulty %s from %d to %d. ", direction, oldLevel, newLevel)

	// Add context
	reason += fmt.Sprintf("BRS=%.1f, click_rate=%.1f%%, trend=%s, simulations=%d.",
		profile.BRSComposite,
		profile.OverallClickRate*100,
		profile.TrendDirection,
		profile.TotalSimulations,
	)

	if newLevel > oldLevel {
		reason += " User performance is improving — raising challenge level."
	} else {
		reason += " User performance is declining — reinforcing fundamentals."
	}

	return reason
}

// SelectTrainingContentForUser picks content from the built-in library
// that matches the user's effective difficulty level.
func SelectTrainingContentForUser(userId int64, category string) []BuiltInTrainingContent {
	level := GetEffectiveDifficulty(userId)
	var pool []BuiltInTrainingContent
	if category != "" {
		pool = GetBuiltInContentByCategory(category)
	} else {
		pool = GetBuiltInContentLibrary()
	}
	return filterByDifficultyWithFallback(pool, level)
}

// filterByDifficultyWithFallback returns content at the exact level, falling
// back to ±1 level if no exact matches exist.
func filterByDifficultyWithFallback(pool []BuiltInTrainingContent, level int) []BuiltInTrainingContent {
	exact := filterByDifficultyRange(pool, level, 0)
	if len(exact) > 0 {
		return exact
	}
	return filterByDifficultyRange(pool, level, 1)
}

// filterByDifficultyRange returns content within ±tolerance of the target level.
func filterByDifficultyRange(pool []BuiltInTrainingContent, level, tolerance int) []BuiltInTrainingContent {
	var result []BuiltInTrainingContent
	for _, c := range pool {
		diff := c.DifficultyLevel - level
		if diff >= -tolerance && diff <= tolerance {
			result = append(result, c)
		}
	}
	return result
}

// ValidateDifficultyLevel checks that a level is within 1-4.
func ValidateDifficultyLevel(level int) error {
	if level < DifficultyEasy || level > DifficultySophisticated {
		return ErrInvalidDifficultyLevel
	}
	return nil
}

// ValidateDifficultyMode checks that a mode is "adaptive" or "manual".
func ValidateDifficultyMode(mode string) error {
	if mode != DifficultyModeAdaptive && mode != DifficultyModeManual {
		return ErrInvalidDifficultyMode
	}
	return nil
}
