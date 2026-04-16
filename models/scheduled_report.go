package models

import (
	"encoding/json"
	"strings"
	"time"

	log "github.com/gophish/gophish/logger"
)

// ── Scheduled Reports ───────────────────────────────────────────
// Allows admins to configure recurring report delivery:
//   "Send me a weekly PDF summary every Monday at 8am"
// Stored in the scheduled_reports table, executed by the
// ScheduledReportWorker.

// Report type slugs that map to available export generators.
const (
	ReportTypeExecutiveSummary = "executive_summary"
	ReportTypeCampaigns        = "campaigns"
	ReportTypeTraining         = "training"
	ReportTypePhishingTickets  = "phishing_tickets"
	ReportTypeEmailSecurity    = "email_security"
	ReportTypeNetworkEvents    = "network_events"
	ReportTypeROI              = "roi"
	ReportTypeCompliance       = "compliance"
	ReportTypeHygiene          = "hygiene"
	ReportTypeRiskScores       = "risk_scores"
)

// Frequency constants.
const (
	FrequencyDaily     = "daily"
	FrequencyWeekly    = "weekly"
	FrequencyBiweekly  = "biweekly"
	FrequencyMonthly   = "monthly"
	FrequencyQuarterly = "quarterly"
)

// Schedule execution statuses.
const (
	ScheduleStatusSuccess = "success"
	ScheduleStatusError   = "error"
	ScheduleStatusRunning = "running"
	ScheduleStatusPending = "pending"
)

// ValidReportTypes lists all supported report types for validation.
var ValidReportTypes = map[string]bool{
	ReportTypeExecutiveSummary: true,
	ReportTypeCampaigns:        true,
	ReportTypeTraining:         true,
	ReportTypePhishingTickets:  true,
	ReportTypeEmailSecurity:    true,
	ReportTypeNetworkEvents:    true,
	ReportTypeROI:              true,
	ReportTypeCompliance:       true,
	ReportTypeHygiene:          true,
	ReportTypeRiskScores:       true,
}

// ValidFrequencies lists all supported recurrence intervals.
var ValidFrequencies = map[string]bool{
	FrequencyDaily:     true,
	FrequencyWeekly:    true,
	FrequencyBiweekly:  true,
	FrequencyMonthly:   true,
	FrequencyQuarterly: true,
}

// ValidExportFormats lists allowed output formats.
var ValidExportFormats = map[string]bool{
	"pdf":  true,
	"xlsx": true,
	"csv":  true,
}

// ScheduledReport represents a recurring report configuration.
type ScheduledReport struct {
	Id              int64      `json:"id" gorm:"primary_key;auto_increment"`
	OrgId           int64      `json:"org_id" gorm:"column:org_id"`
	UserId          int64      `json:"user_id" gorm:"column:user_id"`
	Name            string     `json:"name"`
	ReportType      string     `json:"report_type" gorm:"column:report_type"`
	Format          string     `json:"format"`
	Frequency       string     `json:"frequency"`
	DayOfWeek       int        `json:"day_of_week" gorm:"column:day_of_week"`   // 0=Sun, 1=Mon, ...
	DayOfMonth      int        `json:"day_of_month" gorm:"column:day_of_month"` // 1-28
	Hour            int        `json:"hour"`                                    // 0-23
	Minute          int        `json:"minute"`                                  // 0-59
	Timezone        string     `json:"timezone"`
	Recipients      string     `json:"recipients"` // comma-separated emails
	Subject         string     `json:"subject"`
	IncludeBranding bool       `json:"include_branding" gorm:"column:include_branding"`
	IsActive        bool       `json:"is_active" gorm:"column:is_active"`
	Filters         string     `json:"filters" gorm:"column:filters;type:text"` // JSON filter options
	LastRunAt       *time.Time `json:"last_run_at" gorm:"column:last_run_at"`
	NextRunAt       *time.Time `json:"next_run_at" gorm:"column:next_run_at"`
	LastStatus      string     `json:"last_status" gorm:"column:last_status"`
	LastError       string     `json:"last_error" gorm:"column:last_error"`
	RunCount        int        `json:"run_count" gorm:"column:run_count"`
	CreatedDate     time.Time  `json:"created_date" gorm:"column:created_date"`
	ModifiedDate    time.Time  `json:"modified_date" gorm:"column:modified_date"`
}

// TableName for GORM.
func (ScheduledReport) TableName() string { return "scheduled_reports" }

// ScheduledReportFilters holds optional filter params for report generation.
type ScheduledReportFilters struct {
	PeriodDays  int     `json:"period_days,omitempty"` // lookback days (default: based on frequency)
	GroupIds    []int64 `json:"group_ids,omitempty"`
	CampaignIds []int64 `json:"campaign_ids,omitempty"`
	DateFrom    string  `json:"date_from,omitempty"`
	DateTo      string  `json:"date_to,omitempty"`
}

// ParseFilters decodes the JSON filters field.
func (sr *ScheduledReport) ParseFilters() ScheduledReportFilters {
	var f ScheduledReportFilters
	if sr.Filters != "" && sr.Filters != "{}" {
		json.Unmarshal([]byte(sr.Filters), &f)
	}
	return f
}

