package models

import (
	"fmt"
	"math"
	"time"
)

// ROIReport represents a Return on Investment analysis for the security
// awareness program. It calculates cost avoidance, risk reduction, and
// programme effectiveness metrics that management can use to justify
// continued investment.
type ROIReport struct {
	// Identification
	OrgId       int64     `json:"org_id"`
	PeriodStart time.Time `json:"period_start"`
	PeriodEnd   time.Time `json:"period_end"`
	PeriodLabel string    `json:"period_label"`
	GeneratedAt time.Time `json:"generated_at"`

	// Investment inputs (configurable per org)
	ProgramCost         float64 `json:"program_cost"`           // Total programme spend for the period
	AvgBreachCost       float64 `json:"avg_breach_cost"`        // Industry average breach cost
	AvgIncidentCost     float64 `json:"avg_incident_cost"`      // Average cost per phishing incident
	EstEmployeeSalaryHr float64 `json:"est_employee_salary_hr"` // Average hourly salary for time-cost calculations

	// Computed metrics
	Metrics ROIMetrics `json:"metrics"`

	// Section-level breakdowns
	Phishing    ROIPhishingSection    `json:"phishing"`
	Training    ROITrainingSection    `json:"training"`
	Remediation ROIRemediationSection `json:"remediation"`
	Hygiene     ROIHygieneSection     `json:"hygiene"`
	Compliance  ROIComplianceSection  `json:"compliance"`

	// Industry benchmark comparisons
	Benchmarks []BenchmarkComparison `json:"benchmarks,omitempty"`

	// Monte Carlo confidence intervals
	MonteCarlo *MonteCarloResult `json:"monte_carlo,omitempty"`

	// Executive summary
	KeyFindings     []string `json:"key_findings"`
	Recommendations []string `json:"recommendations"`
}

// ROIMetrics contains the high-level ROI calculations.
type ROIMetrics struct {
	// Cost avoidance
	EstIncidentsAvoided int     `json:"est_incidents_avoided"`     // Estimated phishing incidents avoided
	CostAvoidance       float64 `json:"cost_avoidance"`            // Monetary value of avoided incidents
	BreachRiskReduction float64 `json:"breach_risk_reduction_pct"` // Percentage reduction in breach probability

	// Programme efficiency
	CostPerEmployee     float64 `json:"cost_per_employee"`
	ROIPercentage       float64 `json:"roi_percentage"` // (CostAvoidance - ProgramCost) / ProgramCost * 100
	PaybackPeriodMonths float64 `json:"payback_period_months"`

	// Risk reduction
	ClickRateReduction    float64 `json:"click_rate_reduction_pct"`
	ReportRateImprovement float64 `json:"report_rate_improvement_pct"`
	OverallRiskReduction  float64 `json:"overall_risk_reduction_pct"`

	// Training efficiency
	TrainingHoursSaved    float64 `json:"training_hours_saved"` // Via adaptive difficulty
	TrainingCostSaved     float64 `json:"training_cost_saved"`
	CompletionImprovement float64 `json:"completion_improvement_pct"`
}

// ROIPhishingSection shows phishing-specific ROI metrics.
type ROIPhishingSection struct {
	TotalSimulations   int64   `json:"total_simulations"`
	CurrentClickRate   float64 `json:"current_click_rate"`
	PreviousClickRate  float64 `json:"previous_click_rate"`
	ClickRateReduction float64 `json:"click_rate_reduction"`
	CurrentReportRate  float64 `json:"current_report_rate"`
	PreviousReportRate float64 `json:"previous_report_rate"`
	ReportRateIncrease float64 `json:"report_rate_increase"`
	IncidentsAvoided   int     `json:"incidents_avoided"`
	CostAvoided        float64 `json:"cost_avoided"`
}

// ROITrainingSection shows training-specific ROI metrics.
type ROITrainingSection struct {
	TotalCourses       int64   `json:"total_courses"`
	CompletionRate     float64 `json:"completion_rate"`
	AvgQuizScore       float64 `json:"avg_quiz_score"`
	CertificatesIssued int64   `json:"certificates_issued"`
	OverdueReductions  int64   `json:"overdue_reductions"`
	AvgTimeToComplete  float64 `json:"avg_time_to_complete_hrs"`
	ProductivitySaved  float64 `json:"productivity_saved_hrs"`
}

