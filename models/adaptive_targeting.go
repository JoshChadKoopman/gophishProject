package models

import (
	"math/rand"
	"sort"
	"time"

	log "github.com/gophish/gophish/logger"
)

// Phishing template categories used for vulnerability analysis.
const (
	CategoryCredentialHarvesting = "Credential Harvesting"
	CategoryBEC                  = "Business Email Compromise"
	CategoryDeliveryNotification = "Delivery Notification"
	CategoryITHelpdesk           = "IT Helpdesk"
	CategoryHRPayroll            = "HR / Payroll"
	CategorySocialEngineering    = "Social Engineering"
	CategoryQRCodePhishing       = "QR Code Phishing"
	CategorySMSPhishing          = "SMS Phishing"
)

// AllCategories lists every known phishing category for iteration.
var AllCategories = []string{
	CategoryCredentialHarvesting,
	CategoryBEC,
	CategoryDeliveryNotification,
	CategoryITHelpdesk,
	CategoryHRPayroll,
	CategorySocialEngineering,
	CategoryQRCodePhishing,
	CategorySMSPhishing,
}

// UserTargetingProfile holds the AI-driven targeting recommendation for a user.
type UserTargetingProfile struct {
	UserId                      int64           `json:"user_id"`
	Email                       string          `json:"email"`
	RecommendedDifficulty       int             `json:"recommended_difficulty"`
	EffectiveTrainingDifficulty int             `json:"effective_training_difficulty"` // respects manual override
	TrainingDifficultyMode      string          `json:"training_difficulty_mode"`      // "adaptive" or "manual"
	WeakCategories              []CategoryScore `json:"weak_categories"`
	StrongCategories            []CategoryScore `json:"strong_categories"`
	TotalSimulations            int64           `json:"total_simulations"`
	OverallClickRate            float64         `json:"overall_click_rate"`
	OverallSubmitRate           float64         `json:"overall_submit_rate"`
	OverallReportRate           float64         `json:"overall_report_rate"`
	BRSComposite                float64         `json:"brs_composite"`
	TrendDirection              string          `json:"trend_direction"` // "improving", "declining", "stable"
	LastSimulationDate          *time.Time      `json:"last_simulation_date"`
	RecentCategories            []string        `json:"recent_categories"` // categories used in last 3 campaigns, to avoid repeats
	// Send-time optimization (Feature 1 enhancement)
	PreferredSendDay   string  `json:"preferred_send_day,omitempty"`
	PreferredSendHour  int     `json:"preferred_send_hour,omitempty"`
	SendTimeConfidence float64 `json:"send_time_confidence,omitempty"` // 0-1
	// Department threat context (Feature 1 enhancement)
	Department         string   `json:"department,omitempty"`
	DepartmentThreats  []string `json:"department_threats,omitempty"`
	DepartmentRiskMult float64  `json:"department_risk_multiplier,omitempty"`
}

// CategoryScore tracks a user's click/submit rate for a specific phishing category.
type CategoryScore struct {
	Category   string  `json:"category"`
	Total      int64   `json:"total"`
	Clicked    int64   `json:"clicked"`
	Submitted  int64   `json:"submitted"`
	Reported   int64   `json:"reported"`
	ClickRate  float64 `json:"click_rate"`
	SubmitRate float64 `json:"submit_rate"`
	Score      float64 `json:"score"` // 0 = always clicks (weak), 100 = never clicks (strong)
}