// SetFilters encodes filter options to JSON.
func (sr *ScheduledReport) SetFilters(f ScheduledReportFilters) {
	b, _ := json.Marshal(f)
	sr.Filters = string(b)
}

// RecipientList returns a cleaned slice of email addresses.
func (sr *ScheduledReport) RecipientList() []string {
	raw := strings.Split(sr.Recipients, ",")
	out := make([]string, 0, len(raw))
	for _, r := range raw {
		r = strings.TrimSpace(r)
		if r != "" {
			out = append(out, r)
		}
	}
	return out
}

// ── Validation ──

// Validate ensures the scheduled report configuration is valid.
func (sr *ScheduledReport) Validate() string {
	if sr.Name == "" {
		return "name is required"
	}
	if !ValidReportTypes[sr.ReportType] {
		return "invalid report_type"
	}
	if !ValidExportFormats[sr.Format] {
		return "invalid format (must be pdf, xlsx, or csv)"
	}
	if !ValidFrequencies[sr.Frequency] {
		return "invalid frequency"
	}
	if sr.Hour < 0 || sr.Hour > 23 {
		return "hour must be 0-23"
	}
	if sr.Minute < 0 || sr.Minute > 59 {
		return "minute must be 0-59"
	}
	if sr.DayOfWeek < 0 || sr.DayOfWeek > 6 {
		return "day_of_week must be 0 (Sun) – 6 (Sat)"
	}
	if sr.DayOfMonth < 1 || sr.DayOfMonth > 28 {
		return "day_of_month must be 1-28"
	}
	if len(sr.RecipientList()) == 0 {
		return "at least one recipient email is required"
	}
	if sr.Timezone == "" {
		sr.Timezone = "UTC"
	}
	if _, err := time.LoadLocation(sr.Timezone); err != nil {
		return "invalid timezone"
	}
	return ""
}

// ── Schedule Computation ──

// ComputeNextRun calculates the next scheduled execution time based on
// the frequency, day, hour, minute, and timezone settings.
func (sr *ScheduledReport) ComputeNextRun() time.Time {
	loc, err := time.LoadLocation(sr.Timezone)
	if err != nil {
		loc = time.UTC
	}
	now := time.Now().In(loc)

	var next time.Time
	switch sr.Frequency {
	case FrequencyDaily:
		// Next occurrence at the configured hour:minute
		next = time.Date(now.Year(), now.Month(), now.Day(), sr.Hour, sr.Minute, 0, 0, loc)
		if !next.After(now) {
			next = next.AddDate(0, 0, 1)
		}

	case FrequencyWeekly:
		// Next occurrence of the configured weekday at hour:minute
		next = time.Date(now.Year(), now.Month(), now.Day(), sr.Hour, sr.Minute, 0, 0, loc)
		daysUntil := (sr.DayOfWeek - int(now.Weekday()) + 7) % 7
		if daysUntil == 0 && !next.After(now) {
			daysUntil = 7
		}
		next = next.AddDate(0, 0, daysUntil)

	case FrequencyBiweekly:
		next = time.Date(now.Year(), now.Month(), now.Day(), sr.Hour, sr.Minute, 0, 0, loc)
		daysUntil := (sr.DayOfWeek - int(now.Weekday()) + 7) % 7
		if daysUntil == 0 && !next.After(now) {
			daysUntil = 14
		}
		next = next.AddDate(0, 0, daysUntil)

	case FrequencyMonthly:
		// Next occurrence on day_of_month at hour:minute
		next = time.Date(now.Year(), now.Month(), sr.DayOfMonth, sr.Hour, sr.Minute, 0, 0, loc)
		if !next.After(now) {
			next = next.AddDate(0, 1, 0)
		}

	case FrequencyQuarterly:
		// First day of next quarter, or day_of_month in the first month of next quarter
		qMonth := ((int(now.Month())-1)/3)*3 + 1 // first month of current quarter
		next = time.Date(now.Year(), time.Month(qMonth), sr.DayOfMonth, sr.Hour, sr.Minute, 0, 0, loc)
		if !next.After(now) {
			next = time.Date(now.Year(), time.Month(qMonth+3), sr.DayOfMonth, sr.Hour, sr.Minute, 0, 0, loc)
		}

	default:
		next = now.Add(24 * time.Hour)
	}

	return next.UTC()
}

// DefaultLookbackDays returns a sensible default lookback period based on frequency.
func (sr *ScheduledReport) DefaultLookbackDays() int {
	switch sr.Frequency {
	case FrequencyDaily:
		return 1
	case FrequencyWeekly:
		return 7
	case FrequencyBiweekly:
		return 14
	case FrequencyMonthly:
		return 30
	case FrequencyQuarterly:
		return 90
	default:
		return 30
	}
}

// ── CRUD Operations ──

// GetScheduledReports returns all scheduled reports for an org.
func GetScheduledReports(orgId int64) ([]ScheduledReport, error) {
	var reports []ScheduledReport
	err := db.Where("org_id = ?", orgId).Order("created_date DESC").Find(&reports).Error
	return reports, err
}