// ROIRemediationSection shows remediation-specific ROI metrics.
type ROIRemediationSection struct {
	PathsCreated           int     `json:"paths_created"`
	PathsCompleted         int     `json:"paths_completed"`
	CompletionRate         float64 `json:"completion_rate"`
	RepeatOffendersReduced int     `json:"repeat_offenders_reduced"`
	CriticalResolved       int     `json:"critical_resolved"`
	RiskReduction          float64 `json:"risk_reduction_pct"`
}

// ROIHygieneSection shows cyber hygiene ROI metrics.
type ROIHygieneSection struct {
	DevicesManaged         int     `json:"devices_managed"`
	AvgHygieneScore        float64 `json:"avg_hygiene_score"`
	FullyCompliantPct      float64 `json:"fully_compliant_pct"`
	VulnerabilityReduction float64 `json:"vulnerability_reduction_pct"`
}

// ROIComplianceSection shows compliance-related ROI metrics.
type ROIComplianceSection struct {
	FrameworksCovered  int     `json:"frameworks_covered"`
	OverallScore       float64 `json:"overall_score"`
	PenaltyRiskAvoided float64 `json:"penalty_risk_avoided"`
	AuditReadiness     float64 `json:"audit_readiness_pct"`
}

// ROIConfig holds the configurable cost assumptions for ROI calculations.
// These can be stored per-org or use industry defaults.
type ROIConfig struct {
	Id              int64     `json:"id" gorm:"column:id; primary_key:yes"`
	OrgId           int64     `json:"org_id" gorm:"column:org_id"`
	ProgramCost     float64   `json:"program_cost" gorm:"column:program_cost"`
	AvgBreachCost   float64   `json:"avg_breach_cost" gorm:"column:avg_breach_cost"`
	AvgIncidentCost float64   `json:"avg_incident_cost" gorm:"column:avg_incident_cost"`
	EmployeeCount   int       `json:"employee_count" gorm:"column:employee_count"`
	AvgSalaryHr     float64   `json:"avg_salary_hr" gorm:"column:avg_salary_hr"`
	Currency        string    `json:"currency" gorm:"column:currency"`
	ModifiedDate    time.Time `json:"modified_date" gorm:"column:modified_date"`
}

func (ROIConfig) TableName() string { return "roi_configs" }

// Default industry cost assumptions (IBM / Ponemon 2024 benchmarks)
const (
	DefaultBreachCost    = 4450000.0 // $4.45M average data breach cost
	DefaultIncidentCost  = 1500.0    // $1,500 per phishing incident
	DefaultAvgSalaryHr   = 45.0      // $45/hr average salary
	DefaultProgramCost   = 50000.0   // $50K annual programme cost
	DefaultEmployeeCount = 200
)

// GetROIConfig retrieves the ROI configuration for an org, or returns defaults.
func GetROIConfig(orgId int64) ROIConfig {
	cfg := ROIConfig{}
	err := db.Where("org_id = ?", orgId).First(&cfg).Error
	if err != nil {
		// Return sensible defaults
		return ROIConfig{
			OrgId:           orgId,
			ProgramCost:     DefaultProgramCost,
			AvgBreachCost:   DefaultBreachCost,
			AvgIncidentCost: DefaultIncidentCost,
			EmployeeCount:   DefaultEmployeeCount,
			AvgSalaryHr:     DefaultAvgSalaryHr,
			Currency:        "USD",
		}
	}
	return cfg
}

// SaveROIConfig creates or updates the ROI configuration for an org.
func SaveROIConfig(cfg *ROIConfig) error {
	existing := ROIConfig{}
	err := db.Where("org_id = ?", cfg.OrgId).First(&existing).Error
	if err == nil {
		cfg.Id = existing.Id
	}
	cfg.ModifiedDate = time.Now().UTC()
	return db.Save(cfg).Error
}

