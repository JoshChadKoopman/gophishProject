package models

import (
	"math"
	"sort"
	"time"

	log "github.com/gophish/gophish/logger"
)

// ── Send-Time Optimization ──────────────────────────────────────
// Analyzes per-user click timestamps to determine when they are most
// susceptible to phishing, then recommends optimal send windows.

// SendTimeProfile holds the computed optimal send window for a user.
type SendTimeProfile struct {
	UserId             int64              `json:"user_id"`
	Email              string             `json:"email"`
	OptimalDay         string             `json:"optimal_day"`           // "Monday" … "Sunday"
	OptimalHourStart   int                `json:"optimal_hour_start"`    // 0-23
	OptimalHourEnd     int                `json:"optimal_hour_end"`      // 0-23
	ConfidenceScore    float64            `json:"confidence_score"`      // 0-1 (higher = more data)
	SampleSize         int                `json:"sample_size"`           // total events analysed
	HourlyDistribution [24]float64        `json:"hourly_distribution"`   // normalised click-probability per hour
	DailyDistribution  [7]float64         `json:"daily_distribution"`    // normalised click-probability per weekday (Mon=0)
	Recommendation     string             `json:"recommendation"`        // human-readable
	DayBreakdown       []DayClickProfile  `json:"day_breakdown"`
}

// DayClickProfile is the click probability for a specific day of the week.
type DayClickProfile struct {
	Day       string  `json:"day"`
	EventRate float64 `json:"event_rate"` // relative probability 0-1
	Events    int     `json:"events"`
}

// GetSendTimeProfile computes the optimal send window for a user based on
// their historical click timestamps (EventClicked / EventDataSubmit events).
func GetSendTimeProfile(userId int64) (*SendTimeProfile, error) {
	user, err := GetUser(userId)
	if err != nil {
		return nil, err
	}

	profile := &SendTimeProfile{
		UserId: userId,
		Email:  user.Email,
	}

	// Fetch click/submit event timestamps for this user's email
	type eventRow struct {
		EventTime time.Time
	}
	var events []eventRow
	err = db.Raw(`
		SELECT e.time as event_time
		FROM events e
		JOIN campaigns c ON e.campaign_id = c.id
		JOIN results r ON r.campaign_id = c.id AND r.email = ?
		WHERE c.org_id = ? AND e.email = ? AND e.message IN (?, ?)
		ORDER BY e.time DESC
		LIMIT 200
	`, user.Email, user.OrgId, user.Email, EventClicked, EventDataSubmit).Scan(&events).Error
	if err != nil {
		return nil, err
	}

	profile.SampleSize = len(events)
	if len(events) < 3 {
		// Not enough data — return business-hours default
		profile.OptimalDay = "Tuesday"
		profile.OptimalHourStart = 10
		profile.OptimalHourEnd = 14
		profile.ConfidenceScore = 0.1
		profile.Recommendation = "Insufficient data (< 3 events). Using default business-hours window."
		return profile, nil
	}

	// Count events per hour and per weekday
	hourCounts := [24]int{}
	dayCounts := [7]int{}
	for _, ev := range events {
		hourCounts[ev.EventTime.Hour()]++
		dayCounts[ev.EventTime.Weekday()]++ // Sunday=0
	}

	// Normalise into probability distributions
	total := float64(len(events))
	for h := 0; h < 24; h++ {
		profile.HourlyDistribution[h] = math.Round(float64(hourCounts[h])/total*1000) / 1000
	}

	dayNames := []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}
	for d := 0; d < 7; d++ {
		profile.DailyDistribution[d] = math.Round(float64(dayCounts[d])/total*1000) / 1000
		profile.DayBreakdown = append(profile.DayBreakdown, DayClickProfile{
			Day:       dayNames[d],
			EventRate: profile.DailyDistribution[d],
			Events:    dayCounts[d],
		})
	}

	// Find peak hour window (3-hour block with highest click density)
	bestStart, bestSum := 0, 0.0
	for start := 0; start < 24; start++ {
		windowSum := 0.0
		for offset := 0; offset < 3; offset++ {
			h := (start + offset) % 24
			windowSum += profile.HourlyDistribution[h]
		}
		if windowSum > bestSum {
			bestSum = windowSum
			bestStart = start
		}
	}
	profile.OptimalHourStart = bestStart
	profile.OptimalHourEnd = (bestStart + 3) % 24

	// Find peak day
	peakDay, peakDayVal := 0, 0.0
	for d := 0; d < 7; d++ {
		if profile.DailyDistribution[d] > peakDayVal {
			peakDayVal = profile.DailyDistribution[d]
			peakDay = d
		}
	}
	profile.OptimalDay = dayNames[peakDay]

	// Confidence based on sample size
	switch {
	case len(events) >= 30:
		profile.ConfidenceScore = 0.95
	case len(events) >= 15:
		profile.ConfidenceScore = 0.75
	case len(events) >= 5:
		profile.ConfidenceScore = 0.50
	default:
		profile.ConfidenceScore = 0.25
	}

	profile.Recommendation = generateSendTimeRecommendation(profile)
	return profile, nil
}