// GetUserTargetingProfile builds a comprehensive targeting profile for a user
// by analyzing their BRS, simulation history per category, and recent trends.
func GetUserTargetingProfile(userId int64) (*UserTargetingProfile, error) {
	user, err := GetUser(userId)
	if err != nil {
		return nil, err
	}

	profile := &UserTargetingProfile{
		UserId: userId,
		Email:  user.Email,
	}

	// 1. Get BRS data
	brs, err := GetUserBRS(userId)
	if err == nil {
		profile.BRSComposite = brs.CompositeScore
	}

	// 2. Overall simulation stats
	overallStats := getOverallSimStats(userId, user.OrgId, user.Email)
	profile.TotalSimulations = overallStats.Total
	profile.OverallClickRate = overallStats.ClickRate
	profile.OverallSubmitRate = overallStats.SubmitRate
	profile.OverallReportRate = overallStats.ReportRate

	// 3. Per-category breakdown
	weak, strong := getCategoryBreakdown(userId, user.OrgId, user.Email)
	profile.WeakCategories = weak
	profile.StrongCategories = strong

	// 4. Trend direction
	profile.TrendDirection = getTrendDirection(userId, user.OrgId, user.Email)

	// 5. Recommended difficulty
	profile.RecommendedDifficulty = recommendDifficulty(profile)

	// 5a. Effective training difficulty (respects manual override)
	profile.TrainingDifficultyMode = user.TrainingDifficultyMode
	if profile.TrainingDifficultyMode == "" {
		profile.TrainingDifficultyMode = DifficultyModeAdaptive
	}
	// Compute effective difficulty inline to avoid mutual recursion with GetEffectiveDifficulty
	if profile.TrainingDifficultyMode == DifficultyModeManual &&
		user.TrainingDifficultyManual >= DifficultyEasy &&
		user.TrainingDifficultyManual <= DifficultySophisticated {
		profile.EffectiveTrainingDifficulty = user.TrainingDifficultyManual
	} else {
		profile.EffectiveTrainingDifficulty = profile.RecommendedDifficulty
	}

	// 6. Last simulation date
	profile.LastSimulationDate = getLastSimDate(userId, user.OrgId, user.Email)

	// 7. Recent categories (last 3 campaigns) to avoid repeats
	profile.RecentCategories = getRecentCategories(userId, user.OrgId, user.Email, 3)

	// 8. Send-time optimization
	sendTimeProfile, err := GetSendTimeProfile(userId)
	if err == nil && sendTimeProfile.ConfidenceScore >= 0.3 {
		profile.PreferredSendDay = sendTimeProfile.OptimalDay
		profile.PreferredSendHour = sendTimeProfile.OptimalHourStart
		profile.SendTimeConfidence = sendTimeProfile.ConfidenceScore
	}

	// 9. Department threat profile
	if user.Department != "" {
		profile.Department = user.Department
		deptTP := GetDepartmentThreatProfile(user.Department)
		profile.DepartmentThreats = deptTP.AttackVectors
		profile.DepartmentRiskMult = deptTP.RiskMultiplier
	}

	return profile, nil
}

type simStats struct {
	Total      int64
	Clicked    int64
	Submitted  int64
	Reported   int64
	ClickRate  float64
	SubmitRate float64
	ReportRate float64
}

func getOverallSimStats(userId, orgId int64, email string) simStats {
	var row struct {
		Total     int64
		Clicked   int64
		Submitted int64
		Reported  int64
	}
	err := db.Raw(`
		SELECT COUNT(*) as total,
			SUM(CASE WHEN r.status IN (?, ?) THEN 1 ELSE 0 END) as clicked,
			SUM(CASE WHEN r.status = ? THEN 1 ELSE 0 END) as submitted,
			SUM(CASE WHEN r.reported = 1 THEN 1 ELSE 0 END) as reported
		FROM results r
		JOIN campaigns c ON r.campaign_id = c.id
		WHERE c.org_id = ? AND r.email = ?
	`, EventClicked, EventDataSubmit, EventDataSubmit, orgId, email).Scan(&row).Error

	s := simStats{Total: row.Total, Clicked: row.Clicked, Submitted: row.Submitted, Reported: row.Reported}
	if err != nil || row.Total == 0 {
		return s
	}
	s.ClickRate = float64(row.Clicked) / float64(row.Total)
	s.SubmitRate = float64(row.Submitted) / float64(row.Total)
	s.ReportRate = float64(row.Reported) / float64(row.Total)
	return s
}

// getCategoryBreakdown analyzes the user's click/submit rate per template category.
// Returns weak categories (sorted worst first) and strong categories.
func getCategoryBreakdown(userId, orgId int64, email string) (weak, strong []CategoryScore) {
	type catRow struct {
		Category  string
		Total     int64
		Clicked   int64
		Submitted int64
		Reported  int64
	}
	var rows []catRow
	err := db.Raw(`
		SELECT t.category, COUNT(*) as total,
			SUM(CASE WHEN r.status IN (?, ?) THEN 1 ELSE 0 END) as clicked,
			SUM(CASE WHEN r.status = ? THEN 1 ELSE 0 END) as submitted,
			SUM(CASE WHEN r.reported = 1 THEN 1 ELSE 0 END) as reported
		FROM results r
		JOIN campaigns c ON r.campaign_id = c.id
		JOIN templates t ON c.template_id = t.id
		WHERE c.org_id = ? AND r.email = ? AND t.category != '' AND t.category IS NOT NULL
		GROUP BY t.category
	`, EventClicked, EventDataSubmit, EventDataSubmit, orgId, email).Scan(&rows).Error
	if err != nil {
		log.Errorf("adaptive_targeting: category breakdown error: %v", err)
		return nil, nil
	}

	var scores []CategoryScore
	for _, r := range rows {
		if r.Total == 0 {
			continue
		}
		clickRate := float64(r.Clicked) / float64(r.Total)
		submitRate := float64(r.Submitted) / float64(r.Total)
		// Score: lower click/submit = better. 0-100 scale.
		score := (1 - clickRate*0.6 - submitRate*0.4) * 100
		if score < 0 {
			score = 0
		}
		scores = append(scores, CategoryScore{
			Category:   r.Category,
			Total:      r.Total,
			Clicked:    r.Clicked,
			Submitted:  r.Submitted,
			Reported:   r.Reported,
			ClickRate:  clickRate,
			SubmitRate: submitRate,
			Score:      score,
		})
	}

	// Sort by score ascending (weakest first)
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].Score < scores[j].Score
	})

	// Split into weak (below 60) and strong (60+)
	for _, s := range scores {
		if s.Score < 60 {
			weak = append(weak, s)
		} else {
			strong = append(strong, s)
		}
	}
	return weak, strong
}