// GenerateROIReport computes the full ROI analysis for the given period.
func GenerateROIReport(orgId int64, periodStart, periodEnd time.Time) (*ROIReport, error) {
	cfg := GetROIConfig(orgId)
	scope := OrgScope{OrgId: orgId}

	report := &ROIReport{
		OrgId:               orgId,
		PeriodStart:         periodStart,
		PeriodEnd:           periodEnd,
		PeriodLabel:         fmt.Sprintf("%s — %s", periodStart.Format("Jan 2, 2006"), periodEnd.Format("Jan 2, 2006")),
		GeneratedAt:         time.Now().UTC(),
		ProgramCost:         cfg.ProgramCost,
		AvgBreachCost:       cfg.AvgBreachCost,
		AvgIncidentCost:     cfg.AvgIncidentCost,
		EstEmployeeSalaryHr: cfg.AvgSalaryHr,
	}

	// ─── Phishing ROI ───
	overview, _ := GetReportOverview(scope)
	report.Phishing = ROIPhishingSection{
		TotalSimulations:  overview.TotalCampaigns,
		CurrentClickRate:  overview.AvgClickRate,
		CurrentReportRate: overview.AvgReportRate,
	}

	// Calculate prior-period click/report rates for comparison.
	// If we have enough history, use actual data; otherwise fall back to industry baselines.
	periodDuration := periodEnd.Sub(periodStart)
	priorStart := periodStart.Add(-periodDuration)
	priorEnd := periodStart
	priorClickRate, priorReportRate := getPriorPeriodRates(scope, priorStart, priorEnd)

	baselineClickRate := priorClickRate // actual prior-period or industry baseline

	report.Phishing.PreviousClickRate = baselineClickRate
	report.Phishing.ClickRateReduction = baselineClickRate - overview.AvgClickRate
	if report.Phishing.ClickRateReduction < 0 {
		report.Phishing.ClickRateReduction = 0
	}

	report.Phishing.PreviousReportRate = priorReportRate
	report.Phishing.ReportRateIncrease = overview.AvgReportRate - priorReportRate
	if report.Phishing.ReportRateIncrease < 0 {
		report.Phishing.ReportRateIncrease = 0
	}

	// Estimated incidents avoided = reduction in click rate × total recipients × incident probability
	if overview.TotalRecipients > 0 {
		reductionFraction := report.Phishing.ClickRateReduction / 100.0
		report.Phishing.IncidentsAvoided = int(math.Round(float64(overview.TotalRecipients) * reductionFraction * 0.1))
		report.Phishing.CostAvoided = float64(report.Phishing.IncidentsAvoided) * cfg.AvgIncidentCost
	}

	// ─── Training ROI ───
	trainingSummary, _ := GetTrainingSummaryReport(scope)
	report.Training = ROITrainingSection{
		TotalCourses:       trainingSummary.TotalCourses,
		CompletionRate:     trainingSummary.CompletionRate,
		AvgQuizScore:       trainingSummary.AvgQuizScore,
		CertificatesIssued: trainingSummary.CertificatesIssued,
		OverdueReductions:  0,
	}

	// Estimate productivity saved through efficient adaptive training
	// Assume avg 2 hrs per course, adaptive saves ~20% of time
	avgCourseHours := 2.0
	adaptiveSavingPct := 0.20
	report.Training.ProductivitySaved = float64(trainingSummary.CompletedCount) * avgCourseHours * adaptiveSavingPct
	report.Training.AvgTimeToComplete = avgCourseHours * (1 - adaptiveSavingPct)

	// ─── Remediation ROI ───
	remSummary, _ := GetRemediationSummary(orgId)
	report.Remediation = ROIRemediationSection{
		PathsCreated:     remSummary.TotalPaths,
		PathsCompleted:   remSummary.CompletedPaths,
		CriticalResolved: remSummary.CompletedPaths, // Simplified
	}
	if remSummary.TotalPaths > 0 {
		report.Remediation.CompletionRate = float64(remSummary.CompletedPaths) / float64(remSummary.TotalPaths) * 100
		report.Remediation.RiskReduction = report.Remediation.CompletionRate * 0.6 // 60% weight to remediation
	}

	// ─── Cyber Hygiene ROI ───
	hygSummary, _ := GetOrgHygieneEnrichedSummary(orgId)
	report.Hygiene = ROIHygieneSection{
		DevicesManaged:  hygSummary.TotalDevices,
		AvgHygieneScore: hygSummary.AvgScore,
	}
	if hygSummary.TotalDevices > 0 {
		report.Hygiene.FullyCompliantPct = float64(hygSummary.FullyCompliant) / float64(hygSummary.TotalDevices) * 100
		report.Hygiene.VulnerabilityReduction = hygSummary.AvgScore * 0.7 // 70% of hygiene score maps to vulnerability reduction
	}

	// ─── Compliance ROI ───
	compDashboard, err := GetComplianceDashboard(orgId)
	if err == nil {
		report.Compliance = ROIComplianceSection{
			FrameworksCovered: len(compDashboard.Frameworks),
			OverallScore:      compDashboard.OverallScore,
			AuditReadiness:    compDashboard.OverallScore,
		}
		// Penalty risk avoided: proportion of compliance score × estimated penalty cost
		// Using 5% of breach cost as estimated penalty exposure
		report.Compliance.PenaltyRiskAvoided = (compDashboard.OverallScore / 100.0) * (cfg.AvgBreachCost * 0.05)
	}

	// ─── Aggregate ROI Metrics ───
	totalCostAvoided := report.Phishing.CostAvoided +
		(report.Training.ProductivitySaved * cfg.AvgSalaryHr) +
		report.Compliance.PenaltyRiskAvoided

	report.Metrics = ROIMetrics{
		EstIncidentsAvoided:   report.Phishing.IncidentsAvoided,
		CostAvoidance:         math.Round(totalCostAvoided*100) / 100,
		ClickRateReduction:    report.Phishing.ClickRateReduction,
		ReportRateImprovement: report.Phishing.ReportRateIncrease,
		TrainingHoursSaved:    report.Training.ProductivitySaved,
		TrainingCostSaved:     math.Round(report.Training.ProductivitySaved*cfg.AvgSalaryHr*100) / 100,
		CompletionImprovement: report.Training.CompletionRate,
	}

	// Overall risk reduction (weighted composite)
	var phishRiskReduction float64
	if baselineClickRate > 0 {
		phishRiskReduction = report.Phishing.ClickRateReduction / baselineClickRate * 100
	}
	report.Metrics.OverallRiskReduction = math.Round(
		(phishRiskReduction*0.40+report.Remediation.RiskReduction*0.25+
			report.Hygiene.VulnerabilityReduction*0.20+report.Compliance.AuditReadiness*0.15)*100) / 100

	// Breach risk reduction: click rate reduction proportion
	report.Metrics.BreachRiskReduction = math.Round(phishRiskReduction*100) / 100

	// Cost per employee
	employeeCount := cfg.EmployeeCount
	if employeeCount <= 0 {
		employeeCount = DefaultEmployeeCount
	}
	report.Metrics.CostPerEmployee = math.Round(cfg.ProgramCost/float64(employeeCount)*100) / 100

	// ROI percentage
	if cfg.ProgramCost > 0 {
		report.Metrics.ROIPercentage = math.Round((totalCostAvoided-cfg.ProgramCost)/cfg.ProgramCost*100*100) / 100
	}

	// Payback period (months)
	if totalCostAvoided > 0 {
		monthlyAvoidance := totalCostAvoided / 12.0
		if monthlyAvoidance > 0 {
			report.Metrics.PaybackPeriodMonths = math.Round(cfg.ProgramCost/monthlyAvoidance*10) / 10
		}
	}

	// ─── Key Findings ───
	report.KeyFindings = generateROIFindings(report)
	report.Recommendations = generateROIRecommendations(report)

	// ─── Industry Benchmark Comparisons ───
	report.Benchmarks = CompareOrgToBenchmarks(orgId, report)

	// ─── Monte Carlo Confidence Intervals ───
	mc := RunROIMonteCarlo(report)
	report.MonteCarlo = &mc

	return report, nil
}