func generateSendTimeRecommendation(p *SendTimeProfile) string {
	if p.ConfidenceScore < 0.3 {
		return "Limited click history. Recommend sending during standard business hours (Tue-Thu, 9-14)."
	}
	return "Based on " + string(rune('0'+byte(p.SampleSize/10))) + string(rune('0'+byte(p.SampleSize%10))) +
		"+ interaction events, this user is most susceptible on " + p.OptimalDay +
		" between " + hourLabel(p.OptimalHourStart) + " and " + hourLabel(p.OptimalHourEnd) + "."
}

func hourLabel(h int) string {
	if h < 10 {
		return "0" + string(rune('0'+byte(h))) + ":00"
	}
	return string(rune('0'+byte(h/10))) + string(rune('0'+byte(h%10))) + ":00"
}

// ── Department Threat Profiles ──────────────────────────────────
// Maps departments to their most relevant attack vectors so AI templates
// can be hyper-targeted based on role/department context.

// CategorySupplyChain is the attack category for supply-chain compromise scenarios.
const CategorySupplyChain = "Supply Chain"

// DepartmentThreatProfile maps a department to its most relevant attack vectors.
type DepartmentThreatProfile struct {
	Department        string                `json:"department"`
	PrimaryThreats    []DepartmentThreat    `json:"primary_threats"`
	AttackVectors     []string              `json:"attack_vectors"`       // ordered by relevance
	ContextualTriggers []string             `json:"contextual_triggers"`  // e.g. "quarterly close", "audit season"
	RiskMultiplier    float64               `json:"risk_multiplier"`      // 1.0 = baseline, >1 = higher risk
}

// DepartmentThreat is a specific threat relevant to a department.
type DepartmentThreat struct {
	Category    string  `json:"category"`
	Relevance   float64 `json:"relevance"` // 0-1
	Description string  `json:"description"`
}

