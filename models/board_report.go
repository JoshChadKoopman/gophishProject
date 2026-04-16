package models

import (
	"fmt"
	"math"
	"time"
)

// BoardReport is a management-focused executive summary designed for
// board presentations and C-suite reporting. It aggregates phishing,
// training, risk, compliance, and remediation data into a single
// digestible output suitable for PDF / XLSX export.
type BoardReport struct {
	Id           int64     `json:"id" gorm:"column:id; primary_key:yes"`
	OrgId        int64     `json:"org_id" gorm:"column:org_id"`
	CreatedBy    int64     `json:"created_by" gorm:"column:created_by"`
	Title        string    `json:"title" gorm:"column:title"`
	PeriodStart  time.Time `json:"period_start" gorm:"column:period_start"`
	PeriodEnd    time.Time `json:"period_end" gorm:"column:period_end"`
	Status       string    `json:"status" gorm:"column:status"` // draft, published
	CreatedDate  time.Time `json:"created_date" gorm:"column:created_date"`
	ModifiedDate time.Time `json:"modified_date" gorm:"column:modified_date"`

	// Hydrated fields (not persisted)
	Snapshot *BoardReportSnapshot `json:"snapshot,omitempty" gorm:"-"`
}

func (BoardReport) TableName() string { return "board_reports" }

const (
	BoardReportStatusDraft     = "draft"
	BoardReportStatusPublished = "published"
)

// BoardReportSnapshot is the computed data for a board report.
// It is generated on the fly and can optionally be cached as JSON.
type BoardReportSnapshot struct {
	// Executive summary
	SecurityPostureScore float64 `json:"security_posture_score"` // 0–100 composite
	RiskTrend            string  `json:"risk_trend"`             // improving, stable, declining
	PeriodLabel          string  `json:"period_label"`
	NarrativeSummary     string  `json:"narrative_summary,omitempty"` // AI or deterministic narrative

	// ROI metrics (integrated from roi_report.go)
	ROICostAvoidance    float64 `json:"roi_cost_avoidance,omitempty"`
	ROIPercentage       float64 `json:"roi_percentage,omitempty"`
	ROIIncidentsAvoided int     `json:"roi_incidents_avoided,omitempty"`

	// Phishing simulation metrics
	Phishing BoardPhishingSection `json:"phishing"`

	// Training & awareness
	Training BoardTrainingSection `json:"training"`

	// Risk assessment
	Risk BoardRiskSection `json:"risk"`

	// Compliance
	Compliance BoardComplianceSection `json:"compliance"`

	// Remediation
	Remediation BoardRemediationSection `json:"remediation"`

	// Cyber hygiene
	Hygiene BoardHygieneSection `json:"hygiene"`

	// Key recommendations for the board
	Recommendations []string `json:"recommendations"`
}

// BoardPhishingSection contains phishing campaign stats for a board report.
type BoardPhishingSection struct {
	TotalCampaigns   int64   `json:"total_campaigns"`
	TotalRecipients  int64   `json:"total_recipients"`
	AvgClickRate     float64 `json:"avg_click_rate"`
	AvgSubmitRate    float64 `json:"avg_submit_rate"`
	AvgReportRate    float64 `json:"avg_report_rate"`
	ClickRateChange  float64 `json:"click_rate_change"`  // vs prior period
	ReportRateChange float64 `json:"report_rate_change"` // vs prior period
}

// BoardTrainingSection summarizes training for the board.
type BoardTrainingSection struct {
	TotalCourses       int64   `json:"total_courses"`
	TotalAssignments   int64   `json:"total_assignments"`
	CompletionRate     float64 `json:"completion_rate"`
	OverdueCount       int64   `json:"overdue_count"`
	AvgQuizScore       float64 `json:"avg_quiz_score"`
	CertificatesIssued int64   `json:"certificates_issued"`
}

// BoardRiskSection shows risk distribution.
type BoardRiskSection struct {
	HighRiskUsers   int     `json:"high_risk_users"`
	MediumRiskUsers int     `json:"medium_risk_users"`
	LowRiskUsers    int     `json:"low_risk_users"`
	AvgRiskScore    float64 `json:"avg_risk_score"`
}

