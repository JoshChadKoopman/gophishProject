package models

import (
	"time"

	log "github.com/gophish/gophish/logger"
)

// Badge represents an achievement that users can earn.
type Badge struct {
	Id            int64     `json:"id" gorm:"column:id; primary_key:yes"`
	Slug          string    `json:"slug" gorm:"column:slug"`
	Name          string    `json:"name" gorm:"column:name"`
	Description   string    `json:"description" gorm:"column:description"`
	IconURL       string    `json:"icon_url" gorm:"column:icon_url"`
	Category      string    `json:"category" gorm:"column:category"`
	CriteriaType  string    `json:"criteria_type" gorm:"column:criteria_type"`
	CriteriaValue int       `json:"criteria_value" gorm:"column:criteria_value"`
	CreatedDate   time.Time `json:"created_date" gorm:"column:created_date"`
}

// UserBadge records a badge earned by a user.
type UserBadge struct {
	Id         int64     `json:"id" gorm:"column:id; primary_key:yes"`
	UserId     int64     `json:"user_id" gorm:"column:user_id"`
	BadgeId    int64     `json:"badge_id" gorm:"column:badge_id"`
	EarnedDate time.Time `json:"earned_date" gorm:"column:earned_date"`

	// Populated at query time
	BadgeName        string `json:"badge_name,omitempty" gorm:"-"`
	BadgeDescription string `json:"badge_description,omitempty" gorm:"-"`
	BadgeIconURL     string `json:"badge_icon_url,omitempty" gorm:"-"`
	BadgeCategory    string `json:"badge_category,omitempty" gorm:"-"`
	BadgeSlug        string `json:"badge_slug,omitempty" gorm:"-"`
}

// UserStreak tracks consecutive training activity.
type UserStreak struct {
	Id               int64     `json:"id" gorm:"column:id; primary_key:yes"`
	UserId           int64     `json:"user_id" gorm:"column:user_id"`
	StreakType       string    `json:"streak_type" gorm:"column:streak_type"`
	CurrentStreak    int       `json:"current_streak" gorm:"column:current_streak"`
	LongestStreak    int       `json:"longest_streak" gorm:"column:longest_streak"`
	LastActivityDate time.Time `json:"last_activity_date" gorm:"column:last_activity_date"`
	CreatedDate      time.Time `json:"created_date" gorm:"column:created_date"`
}

// LeaderboardEntry represents a cached leaderboard row.
type LeaderboardEntry struct {
	Id             int64     `json:"id" gorm:"column:id; primary_key:yes"`
	OrgId          int64     `json:"org_id" gorm:"column:org_id"`
	Department     string    `json:"department" gorm:"column:department"`
	UserId         int64     `json:"user_id" gorm:"column:user_id"`
	Score          int       `json:"score" gorm:"column:score"`
	Rank           int       `json:"rank" gorm:"column:rank"`
	Period         string    `json:"period" gorm:"column:period"`
	CalculatedDate time.Time `json:"calculated_date" gorm:"column:calculated_date"`

	// Populated at query time
	UserName   string `json:"user_name,omitempty" gorm:"-"`
	UserEmail  string `json:"user_email,omitempty" gorm:"-"`
	BadgeCount int    `json:"badge_count" gorm:"-"`
}

// --- Badge functions ---

// GetAllBadges returns all defined badges.
func GetAllBadges() ([]Badge, error) {
	badges := []Badge{}
	err := db.Order("category asc, criteria_value asc").Find(&badges).Error
	return badges, err
}

