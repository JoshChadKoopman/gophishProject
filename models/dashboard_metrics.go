package models

import (
	"math"
	"time"
)

// ── Real-Time Dashboard Metrics ─────────────────────────────────
// Provides configurable time-window dashboard cards with sparkline
// data, live counts, and trend indicators.

// TimeWindow represents one of the supported dashboard windows.
type TimeWindow string

const (
	TimeWindow7D  TimeWindow = "7d"
	TimeWindow30D TimeWindow = "30d"
	TimeWindow90D TimeWindow = "90d"
	TimeWindowYTD TimeWindow = "ytd"
)

// ValidTimeWindows lists all valid window values.
var ValidTimeWindows = map[TimeWindow]bool{
	TimeWindow7D: true, TimeWindow30D: true,
	TimeWindow90D: true, TimeWindowYTD: true,
}

// TimeWindowDays converts a TimeWindow to a number of days.
// For YTD it returns the number of days elapsed this year so far.
func TimeWindowDays(tw TimeWindow) int {
	switch tw {
	case TimeWindow7D:
		return 7
	case TimeWindow30D:
		return 30
	case TimeWindow90D:
		return 90
	case TimeWindowYTD:
		now := time.Now().UTC()
		yearStart := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, time.UTC)
		return int(now.Sub(yearStart).Hours()/24) + 1
	default:
		return 30
	}
}

// DashboardMetrics is the top-level response for GET /api/dashboard/metrics.
type DashboardMetrics struct {
	TimeWindow  TimeWindow       `json:"time_window"`
	GeneratedAt time.Time        `json:"generated_at"`
	Cards       DashboardCards   `json:"cards"`
}

// DashboardCards groups all summary cards shown on the admin dashboard.
type DashboardCards struct {
	Campaigns  CampaignCard  `json:"campaigns"`
	ClickRate  RateCard      `json:"click_rate"`
	ReportRate RateCard      `json:"report_rate"`
	Training   TrainingCard  `json:"training"`
	Tickets    TicketCard    `json:"tickets"`
	RiskScore  RiskCard      `json:"risk_score"`
	Compliance ComplianceCard `json:"compliance"`
}

// SparklinePoint is a single data point for mini-charts.
type SparklinePoint struct {
	Date  string  `json:"date"`
	Value float64 `json:"value"`
}

// TrendDirection indicates the direction of a metric over the window.
type TrendDirection string

const (
	TrendUp    TrendDirection = "up"
	TrendDown  TrendDirection = "down"
	TrendFlat  TrendDirection = "flat"
)

// ── Individual card types ──

// CampaignCard shows campaign summary with sparkline.
type CampaignCard struct {
	ActiveCount    int64            `json:"active_count"`
	TotalInWindow  int64            `json:"total_in_window"`
	EmailsSent     int64            `json:"emails_sent"`
	Sparkline      []SparklinePoint `json:"sparkline"` // emails sent per day
	Trend          TrendDirection   `json:"trend"`
}

// RateCard is used for click-rate and report-rate cards.
type RateCard struct {
	CurrentRate float64          `json:"current_rate"`
	PreviousRate float64         `json:"previous_rate"`
	Delta       float64          `json:"delta"`
	Sparkline   []SparklinePoint `json:"sparkline"` // daily rate
	Trend       TrendDirection   `json:"trend"`
}

// TrainingCard shows training completion at a glance.
type TrainingCard struct {
	CompletionRate float64          `json:"completion_rate"`
	Completed      int64            `json:"completed"`
	Overdue        int64            `json:"overdue"`
	Sparkline      []SparklinePoint `json:"sparkline"` // daily completions
	Trend          TrendDirection   `json:"trend"`
}

// TicketCard shows open/resolved phishing ticket counts.
type TicketCard struct {
	OpenCount     int64            `json:"open_count"`
	ResolvedToday int64            `json:"resolved_today"`
	Sparkline     []SparklinePoint `json:"sparkline"` // daily new tickets
	Trend         TrendDirection   `json:"trend"`
}