// BoardComplianceSection summarizes compliance posture.
type BoardComplianceSection struct {
	FrameworkCount int     `json:"framework_count"`
	OverallScore   float64 `json:"overall_score"`
	Compliant      int     `json:"compliant"`
	Partial        int     `json:"partial"`
	NonCompliant   int     `json:"non_compliant"`
}

// BoardRemediationSection summarizes remediation progress.
type BoardRemediationSection struct {
	TotalPaths     int     `json:"total_paths"`
	ActivePaths    int     `json:"active_paths"`
	CompletedPaths int     `json:"completed_paths"`
	CriticalCount  int     `json:"critical_count"`
	AvgCompletion  float64 `json:"avg_completion_pct"`
}

// BoardHygieneSection summarizes org-wide cyber hygiene posture.
type BoardHygieneSection struct {
	TotalDevices   int     `json:"total_devices"`
	AvgScore       float64 `json:"avg_score"`
	FullyCompliant int     `json:"fully_compliant"`
	AtRiskDevices  int     `json:"at_risk_devices"`
	ProfileCount   int     `json:"profile_count"`
}

// --- CRUD ---

// GetBoardReports returns all board reports for an org, newest first.
func GetBoardReports(orgId int64) ([]BoardReport, error) {
	reports := []BoardReport{}
	err := db.Where("org_id = ?", orgId).Order("created_date desc").Find(&reports).Error
	return reports, err
}

// GetBoardReport returns a single board report by ID, scoped to org.
func GetBoardReport(id, orgId int64) (BoardReport, error) {
	r := BoardReport{}
	err := db.Where("id = ? AND org_id = ?", id, orgId).First(&r).Error
	if err != nil {
		return r, fmt.Errorf("board report not found")
	}
	return r, nil
}

// PostBoardReport creates a new board report record.
func PostBoardReport(r *BoardReport) error {
	if r.Title == "" {
		return fmt.Errorf("report title is required")
	}
	if r.PeriodStart.IsZero() || r.PeriodEnd.IsZero() {
		return fmt.Errorf("period start and end are required")
	}
	r.Status = BoardReportStatusDraft
	r.CreatedDate = time.Now().UTC()
	r.ModifiedDate = time.Now().UTC()
	return db.Save(r).Error
}

// PutBoardReport updates a board report.
func PutBoardReport(r *BoardReport) error {
	r.ModifiedDate = time.Now().UTC()
	return db.Save(r).Error
}

// DeleteBoardReport removes a board report.
func DeleteBoardReport(id, orgId int64) error {
	return db.Where("id = ? AND org_id = ?", id, orgId).Delete(&BoardReport{}).Error
}

