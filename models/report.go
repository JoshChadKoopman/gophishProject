package models

import (
	"fmt"
	"math"
	"time"

	log "github.com/gophish/gophish/logger"
)

// ReportOverview contains aggregated stats across all campaigns for a user.
type ReportOverview struct {
	TotalCampaigns  int64         `json:"total_campaigns"`
	ActiveCampaigns int64         `json:"active_campaigns"`
	TotalRecipients int64         `json:"total_recipients"`
	AvgClickRate    float64       `json:"avg_click_rate"`
	AvgSubmitRate   float64       `json:"avg_submit_rate"`
	AvgReportRate   float64       `json:"avg_report_rate"`
	Stats           CampaignStats `json:"stats"`
}

// TrendPoint represents a single day's phishing metrics for trend charts.
type TrendPoint struct {
	Date          string  `json:"date"`
	Sent          int64   `json:"sent"`
	Opened        int64   `json:"opened"`
	Clicked       int64   `json:"clicked"`
	SubmittedData int64   `json:"submitted_data"`
	Reported      int64   `json:"reported"`
	ClickRate     float64 `json:"click_rate"`
}

// UserRiskScore represents a target's phishing susceptibility score.
type UserRiskScore struct {
	UserId    int64   `json:"user_id"`
	Email     string  `json:"email"`
	FirstName string  `json:"first_name"`
	LastName  string  `json:"last_name"`
	Total     int64   `json:"total_emails"`
	Clicked   int64   `json:"clicked"`
	Submitted int64   `json:"submitted"`
	Reported  int64   `json:"reported"`
	RiskScore float64 `json:"risk_score"`
}

// TrainingSummary contains aggregated training completion metrics.
type TrainingSummary struct {
	TotalCourses       int64   `json:"total_courses"`
	TotalAssignments   int64   `json:"total_assignments"`
	CompletedCount     int64   `json:"completed_count"`
	InProgressCount    int64   `json:"in_progress_count"`
	NotStartedCount    int64   `json:"not_started_count"`
	OverdueCount       int64   `json:"overdue_count"`
	CompletionRate     float64 `json:"completion_rate"`
	CertificatesIssued int64   `json:"certificates_issued"`
	AvgQuizScore       float64 `json:"avg_quiz_score"`
}

// GroupComparison contains phishing stats for a single group.
type GroupComparison struct {
	GroupId    int64         `json:"group_id"`
	GroupName  string        `json:"group_name"`
	Stats      CampaignStats `json:"stats"`
	ClickRate  float64       `json:"click_rate"`
	SubmitRate float64       `json:"submit_rate"`
}

// GetReportOverview aggregates stats across all campaigns for the given org scope.
func GetReportOverview(scope OrgScope) (ReportOverview, error) {
	overview := ReportOverview{}

	// Get all campaign IDs for this org
	type campaignRow struct {
		Id     int64
		Status string
	}
	var campaigns []campaignRow
	err := scopeQuery(db.Table("campaigns"), scope).Select("id, status").Scan(&campaigns).Error
	if err != nil {
		log.Error(err)
		return overview, err
	}

	overview.TotalCampaigns = int64(len(campaigns))
	if overview.TotalCampaigns == 0 {
		return overview, nil
	}

	for _, c := range campaigns {
		if c.Status == CampaignInProgress {
			overview.ActiveCampaigns++
		}
		s, err := getCampaignStats(c.Id)
		if err != nil {
			log.Error(err)
			continue
		}
		overview.Stats.Total += s.Total
		overview.Stats.EmailsSent += s.EmailsSent
		overview.Stats.OpenedEmail += s.OpenedEmail
		overview.Stats.ClickedLink += s.ClickedLink
		overview.Stats.SubmittedData += s.SubmittedData
		overview.Stats.EmailReported += s.EmailReported
		overview.Stats.Error += s.Error
	}

	overview.TotalRecipients = overview.Stats.Total
	if overview.Stats.Total > 0 {
		overview.AvgClickRate = math.Round(float64(overview.Stats.ClickedLink)*10000/float64(overview.Stats.Total)) / 100
		overview.AvgSubmitRate = math.Round(float64(overview.Stats.SubmittedData)*10000/float64(overview.Stats.Total)) / 100
		overview.AvgReportRate = math.Round(float64(overview.Stats.EmailReported)*10000/float64(overview.Stats.Total)) / 100
	}

	return overview, nil
}

