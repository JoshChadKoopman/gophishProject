package models

import (
	"math"
	"time"

	log "github.com/gophish/gophish/logger"
)

// ── Materialized Daily Metrics ──────────────────────────────────
// Pre-computed daily rollups per org, populated by a nightly worker.
// Dashboard endpoints query these instead of scanning raw tables,
// dramatically improving load times as data grows.

// ReportDailyMetric represents a single day's pre-aggregated metrics for an org.
type ReportDailyMetric struct {
	Id         int64  `json:"id" gorm:"primary_key;auto_increment"`
	OrgId      int64  `json:"org_id" gorm:"column:org_id"`
	MetricDate string `json:"metric_date" gorm:"column:metric_date"`

	// Campaign / email
	EmailsSent         int64 `json:"emails_sent" gorm:"column:emails_sent"`
	EmailsOpened       int64 `json:"emails_opened" gorm:"column:emails_opened"`
	LinksClicked       int64 `json:"links_clicked" gorm:"column:links_clicked"`
	DataSubmitted      int64 `json:"data_submitted" gorm:"column:data_submitted"`
	EmailsReported     int64 `json:"emails_reported" gorm:"column:emails_reported"`
	CampaignsLaunched  int64 `json:"campaigns_launched" gorm:"column:campaigns_launched"`
	CampaignsCompleted int64 `json:"campaigns_completed" gorm:"column:campaigns_completed"`

	// Training
	TrainingAssigned   int64   `json:"training_assigned" gorm:"column:training_assigned"`
	TrainingCompleted  int64   `json:"training_completed" gorm:"column:training_completed"`
	TrainingOverdue    int64   `json:"training_overdue" gorm:"column:training_overdue"`
	AvgQuizScore       float64 `json:"avg_quiz_score" gorm:"column:avg_quiz_score"`
	CertificatesIssued int64   `json:"certificates_issued" gorm:"column:certificates_issued"`

	// Tickets / incidents
	TicketsOpened         int64 `json:"tickets_opened" gorm:"column:tickets_opened"`
	TicketsResolved       int64 `json:"tickets_resolved" gorm:"column:tickets_resolved"`
	IncidentsCreated      int64 `json:"incidents_created" gorm:"column:incidents_created"`
	IncidentsResolved     int64 `json:"incidents_resolved" gorm:"column:incidents_resolved"`
	NetworkEventsIngested int64 `json:"network_events_ingested" gorm:"column:network_events_ingested"`

	// Risk / compliance / hygiene snapshots
	AvgRiskScore      float64 `json:"avg_risk_score" gorm:"column:avg_risk_score"`
	HighRiskUserCount int64   `json:"high_risk_user_count" gorm:"column:high_risk_user_count"`
	ComplianceScore   float64 `json:"compliance_score" gorm:"column:compliance_score"`
	AvgHygieneScore   float64 `json:"avg_hygiene_score" gorm:"column:avg_hygiene_score"`
	DevicesCompliant  int64   `json:"devices_compliant" gorm:"column:devices_compliant"`
	DevicesTotal      int64   `json:"devices_total" gorm:"column:devices_total"`

	// Computed rates
	ClickRate              float64 `json:"click_rate" gorm:"column:click_rate"`
	ReportRate             float64 `json:"report_rate" gorm:"column:report_rate"`
	TrainingCompletionRate float64 `json:"training_completion_rate" gorm:"column:training_completion_rate"`

	// Active state
	ActiveCampaigns int64 `json:"active_campaigns" gorm:"column:active_campaigns"`
	TotalUsers      int64 `json:"total_users" gorm:"column:total_users"`

	ComputedAt time.Time `json:"computed_at" gorm:"column:computed_at"`
}

// TableName for GORM mapping.
func (ReportDailyMetric) TableName() string { return "report_daily_metrics" }

// ── Rollup Computation ──────────────────────────────────────────