// GetUserBadges returns all badges earned by a user.
func GetUserBadges(userId int64) ([]UserBadge, error) {
	ubs := []UserBadge{}
	err := db.Where("user_id = ?", userId).Order("earned_date desc").Find(&ubs).Error
	if err != nil {
		return ubs, err
	}
	for i := range ubs {
		b := Badge{}
		if err := db.Where("id = ?", ubs[i].BadgeId).First(&b).Error; err == nil {
			ubs[i].BadgeName = b.Name
			ubs[i].BadgeDescription = b.Description
			ubs[i].BadgeIconURL = b.IconURL
			ubs[i].BadgeCategory = b.Category
			ubs[i].BadgeSlug = b.Slug
		}
	}
	return ubs, nil
}

// HasBadge checks if a user has already earned a specific badge.
func HasBadge(userId int64, badgeSlug string) bool {
	var count int
	db.Table("user_badges").
		Joins("JOIN badges ON badges.id = user_badges.badge_id").
		Where("user_badges.user_id = ? AND badges.slug = ?", userId, badgeSlug).
		Count(&count)
	return count > 0
}

// AwardBadge gives a badge to a user if not already earned.
func AwardBadge(userId int64, badgeSlug string) (*UserBadge, bool) {
	if HasBadge(userId, badgeSlug) {
		return nil, false
	}
	badge := Badge{}
	if err := db.Where("slug = ?", badgeSlug).First(&badge).Error; err != nil {
		return nil, false
	}
	ub := UserBadge{
		UserId:     userId,
		BadgeId:    badge.Id,
		EarnedDate: time.Now().UTC(),
	}
	if err := db.Save(&ub).Error; err != nil {
		log.Error(err)
		return nil, false
	}
	ub.BadgeName = badge.Name
	ub.BadgeSlug = badge.Slug
	return &ub, true
}

// GetUserBadgeCount returns the number of badges a user has earned.
func GetUserBadgeCount(userId int64) int {
	var count int
	db.Table("user_badges").Where("user_id = ?", userId).Count(&count)
	return count
}

// CheckAndAwardBadges evaluates all badge criteria for a user and awards any newly earned badges.
// Returns a list of newly awarded badges.
func CheckAndAwardBadges(userId int64) []UserBadge {
	awarded := []UserBadge{}

	// Count completed courses
	var coursesCompleted int
	db.Table("course_progress").Where("user_id = ? AND status = 'complete'", userId).Count(&coursesCompleted)

	// Count passed quizzes
	var quizzesPassed int
	db.Table("quiz_attempts").Where("user_id = ? AND passed = 1", userId).Count(&quizzesPassed)

	// Check for perfect quiz scores
	var perfectQuizzes int
	db.Table("quiz_attempts").Where("user_id = ? AND passed = 1 AND score = total_questions", userId).Count(&perfectQuizzes)

	// Course completion badges
	if coursesCompleted >= 1 {
		if ub, ok := AwardBadge(userId, "first_course"); ok {
			awarded = append(awarded, *ub)
		}
	}
	if coursesCompleted >= 5 {
		if ub, ok := AwardBadge(userId, "five_courses"); ok {
			awarded = append(awarded, *ub)
		}
	}
	if coursesCompleted >= 10 {
		if ub, ok := AwardBadge(userId, "ten_courses"); ok {
			awarded = append(awarded, *ub)
		}
	}

	// Quiz badges
	if perfectQuizzes >= 1 {
		if ub, ok := AwardBadge(userId, "perfect_quiz"); ok {
			awarded = append(awarded, *ub)
		}
	}
	if quizzesPassed >= 5 {
		if ub, ok := AwardBadge(userId, "five_quizzes"); ok {
			awarded = append(awarded, *ub)
		}
	}

	// Academy tier badges
	completedSlugs := GetCompletedTierSlugs(userId)
	tierBadgeMap := map[string]string{
		"bronze":   "bronze_tier",
		"silver":   "silver_tier",
		"gold":     "gold_tier",
		"platinum": "platinum_tier",
	}
	for _, slug := range completedSlugs {
		if badgeSlug, ok := tierBadgeMap[slug]; ok {
			if ub, ok := AwardBadge(userId, badgeSlug); ok {
				awarded = append(awarded, *ub)
			}
		}
	}

	// Streak badges
	streak, err := GetUserStreak(userId, "weekly")
	if err == nil {
		if streak.CurrentStreak >= 3 {
			if ub, ok := AwardBadge(userId, "week_streak_3"); ok {
				awarded = append(awarded, *ub)
			}
		}
		if streak.CurrentStreak >= 8 {
			if ub, ok := AwardBadge(userId, "week_streak_8"); ok {
				awarded = append(awarded, *ub)
			}
		}
		if streak.CurrentStreak >= 16 {
			if ub, ok := AwardBadge(userId, "week_streak_16"); ok {
				awarded = append(awarded, *ub)
			}
		}
	}

	// Compliance cert badges
	complianceCount := GetComplianceCertCount(userId)
	if complianceCount >= 1 {
		if ub, ok := AwardBadge(userId, "compliance_cert"); ok {
			awarded = append(awarded, *ub)
		}
	}

	return awarded
}

