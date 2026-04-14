package models

import (
	"encoding/json"
	"time"

	log "github.com/gophish/gophish/logger"
)

// AcademyTier represents a tier in the academy progression (Bronze, Silver, Gold, Platinum).
type AcademyTier struct {
	Id           int64     `json:"id" gorm:"column:id; primary_key:yes"`
	OrgId        int64     `json:"org_id" gorm:"column:org_id"`
	Slug         string    `json:"slug" gorm:"column:slug"`
	Name         string    `json:"name" gorm:"column:name"`
	Description  string    `json:"description" gorm:"column:description"`
	BadgeIconURL string    `json:"badge_icon_url" gorm:"column:badge_icon_url"`
	SortOrder    int       `json:"sort_order" gorm:"column:sort_order"`
	IsActive     bool      `json:"is_active" gorm:"column:is_active"`
	CreatedDate  time.Time `json:"created_date" gorm:"column:created_date"`

	// Populated at query time, not stored
	Sessions         []AcademySession     `json:"sessions,omitempty" gorm:"-"`
	TotalSessions    int                  `json:"total_sessions" gorm:"-"`
	RequiredSessions int                  `json:"required_sessions" gorm:"-"`
	UserProgress     *AcademyUserProgress `json:"user_progress,omitempty" gorm:"-"`
}

// AcademySession links a training presentation to an academy tier.
type AcademySession struct {
	Id               int64     `json:"id" gorm:"column:id; primary_key:yes"`
	TierId           int64     `json:"tier_id" gorm:"column:tier_id"`
	PresentationId   int64     `json:"presentation_id" gorm:"column:presentation_id"`
	SortOrder        int       `json:"sort_order" gorm:"column:sort_order"`
	EstimatedMinutes int       `json:"estimated_minutes" gorm:"column:estimated_minutes"`
	IsRequired       bool      `json:"is_required" gorm:"column:is_required"`
	CreatedDate      time.Time `json:"created_date" gorm:"column:created_date"`

	// Populated at query time
	PresentationName string `json:"presentation_name,omitempty" gorm:"-"`
	Completed        bool   `json:"completed" gorm:"-"`
}

// AcademyUserProgress tracks a user's progress through an academy tier.
type AcademyUserProgress struct {
	Id                int64     `json:"id" gorm:"column:id; primary_key:yes"`
	UserId            int64     `json:"user_id" gorm:"column:user_id"`
	TierId            int64     `json:"tier_id" gorm:"column:tier_id"`
	SessionsCompleted int       `json:"sessions_completed" gorm:"column:sessions_completed"`
	TierUnlocked      bool      `json:"tier_unlocked" gorm:"column:tier_unlocked"`
	TierCompleted     bool      `json:"tier_completed" gorm:"column:tier_completed"`
	CompletedDate     time.Time `json:"completed_date" gorm:"column:completed_date"`
	CreatedDate       time.Time `json:"created_date" gorm:"column:created_date"`
}

// TableName overrides the default GORM table name.
func (AcademyUserProgress) TableName() string {
	return "academy_user_progress"
}

// Shared WHERE clause constants for academy queries.
const (
	orderSortOrderAsc       = "sort_order asc"
	queryWhereTierID        = "tier_id = ?"
	queryWhereUserAndTierID = "user_id = ? AND tier_id = ?"
)

// GetAcademyTiers returns all active tiers for an org (falls back to system defaults org_id=0).
func GetAcademyTiers(orgId int64) ([]AcademyTier, error) {
	tiers := []AcademyTier{}
	err := db.Where("(org_id = ? OR org_id = 0) AND is_active = 1", orgId).
		Order(orderSortOrderAsc).Find(&tiers).Error
	if err != nil {
		log.Error(err)
		return tiers, err
	}
	// Populate session counts
	for i := range tiers {
		var total, required int
		db.Table("academy_sessions").Where(queryWhereTierID, tiers[i].Id).Count(&total)
		db.Table("academy_sessions").Where("tier_id = ? AND is_required = 1", tiers[i].Id).Count(&required)
		tiers[i].TotalSessions = total
		tiers[i].RequiredSessions = required
	}
	return tiers, nil
}