// ComputeAndStoreDailyMetrics calculates metrics for a specific org and
// date, then upserts the row. This is called by the nightly worker.
func ComputeAndStoreDailyMetrics(orgId int64, date time.Time) error {
	dateStr := date.Format("2006-01-02")
	m := ReportDailyMetric{
		OrgId:      orgId,
		MetricDate: dateStr,
		ComputedAt: time.Now().UTC(),
	}

	// ── Campaign / email events ──
	type cntRow struct{ Count int64 }
	var row cntRow

	db.Raw(`SELECT COUNT(*) as count FROM events e
		JOIN campaigns c ON e.campaign_id = c.id
		WHERE c.org_id = ? AND e.message = ? AND DATE(e.time) = ?`,
		orgId, EventSent, dateStr).Scan(&row)
	m.EmailsSent = row.Count

	db.Raw(`SELECT COUNT(*) as count FROM events e
		JOIN campaigns c ON e.campaign_id = c.id
		WHERE c.org_id = ? AND e.message = ? AND DATE(e.time) = ?`,
		orgId, EventOpened, dateStr).Scan(&row)
	m.EmailsOpened = row.Count

	db.Raw(`SELECT COUNT(*) as count FROM events e
		JOIN campaigns c ON e.campaign_id = c.id
		WHERE c.org_id = ? AND e.message = ? AND DATE(e.time) = ?`,
		orgId, EventClicked, dateStr).Scan(&row)
	m.LinksClicked = row.Count

	db.Raw(`SELECT COUNT(*) as count FROM events e
		JOIN campaigns c ON e.campaign_id = c.id
		WHERE c.org_id = ? AND e.message = ? AND DATE(e.time) = ?`,
		orgId, EventDataSubmit, dateStr).Scan(&row)
	m.DataSubmitted = row.Count

	db.Raw(`SELECT COUNT(*) as count FROM events e
		JOIN campaigns c ON e.campaign_id = c.id
		WHERE c.org_id = ? AND e.message = ? AND DATE(e.time) = ?`,
		orgId, EventReported, dateStr).Scan(&row)
	m.EmailsReported = row.Count

	// Campaigns launched on this date
	db.Raw(`SELECT COUNT(*) as count FROM campaigns
		WHERE org_id = ? AND DATE(created_date) = ?`,
		orgId, dateStr).Scan(&row)
	m.CampaignsLaunched = row.Count

	// Campaigns completed on this date
	db.Raw(`SELECT COUNT(*) as count FROM campaigns
		WHERE org_id = ? AND status = ? AND DATE(completed_date) = ?`,
		orgId, CampaignComplete, dateStr).Scan(&row)
	m.CampaignsCompleted = row.Count

	// Active campaigns as of this date
	db.Raw(`SELECT COUNT(*) as count FROM campaigns
		WHERE org_id = ? AND status = ? AND DATE(created_date) <= ?`,
		orgId, CampaignInProgress, dateStr).Scan(&row)
	m.ActiveCampaigns = row.Count

	// ── Training metrics ──
	db.Raw(`SELECT COUNT(*) as count FROM training_assignments
		WHERE org_id = ? AND DATE(assigned_date) = ?`,
		orgId, dateStr).Scan(&row)
	m.TrainingAssigned = row.Count

	db.Raw(`SELECT COUNT(*) as count FROM training_assignments
		WHERE org_id = ? AND status = 'completed' AND DATE(completed_date) = ?`,
		orgId, dateStr).Scan(&row)
	m.TrainingCompleted = row.Count

	db.Raw(`SELECT COUNT(*) as count FROM training_assignments
		WHERE org_id = ? AND status IN ('assigned','in_progress') AND due_date < ?`,
		orgId, dateStr).Scan(&row)
	m.TrainingOverdue = row.Count

	// Avg quiz score for completions on this date
	type avgRow struct{ Avg float64 }
	var ar avgRow
	db.Raw(`SELECT COALESCE(AVG(score), 0) as avg FROM quiz_attempts
		WHERE org_id = ? AND DATE(completed_at) = ?`,
		orgId, dateStr).Scan(&ar)
	m.AvgQuizScore = math.Round(ar.Avg*100) / 100

	// Certificates issued
	db.Raw(`SELECT COUNT(*) as count FROM completion_certificates
		WHERE org_id = ? AND DATE(issued_date) = ?`,
		orgId, dateStr).Scan(&row)
	m.CertificatesIssued = row.Count

	// Training completion rate (cumulative as of this date)
	var totalAssign, completedAssign int64
	db.Raw(`SELECT COUNT(*) as count FROM training_assignments
		WHERE org_id = ? AND DATE(assigned_date) <= ?`,
		orgId, dateStr).Scan(&row)
	totalAssign = row.Count
	db.Raw(`SELECT COUNT(*) as count FROM training_assignments
		WHERE org_id = ? AND status = 'completed' AND DATE(completed_date) <= ?`,
		orgId, dateStr).Scan(&row)
	completedAssign = row.Count
	if totalAssign > 0 {
		m.TrainingCompletionRate = math.Round(float64(completedAssign)*10000/float64(totalAssign)) / 100
	}

	// ── Tickets / incidents ──
	db.Raw(`SELECT COUNT(*) as count FROM phishing_tickets
		WHERE org_id = ? AND DATE(created_date) = ?`,
		orgId, dateStr).Scan(&row)
	m.TicketsOpened = row.Count

	db.Raw(`SELECT COUNT(*) as count FROM phishing_tickets
		WHERE org_id = ? AND status = 'resolved' AND DATE(resolved_at) = ?`,
		orgId, dateStr).Scan(&row)
	m.TicketsResolved = row.Count

	// Network events (may not exist, ignore errors)
	db.Raw(`SELECT COUNT(*) as count FROM network_events
		WHERE org_id = ? AND DATE(event_date) = ?`,
		orgId, dateStr).Scan(&row)
	m.NetworkEventsIngested = row.Count

	// Incidents created/resolved on this date
	db.Raw(`SELECT COUNT(*) as count FROM network_incidents
		WHERE org_id = ? AND DATE(created_date) = ?`,
		orgId, dateStr).Scan(&row)
	m.IncidentsCreated = row.Count

	db.Raw(`SELECT COUNT(*) as count FROM network_incidents
		WHERE org_id = ? AND status = 'closed' AND DATE(updated_at) = ?`,
		orgId, dateStr).Scan(&row)
	m.IncidentsResolved = row.Count

	// ── Risk / compliance / hygiene snapshots ──
	db.Raw(`SELECT COALESCE(AVG(overall_score), 0) as avg FROM user_risk_scores
		WHERE org_id = ?`, orgId).Scan(&ar)
	m.AvgRiskScore = math.Round(ar.Avg*100) / 100

	db.Raw(`SELECT COUNT(*) as count FROM user_risk_scores
		WHERE org_id = ? AND overall_score >= 70`, orgId).Scan(&row)
	m.HighRiskUserCount = row.Count

	// Compliance score
	db.Raw(`SELECT COALESCE(AVG(CASE WHEN status='met' THEN 100 WHEN status='partial' THEN 50 ELSE 0 END), 0) as avg
		FROM compliance_control_assessments WHERE org_id = ?`, orgId).Scan(&ar)
	m.ComplianceScore = math.Round(ar.Avg*100) / 100

	// Hygiene
	db.Raw(`SELECT COALESCE(AVG(score), 0) as avg FROM hygiene_devices
		WHERE org_id = ?`, orgId).Scan(&ar)
	m.AvgHygieneScore = math.Round(ar.Avg*100) / 100

	db.Raw(`SELECT COUNT(*) as count FROM hygiene_devices
		WHERE org_id = ? AND compliant = 1`, orgId).Scan(&row)
	m.DevicesCompliant = row.Count

	db.Raw(`SELECT COUNT(*) as count FROM hygiene_devices
		WHERE org_id = ?`, orgId).Scan(&row)
	m.DevicesTotal = row.Count

	// ── Computed rates ──
	if m.EmailsSent > 0 {
		m.ClickRate = math.Round(float64(m.LinksClicked)*10000/float64(m.EmailsSent)) / 100
		m.ReportRate = math.Round(float64(m.EmailsReported)*10000/float64(m.EmailsSent)) / 100
	}

	// Total users in org
	db.Raw(`SELECT COUNT(*) as count FROM users WHERE org_id = ?`, orgId).Scan(&row)
	m.TotalUsers = row.Count

	// ── Upsert ──
	existing := ReportDailyMetric{}
	err := db.Where("org_id = ? AND metric_date = ?", orgId, dateStr).First(&existing).Error
	if err == nil {
		m.Id = existing.Id
	}
	return db.Save(&m).Error
}