// RiskCard shows the org-average behavioural risk score.
type RiskCard struct {
	AvgScore  float64          `json:"avg_score"`
	Sparkline []SparklinePoint `json:"sparkline"` // daily avg risk
	Trend     TrendDirection   `json:"trend"`
}

// ComplianceCard shows overall compliance posture.
type ComplianceCard struct {
	OverallScore  float64          `json:"overall_score"`
	FrameworksActive int           `json:"frameworks_active"`
	Sparkline     []SparklinePoint `json:"sparkline"` // daily compliance score
	Trend         TrendDirection   `json:"trend"`
}

// ── Query functions ──

// GetDashboardMetrics builds the full dashboard for the given time window.
func GetDashboardMetrics(scope OrgScope, tw TimeWindow) (*DashboardMetrics, error) {
	days := TimeWindowDays(tw)
	cutoff := time.Now().UTC().AddDate(0, 0, -days)

	metrics := &DashboardMetrics{
		TimeWindow:  tw,
		GeneratedAt: time.Now().UTC(),
	}

	metrics.Cards.Campaigns = getCampaignCard(scope, cutoff, days)
	metrics.Cards.ClickRate = getClickRateCard(scope, cutoff, days)
	metrics.Cards.ReportRate = getReportRateCard(scope, cutoff, days)
	metrics.Cards.Training = getTrainingCard(scope, cutoff, days)
	metrics.Cards.Tickets = getTicketCard(scope, cutoff)
	metrics.Cards.RiskScore = getRiskScoreCard(scope, days)
	metrics.Cards.Compliance = getComplianceCard(scope)

	return metrics, nil
}

// ── Campaign card ──

func getCampaignCard(scope OrgScope, cutoff time.Time, days int) CampaignCard {
	card := CampaignCard{}

	// Active campaigns
	var active int64
	scopeQuery(db.Table("campaigns"), scope).
		Where("status = ?", CampaignInProgress).Count(&active)
	card.ActiveCount = active

	// Total campaigns in window
	var total int64
	scopeQuery(db.Table("campaigns"), scope).
		Where("created_date >= ?", cutoff).Count(&total)
	card.TotalInWindow = total

	// Emails sent in window
	type sentRow struct{ Count int64 }
	var sr sentRow
	if scope.IsSuperAdmin {
		db.Raw(`SELECT COUNT(*) as count FROM events e
			JOIN campaigns c ON e.campaign_id = c.id
			WHERE e.message = ? AND e.time >= ?`, EventSent, cutoff).Scan(&sr)
	} else {
		db.Raw(`SELECT COUNT(*) as count FROM events e
			JOIN campaigns c ON e.campaign_id = c.id
			WHERE c.org_id = ? AND e.message = ? AND e.time >= ?`, scope.OrgId, EventSent, cutoff).Scan(&sr)
	}
	card.EmailsSent = sr.Count

	// Sparkline: emails sent per day
	card.Sparkline = buildEventSparkline(scope, EventSent, days)

	card.Trend = sparklineTrend(card.Sparkline)
	return card
}

// ── Click-rate card ──

func getClickRateCard(scope OrgScope, cutoff time.Time, days int) RateCard {
	card := RateCard{}
	card.Sparkline = buildRateSparkline(scope, EventClicked, EventSent, days)
	if len(card.Sparkline) > 0 {
		card.CurrentRate = card.Sparkline[len(card.Sparkline)-1].Value
	}

	// Previous rate = average of first half of the sparkline
	mid := len(card.Sparkline) / 2
	if mid > 0 {
		sum := 0.0
		for _, p := range card.Sparkline[:mid] {
			sum += p.Value
		}
		card.PreviousRate = math.Round(sum/float64(mid)*100) / 100
	}
	card.Delta = math.Round((card.CurrentRate-card.PreviousRate)*100) / 100
	// For click rate, down is good
	if card.Delta < -1 {
		card.Trend = TrendDown
	} else if card.Delta > 1 {
		card.Trend = TrendUp
	} else {
		card.Trend = TrendFlat
	}
	return card
}