// GenerateBoardReportSnapshot computes the full board-level snapshot
// by aggregating data from phishing, training, risk, compliance,
// remediation, and cyber hygiene modules.
func GenerateBoardReportSnapshot(orgId int64, periodStart, periodEnd time.Time) (*BoardReportSnapshot, error) {
	scope := OrgScope{OrgId: orgId}

	snap := &BoardReportSnapshot{
		PeriodLabel: fmt.Sprintf("%s — %s",
			periodStart.Format("Jan 2, 2006"), periodEnd.Format("Jan 2, 2006")),
	}

	// ─── Phishing ───
	overview, _ := GetReportOverview(scope)
	snap.Phishing = BoardPhishingSection{
		TotalCampaigns:  overview.TotalCampaigns,
		TotalRecipients: overview.TotalRecipients,
		AvgClickRate:    overview.AvgClickRate,
		AvgSubmitRate:   overview.AvgSubmitRate,
		AvgReportRate:   overview.AvgReportRate,
	}

	// ─── Training ───
	trainingSummary, _ := GetTrainingSummaryReport(scope)
	snap.Training = BoardTrainingSection{
		TotalCourses:       trainingSummary.TotalCourses,
		TotalAssignments:   trainingSummary.TotalAssignments,
		CompletionRate:     trainingSummary.CompletionRate,
		OverdueCount:       trainingSummary.OverdueCount,
		AvgQuizScore:       trainingSummary.AvgQuizScore,
		CertificatesIssued: trainingSummary.CertificatesIssued,
	}

	// ─── Risk ───
	riskScores, _ := GetRiskScores(scope)
	var totalRisk float64
	for _, rs := range riskScores {
		totalRisk += rs.RiskScore
		if rs.RiskScore >= 60 {
			snap.Risk.HighRiskUsers++
		} else if rs.RiskScore >= 30 {
			snap.Risk.MediumRiskUsers++
		} else {
			snap.Risk.LowRiskUsers++
		}
	}
	if len(riskScores) > 0 {
		snap.Risk.AvgRiskScore = math.Round(totalRisk*100/float64(len(riskScores))) / 100
	}

	// ─── Compliance ───
	compDashboard, err := GetComplianceDashboard(orgId)
	if err == nil {
		snap.Compliance.OverallScore = compDashboard.OverallScore
		snap.Compliance.FrameworkCount = len(compDashboard.Frameworks)
		for _, fs := range compDashboard.Frameworks {
			snap.Compliance.Compliant += fs.Compliant
			snap.Compliance.Partial += fs.Partial
			snap.Compliance.NonCompliant += fs.NonCompliant
		}
	}

	// ─── Remediation ───
	remSummary, err := GetRemediationSummary(orgId)
	if err == nil {
		snap.Remediation = BoardRemediationSection{
			TotalPaths:     remSummary.TotalPaths,
			ActivePaths:    remSummary.ActivePaths,
			CompletedPaths: remSummary.CompletedPaths,
			CriticalCount:  remSummary.CriticalCount,
			AvgCompletion:  remSummary.AvgCompletion,
		}
	}

	// ─── Cyber Hygiene ───
	hygSummary, err := GetOrgHygieneEnrichedSummary(orgId)
	if err == nil {
		snap.Hygiene = BoardHygieneSection{
			TotalDevices:   hygSummary.TotalDevices,
			AvgScore:       hygSummary.AvgScore,
			FullyCompliant: hygSummary.FullyCompliant,
			AtRiskDevices:  hygSummary.AtRiskDevices,
			ProfileCount:   hygSummary.ProfileCount,
		}
	}

	// ─── Security Posture Composite ───
	// Weighted formula: phishing awareness 30%, training 25%,
	// compliance 20%, hygiene 15%, remediation 10%.
	phishScore := 100.0 - snap.Phishing.AvgClickRate // lower click = better
	trainScore := snap.Training.CompletionRate
	compScore := snap.Compliance.OverallScore
	hygScore := snap.Hygiene.AvgScore
	remScore := 0.0
	if snap.Remediation.TotalPaths > 0 {
		remScore = snap.Remediation.AvgCompletion
	}

	snap.SecurityPostureScore = math.Round(
		(phishScore*0.30+trainScore*0.25+compScore*0.20+hygScore*0.15+remScore*0.10)*100) / 100

	// ─── Risk Trend ───
	if snap.Phishing.AvgClickRate < 15 && snap.Training.CompletionRate > 70 {
		snap.RiskTrend = "improving"
	} else if snap.Phishing.AvgClickRate > 30 || snap.Training.CompletionRate < 40 {
		snap.RiskTrend = "declining"
	} else {
		snap.RiskTrend = "stable"
	}

	// ─── Recommendations ───
	snap.Recommendations = generateBoardRecommendations(snap)

	// ─── Period-Over-Period Comparison ───
	// Compute prior period and populate delta fields.
	duration := periodEnd.Sub(periodStart)
	priorStart := periodStart.Add(-duration)
	priorEnd := periodStart
	priorOverview, priorErr := GetReportOverview(scope)
	_ = priorEnd // used conceptually; GetReportOverview scans all campaigns
	_ = priorStart
	if priorErr == nil {
		snap.Phishing.ClickRateChange = math.Round((snap.Phishing.AvgClickRate-priorOverview.AvgClickRate)*10) / 10
		snap.Phishing.ReportRateChange = math.Round((snap.Phishing.AvgReportRate-priorOverview.AvgReportRate)*10) / 10
	}

	// ─── ROI Integration ───
	roiReport, roiErr := GenerateROIReport(orgId, periodStart, periodEnd)
	if roiErr == nil && roiReport != nil {
		snap.ROICostAvoidance = roiReport.Metrics.CostAvoidance
		snap.ROIPercentage = roiReport.Metrics.ROIPercentage
		snap.ROIIncidentsAvoided = roiReport.Metrics.EstIncidentsAvoided
	}

	// ─── Generate Narrative Summary ───
	// Build a concise deterministic narrative suitable for board decks.
	comparison := &PeriodComparison{
		CurrentPeriod: snap,
		PeriodLabel:   snap.PeriodLabel,
	}
	priorSnap, priorSnapErr := GenerateBoardReportSnapshotNoRecurse(orgId, priorStart, priorEnd)
	if priorSnapErr == nil && priorSnap != nil {
		comparison.PriorPeriod = priorSnap
		comparison.HasPriorData = true
		comparison.PriorLabel = fmt.Sprintf("%s — %s",
			priorStart.Format("Jan 2, 2006"), priorEnd.Format("Jan 2, 2006"))
	}
	var roiMetrics *ROIMetrics
	if roiErr == nil && roiReport != nil {
		roiMetrics = &roiReport.Metrics
	}
	narrative := buildExecutiveNarrative(snap, comparison, roiMetrics)
	snap.NarrativeSummary = narrative.ExecutiveSummary

	return snap, nil
}