func generateROIFindings(r *ROIReport) []string {
	var findings []string

	if r.Metrics.ROIPercentage > 0 {
		findings = append(findings, fmt.Sprintf("The security awareness programme generated a %.0f%% return on investment.", r.Metrics.ROIPercentage))
	}
	if r.Metrics.CostAvoidance > 0 {
		findings = append(findings, fmt.Sprintf("Estimated cost avoidance of $%.0f through reduced phishing incidents and improved compliance.", r.Metrics.CostAvoidance))
	}
	if r.Phishing.ClickRateReduction > 0 {
		findings = append(findings, fmt.Sprintf("Phishing click rate reduced by %.1f percentage points (from %.1f%% baseline to %.1f%%).",
			r.Phishing.ClickRateReduction, r.Phishing.PreviousClickRate, r.Phishing.CurrentClickRate))
	}
	if r.Training.CompletionRate > 70 {
		findings = append(findings, fmt.Sprintf("Training completion rate of %.0f%% exceeds the 70%% industry benchmark.", r.Training.CompletionRate))
	}
	if r.Phishing.IncidentsAvoided > 0 {
		findings = append(findings, fmt.Sprintf("An estimated %d phishing incidents were avoided during this period.", r.Phishing.IncidentsAvoided))
	}
	if r.Metrics.TrainingHoursSaved > 0 {
		findings = append(findings, fmt.Sprintf("Adaptive training saved approximately %.0f employee-hours ($%.0f in productivity).",
			r.Metrics.TrainingHoursSaved, r.Metrics.TrainingCostSaved))
	}
	if r.Remediation.PathsCompleted > 0 {
		findings = append(findings, fmt.Sprintf("%d remediation paths completed, reducing repeat-offender risk by %.0f%%.",
			r.Remediation.PathsCompleted, r.Remediation.RiskReduction))
	}
	if r.Compliance.OverallScore > 0 {
		findings = append(findings, fmt.Sprintf("Compliance audit readiness at %.0f%% across %d framework(s).",
			r.Compliance.AuditReadiness, r.Compliance.FrameworksCovered))
	}

	if len(findings) == 0 {
		findings = append(findings, "Insufficient data to generate ROI findings. Run more campaigns and training to build data.")
	}

	return findings
}