// ── Report-rate card ──

func getReportRateCard(scope OrgScope, cutoff time.Time, days int) RateCard {
	card := RateCard{}
	card.Sparkline = buildRateSparkline(scope, EventReported, EventSent, days)
	if len(card.Sparkline) > 0 {
		card.CurrentRate = card.Sparkline[len(card.Sparkline)-1].Value
	}
	mid := len(card.Sparkline) / 2
	if mid > 0 {
		sum := 0.0
		for _, p := range card.Sparkline[:mid] {
			sum += p.Value
		}
		card.PreviousRate = math.Round(sum/float64(mid)*100) / 100
	}
	card.Delta = math.Round((card.CurrentRate-card.PreviousRate)*100) / 100
	// For report rate, up is good
	if card.Delta > 1 {
		card.Trend = TrendUp
	} else if card.Delta < -1 {
		card.Trend = TrendDown
	} else {
		card.Trend = TrendFlat
	}
	return card
}

// ── Training card ──

func getTrainingCard(scope OrgScope, cutoff time.Time, days int) TrainingCard {
	card := TrainingCard{}

	// Completion rate from assignments
	var totalAssign, completedAssign int64
	q := db.Table("training_assignments")
	if !scope.IsSuperAdmin {
		q = q.Where("org_id = ?", scope.OrgId)
	}
	q.Count(&totalAssign)
	q2 := db.Table("training_assignments").Where("status = ?", "completed")
	if !scope.IsSuperAdmin {
		q2 = q2.Where("org_id = ?", scope.OrgId)
	}
	q2.Count(&completedAssign)
	card.Completed = completedAssign
	if totalAssign > 0 {
		card.CompletionRate = math.Round(float64(completedAssign)*10000/float64(totalAssign)) / 100
	}

	// Overdue
	var overdue int64
	q3 := db.Table("training_assignments").Where("status IN (?) AND due_date < ?", []string{"assigned", "in_progress"}, time.Now().UTC())
	if !scope.IsSuperAdmin {
		q3 = q3.Where("org_id = ?", scope.OrgId)
	}
	q3.Count(&overdue)
	card.Overdue = overdue

	// Sparkline: daily completions in window
	card.Sparkline = buildTrainingSparkline(scope, days)
	card.Trend = sparklineTrend(card.Sparkline)
	return card
}

// ── Ticket card ──

func getTicketCard(scope OrgScope, cutoff time.Time) TicketCard {
	card := TicketCard{}

	q := db.Table("phishing_tickets")
	if !scope.IsSuperAdmin {
		q = q.Where("org_id = ?", scope.OrgId)
	}
	var openCount int64
	q.Where("status IN (?)", []string{"open", "investigating"}).Count(&openCount)
	card.OpenCount = openCount

	var resolvedToday int64
	today := time.Now().UTC().Format("2006-01-02")
	q2 := db.Table("phishing_tickets").Where("status = ? AND DATE(resolved_at) = ?", "resolved", today)
	if !scope.IsSuperAdmin {
		q2 = q2.Where("org_id = ?", scope.OrgId)
	}
	q2.Count(&resolvedToday)
	card.ResolvedToday = resolvedToday

	card.Sparkline = buildTicketSparkline(scope, 7)
	card.Trend = sparklineTrend(card.Sparkline)
	return card
}

// ── Risk score card ──

func getRiskScoreCard(scope OrgScope, days int) RiskCard {
	card := RiskCard{}

	type avgRow struct{ Avg float64 }
	var ar avgRow
	q := db.Table("user_risk_scores").Select("COALESCE(AVG(overall_score),0) as avg")
	if !scope.IsSuperAdmin {
		q = q.Where("org_id = ?", scope.OrgId)
	}
	q.Scan(&ar)
	card.AvgScore = math.Round(ar.Avg*100) / 100

	card.Sparkline = buildRiskSparkline(scope, days)
	card.Trend = sparklineTrend(card.Sparkline)
	return card
}

// ── Compliance card ──