// ComputeAllOrgsDailyMetrics runs the rollup for every org for the given date.
func ComputeAllOrgsDailyMetrics(date time.Time) error {
	type orgRow struct{ Id int64 }
	var orgs []orgRow
	if err := db.Raw("SELECT id FROM organizations").Scan(&orgs).Error; err != nil {
		return err
	}

	// Also compute for org_id = 0 (default org / legacy data)
	orgIds := []int64{0}
	for _, o := range orgs {
		orgIds = append(orgIds, o.Id)
	}

	for _, oid := range orgIds {
		if err := ComputeAndStoreDailyMetrics(oid, date); err != nil {
			log.Errorf("ReportRollup: failed for org %d on %s: %v", oid, date.Format("2006-01-02"), err)
			// Continue to other orgs — don't fail the entire run
		}
	}
	return nil
}

// BackfillDailyMetrics computes rollups for the last N days for all orgs.
// Useful for initial population or catching up after downtime.
func BackfillDailyMetrics(days int) error {
	now := time.Now().UTC()
	for i := days; i >= 0; i-- {
		date := now.AddDate(0, 0, -i)
		if err := ComputeAllOrgsDailyMetrics(date); err != nil {
			log.Errorf("ReportRollup: backfill error for %s: %v", date.Format("2006-01-02"), err)
		}
	}
	return nil
}