// --- Streak functions ---

// GetUserStreak returns a user's streak by type.
func GetUserStreak(userId int64, streakType string) (UserStreak, error) {
	s := UserStreak{}
	err := db.Where("user_id = ? AND streak_type = ?", userId, streakType).First(&s).Error
	return s, err
}

// GetUserStreaks returns all streaks for a user.
func GetUserStreaks(userId int64) ([]UserStreak, error) {
	streaks := []UserStreak{}
	err := db.Where("user_id = ?", userId).Find(&streaks).Error
	return streaks, err
}

// RecordTrainingActivity updates the user's weekly streak.
// Call this when a user completes a course or quiz.
func RecordTrainingActivity(userId int64) {
	now := time.Now().UTC()
	streak, err := GetUserStreak(userId, "weekly")
	if err != nil {
		// Create new streak
		streak = UserStreak{
			UserId:           userId,
			StreakType:       "weekly",
			CurrentStreak:    1,
			LongestStreak:    1,
			LastActivityDate: now,
			CreatedDate:      now,
		}
		if err := db.Save(&streak).Error; err != nil {
			log.Error(err)
		}
		return
	}

	// Calculate weeks since last activity
	if streak.LastActivityDate.IsZero() {
		streak.CurrentStreak = 1
	} else {
		daysSince := int(now.Sub(streak.LastActivityDate).Hours() / 24)
		if daysSince <= 7 {
			// Same week or consecutive week — already counted or increment
			_, lastWeek := streak.LastActivityDate.ISOWeek()
			_, thisWeek := now.ISOWeek()
			if thisWeek != lastWeek {
				streak.CurrentStreak++
			}
			// Same week = no change to streak count
		} else if daysSince <= 14 {
			// Within grace period — consecutive
			streak.CurrentStreak++
		} else {
			// Streak broken — reset
			streak.CurrentStreak = 1
		}
	}

	if streak.CurrentStreak > streak.LongestStreak {
		streak.LongestStreak = streak.CurrentStreak
	}
	streak.LastActivityDate = now
	if err := db.Save(&streak).Error; err != nil {
		log.Error(err)
	}
}

// ExpireStaleStreaks resets streaks for users who haven't had activity in over 14 days.
func ExpireStaleStreaks() {
	cutoff := time.Now().UTC().AddDate(0, 0, -14)
	db.Table("user_streaks").
		Where("last_activity_date < ? AND current_streak > 0", cutoff).
		Updates(map[string]interface{}{"current_streak": 0})
}

// --- Leaderboard functions ---