// GetAcademyTiersWithProgress returns tiers with user progress attached.
func GetAcademyTiersWithProgress(orgId, userId int64) ([]AcademyTier, error) {
	tiers, err := GetAcademyTiers(orgId)
	if err != nil {
		return tiers, err
	}
	for i := range tiers {
		progress := AcademyUserProgress{}
		err := db.Where(queryWhereUserAndTierID, userId, tiers[i].Id).First(&progress).Error
		if err == nil {
			tiers[i].UserProgress = &progress
		}
		// First tier is always unlocked
		if i == 0 && tiers[i].UserProgress == nil {
			tiers[i].UserProgress = &AcademyUserProgress{
				UserId:       userId,
				TierId:       tiers[i].Id,
				TierUnlocked: true,
			}
		}
	}
	return tiers, nil
}

// GetAcademyTierBySlug returns a single tier by slug.
func GetAcademyTierBySlug(orgId int64, slug string) (AcademyTier, error) {
	tier := AcademyTier{}
	err := db.Where("(org_id = ? OR org_id = 0) AND slug = ? AND is_active = 1", orgId, slug).
		First(&tier).Error
	return tier, err
}

// GetAcademySessions returns all sessions for a tier with presentation names.
func GetAcademySessions(tierId int64) ([]AcademySession, error) {
	sessions := []AcademySession{}
	err := db.Where(queryWhereTierID, tierId).Order(orderSortOrderAsc).Find(&sessions).Error
	if err != nil {
		return sessions, err
	}
	for i := range sessions {
		tp := TrainingPresentation{}
		if err := db.Table("training_presentations").Where(queryWhereID, sessions[i].PresentationId).First(&tp).Error; err == nil {
			sessions[i].PresentationName = tp.Name
		}
	}
	return sessions, nil
}

// GetAcademySessionsWithUserProgress returns sessions with completion status for a user.
func GetAcademySessionsWithUserProgress(tierId, userId int64) ([]AcademySession, error) {
	sessions, err := GetAcademySessions(tierId)
	if err != nil {
		return sessions, err
	}
	for i := range sessions {
		cp := CourseProgress{}
		if err := db.Where("user_id = ? AND presentation_id = ? AND status = 'complete'", userId, sessions[i].PresentationId).First(&cp).Error; err == nil {
			sessions[i].Completed = true
		}
	}
	return sessions, nil
}

// CreateAcademySession adds a session to a tier.
func CreateAcademySession(s *AcademySession) error {
	s.CreatedDate = time.Now().UTC()
	return db.Save(s).Error
}

// UpdateAcademySession updates a session's sort order, estimated minutes, or required flag.
func UpdateAcademySession(s *AcademySession) error {
	return db.Table("academy_sessions").Where(queryWhereID, s.Id).Updates(map[string]interface{}{
		"sort_order":        s.SortOrder,
		"estimated_minutes": s.EstimatedMinutes,
		"is_required":       s.IsRequired,
	}).Error
}

// DeleteAcademySession removes a session from a tier.
func DeleteAcademySession(id int64) error {
	return db.Where(queryWhereID, id).Delete(&AcademySession{}).Error
}

// GetAcademyUserProgress returns progress for a user on a tier.
func GetAcademyUserProgress(userId, tierId int64) (AcademyUserProgress, error) {
	p := AcademyUserProgress{}
	err := db.Where(queryWhereUserAndTierID, userId, tierId).First(&p).Error
	return p, err
}

// GetUserAcademyOverview returns overall academy progress for a user (all tiers).
func GetUserAcademyOverview(userId int64) ([]AcademyUserProgress, error) {
	progress := []AcademyUserProgress{}
	err := db.Where("user_id = ?", userId).Find(&progress).Error
	return progress, err
}

// UpdateAcademyProgress recalculates a user's progress for a specific tier.
// It counts completed required sessions and marks the tier as complete if all required sessions are done.
func UpdateAcademyProgress(userId, tierId int64) error {
	sessions, err := GetAcademySessionsWithUserProgress(tierId, userId)
	if err != nil {
		return err
	}

	requiredTotal := 0
	requiredDone := 0
	for _, s := range sessions {
		if s.IsRequired {
			requiredTotal++
			if s.Completed {
				requiredDone++
			}
		}
	}

	progress, err := GetAcademyUserProgress(userId, tierId)
	if err != nil {
		// Create new progress record
		progress = AcademyUserProgress{
			UserId:            userId,
			TierId:            tierId,
			SessionsCompleted: requiredDone,
			TierUnlocked:      true,
			CreatedDate:       time.Now().UTC(),
		}
	} else {
		progress.SessionsCompleted = requiredDone
	}

	if requiredTotal > 0 && requiredDone >= requiredTotal && !progress.TierCompleted {
		progress.TierCompleted = true
		progress.CompletedDate = time.Now().UTC()
		// Unlock next tier
		unlockNextTier(userId, tierId)
	}

	return db.Save(&progress).Error
}