// GetReportTrend returns daily phishing event counts for the last N days.
func GetReportTrend(scope OrgScope, days int) ([]TrendPoint, error) {
	if days <= 0 {
		days = 30
	}
	cutoff := time.Now().AddDate(0, 0, -days).Format("2006-01-02")

	// Build the org scope SQL fragment
	scopeCol, scopeVal := scopeRawSQL(scope)
	_ = scopeCol // used in raw SQL below

	// Raw SQL query: count events by type per day across all org campaigns
	type dayRow struct {
		Date    string
		Message string
		Count   int64
	}
	var rows []dayRow
	var err error
	if scope.IsSuperAdmin {
		err = db.Raw(`
			SELECT DATE(e.time) as date, e.message, COUNT(*) as count
			FROM events e
			JOIN campaigns c ON e.campaign_id = c.id
			WHERE DATE(e.time) >= ?
			GROUP BY DATE(e.time), e.message
			ORDER BY date
		`, cutoff).Scan(&rows).Error
	} else {
		err = db.Raw(`
			SELECT DATE(e.time) as date, e.message, COUNT(*) as count
			FROM events e
			JOIN campaigns c ON e.campaign_id = c.id
			WHERE c.org_id = ? AND DATE(e.time) >= ?
			GROUP BY DATE(e.time), e.message
			ORDER BY date
		`, scopeVal, cutoff).Scan(&rows).Error
	}
	if err != nil {
		log.Error(err)
		return nil, err
	}

	// Also count reported emails per day from results table
	type reportedRow struct {
		Date  string
		Count int64
	}
	var reportedRows []reportedRow
	if scope.IsSuperAdmin {
		err = db.Raw(`
			SELECT DATE(r.modified_date) as date, COUNT(*) as count
			FROM results r
			JOIN campaigns c ON r.campaign_id = c.id
			WHERE r.reported = 1 AND DATE(r.modified_date) >= ?
			GROUP BY DATE(r.modified_date)
		`, cutoff).Scan(&reportedRows).Error
	} else {
		err = db.Raw(`
			SELECT DATE(r.modified_date) as date, COUNT(*) as count
			FROM results r
			JOIN campaigns c ON r.campaign_id = c.id
			WHERE c.org_id = ? AND r.reported = 1 AND DATE(r.modified_date) >= ?
			GROUP BY DATE(r.modified_date)
		`, scopeVal, cutoff).Scan(&reportedRows).Error
	}
	if err != nil {
		log.Error(err)
		// Non-fatal, continue without reported data
	}

	// Build a map of date -> TrendPoint
	pointMap := make(map[string]*TrendPoint)
	for _, row := range rows {
		tp, ok := pointMap[row.Date]
		if !ok {
			tp = &TrendPoint{Date: row.Date}
			pointMap[row.Date] = tp
		}
		switch row.Message {
		case EventSent:
			tp.Sent += row.Count
		case EventOpened:
			tp.Opened += row.Count
		case EventClicked:
			tp.Clicked += row.Count
		case EventDataSubmit:
			tp.SubmittedData += row.Count
		}
	}

	for _, rr := range reportedRows {
		tp, ok := pointMap[rr.Date]
		if !ok {
			tp = &TrendPoint{Date: rr.Date}
			pointMap[rr.Date] = tp
		}
		tp.Reported += rr.Count
	}

	// Convert map to sorted slice and compute click rates
	var points []TrendPoint
	start := time.Now().AddDate(0, 0, -days)
	for i := 0; i <= days; i++ {
		d := start.AddDate(0, 0, i).Format("2006-01-02")
		if tp, ok := pointMap[d]; ok {
			if tp.Sent > 0 {
				tp.ClickRate = math.Round(float64(tp.Clicked)*10000/float64(tp.Sent)) / 100
			}
			points = append(points, *tp)
		} else {
			points = append(points, TrendPoint{Date: d})
		}
	}

	return points, nil
}