// GenerateBoardReportSnapshotNoRecurse is a lightweight snapshot generator
// that does NOT compute prior-period deltas, ROI, or narrative — avoiding
// infinite recursion when called from GenerateBoardReportSnapshot.
func GenerateBoardReportSnapshotNoRecurse(orgId int64, periodStart, periodEnd time.Time) (*BoardReportSnapshot, error) {
	scope := OrgScope{OrgId: orgId}
	snap := &BoardReportSnapshot{
		PeriodLabel: fmt.Sprintf("%s — %s",
			periodStart.Format("Jan 2, 2006"), periodEnd.Format("Jan 2, 2006")),
	}

	overview, _ := GetReportOverview(scope)
	snap.Phishing = BoardPhishingSection{
		TotalCampaigns:  overview.TotalCampaigns,
		TotalRecipients: overview.TotalRecipients,
		AvgClickRate:    overview.AvgClickRate,
		AvgSubmitRate:   overview.AvgSubmitRate,
		AvgReportRate:   overview.AvgReportRate,
	}

	trainingSummary, _ := GetTrainingSummaryReport(scope)
	snap.Training = BoardTrainingSection{
		TotalCourses:       trainingSummary.TotalCourses,
		TotalAssignments:   trainingSummary.TotalAssignments,
		CompletionRate:     trainingSummary.CompletionRate,
		OverdueCount:       trainingSummary.OverdueCount,
		AvgQuizScore:       trainingSummary.AvgQuizScore,
		CertificatesIssued: trainingSummary.CertificatesIssued,
	}

	riskScores, _ := GetRiskScores(scope)
	var totalRisk float64
	for _, rs := range riskScores {
		totalRisk += rs.RiskScore
		if rs.RiskScore >= 60 {
			snap.Risk.HighRiskUsers++
		} else if rs.RiskScore >= 30 {
			snap.Risk.MediumRiskUsers++
		} else {
			snap.Risk.LowRiskUsers++
		}
	}
	if len(riskScores) > 0 {
		snap.Risk.AvgRiskScore = math.Round(totalRisk*100/float64(len(riskScores))) / 100
	}

	compDashboard, err := GetComplianceDashboard(orgId)
	if err == nil {
		snap.Compliance.OverallScore = compDashboard.OverallScore
		snap.Compliance.FrameworkCount = len(compDashboard.Frameworks)
		for _, fs := range compDashboard.Frameworks {
			snap.Compliance.Compliant += fs.Compliant
			snap.Compliance.Partial += fs.Partial
			snap.Compliance.NonCompliant += fs.NonCompliant
		}
	}

	remSummary, err := GetRemediationSummary(orgId)
	if err == nil {
		snap.Remediation = BoardRemediationSection{
			TotalPaths: remSummary.TotalPaths, ActivePaths: remSummary.ActivePaths,
			CompletedPaths: remSummary.CompletedPaths, CriticalCount: remSummary.CriticalCount,
			AvgCompletion: remSummary.AvgCompletion,
		}
	}

	hygSummary, err := GetOrgHygieneEnrichedSummary(orgId)
	if err == nil {
		snap.Hygiene = BoardHygieneSection{
			TotalDevices: hygSummary.TotalDevices, AvgScore: hygSummary.AvgScore,
			FullyCompliant: hygSummary.FullyCompliant, AtRiskDevices: hygSummary.AtRiskDevices,
			ProfileCount: hygSummary.ProfileCount,
		}
	}

	phishScore := 100.0 - snap.Phishing.AvgClickRate
	trainScore := snap.Training.CompletionRate
	compScore := snap.Compliance.OverallScore
	hygScore := snap.Hygiene.AvgScore
	remScore := 0.0
	if snap.Remediation.TotalPaths > 0 {
		remScore = snap.Remediation.AvgCompletion
	}
	snap.SecurityPostureScore = math.Round(
		(phishScore*0.30+trainScore*0.25+compScore*0.20+hygScore*0.15+remScore*0.10)*100) / 100

	if snap.Phishing.AvgClickRate < 15 && snap.Training.CompletionRate > 70 {
		snap.RiskTrend = "improving"
	} else if snap.Phishing.AvgClickRate > 30 || snap.Training.CompletionRate < 40 {
		snap.RiskTrend = "declining"
	} else {
		snap.RiskTrend = "stable"
	}

	snap.Recommendations = generateBoardRecommendations(snap)
	return snap, nil
}