func getTrendDirection(userId, orgId int64, email string) string {
	type periodRow struct {
		Total   int64
		Clicked int64
	}
	now := time.Now()
	recent30 := now.AddDate(0, 0, -30).Format(DateFormat)
	older90 := now.AddDate(0, 0, -90).Format(DateFormat)

	var recent, older periodRow
	db.Raw(`
		SELECT COUNT(*) as total,
			SUM(CASE WHEN r.status IN (?, ?) THEN 1 ELSE 0 END) as clicked
		FROM results r JOIN campaigns c ON r.campaign_id = c.id
		WHERE c.org_id = ? AND r.email = ? AND DATE(c.created_date) >= ?
	`, EventClicked, EventDataSubmit, orgId, email, recent30).Scan(&recent)

	db.Raw(`
		SELECT COUNT(*) as total,
			SUM(CASE WHEN r.status IN (?, ?) THEN 1 ELSE 0 END) as clicked
		FROM results r JOIN campaigns c ON r.campaign_id = c.id
		WHERE c.org_id = ? AND r.email = ? AND DATE(c.created_date) >= ? AND DATE(c.created_date) < ?
	`, EventClicked, EventDataSubmit, orgId, email, older90, recent30).Scan(&older)

	if recent.Total < 2 || older.Total < 2 {
		return "stable"
	}
	recentRate := float64(recent.Clicked) / float64(recent.Total)
	olderRate := float64(older.Clicked) / float64(older.Total)
	diff := olderRate - recentRate
	if diff > 0.1 {
		return "improving"
	}
	if diff < -0.1 {
		return "declining"
	}
	return "stable"
}

// recommendDifficulty maps a user's behavioral profile to an optimal difficulty level.
//
// Logic:
//   - New users (< 3 simulations): start at Easy (1)
//   - High click rate (> 50%) or low BRS (< 30): Easy (1)
//   - Moderate click rate (20-50%) or medium BRS (30-60): Medium (2)
//   - Low click rate (5-20%) or good BRS (60-80): Hard (3)
//   - Very low click rate (< 5%) and high BRS (> 80): Sophisticated (4)
//   - Declining trend: drop one level to reinforce basics
//   - Improving trend: bump up one level to keep challenging
func recommendDifficulty(p *UserTargetingProfile) int {
	// New users start easy
	if p.TotalSimulations < 3 {
		return 1
	}

	var level int

	// Primary signal: BRS composite score
	switch {
	case p.BRSComposite < 30:
		level = 1
	case p.BRSComposite < 55:
		level = 2
	case p.BRSComposite < 80:
		level = 3
	default:
		level = 4
	}

	// Secondary signal: click rate can override up/down
	switch {
	case p.OverallClickRate > 0.50:
		if level > 1 {
			level = 1
		}
	case p.OverallClickRate > 0.30:
		if level > 2 {
			level = 2
		}
	case p.OverallClickRate < 0.05 && p.TotalSimulations >= 10:
		if level < 4 {
			level++
		}
	}

	// Trend modifier: adjust by ±1
	switch p.TrendDirection {
	case "declining":
		if level > 1 {
			level--
		}
	case "improving":
		if level < 4 {
			level++
		}
	}

	return level
}

func getLastSimDate(userId, orgId int64, email string) *time.Time {
	type row struct {
		LastDate *time.Time
	}
	var r row
	db.Raw(`
		SELECT MAX(r.send_date) as last_date
		FROM results r
		JOIN campaigns c ON r.campaign_id = c.id
		WHERE c.org_id = ? AND r.email = ?
	`, orgId, email).Scan(&r)
	return r.LastDate
}