// GetLeaderboard returns the cached leaderboard for an org and period.
func GetLeaderboard(orgId int64, period string, department string, limit int) ([]LeaderboardEntry, error) {
	entries := []LeaderboardEntry{}
	q := db.Where("org_id = ? AND period = ?", orgId, period)
	if department != "" {
		q = q.Where("department = ?", department)
	}
	if limit <= 0 {
		limit = 50
	}
	err := q.Order("`rank` asc").Limit(limit).Find(&entries).Error
	if err != nil {
		return entries, err
	}
	// Enrich with user info
	for i := range entries {
		u := User{}
		if err := db.Where("id = ?", entries[i].UserId).First(&u).Error; err == nil {
			entries[i].UserName = u.FirstName + " " + u.LastName
			entries[i].UserEmail = u.Username
		}
		entries[i].BadgeCount = GetUserBadgeCount(entries[i].UserId)
	}
	return entries, nil
}

// GetUserLeaderboardPosition returns a user's position in the leaderboard.
func GetUserLeaderboardPosition(userId int64, orgId int64, period string) (*LeaderboardEntry, error) {
	entry := LeaderboardEntry{}
	err := db.Where("user_id = ? AND org_id = ? AND period = ?", userId, orgId, period).First(&entry).Error
	if err != nil {
		return nil, err
	}
	u := User{}
	if err := db.Where("id = ?", userId).First(&u).Error; err == nil {
		entry.UserName = u.FirstName + " " + u.LastName
		entry.UserEmail = u.Username
	}
	entry.BadgeCount = GetUserBadgeCount(userId)
	return &entry, nil
}

// RecalculateLeaderboard recalculates the leaderboard for an org.
// Score formula: (courses_completed * 10) + (quizzes_passed * 15) + (badges * 20) + (streak_weeks * 5) + (compliance_certs * 25)
func RecalculateLeaderboard(orgId int64) error {
	// Get all users in org
	users := []User{}
	if err := db.Where("org_id = ?", orgId).Find(&users).Error; err != nil {
		return err
	}

	now := time.Now().UTC()

	// Clear existing cache for this org
	db.Where("org_id = ?", orgId).Delete(&LeaderboardEntry{})

	type userScore struct {
		userId     int64
		department string
		score      int
	}

	scores := []userScore{}
	for _, u := range users {
		var coursesCompleted int
		db.Table("course_progress").Where("user_id = ? AND status = 'complete'", u.Id).Count(&coursesCompleted)

		var quizzesPassed int
		db.Table("quiz_attempts").Where("user_id = ? AND passed = 1", u.Id).Count(&quizzesPassed)

		badgeCount := GetUserBadgeCount(u.Id)
		complianceCount := GetComplianceCertCount(u.Id)

		streakWeeks := 0
		streak, err := GetUserStreak(u.Id, "weekly")
		if err == nil {
			streakWeeks = streak.CurrentStreak
		}

		score := (coursesCompleted * 10) + (quizzesPassed * 15) + (badgeCount * 20) + (streakWeeks * 5) + (complianceCount * 25)
		scores = append(scores, userScore{userId: u.Id, department: u.Department, score: score})
	}

	// Sort by score descending (simple bubble sort for small datasets)
	for i := 0; i < len(scores); i++ {
		for j := i + 1; j < len(scores); j++ {
			if scores[j].score > scores[i].score {
				scores[i], scores[j] = scores[j], scores[i]
			}
		}
	}

	// Insert all_time entries
	for rank, s := range scores {
		entry := LeaderboardEntry{
			OrgId:          orgId,
			Department:     s.department,
			UserId:         s.userId,
			Score:          s.score,
			Rank:           rank + 1,
			Period:         "all_time",
			CalculatedDate: now,
		}
		if err := db.Save(&entry).Error; err != nil {
			log.Error(err)
		}
	}

	return nil
}

// RecalculateAllLeaderboards recalculates leaderboards for all orgs.
func RecalculateAllLeaderboards() {
	type orgRow struct {
		Id int64
	}
	orgs := []orgRow{}
	db.Table("organizations").Select("id").Scan(&orgs)
	for _, o := range orgs {
		if err := RecalculateLeaderboard(o.Id); err != nil {
			log.Error(err)
		}
	}
}