func getComplianceCard(scope OrgScope) ComplianceCard {
	card := ComplianceCard{}

	// Frameworks enabled for org
	var fwCount int
	if !scope.IsSuperAdmin {
		db.Table("org_compliance_frameworks").Where("org_id = ? AND enabled = ?", scope.OrgId, true).Count(&fwCount)
	}
	card.FrameworksActive = fwCount

	// Overall compliance score from framework assessments
	type scoreRow struct{ Avg float64 }
	var sr scoreRow
	q := db.Table("compliance_control_assessments").
		Select("COALESCE(AVG(CASE WHEN status='met' THEN 100 WHEN status='partial' THEN 50 ELSE 0 END),0) as avg")
	if !scope.IsSuperAdmin {
		q = q.Where("org_id = ?", scope.OrgId)
	}
	q.Scan(&sr)
	card.OverallScore = math.Round(sr.Avg*100) / 100

	// Static sparkline placeholder (compliance doesn't change daily normally)
	card.Sparkline = []SparklinePoint{}
	card.Trend = TrendFlat
	return card
}

// ── Sparkline builder helpers ──

// buildEventSparkline counts events of a given type per day for the last N days.
func buildEventSparkline(scope OrgScope, eventMsg string, days int) []SparklinePoint {
	var rows []sparkRow
	cutoff := time.Now().UTC().AddDate(0, 0, -days).Format("2006-01-02")

	if scope.IsSuperAdmin {
		db.Raw(`SELECT DATE(e.time) as date, COUNT(*) as count
			FROM events e JOIN campaigns c ON e.campaign_id = c.id
			WHERE e.message = ? AND DATE(e.time) >= ?
			GROUP BY DATE(e.time) ORDER BY date`, eventMsg, cutoff).Scan(&rows)
	} else {
		db.Raw(`SELECT DATE(e.time) as date, COUNT(*) as count
			FROM events e JOIN campaigns c ON e.campaign_id = c.id
			WHERE c.org_id = ? AND e.message = ? AND DATE(e.time) >= ?
			GROUP BY DATE(e.time) ORDER BY date`, scope.OrgId, eventMsg, cutoff).Scan(&rows)
	}

	return padSparkline(rows, days)
}

// buildRateSparkline builds a daily rate (numerator/denominator) sparkline.
func buildRateSparkline(scope OrgScope, numeratorEvent, denominatorEvent string, days int) []SparklinePoint {
	type dayRow struct {
		Date string
		Num  float64
		Den  float64
	}
	var rows []dayRow
	cutoff := time.Now().UTC().AddDate(0, 0, -days).Format("2006-01-02")

	if scope.IsSuperAdmin {
		db.Raw(`SELECT DATE(e.time) as date,
			SUM(CASE WHEN e.message = ? THEN 1 ELSE 0 END) as num,
			SUM(CASE WHEN e.message = ? THEN 1 ELSE 0 END) as den
			FROM events e JOIN campaigns c ON e.campaign_id = c.id
			WHERE DATE(e.time) >= ?
			GROUP BY DATE(e.time) ORDER BY date`,
			numeratorEvent, denominatorEvent, cutoff).Scan(&rows)
	} else {
		db.Raw(`SELECT DATE(e.time) as date,
			SUM(CASE WHEN e.message = ? THEN 1 ELSE 0 END) as num,
			SUM(CASE WHEN e.message = ? THEN 1 ELSE 0 END) as den
			FROM events e JOIN campaigns c ON e.campaign_id = c.id
			WHERE c.org_id = ? AND DATE(e.time) >= ?
			GROUP BY DATE(e.time) ORDER BY date`,
			numeratorEvent, denominatorEvent, scope.OrgId, cutoff).Scan(&rows)
	}

	points := make([]SparklinePoint, 0, len(rows))
	for _, r := range rows {
		v := 0.0
		if r.Den > 0 {
			v = math.Round(r.Num/r.Den*10000) / 100
		}
		points = append(points, SparklinePoint{Date: r.Date, Value: v})
	}
	return points
}