// unlockNextTier finds the next tier by sort_order and marks it as unlocked for the user.
func unlockNextTier(userId, currentTierId int64) {
	currentTier := AcademyTier{}
	if err := db.Where(queryWhereID, currentTierId).First(&currentTier).Error; err != nil {
		return
	}
	nextTier := AcademyTier{}
	if err := db.Where("(org_id = ? OR org_id = 0) AND sort_order > ? AND is_active = 1",
		currentTier.OrgId, currentTier.SortOrder).Order(orderSortOrderAsc).First(&nextTier).Error; err != nil {
		return // No next tier
	}
	// Ensure progress record exists
	existing := AcademyUserProgress{}
	if err := db.Where(queryWhereUserAndTierID, userId, nextTier.Id).First(&existing).Error; err != nil {
		existing = AcademyUserProgress{
			UserId:       userId,
			TierId:       nextTier.Id,
			TierUnlocked: true,
			CreatedDate:  time.Now().UTC(),
		}
		if err := db.Save(&existing).Error; err != nil {
			log.Error(err)
		}
	} else if !existing.TierUnlocked {
		db.Table("academy_user_progress").Where(queryWhereID, existing.Id).Update("tier_unlocked", true)
	}
}

// CreateAcademyTier creates a new academy tier (admin use).
func CreateAcademyTier(t *AcademyTier) error {
	t.CreatedDate = time.Now().UTC()
	return db.Save(t).Error
}

// UpdateAcademyTier updates a tier's details.
func UpdateAcademyTier(t *AcademyTier) error {
	return db.Table("academy_tiers").Where(queryWhereID, t.Id).Updates(map[string]interface{}{
		"name":           t.Name,
		"description":    t.Description,
		"badge_icon_url": t.BadgeIconURL,
		"sort_order":     t.SortOrder,
		"is_active":      t.IsActive,
	}).Error
}

// DeleteAcademyTier removes a tier and its sessions.
func DeleteAcademyTier(id int64) error {
	db.Where(queryWhereTierID, id).Delete(&AcademySession{})
	db.Where(queryWhereTierID, id).Delete(&AcademyUserProgress{})
	return db.Where(queryWhereID, id).Delete(&AcademyTier{}).Error
}

// GetCompletedTierCount returns the number of tiers a user has completed.
func GetCompletedTierCount(userId int64) int {
	var count int
	db.Table("academy_user_progress").Where("user_id = ? AND tier_completed = 1", userId).Count(&count)
	return count
}

// GetCompletedTierSlugs returns slugs of completed tiers for badge checking.
func GetCompletedTierSlugs(userId int64) []string {
	type result struct {
		Slug string
	}
	results := []result{}
	db.Table("academy_user_progress p").
		Select("t.slug").
		Joins("JOIN academy_tiers t ON t.id = p.tier_id").
		Where("p.user_id = ? AND p.tier_completed = 1", userId).
		Scan(&results)
	slugs := make([]string, len(results))
	for i, r := range results {
		slugs[i] = r.Slug
	}
	return slugs
}

// GetTierSessionIDs returns all session IDs for a tier.
func GetTierSessionIDs(tierId int64) []int64 {
	sessions := []AcademySession{}
	db.Where(queryWhereTierID, tierId).Find(&sessions)
	ids := make([]int64, len(sessions))
	for i, s := range sessions {
		ids[i] = s.Id
	}
	return ids
}

// ParseSessionIDs parses a JSON array of session IDs from a string.
func ParseSessionIDs(jsonStr string) []int64 {
	var ids []int64
	if jsonStr == "" || jsonStr == "null" {
		return ids
	}
	if err := json.Unmarshal([]byte(jsonStr), &ids); err != nil {
		log.Errorf("ParseSessionIDs: invalid JSON %q: %v", jsonStr, err)
	}
	return ids
}