// GetRiskScores computes per-target risk scores across all campaigns for the given org scope.
func GetRiskScores(scope OrgScope) ([]UserRiskScore, error) {
	type riskRow struct {
		Email     string
		FirstName string
		LastName  string
		Total     int64
		Clicked   int64
		Submitted int64
		Reported  int64
	}

	var rows []riskRow
	var err error
	if scope.IsSuperAdmin {
		err = db.Raw(`
			SELECT r.email, r.first_name, r.last_name,
				COUNT(*) as total,
				SUM(CASE WHEN r.status IN (?, ?) THEN 1 ELSE 0 END) as clicked,
				SUM(CASE WHEN r.status = ? THEN 1 ELSE 0 END) as submitted,
				SUM(CASE WHEN r.reported = 1 THEN 1 ELSE 0 END) as reported
			FROM results r
			JOIN campaigns c ON r.campaign_id = c.id
			GROUP BY r.email
			ORDER BY clicked DESC, submitted DESC
		`, EventClicked, EventDataSubmit, EventDataSubmit).Scan(&rows).Error
	} else {
		err = db.Raw(`
			SELECT r.email, r.first_name, r.last_name,
				COUNT(*) as total,
				SUM(CASE WHEN r.status IN (?, ?) THEN 1 ELSE 0 END) as clicked,
				SUM(CASE WHEN r.status = ? THEN 1 ELSE 0 END) as submitted,
				SUM(CASE WHEN r.reported = 1 THEN 1 ELSE 0 END) as reported
			FROM results r
			JOIN campaigns c ON r.campaign_id = c.id
			WHERE c.org_id = ?
			GROUP BY r.email
			ORDER BY clicked DESC, submitted DESC
		`, EventClicked, EventDataSubmit, EventDataSubmit, scope.OrgId).Scan(&rows).Error
	}
	if err != nil {
		log.Error(err)
		return nil, err
	}

	scores := make([]UserRiskScore, 0, len(rows))
	for _, row := range rows {
		score := UserRiskScore{
			Email:     row.Email,
			FirstName: row.FirstName,
			LastName:  row.LastName,
			Total:     row.Total,
			Clicked:   row.Clicked,
			Submitted: row.Submitted,
			Reported:  row.Reported,
		}

		if row.Total > 0 {
			// Risk formula: weight clicks (1x) and submissions (2x), subtract reports
			// Normalize to 0-100 scale
			raw := float64(row.Clicked+row.Submitted*2-row.Reported) / float64(row.Total*3) * 100
			score.RiskScore = math.Round(math.Max(0, math.Min(100, raw))*100) / 100
		}

		scores = append(scores, score)
	}

	// Sort by risk score descending
	for i := 0; i < len(scores); i++ {
		for j := i + 1; j < len(scores); j++ {
			if scores[j].RiskScore > scores[i].RiskScore {
				scores[i], scores[j] = scores[j], scores[i]
			}
		}
	}

	return scores, nil
}

// GetTrainingSummaryReport returns aggregated training completion metrics.
func GetTrainingSummaryReport(scope OrgScope) (TrainingSummary, error) {
	summary := TrainingSummary{}

	// Count total courses
	scopeQuery(db.Table("training_presentations"), scope).Count(&summary.TotalCourses)

	// Count assignments by status
	scopeQuery(db.Table("course_assignments"), scope).Count(&summary.TotalAssignments)
	scopeQuery(db.Table("course_assignments"), scope).Where("status = ?", AssignmentStatusCompleted).Count(&summary.CompletedCount)
	scopeQuery(db.Table("course_assignments"), scope).Where("status = ?", AssignmentStatusInProgress).Count(&summary.InProgressCount)
	scopeQuery(db.Table("course_assignments"), scope).Where("status = ?", AssignmentStatusPending).Count(&summary.NotStartedCount)

	// Count overdue (pending or in_progress with due_date in the past)
	scopeQuery(db.Table("course_assignments"), scope).
		Where("status != ? AND due_date < ? AND due_date > ?",
			AssignmentStatusCompleted, time.Now(), time.Time{}).
		Count(&summary.OverdueCount)

	// Completion rate
	if summary.TotalAssignments > 0 {
		summary.CompletionRate = math.Round(float64(summary.CompletedCount)*10000/float64(summary.TotalAssignments)) / 100
	}

	// Count certificates
	scopeQuery(db.Table("certificates"), scope).Count(&summary.CertificatesIssued)

	// Average quiz score from passed attempts
	type avgRow struct {
		AvgScore float64
	}
	var avg avgRow
	err := db.Raw(`
		SELECT COALESCE(AVG(CASE WHEN total_questions > 0 THEN score * 100.0 / total_questions ELSE 0 END), 0) as avg_score
		FROM quiz_attempts
	`).Scan(&avg).Error
	if err == nil {
		summary.AvgQuizScore = math.Round(avg.AvgScore*100) / 100
	}

	return summary, nil
}