// buildTrainingSparkline counts daily training completions.
func buildTrainingSparkline(scope OrgScope, days int) []SparklinePoint {
	var rows []sparkRow
	cutoff := time.Now().UTC().AddDate(0, 0, -days).Format("2006-01-02")

	q := `SELECT DATE(completed_date) as date, COUNT(*) as count
		FROM training_assignments WHERE status = 'completed' AND DATE(completed_date) >= ?`
	if scope.IsSuperAdmin {
		db.Raw(q+" GROUP BY DATE(completed_date) ORDER BY date", cutoff).Scan(&rows)
	} else {
		db.Raw(q+" AND org_id = ? GROUP BY DATE(completed_date) ORDER BY date", cutoff, scope.OrgId).Scan(&rows)
	}
	return padSparkline(rows, days)
}

// buildTicketSparkline counts daily new phishing tickets.
func buildTicketSparkline(scope OrgScope, days int) []SparklinePoint {
	var rows []sparkRow
	cutoff := time.Now().UTC().AddDate(0, 0, -days).Format("2006-01-02")

	q := `SELECT DATE(created_date) as date, COUNT(*) as count
		FROM phishing_tickets WHERE DATE(created_date) >= ?`
	if scope.IsSuperAdmin {
		db.Raw(q+" GROUP BY DATE(created_date) ORDER BY date", cutoff).Scan(&rows)
	} else {
		db.Raw(q+" AND org_id = ? GROUP BY DATE(created_date) ORDER BY date", cutoff, scope.OrgId).Scan(&rows)
	}
	return padSparkline(rows, days)
}

// buildRiskSparkline builds a daily average risk score sparkline from BRS history.
func buildRiskSparkline(scope OrgScope, days int) []SparklinePoint {
	var rows []sparkRow
	cutoff := time.Now().UTC().AddDate(0, 0, -days).Format("2006-01-02")

	q := `SELECT DATE(recorded_at) as date, AVG(score) as count
		FROM brs_history WHERE DATE(recorded_at) >= ?`
	if scope.IsSuperAdmin {
		db.Raw(q+" GROUP BY DATE(recorded_at) ORDER BY date", cutoff).Scan(&rows)
	} else {
		db.Raw(q+" AND org_id = ? GROUP BY DATE(recorded_at) ORDER BY date", cutoff, scope.OrgId).Scan(&rows)
	}
	return padSparkline(rows, days)
}

// sparkRow is a helper type used by sparkline builders.
type sparkRow struct {
	Date  string
	Count float64
}

// padSparkline fills in missing dates with zero values so the sparkline
// has one entry per day.
func padSparkline(rows []sparkRow, days int) []SparklinePoint {
	lookup := make(map[string]float64)
	for _, r := range rows {
		lookup[r.Date] = r.Count
	}

	points := make([]SparklinePoint, 0, days)
	for i := days - 1; i >= 0; i-- {
		d := time.Now().UTC().AddDate(0, 0, -i).Format("2006-01-02")
		v := lookup[d]
		points = append(points, SparklinePoint{Date: d, Value: v})
	}
	return points
}

// sparklineTrend determines if the sparkline is trending up, down, or flat.
func sparklineTrend(points []SparklinePoint) TrendDirection {
	if len(points) < 2 {
		return TrendFlat
	}
	mid := len(points) / 2
	firstHalfSum, secondHalfSum := 0.0, 0.0
	for _, p := range points[:mid] {
		firstHalfSum += p.Value
	}
	for _, p := range points[mid:] {
		secondHalfSum += p.Value
	}
	firstAvg := firstHalfSum / float64(mid)
	secondAvg := secondHalfSum / float64(len(points)-mid)

	diff := secondAvg - firstAvg
	threshold := firstAvg * 0.05 // 5% change threshold
	if threshold < 0.5 {
		threshold = 0.5
	}

	if diff > threshold {
		return TrendUp
	} else if diff < -threshold {
		return TrendDown
	}
	return TrendFlat
}

// ── Dashboard Preference (DB-backed) ──