// departmentThreatMap is the built-in mapping of departments to their threat profiles.
var departmentThreatMap = map[string]DepartmentThreatProfile{
	"Finance": {
		Department:    "Finance",
		RiskMultiplier: 1.5,
		PrimaryThreats: []DepartmentThreat{
			{Category: CategoryBEC, Relevance: 0.95, Description: "Wire transfer fraud, invoice manipulation, vendor payment redirect"},
			{Category: CategoryCredentialHarvesting, Relevance: 0.80, Description: "Banking portal credential theft, ERP system access"},
			{Category: CategoryHRPayroll, Relevance: 0.70, Description: "Payroll diversion, tax form fraud"},
		},
		AttackVectors:      []string{CategoryBEC, CategoryCredentialHarvesting, CategoryHRPayroll},
		ContextualTriggers: []string{"quarter-end close", "audit season", "tax filing deadline", "budget approval cycle"},
	},
	"Human Resources": {
		Department:    "Human Resources",
		RiskMultiplier: 1.3,
		PrimaryThreats: []DepartmentThreat{
			{Category: CategoryHRPayroll, Relevance: 0.95, Description: "Benefits enrollment fraud, W-2/tax phishing"},
			{Category: CategoryBEC, Relevance: 0.85, Description: "CEO requesting employee PII, organizational chart exfiltration"},
			{Category: CategorySocialEngineering, Relevance: 0.75, Description: "Fake job applicant with malicious attachments"},
		},
		AttackVectors:      []string{CategoryHRPayroll, CategoryBEC, CategorySocialEngineering},
		ContextualTriggers: []string{"open enrollment", "annual review cycle", "new hire onboarding", "offboarding"},
	},
	"IT": {
		Department:    "IT",
		RiskMultiplier: 1.4,
		PrimaryThreats: []DepartmentThreat{
			{Category: CategoryCredentialHarvesting, Relevance: 0.90, Description: "Admin credential phishing, SSO portal spoofing"},
			{Category: CategorySupplyChain, Relevance: 0.85, Description: "Compromised vendor update, fake security advisory"},
			{Category: CategoryITHelpdesk, Relevance: 0.70, Description: "Spoofed helpdesk ticket, fake vulnerability scanner notification"},
		},
		AttackVectors:      []string{CategoryCredentialHarvesting, CategorySupplyChain, CategoryITHelpdesk},
		ContextualTriggers: []string{"patch Tuesday", "major software release", "security incident response", "cloud migration"},
	},
	"Engineering": {
		Department:    "Engineering",
		RiskMultiplier: 1.2,
		PrimaryThreats: []DepartmentThreat{
			{Category: CategorySupplyChain, Relevance: 0.90, Description: "Compromised package manager, fake GitHub notification"},
			{Category: CategoryCredentialHarvesting, Relevance: 0.85, Description: "CI/CD credential theft, cloud console phishing"},
			{Category: CategorySocialEngineering, Relevance: 0.65, Description: "Fake recruiter, conference invitation"},
		},
		AttackVectors:      []string{CategorySupplyChain, CategoryCredentialHarvesting, CategorySocialEngineering},
		ContextualTriggers: []string{"sprint planning", "deployment freeze", "conference season", "open source advisory"},
	},
	"Executive": {
		Department:    "Executive",
		RiskMultiplier: 2.0,
		PrimaryThreats: []DepartmentThreat{
			{Category: CategoryBEC, Relevance: 0.95, Description: "Whaling attacks, board communication spoofing"},
			{Category: CategoryCredentialHarvesting, Relevance: 0.85, Description: "Executive account takeover, M365 admin phishing"},
			{Category: CategorySocialEngineering, Relevance: 0.80, Description: "Fake legal subpoena, regulatory action notification"},
		},
		AttackVectors:      []string{CategoryBEC, CategoryCredentialHarvesting, CategorySocialEngineering},
		ContextualTriggers: []string{"board meeting", "M&A activity", "earnings report", "regulatory filing"},
	},
	"Sales": {
		Department:    "Sales",
		RiskMultiplier: 1.1,
		PrimaryThreats: []DepartmentThreat{
			{Category: CategoryCredentialHarvesting, Relevance: 0.85, Description: "CRM credential theft, Salesforce/HubSpot phishing"},
			{Category: CategorySocialEngineering, Relevance: 0.80, Description: "Fake customer inquiry, LinkedIn connection bait"},
			{Category: CategoryBEC, Relevance: 0.65, Description: "Fake deal approval, commission statement fraud"},
		},
		AttackVectors:      []string{CategoryCredentialHarvesting, CategorySocialEngineering, CategoryBEC},
		ContextualTriggers: []string{"end of quarter", "pipeline review", "conference season", "territory change"},
	},
	"Legal": {
		Department:    "Legal",
		RiskMultiplier: 1.3,
		PrimaryThreats: []DepartmentThreat{
			{Category: CategoryBEC, Relevance: 0.90, Description: "Fake opposing counsel, settlement document phishing"},
			{Category: CategoryCredentialHarvesting, Relevance: 0.85, Description: "Legal portal credential theft, e-discovery platform phishing"},
			{Category: CategorySocialEngineering, Relevance: 0.75, Description: "Fake subpoena, regulatory inquiry"},
		},
		AttackVectors:      []string{CategoryBEC, CategoryCredentialHarvesting, CategorySocialEngineering},
		ContextualTriggers: []string{"litigation deadline", "regulatory audit", "contract renewal", "board governance review"},
	},
}

