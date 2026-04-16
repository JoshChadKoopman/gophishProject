package models

import (
	"fmt"
	"time"
)

// ── Board Report Scheduled Generation ───────────────────────────
// Cron-style scheduler that auto-generates monthly/quarterly board reports.

const (
	BoardScheduleMonthly   = "monthly"
	BoardScheduleQuarterly = "quarterly"
)

// BoardReportSchedule defines an auto-generation schedule for board reports.
type BoardReportSchedule struct {
	Id           int64     `json:"id" gorm:"primary_key"`
	OrgId        int64     `json:"org_id" gorm:"column:org_id;index"`
	Frequency    string    `json:"frequency" gorm:"column:frequency;size:20"` // monthly, quarterly
	DayOfMonth   int       `json:"day_of_month" gorm:"column:day_of_month"`   // 1-28
	Enabled      bool      `json:"enabled" gorm:"column:enabled;default:true"`
	AutoPublish  bool      `json:"auto_publish" gorm:"column:auto_publish;default:false"`
	NotifyEmails string    `json:"notify_emails" gorm:"column:notify_emails;type:text"` // comma-separated
	CreatedBy    int64     `json:"created_by" gorm:"column:created_by"`
	LastRunDate  time.Time `json:"last_run_date" gorm:"column:last_run_date"`
	NextRunDate  time.Time `json:"next_run_date" gorm:"column:next_run_date"`
	CreatedDate  time.Time `json:"created_date" gorm:"column:created_date"`
	ModifiedDate time.Time `json:"modified_date" gorm:"column:modified_date"`
}

func (BoardReportSchedule) TableName() string { return "board_report_schedules" }

// GetBoardReportSchedules returns all schedules for an org.
func GetBoardReportSchedules(orgId int64) ([]BoardReportSchedule, error) {
	var schedules []BoardReportSchedule
	err := db.Where("org_id = ?", orgId).Order("created_date DESC").Find(&schedules).Error
	return schedules, err
}

// GetBoardReportSchedule returns a single schedule by ID.
func GetBoardReportSchedule(id, orgId int64) (BoardReportSchedule, error) {
	var s BoardReportSchedule
	err := db.Where("id = ? AND org_id = ?", id, orgId).First(&s).Error
	return s, err
}

// SaveBoardReportSchedule creates or updates a schedule.
func SaveBoardReportSchedule(s *BoardReportSchedule) error {
	if s.Frequency != BoardScheduleMonthly && s.Frequency != BoardScheduleQuarterly {
		return fmt.Errorf("frequency must be 'monthly' or 'quarterly'")
	}
	if s.DayOfMonth < 1 || s.DayOfMonth > 28 {
		s.DayOfMonth = 1
	}
	s.ModifiedDate = time.Now().UTC()
	if s.CreatedDate.IsZero() {
		s.CreatedDate = time.Now().UTC()
	}
	if s.NextRunDate.IsZero() {
		s.NextRunDate = computeNextRun(s.Frequency, s.DayOfMonth, time.Now().UTC())
	}
	return db.Save(s).Error
}

// DeleteBoardReportSchedule removes a schedule.
func DeleteBoardReportSchedule(id, orgId int64) error {
	return db.Where("id = ? AND org_id = ?", id, orgId).Delete(&BoardReportSchedule{}).Error
}

// GetDueBoardReportSchedules returns schedules that are due to run.
func GetDueBoardReportSchedules() ([]BoardReportSchedule, error) {
	var schedules []BoardReportSchedule
	err := db.Where("enabled = ? AND next_run_date <= ?", true, time.Now().UTC()).Find(&schedules).Error
	return schedules, err
}

// RunScheduledBoardReports checks for due schedules and generates reports.
// Returns the number of reports generated.
func RunScheduledBoardReports() (int, error) {
	schedules, err := GetDueBoardReportSchedules()
	if err != nil {
		return 0, err
	}

	generated := 0
	for _, s := range schedules {
		// Compute period based on frequency
		now := time.Now().UTC()
		var periodStart, periodEnd time.Time

		switch s.Frequency {
		case BoardScheduleQuarterly:
			periodEnd = now
			periodStart = now.AddDate(0, -3, 0)
		default: // monthly
			periodEnd = now
			periodStart = now.AddDate(0, -1, 0)
		}

		// Generate the report
		title := fmt.Sprintf("Auto-Generated %s Board Report — %s",
			capitalize(s.Frequency),
			now.Format("January 2006"))

		br := &BoardReport{
			OrgId:       s.OrgId,
			CreatedBy:   s.CreatedBy,
			Title:       title,
			PeriodStart: periodStart,
			PeriodEnd:   periodEnd,
		}

		if err := PostBoardReport(br); err != nil {
			continue
		}

		// Generate and store a narrative
		snap, snapErr := GenerateBoardReportSnapshot(s.OrgId, periodStart, periodEnd)
		if snapErr == nil && snap != nil {
			deltas := ComputePeriodDeltas(snap, nil)
			heatmap, _ := GenerateDeptHeatmap(s.OrgId)
			narrative := BuildDeterministicNarrative(snap, deltas, heatmap)
			narrative.OrgId = s.OrgId
			narrative.ReportId = br.Id
			SaveBoardReportNarrative(narrative)
		}

		// Auto-publish if configured
		if s.AutoPublish {
			br.Status = BoardReportStatusPublished
			PutBoardReport(br)
		}

		// Update schedule
		s.LastRunDate = now
		s.NextRunDate = computeNextRun(s.Frequency, s.DayOfMonth, now)
		db.Save(&s)

		generated++
	}

	return generated, nil
}

// computeNextRun calculates the next run date based on frequency.
func computeNextRun(frequency string, dayOfMonth int, after time.Time) time.Time {
	year, month, _ := after.Date()

	switch frequency {
	case BoardScheduleQuarterly:
		// Next quarter start
		nextMonth := int(month) + 3 - ((int(month) - 1) % 3)
		if nextMonth > 12 {
			nextMonth -= 12
			year++
		}
		return time.Date(year, time.Month(nextMonth), dayOfMonth, 8, 0, 0, 0, time.UTC)
	default: // monthly
		nextMonth := month + 1
		nextYear := year
		if nextMonth > 12 {
			nextMonth = 1
			nextYear++
		}
		return time.Date(nextYear, nextMonth, dayOfMonth, 8, 0, 0, 0, time.UTC)
	}
}

func capitalize(s string) string {
	if len(s) == 0 {
		return s
	}
	return string(s[0]-32) + s[1:]
}