// DashboardPreference stores an admin's preferred time window.
type DashboardPreference struct {
	Id         int64  `json:"id" gorm:"primary_key"`
	UserId     int64  `json:"user_id" gorm:"column:user_id;unique_index"`
	OrgId      int64  `json:"org_id" gorm:"column:org_id"`
	TimeWindow string `json:"time_window" gorm:"column:time_window;default:'30d'"`
}

// TableName for DashboardPreference.
func (DashboardPreference) TableName() string { return "dashboard_preferences" }

// GetDashboardPreference returns the user's saved time window, or "30d" by default.
func GetDashboardPreference(userId int64) DashboardPreference {
	pref := DashboardPreference{}
	err := db.Where("user_id = ?", userId).First(&pref).Error
	if err != nil {
		return DashboardPreference{
			UserId:     userId,
			TimeWindow: string(TimeWindow30D),
		}
	}
	return pref
}

// SaveDashboardPreference upserts the dashboard time window preference.
func SaveDashboardPreference(pref *DashboardPreference) error {
	existing := DashboardPreference{}
	err := db.Where("user_id = ?", pref.UserId).First(&existing).Error
	if err == nil {
		pref.Id = existing.Id
	}
	return db.Save(pref).Error
}

// ── Live counts (quick queries for WebSocket pulse) ──

// DashboardLiveCounts contains fast-computed counts for the WS pulse.
type DashboardLiveCounts struct {
	ActiveCampaigns int64   `json:"active_campaigns"`
	EmailsSentToday int64   `json:"emails_sent_today"`
	OpenTickets     int64   `json:"open_tickets"`
	AvgClickRate    float64 `json:"avg_click_rate"`
	AvgReportRate   float64 `json:"avg_report_rate"`
	OnlineAdmins    int     `json:"online_admins"`
}

// GetDashboardLiveCounts returns a lightweight snapshot for the WS pulse.
func GetDashboardLiveCounts(scope OrgScope) DashboardLiveCounts {
	counts := DashboardLiveCounts{}

	// Active campaigns
	scopeQuery(db.Table("campaigns"), scope).
		Where("status = ?", CampaignInProgress).Count(&counts.ActiveCampaigns)

	// Emails sent today
	today := time.Now().UTC().Format("2006-01-02")
	type cntRow struct{ Count int64 }
	var sr cntRow
	if scope.IsSuperAdmin {
		db.Raw(`SELECT COUNT(*) as count FROM events e
			JOIN campaigns c ON e.campaign_id = c.id
			WHERE e.message = ? AND DATE(e.time) = ?`, EventSent, today).Scan(&sr)
	} else {
		db.Raw(`SELECT COUNT(*) as count FROM events e
			JOIN campaigns c ON e.campaign_id = c.id
			WHERE c.org_id = ? AND e.message = ? AND DATE(e.time) = ?`, scope.OrgId, EventSent, today).Scan(&sr)
	}
	counts.EmailsSentToday = sr.Count

	// Open tickets
	q := db.Table("phishing_tickets").Where("status IN (?)", []string{"open", "investigating"})
	if !scope.IsSuperAdmin {
		q = q.Where("org_id = ?", scope.OrgId)
	}
	q.Count(&counts.OpenTickets)

	// Click rate (last 30 days)
	overview, err := GetReportOverview(scope)
	if err == nil {
		counts.AvgClickRate = overview.AvgClickRate
		counts.AvgReportRate = overview.AvgReportRate
	}

	counts.OnlineAdmins = GetWSHub().SubscriberCount(scope.OrgId)
	return counts
}

// ── Exported sparkline builders (used by controller) ──

// BuildEventSparklinePublic is the exported version of buildEventSparkline.
func BuildEventSparklinePublic(scope OrgScope, eventMsg string, days int) []SparklinePoint {
	return buildEventSparkline(scope, eventMsg, days)
}

// BuildRateSparklinePublic is the exported version of buildRateSparkline.
func BuildRateSparklinePublic(scope OrgScope, numeratorEvent, denominatorEvent string, days int) []SparklinePoint {
	return buildRateSparkline(scope, numeratorEvent, denominatorEvent, days)
}