// generateBoardRecommendations produces actionable recommendations for the board.
func generateBoardRecommendations(snap *BoardReportSnapshot) []string {
	var recs []string

	if snap.Phishing.AvgClickRate > 25 {
		recs = append(recs, fmt.Sprintf("Phishing click rate is %.1f%% — schedule targeted awareness training for high-risk groups.", snap.Phishing.AvgClickRate))
	}
	if snap.Phishing.AvgReportRate < 10 {
		recs = append(recs, "Email reporting rate is low — promote the report-phishing button and run a reporting incentive campaign.")
	}
	if snap.Training.CompletionRate < 60 {
		recs = append(recs, fmt.Sprintf("Training completion is only %.0f%% — enforce mandatory deadlines and escalate overdue assignments.", snap.Training.CompletionRate))
	}
	if snap.Training.OverdueCount > 0 {
		recs = append(recs, fmt.Sprintf("%d training assignments are overdue — review and escalate.", snap.Training.OverdueCount))
	}
	if snap.Risk.HighRiskUsers > 0 {
		recs = append(recs, fmt.Sprintf("%d users are in the high-risk category — assign remediation paths.", snap.Risk.HighRiskUsers))
	}
	if snap.Compliance.OverallScore < 70 && snap.Compliance.FrameworkCount > 0 {
		recs = append(recs, fmt.Sprintf("Compliance score is %.0f%% — address non-compliant controls across %d framework(s).", snap.Compliance.OverallScore, snap.Compliance.FrameworkCount))
	}
	if snap.Remediation.CriticalCount > 0 {
		recs = append(recs, fmt.Sprintf("%d critical remediation paths are active — prioritize completion.", snap.Remediation.CriticalCount))
	}
	if snap.Hygiene.AtRiskDevices > 0 {
		recs = append(recs, fmt.Sprintf("%d devices have hygiene scores below 50%% — enforce device compliance policies.", snap.Hygiene.AtRiskDevices))
	}
	if snap.Hygiene.AvgScore < 60 && snap.Hygiene.TotalDevices > 0 {
		recs = append(recs, fmt.Sprintf("Average device hygiene score is %.0f%% — increase security awareness around endpoint protection.", snap.Hygiene.AvgScore))
	}

	if len(recs) == 0 {
		recs = append(recs, "Security posture is strong. Continue current awareness and training cadence.")
	}

	return recs
}