// GetDepartmentThreatProfile returns the threat profile for a department.
// Falls back to a generic profile if the department is not mapped.
func GetDepartmentThreatProfile(department string) DepartmentThreatProfile {
	if p, ok := departmentThreatMap[department]; ok {
		return p
	}
	// Generic fallback
	return DepartmentThreatProfile{
		Department:     department,
		RiskMultiplier: 1.0,
		PrimaryThreats: []DepartmentThreat{
			{Category: CategoryCredentialHarvesting, Relevance: 0.80, Description: "Password reset and account verification phishing"},
			{Category: CategorySocialEngineering, Relevance: 0.70, Description: "Generic social engineering via email"},
			{Category: CategoryDeliveryNotification, Relevance: 0.60, Description: "Package delivery and service notifications"},
		},
		AttackVectors:      []string{CategoryCredentialHarvesting, CategorySocialEngineering, CategoryDeliveryNotification},
		ContextualTriggers: []string{"policy update", "system migration", "annual training"},
	}
}

// GetAllDepartmentThreatProfiles returns all known department threat profiles.
func GetAllDepartmentThreatProfiles() []DepartmentThreatProfile {
	profiles := make([]DepartmentThreatProfile, 0, len(departmentThreatMap))
	for _, p := range departmentThreatMap {
		profiles = append(profiles, p)
	}
	sort.Slice(profiles, func(i, j int) bool {
		return profiles[i].Department < profiles[j].Department
	})
	return profiles
}

// GetOrgDepartmentStats returns department-level phishing stats for an org,
// enriched with threat profile metadata.
func GetOrgDepartmentStats(orgId int64) ([]DepartmentThreatStat, error) {
	type deptRow struct {
		Department string
		Total      int64
		Clicked    int64
		Submitted  int64
		Reported   int64
	}
	var rows []deptRow
	err := db.Raw(`
		SELECT u.department, COUNT(*) as total,
			SUM(CASE WHEN r.status IN (?, ?) THEN 1 ELSE 0 END) as clicked,
			SUM(CASE WHEN r.status = ? THEN 1 ELSE 0 END) as submitted,
			SUM(CASE WHEN r.reported = 1 THEN 1 ELSE 0 END) as reported
		FROM results r
		JOIN campaigns c ON r.campaign_id = c.id
		JOIN users u ON u.email = r.email AND u.org_id = c.org_id
		WHERE c.org_id = ? AND u.department != '' AND u.department IS NOT NULL
		GROUP BY u.department
		ORDER BY total DESC
	`, EventClicked, EventDataSubmit, EventDataSubmit, orgId).Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	var stats []DepartmentThreatStat
	for _, r := range rows {
		tp := GetDepartmentThreatProfile(r.Department)
		clickRate := 0.0
		if r.Total > 0 {
			clickRate = float64(r.Clicked) / float64(r.Total) * 100
		}
		stats = append(stats, DepartmentThreatStat{
			Department:     r.Department,
			TotalSims:      r.Total,
			Clicked:        r.Clicked,
			Submitted:      r.Submitted,
			Reported:       r.Reported,
			ClickRate:      math.Round(clickRate*100) / 100,
			ThreatProfile:  tp,
			RiskMultiplier: tp.RiskMultiplier,
		})
	}
	return stats, nil
}

// DepartmentThreatStat combines observed phishing stats with the department's threat profile.
type DepartmentThreatStat struct {
	Department     string                  `json:"department"`
	TotalSims      int64                   `json:"total_simulations"`
	Clicked        int64                   `json:"clicked"`
	Submitted      int64                   `json:"submitted"`
	Reported       int64                   `json:"reported"`
	ClickRate      float64                 `json:"click_rate"`
	ThreatProfile  DepartmentThreatProfile `json:"threat_profile"`
	RiskMultiplier float64                 `json:"risk_multiplier"`
}

// ── A/B Testing Support ─────────────────────────────────────────