// ── Fast Query Functions (replace raw table scans) ──────────────

// GetDailyMetricSparkline returns daily metric values for a given org
// over the last N days, using the rollup table.
func GetDailyMetricSparkline(scope OrgScope, metric string, days int) []SparklinePoint {
	cutoff := time.Now().UTC().AddDate(0, 0, -days).Format("2006-01-02")
	type row struct {
		MetricDate string  `gorm:"column:metric_date"`
		Value      float64 `gorm:"column:value"`
	}
	var rows []row

	col := metricToColumn(metric)
	if col == "" {
		return []SparklinePoint{}
	}

	q := "SELECT metric_date, " + col + " as value FROM report_daily_metrics WHERE metric_date >= ?"
	if scope.IsSuperAdmin {
		// Aggregate across all orgs
		q = "SELECT metric_date, SUM(" + col + ") as value FROM report_daily_metrics WHERE metric_date >= ? GROUP BY metric_date ORDER BY metric_date"
		db.Raw(q, cutoff).Scan(&rows)
	} else {
		q += " AND org_id = ? ORDER BY metric_date"
		db.Raw(q, cutoff, scope.OrgId).Scan(&rows)
	}

	// Pad missing dates
	lookup := make(map[string]float64)
	for _, r := range rows {
		lookup[r.MetricDate] = r.Value
	}
	points := make([]SparklinePoint, 0, days)
	for i := days - 1; i >= 0; i-- {
		d := time.Now().UTC().AddDate(0, 0, -i).Format("2006-01-02")
		points = append(points, SparklinePoint{Date: d, Value: lookup[d]})
	}
	return points
}