// GetScheduledReport returns a single scheduled report by ID.
func GetScheduledReport(id, orgId int64) (*ScheduledReport, error) {
	sr := &ScheduledReport{}
	err := db.Where("id = ? AND org_id = ?", id, orgId).First(sr).Error
	return sr, err
}

// CreateScheduledReport inserts a new scheduled report and computes the
// first next_run_at time.
func CreateScheduledReport(sr *ScheduledReport) error {
	sr.CreatedDate = time.Now().UTC()
	sr.ModifiedDate = sr.CreatedDate
	sr.LastStatus = ScheduleStatusPending
	next := sr.ComputeNextRun()
	sr.NextRunAt = &next
	return db.Save(sr).Error
}

// UpdateScheduledReport updates an existing scheduled report.
func UpdateScheduledReport(sr *ScheduledReport) error {
	sr.ModifiedDate = time.Now().UTC()
	next := sr.ComputeNextRun()
	sr.NextRunAt = &next
	return db.Save(sr).Error
}

// DeleteScheduledReport removes a scheduled report.
func DeleteScheduledReport(id, orgId int64) error {
	return db.Where("id = ? AND org_id = ?", id, orgId).Delete(&ScheduledReport{}).Error
}

// ToggleScheduledReport enables or disables a scheduled report.
func ToggleScheduledReport(id, orgId int64, active bool) error {
	updates := map[string]interface{}{
		"is_active":     active,
		"modified_date": time.Now().UTC(),
	}
	if active {
		sr, err := GetScheduledReport(id, orgId)
		if err != nil {
			return err
		}
		next := sr.ComputeNextRun()
		updates["next_run_at"] = next
	}
	return db.Table("scheduled_reports").Where("id = ? AND org_id = ?", id, orgId).Updates(updates).Error
}

// ── Worker Query Functions ──

// GetDueScheduledReports returns all active reports whose next_run_at
// has passed, i.e., they're ready to be executed by the worker.
func GetDueScheduledReports() ([]ScheduledReport, error) {
	var reports []ScheduledReport
	err := db.Where("is_active = ? AND next_run_at <= ?", true, time.Now().UTC()).
		Find(&reports).Error
	return reports, err
}

// MarkScheduledReportRunning marks the report as currently executing.
func MarkScheduledReportRunning(id int64) error {
	return db.Table("scheduled_reports").Where("id = ?", id).
		Updates(map[string]interface{}{
			"last_status": ScheduleStatusRunning,
		}).Error
}

// MarkScheduledReportComplete records a successful execution and
// advances next_run_at.
func MarkScheduledReportComplete(sr *ScheduledReport) error {
	now := time.Now().UTC()
	next := sr.ComputeNextRun()
	return db.Table("scheduled_reports").Where("id = ?", sr.Id).
		Updates(map[string]interface{}{
			"last_run_at":   now,
			"next_run_at":   next,
			"last_status":   ScheduleStatusSuccess,
			"last_error":    "",
			"run_count":     sr.RunCount + 1,
			"modified_date": now,
		}).Error
}

// MarkScheduledReportFailed records a failed execution and still
// advances next_run_at so it retries next period.
func MarkScheduledReportFailed(sr *ScheduledReport, errMsg string) error {
	now := time.Now().UTC()
	next := sr.ComputeNextRun()
	log.Errorf("ScheduledReport %d (%s) failed: %s", sr.Id, sr.Name, errMsg)
	return db.Table("scheduled_reports").Where("id = ?", sr.Id).
		Updates(map[string]interface{}{
			"last_run_at":   now,
			"next_run_at":   next,
			"last_status":   ScheduleStatusError,
			"last_error":    errMsg,
			"run_count":     sr.RunCount + 1,
			"modified_date": now,
		}).Error
}

// ── Summary for admin display ──

// ScheduledReportSummary provides aggregate stats for the org.
type ScheduledReportSummary struct {
	TotalSchedules  int64 `json:"total_schedules"`
	ActiveSchedules int64 `json:"active_schedules"`
	TotalRuns       int64 `json:"total_runs"`
	FailedLast24h   int64 `json:"failed_last_24h"`
}

// GetScheduledReportSummary returns aggregate stats for scheduled reports.
func GetScheduledReportSummary(orgId int64) ScheduledReportSummary {
	s := ScheduledReportSummary{}
	db.Table("scheduled_reports").Where("org_id = ?", orgId).Count(&s.TotalSchedules)
	db.Table("scheduled_reports").Where("org_id = ? AND is_active = ?", orgId, true).Count(&s.ActiveSchedules)

	type sumRow struct{ Total int64 }
	var sr sumRow
	db.Table("scheduled_reports").Where("org_id = ?", orgId).
		Select("COALESCE(SUM(run_count), 0) as total").Scan(&sr)
	s.TotalRuns = sr.Total

	cutoff := time.Now().UTC().Add(-24 * time.Hour)
	db.Table("scheduled_reports").
		Where("org_id = ? AND last_status = ? AND last_run_at >= ?", orgId, ScheduleStatusError, cutoff).
		Count(&s.FailedLast24h)

	return s
}