// getRecentCategories returns the categories used in the user's N most recent campaigns.
func getRecentCategories(userId, orgId int64, email string, n int) []string {
	type catRow struct {
		Category string
	}
	var rows []catRow
	db.Raw(`
		SELECT DISTINCT t.category
		FROM results r
		JOIN campaigns c ON r.campaign_id = c.id
		JOIN templates t ON c.template_id = t.id
		WHERE c.org_id = ? AND r.email = ? AND t.category != '' AND t.category IS NOT NULL
		ORDER BY c.created_date DESC
		LIMIT ?
	`, orgId, email, n).Scan(&rows)

	cats := make([]string, 0, len(rows))
	for _, r := range rows {
		cats = append(cats, r.Category)
	}
	return cats
}

// SelectTemplate picks the best template for a user from a list of available templates.
// Strategy:
//  1. Filter to templates matching the recommended difficulty (±1 level)
//  2. Prefer templates in the user's weak categories
//  3. Avoid categories used in the user's last 3 campaigns
//  4. Fall back to random if no data available
func SelectTemplate(profile *UserTargetingProfile, templates []Template) Template {
	if len(templates) == 0 {
		return Template{}
	}
	if profile == nil || profile.TotalSimulations < 3 {
		return templates[rand.Intn(len(templates))]
	}

	recentSet := buildStringSet(profile.RecentCategories)
	weakSet := buildWeakMap(profile.WeakCategories)

	type scored struct {
		template Template
		score    float64
	}
	candidates := make([]scored, 0, len(templates))
	for _, t := range templates {
		s := scoreTemplate(t, profile.RecommendedDifficulty, weakSet, recentSet)
		candidates = append(candidates, scored{template: t, score: s})
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].score > candidates[j].score
	})
	return candidates[0].template
}

// scoreTemplate computes a desirability score for a single template.
func scoreTemplate(t Template, difficulty int, weakSet map[string]float64, recentSet map[string]bool) float64 {
	s := scoreDifficultyMatch(t.DifficultyLevel, difficulty)
	s += scoreCategoryMatch(t.Category, weakSet, recentSet)
	s += rand.Float64() * 2 // jitter
	return s
}

// scoreDifficultyMatch awards points for how closely the template matches the target difficulty.
func scoreDifficultyMatch(templateLevel, targetLevel int) float64 {
	switch abs(templateLevel - targetLevel) {
	case 0:
		return 10
	case 1:
		return 5
	default:
		return 0
	}
}

// scoreCategoryMatch awards/penalizes based on category weakness and recency.
func scoreCategoryMatch(category string, weakSet map[string]float64, recentSet map[string]bool) float64 {
	if category == "" {
		return 0
	}
	s := 0.0
	if clickRate, isWeak := weakSet[category]; isWeak {
		s += 8 + clickRate*5
	}
	if recentSet[category] {
		s -= 6
	}
	return s
}

// buildStringSet creates a set from a string slice.
func buildStringSet(items []string) map[string]bool {
	m := make(map[string]bool, len(items))
	for _, item := range items {
		m[item] = true
	}
	return m
}

// buildWeakMap creates a map of category -> click rate from weak category scores.
func buildWeakMap(weak []CategoryScore) map[string]float64 {
	m := make(map[string]float64, len(weak))
	for _, w := range weak {
		m[w.Category] = w.ClickRate
	}
	return m
}

// SelectLibraryTemplate picks the best library template for a user.
// Used when the org has no custom templates or for AI-augmented autopilot.
func SelectLibraryTemplate(profile *UserTargetingProfile) *LibraryTemplate {
	all := GetTemplateLibrary("", 0)
	if len(all) == 0 {
		return nil
	}
	if profile == nil || profile.TotalSimulations < 3 {
		t := all[rand.Intn(len(all))]
		return &t
	}

	recentSet := buildStringSet(profile.RecentCategories)
	weakSet := buildWeakMap(profile.WeakCategories)

	type scored struct {
		template LibraryTemplate
		score    float64
	}
	candidates := make([]scored, 0, len(all))

	for _, t := range all {
		s := scoreDifficultyMatch(t.DifficultyLevel, profile.RecommendedDifficulty)
		s += scoreCategoryMatch(t.Category, weakSet, recentSet)
		s += rand.Float64() * 2
		candidates = append(candidates, scored{template: t, score: s})
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].score > candidates[j].score
	})

	return &candidates[0].template
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