// GetDailyMetricSum returns the total of a metric column over a date range.
func GetDailyMetricSum(scope OrgScope, metric string, days int) float64 {
	cutoff := time.Now().UTC().AddDate(0, 0, -days).Format("2006-01-02")
	col := metricToColumn(metric)
	if col == "" {
		return 0
	}

	type sumRow struct{ Total float64 }
	var sr sumRow
	q := "SELECT COALESCE(SUM(" + col + "), 0) as total FROM report_daily_metrics WHERE metric_date >= ?"
	if scope.IsSuperAdmin {
		db.Raw(q, cutoff).Scan(&sr)
	} else {
		db.Raw(q+" AND org_id = ?", cutoff, scope.OrgId).Scan(&sr)
	}
	return sr.Total
}

// GetDailyMetricAvg returns the average of a metric column over a date range.
func GetDailyMetricAvg(scope OrgScope, metric string, days int) float64 {
	cutoff := time.Now().UTC().AddDate(0, 0, -days).Format("2006-01-02")
	col := metricToColumn(metric)
	if col == "" {
		return 0
	}

	type avgRow struct{ Avg float64 }
	var ar avgRow
	q := "SELECT COALESCE(AVG(" + col + "), 0) as avg FROM report_daily_metrics WHERE metric_date >= ?"
	if scope.IsSuperAdmin {
		db.Raw(q, cutoff).Scan(&ar)
	} else {
		db.Raw(q+" AND org_id = ?", cutoff, scope.OrgId).Scan(&ar)
	}
	return math.Round(ar.Avg*100) / 100
}

// GetLatestDailyMetric returns the most recent rollup row for an org.
func GetLatestDailyMetric(orgId int64) (*ReportDailyMetric, error) {
	m := &ReportDailyMetric{}
	err := db.Where("org_id = ?", orgId).Order("metric_date DESC").First(m).Error
	if err != nil {
		return nil, err
	}
	return m, nil
}

// HasRollupData checks if the rollup table has been populated.
func HasRollupData(orgId int64) bool {
	var count int64
	db.Table("report_daily_metrics").Where("org_id = ?", orgId).Count(&count)
	return count > 0
}

// ── Rate sparkline from rollup ──

// GetDailyRateSparkline returns a rate metric (e.g., click_rate) sparkline.
func GetDailyRateSparkline(scope OrgScope, rateColumn string, days int) []SparklinePoint {
	return GetDailyMetricSparkline(scope, rateColumn, days)
}

// ── Helper: map metric names to DB columns ──

func metricToColumn(metric string) string {
	columns := map[string]string{
		"emails_sent":              "emails_sent",
		"emails_opened":            "emails_opened",
		"links_clicked":            "links_clicked",
		"data_submitted":           "data_submitted",
		"emails_reported":          "emails_reported",
		"campaigns_launched":       "campaigns_launched",
		"campaigns_completed":      "campaigns_completed",
		"training_assigned":        "training_assigned",
		"training_completed":       "training_completed",
		"training_overdue":         "training_overdue",
		"avg_quiz_score":           "avg_quiz_score",
		"certificates_issued":      "certificates_issued",
		"tickets_opened":           "tickets_opened",
		"tickets_resolved":         "tickets_resolved",
		"incidents_created":        "incidents_created",
		"incidents_resolved":       "incidents_resolved",
		"network_events_ingested":  "network_events_ingested",
		"avg_risk_score":           "avg_risk_score",
		"high_risk_user_count":     "high_risk_user_count",
		"compliance_score":         "compliance_score",
		"avg_hygiene_score":        "avg_hygiene_score",
		"click_rate":               "click_rate",
		"report_rate":              "report_rate",
		"training_completion_rate": "training_completion_rate",
		"active_campaigns":         "active_campaigns",
		"total_users":              "total_users",
	}
	return columns[metric]
}