// ABTestResult tracks the outcome of a template variant test for a user.
type ABTestResult struct {
	Id            int64     `json:"id" gorm:"primary_key"`
	OrgId         int64     `json:"org_id"`
	CampaignId    int64     `json:"campaign_id"`
	UserId        int64     `json:"user_id"`
	Email         string    `json:"email"`
	VariantId     string    `json:"variant_id"`     // "A" or "B"
	TemplateId    int64     `json:"template_id"`
	TemplateName  string    `json:"template_name"`
	Clicked       bool      `json:"clicked"`
	Submitted     bool      `json:"submitted"`
	Reported      bool      `json:"reported"`
	TimeToClick   int64     `json:"time_to_click_s"` // seconds from send to click, 0 = no click
	CreatedDate   time.Time `json:"created_date"`
}

func (ABTestResult) TableName() string { return "ab_test_results" }

// RecordABTestResult saves an A/B test outcome.
func RecordABTestResult(r *ABTestResult) error {
	r.CreatedDate = time.Now().UTC()
	return db.Save(r).Error
}

// GetABTestSummary returns aggregated results for an A/B test campaign.
func GetABTestSummary(campaignId int64) ([]ABTestVariantSummary, error) {
	type variantRow struct {
		VariantId   string
		Total       int64
		Clicked     int64
		Submitted   int64
		Reported    int64
		AvgTimeToClick float64
	}
	var rows []variantRow
	err := db.Raw(`
		SELECT variant_id, COUNT(*) as total,
			SUM(CASE WHEN clicked = 1 THEN 1 ELSE 0 END) as clicked,
			SUM(CASE WHEN submitted = 1 THEN 1 ELSE 0 END) as submitted,
			SUM(CASE WHEN reported = 1 THEN 1 ELSE 0 END) as reported,
			AVG(CASE WHEN time_to_click_s > 0 THEN time_to_click_s END) as avg_time_to_click
		FROM ab_test_results
		WHERE campaign_id = ?
		GROUP BY variant_id
	`, campaignId).Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	var summaries []ABTestVariantSummary
	for _, r := range rows {
		clickRate := 0.0
		if r.Total > 0 {
			clickRate = float64(r.Clicked) / float64(r.Total) * 100
		}
		summaries = append(summaries, ABTestVariantSummary{
			VariantId:      r.VariantId,
			Total:          r.Total,
			Clicked:        r.Clicked,
			Submitted:      r.Submitted,
			Reported:       r.Reported,
			ClickRate:      math.Round(clickRate*100) / 100,
			AvgTimeToClick: math.Round(r.AvgTimeToClick*100) / 100,
		})
	}
	return summaries, nil
}

// ABTestVariantSummary is the aggregated outcome for one variant.
type ABTestVariantSummary struct {
	VariantId      string  `json:"variant_id"`
	Total          int64   `json:"total"`
	Clicked        int64   `json:"clicked"`
	Submitted      int64   `json:"submitted"`
	Reported       int64   `json:"reported"`
	ClickRate      float64 `json:"click_rate"`
	AvgTimeToClick float64 `json:"avg_time_to_click_s"`
}

// ── Category-Aware Reward Signals ───────────────────────────────

// AwardCategoryReportBonus gives a BRS bonus when a user correctly reports
// a phishing email, weighted by the category they detected.
func AwardCategoryReportBonus(userId int64, templateCategory string) {
	if templateCategory == "" {
		return
	}

	// Check if user is weak in this category — bigger reward if so
	profile, err := GetUserTargetingProfile(userId)
	if err != nil {
		log.Warnf("send_time_optimizer: failed to get targeting profile for user %d: %v", userId, err)
		return
	}

	bonus := 2.0 // Base bonus for any report
	for _, wc := range profile.WeakCategories {
		if wc.Category == templateCategory {
			// Extra reward for detecting attacks in weak categories
			bonus = 5.0 + (1.0-wc.Score/100.0)*3.0 // 5-8 points based on weakness severity
			break
		}
	}

	// Apply to BRS via trend nudge
	var currentBRS UserRiskScoreRecord
	if err := db.Where(queryWhereUserID, userId).First(&currentBRS).Error; err == nil {
		newTrend := math.Min(100, currentBRS.TrendScore+bonus)
		db.Model(&UserRiskScoreRecord{}).Where(queryWhereUserID, userId).
			Update("trend_score", newTrend)
		log.Infof("send_time_optimizer: awarded %.1f BRS trend bonus to user %d for reporting %s attack",
			bonus, userId, templateCategory)
	}
}