func generateROIRecommendations(r *ROIReport) []string {
	var recs []string

	if r.Metrics.ROIPercentage < 100 {
		recs = append(recs, "Increase phishing simulation frequency to drive click rate down further and improve cost avoidance.")
	}
	if r.Training.CompletionRate < 80 {
		recs = append(recs, fmt.Sprintf("Training completion is at %.0f%% — enforce mandatory deadlines to reach the 80%% target.", r.Training.CompletionRate))
	}
	if r.Phishing.CurrentClickRate > 15 {
		recs = append(recs, fmt.Sprintf("Click rate of %.1f%% remains above the 15%% target — schedule targeted awareness campaigns.", r.Phishing.CurrentClickRate))
	}
	if r.Hygiene.FullyCompliantPct < 80 {
		recs = append(recs, "Less than 80% of devices are fully compliant — enforce endpoint security policies.")
	}
	if r.Remediation.PathsCreated > 0 && r.Remediation.CompletionRate < 70 {
		recs = append(recs, fmt.Sprintf("Remediation completion at %.0f%% — prioritise critical paths and escalate overdue items.", r.Remediation.CompletionRate))
	}
	if r.Compliance.OverallScore < 70 && r.Compliance.FrameworksCovered > 0 {
		recs = append(recs, "Compliance score is below 70% — address non-compliant controls before the next audit cycle.")
	}

	if len(recs) == 0 {
		recs = append(recs, "Security programme is performing well. Maintain current investment and cadence.")
	}

	return recs
}

// getPriorPeriodRates queries campaign results from a prior period to calculate
// actual click and report rates. Falls back to industry baselines (35% click, 5% report)
// if no prior data exists.
func getPriorPeriodRates(scope OrgScope, priorStart, priorEnd time.Time) (clickRate float64, reportRate float64) {
	// Industry baseline fallback values
	const defaultClickRate = 35.0
	const defaultReportRate = 5.0

	// Find campaigns in the prior period
	var campaignIds []int64
	q := db.Table("campaigns").
		Where("launch_date >= ? AND launch_date < ?", priorStart, priorEnd)
	if !scope.IsSuperAdmin {
		q = q.Where(queryWhereOrgID, scope.OrgId)
	}
	q.Pluck("id", &campaignIds)

	if len(campaignIds) == 0 {
		return defaultClickRate, defaultReportRate
	}

	// Aggregate results across prior-period campaigns
	var total, clicked, reported int64
	db.Table("results").
		Where("campaign_id IN (?)", campaignIds).
		Count(&total)
	if total == 0 {
		return defaultClickRate, defaultReportRate
	}

	db.Table("results").
		Where("campaign_id IN (?) AND status IN (?)", campaignIds, []string{EventClicked, EventDataSubmit}).
		Count(&clicked)
	db.Table("results").
		Where("campaign_id IN (?) AND reported = ?", campaignIds, true).
		Count(&reported)

	clickRate = math.Round(float64(clicked)*10000/float64(total)) / 100
	reportRate = math.Round(float64(reported)*10000/float64(total)) / 100

	// If prior period data is very sparse, blend with baseline
	if total < 10 {
		clickRate = (clickRate + defaultClickRate) / 2
		reportRate = (reportRate + defaultReportRate) / 2
	}

	return clickRate, reportRate
}