// GetGroupComparison returns phishing stats broken down by group.
func GetGroupComparison(scope OrgScope) ([]GroupComparison, error) {
	// Get all groups for this org
	var groups []Group
	err := scopeQuery(db.Table("groups"), scope).Find(&groups).Error
	if err != nil {
		log.Error(err)
		return nil, err
	}

	comparisons := make([]GroupComparison, 0, len(groups))
	for _, g := range groups {
		gc := GroupComparison{
			GroupId:   g.Id,
			GroupName: g.Name,
		}

		// Get target emails for this group
		targets, err := GetTargets(g.Id)
		if err != nil {
			log.Error(err)
			continue
		}
		if len(targets) == 0 {
			comparisons = append(comparisons, gc)
			continue
		}

		emails := make([]interface{}, len(targets))
		placeholders := ""
		for i, t := range targets {
			emails[i] = t.Email
			if i > 0 {
				placeholders += ","
			}
			placeholders += "?"
		}

		// Aggregate results for these target emails across all org campaigns
		var query string
		var args []interface{}
		if scope.IsSuperAdmin {
			query = fmt.Sprintf(`
				SELECT COUNT(*) as total,
					SUM(CASE WHEN r.status = '%s' THEN 1 ELSE 0 END) as submitted_data,
					SUM(CASE WHEN r.status = '%s' THEN 1 ELSE 0 END) as clicked_link,
					SUM(CASE WHEN r.status = '%s' THEN 1 ELSE 0 END) as opened_email,
					SUM(CASE WHEN r.status = '%s' THEN 1 ELSE 0 END) as emails_sent,
					SUM(CASE WHEN r.reported = 1 THEN 1 ELSE 0 END) as email_reported
				FROM results r
				JOIN campaigns c ON r.campaign_id = c.id
				WHERE r.email IN (%s)
			`, EventDataSubmit, EventClicked, EventOpened, EventSent, placeholders)
			args = emails
		} else {
			query = fmt.Sprintf(`
				SELECT COUNT(*) as total,
					SUM(CASE WHEN r.status = '%s' THEN 1 ELSE 0 END) as submitted_data,
					SUM(CASE WHEN r.status = '%s' THEN 1 ELSE 0 END) as clicked_link,
					SUM(CASE WHEN r.status = '%s' THEN 1 ELSE 0 END) as opened_email,
					SUM(CASE WHEN r.status = '%s' THEN 1 ELSE 0 END) as emails_sent,
					SUM(CASE WHEN r.reported = 1 THEN 1 ELSE 0 END) as email_reported
				FROM results r
				JOIN campaigns c ON r.campaign_id = c.id
				WHERE c.org_id = ? AND r.email IN (%s)
			`, EventDataSubmit, EventClicked, EventOpened, EventSent, placeholders)
			args = append([]interface{}{scope.OrgId}, emails...)
		}
		var stats CampaignStats
		err = db.Raw(query, args...).Scan(&stats).Error
		if err != nil {
			log.Error(err)
			continue
		}

		// Apply backfill logic (same as getCampaignStats)
		stats.ClickedLink += stats.SubmittedData
		stats.OpenedEmail += stats.ClickedLink
		stats.EmailsSent += stats.OpenedEmail

		gc.Stats = stats
		if stats.Total > 0 {
			gc.ClickRate = math.Round(float64(stats.ClickedLink)*10000/float64(stats.Total)) / 100
			gc.SubmitRate = math.Round(float64(stats.SubmittedData)*10000/float64(stats.Total)) / 100
		}

		comparisons = append(comparisons, gc)
	}

	return comparisons, nil
}
