package models

import (
	"encoding/json"
	"time"

	log "github.com/gophish/gophish/logger"
)

// ── ROI Report History (Time-Series) ────────────────────────────
// Persists each generated ROI report snapshot so that historical trends
// can be plotted on the page without recomputing past periods.

// ROIReportRecord stores a serialised snapshot of a generated ROI report.
type ROIReportRecord struct {
	Id                 int64     `json:"id" gorm:"primary_key"`
	OrgId              int64     `json:"org_id" gorm:"column:org_id;index"`
	PeriodStart        time.Time `json:"period_start" gorm:"column:period_start"`
	PeriodEnd          time.Time `json:"period_end" gorm:"column:period_end"`
	Quarter            string    `json:"quarter" gorm:"column:quarter;size:10"` // e.g. "2026-Q1"
	ROIPercentage      float64   `json:"roi_percentage" gorm:"column:roi_percentage"`
	CostAvoidance      float64   `json:"cost_avoidance" gorm:"column:cost_avoidance"`
	ProgramCost        float64   `json:"program_cost" gorm:"column:program_cost"`
	ClickRate          float64   `json:"click_rate" gorm:"column:click_rate"`
	ReportRate         float64   `json:"report_rate" gorm:"column:report_rate"`
	IncidentsAvoided   int       `json:"incidents_avoided" gorm:"column:incidents_avoided"`
	RiskReduction      float64   `json:"risk_reduction" gorm:"column:risk_reduction"`
	TrainingCompletion float64   `json:"training_completion" gorm:"column:training_completion"`
	ComplianceScore    float64   `json:"compliance_score" gorm:"column:compliance_score"`
	ReportJSON         string    `json:"-" gorm:"column:report_json;type:text"` // Full report JSON
	GeneratedAt        time.Time `json:"generated_at" gorm:"column:generated_at"`
	GeneratedBy        int64     `json:"generated_by" gorm:"column:generated_by"`
}

// TableName returns the GORM table name.
func (ROIReportRecord) TableName() string { return "roi_reports" }

// QuarterLabel returns a "YYYY-QN" label for a given date.
func QuarterLabel(t time.Time) string {
	q := (int(t.Month())-1)/3 + 1
	return t.Format("2006") + "-Q" + string(rune('0'+q))
}

// SaveROIReportRecord persists a generated ROI report as a historical record.
func SaveROIReportRecord(rpt *ROIReport, generatedBy int64) error {
	reportJSON, err := json.Marshal(rpt)
	if err != nil {
		return err
	}

	record := ROIReportRecord{
		OrgId:              rpt.OrgId,
		PeriodStart:        rpt.PeriodStart,
		PeriodEnd:          rpt.PeriodEnd,
		Quarter:            QuarterLabel(rpt.PeriodEnd),
		ROIPercentage:      rpt.Metrics.ROIPercentage,
		CostAvoidance:      rpt.Metrics.CostAvoidance,
		ProgramCost:        rpt.ProgramCost,
		ClickRate:          rpt.Phishing.CurrentClickRate,
		ReportRate:         rpt.Phishing.CurrentReportRate,
		IncidentsAvoided:   rpt.Metrics.EstIncidentsAvoided,
		RiskReduction:      rpt.Metrics.OverallRiskReduction,
		TrainingCompletion: rpt.Training.CompletionRate,
		ComplianceScore:    rpt.Compliance.OverallScore,
		ReportJSON:         string(reportJSON),
		GeneratedAt:        time.Now().UTC(),
		GeneratedBy:        generatedBy,
	}

	return db.Create(&record).Error
}

// GetROIReportHistory returns all stored report records for an org, newest first.
func GetROIReportHistory(orgId int64) ([]ROIReportRecord, error) {
	var records []ROIReportRecord
	err := db.Where(queryWhereOrgID, orgId).
		Order("period_end DESC").
		Find(&records).Error
	return records, err
}

// GetROIReportHistoryLimited returns the N most recent records.
func GetROIReportHistoryLimited(orgId int64, limit int) ([]ROIReportRecord, error) {
	var records []ROIReportRecord
	err := db.Where(queryWhereOrgID, orgId).
		Order("period_end DESC").
		Limit(limit).
		Find(&records).Error
	return records, err
}

// GetROIQuarterlyTrend returns one record per quarter for time-series plotting.
func GetROIQuarterlyTrend(orgId int64) []ROIQuarterPoint {
	var records []ROIReportRecord
	err := db.Where(queryWhereOrgID, orgId).
		Order("period_end ASC").
		Find(&records).Error
	if err != nil {
		log.Warnf("roi quarterly trend: %v", err)
		return nil
	}

	// Deduplicate by quarter, keeping the latest per quarter
	qmap := map[string]ROIReportRecord{}
	for _, r := range records {
		existing, ok := qmap[r.Quarter]
		if !ok || r.GeneratedAt.After(existing.GeneratedAt) {
			qmap[r.Quarter] = r
		}
	}

	// Sort quarters
	quarters := make([]string, 0, len(qmap))
	for q := range qmap {
		quarters = append(quarters, q)
	}
	sortStrings(quarters)

	points := make([]ROIQuarterPoint, 0, len(quarters))
	for _, q := range quarters {
		r := qmap[q]
		points = append(points, ROIQuarterPoint{
			Quarter:            q,
			ROIPercentage:      r.ROIPercentage,
			CostAvoidance:      r.CostAvoidance,
			ProgramCost:        r.ProgramCost,
			ClickRate:          r.ClickRate,
			ReportRate:         r.ReportRate,
			IncidentsAvoided:   r.IncidentsAvoided,
			RiskReduction:      r.RiskReduction,
			TrainingCompletion: r.TrainingCompletion,
			ComplianceScore:    r.ComplianceScore,
		})
	}
	return points
}

// ROIQuarterPoint is one data point for the quarterly ROI trend chart.
type ROIQuarterPoint struct {
	Quarter            string  `json:"quarter"`
	ROIPercentage      float64 `json:"roi_percentage"`
	CostAvoidance      float64 `json:"cost_avoidance"`
	ProgramCost        float64 `json:"program_cost"`
	ClickRate          float64 `json:"click_rate"`
	ReportRate         float64 `json:"report_rate"`
	IncidentsAvoided   int     `json:"incidents_avoided"`
	RiskReduction      float64 `json:"risk_reduction"`
	TrainingCompletion float64 `json:"training_completion"`
	ComplianceScore    float64 `json:"compliance_score"`
}

// GetROIReportByID loads a specific historical report record.
func GetROIReportByID(id, orgId int64) (*ROIReportRecord, error) {
	var record ROIReportRecord
	err := db.Where("id = ? AND org_id = ?", id, orgId).First(&record).Error
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// DeleteROIReportRecord removes a historical record.
func DeleteROIReportRecord(id, orgId int64) error {
	return db.Where("id = ? AND org_id = ?", id, orgId).Delete(&ROIReportRecord{}).Error
}

// sortStrings sorts a string slice in place.
func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}
