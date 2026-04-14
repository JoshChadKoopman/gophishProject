package models

import (
	"fmt"
	"math"
	"time"

	log "github.com/gophish/gophish/logger"
)

// ── ROI Reporting Dashboard ─────────────────────────────────────
// Provides an executive-level dashboard with period-over-period comparison,
// trend data, and leadership-ready summaries that demonstrate the value
// of the security awareness programme.

// ROIDashboard is the top-level structure returned by the dashboard endpoint.
type ROIDashboard struct {
	OrgId            int64                `json:"org_id"`
	GeneratedAt      time.Time            `json:"generated_at"`
	CurrentPeriod    ROIPeriodSummary     `json:"current_period"`
	PreviousPeriod   ROIPeriodSummary     `json:"previous_period"`
	Deltas           ROIDelta             `json:"deltas"`
	Trend            []ROITrendPoint      `json:"trend"`
	InvestmentBreak  ROIInvestmentBreak   `json:"investment_breakdown"`
	LeadershipBrief  ROILeadershipBrief   `json:"leadership_brief"`
}

// ROIPeriodSummary captures headline numbers for one reporting period.
type ROIPeriodSummary struct {
	PeriodStart       time.Time `json:"period_start"`
	PeriodEnd         time.Time `json:"period_end"`
	ProgramCost       float64   `json:"program_cost"`
	CostAvoidance     float64   `json:"cost_avoidance"`
	ROIPercentage     float64   `json:"roi_percentage"`
	ClickRate         float64   `json:"click_rate"`
	ReportRate        float64   `json:"report_rate"`
	TrainingCompletion float64  `json:"training_completion"`
	ComplianceScore   float64   `json:"compliance_score"`
	RiskReduction     float64   `json:"risk_reduction"`
	IncidentsAvoided  int       `json:"incidents_avoided"`
}

// ROIDelta shows the change between the current and previous periods.
type ROIDelta struct {
	CostAvoidanceChange     float64 `json:"cost_avoidance_change"`
	ROIChange               float64 `json:"roi_change"`
	ClickRateChange         float64 `json:"click_rate_change"`
	ReportRateChange        float64 `json:"report_rate_change"`
	TrainingCompletionChange float64 `json:"training_completion_change"`
	ComplianceScoreChange   float64 `json:"compliance_score_change"`
	RiskReductionChange     float64 `json:"risk_reduction_change"`
	OverallTrend            string  `json:"overall_trend"` // "improving", "stable", "declining"
}

// ROITrendPoint is one data point in the ROI time-series.
type ROITrendPoint struct {
	Date           string  `json:"date"`
	CostAvoidance  float64 `json:"cost_avoidance"`
	ClickRate      float64 `json:"click_rate"`
	ReportRate     float64 `json:"report_rate"`
	TrainingPct    float64 `json:"training_completion"`
	RiskScore      float64 `json:"risk_score"`
}

// ROIInvestmentBreak shows how programme costs break down by category.
type ROIInvestmentBreak struct {
	TotalCost          float64 `json:"total_cost"`
	PhishingSimCost    float64 `json:"phishing_sim_cost"`
	TrainingCost       float64 `json:"training_cost"`
	ToolingCost        float64 `json:"tooling_cost"`
	PersonnelCost      float64 `json:"personnel_cost"`
	CostPerEmployee    float64 `json:"cost_per_employee"`
	CostPerIncident    float64 `json:"cost_per_incident_avoided"`
}

// ROILeadershipBrief is a plain-English summary generated for board / C-level.
type ROILeadershipBrief struct {
	Headline         string   `json:"headline"`
	KeyMetrics       []string `json:"key_metrics"`
	Risks            []string `json:"risks"`
	Recommendations  []string `json:"recommendations"`
	ExecutiveSummary string   `json:"executive_summary"`
}

// ROIInvestmentConfig stores the cost allocation for ROI investment breakdown.
type ROIInvestmentConfig struct {
	Id             int64   `json:"id" gorm:"primary_key"`
	OrgId          int64   `json:"org_id" gorm:"column:org_id;unique_index"`
	PhishingSimPct float64 `json:"phishing_sim_pct" gorm:"column:phishing_sim_pct;default:30"`
	TrainingPct    float64 `json:"training_pct" gorm:"column:training_pct;default:25"`
	ToolingPct     float64 `json:"tooling_pct" gorm:"column:tooling_pct;default:25"`
	PersonnelPct   float64 `json:"personnel_pct" gorm:"column:personnel_pct;default:20"`
}

// DefaultInvestmentSplit is the fallback cost allocation.
var DefaultInvestmentSplit = ROIInvestmentConfig{
	PhishingSimPct: 30,
	TrainingPct:    25,
	ToolingPct:     25,
	PersonnelPct:   20,
}

// TableName returns the table name for ROIInvestmentConfig.
func (ROIInvestmentConfig) TableName() string { return "roi_investment_configs" }

// OverallTrend constants.
const (
	TrendImproving = "improving"
	TrendStable    = "stable"
	TrendDeclining = "declining"
)

// DeltaImprovingThreshold is the minimum delta sum to consider "improving".
const DeltaImprovingThreshold = 2.0

// DeltaDecliningThreshold is the maximum delta sum to consider "declining".
const DeltaDecliningThreshold = -2.0

// GetROIInvestmentConfig returns the investment config for an org (or defaults).
func GetROIInvestmentConfig(orgId int64) ROIInvestmentConfig {
	cfg := ROIInvestmentConfig{}
	err := db.Where("org_id = ?", orgId).First(&cfg).Error
	if err != nil {
		cfg = DefaultInvestmentSplit
		cfg.OrgId = orgId
	}
	return cfg
}

// SaveROIInvestmentConfig upserts the investment breakdown config.
func SaveROIInvestmentConfig(cfg *ROIInvestmentConfig) error {
	existing := ROIInvestmentConfig{}
	err := db.Where("org_id = ?", cfg.OrgId).First(&existing).Error
	if err != nil {
		return db.Save(cfg).Error
	}
	cfg.Id = existing.Id
	return db.Save(cfg).Error
}

// GenerateROIDashboard creates the full executive ROI dashboard.
func GenerateROIDashboard(orgId int64, periodEnd time.Time, periodMonths int) (*ROIDashboard, error) {
	if periodMonths <= 0 {
		periodMonths = 12
	}
	if periodEnd.IsZero() {
		periodEnd = time.Now().UTC()
	}
	periodStart := periodEnd.AddDate(0, -periodMonths, 0)
	prevEnd := periodStart
	prevStart := prevEnd.AddDate(0, -periodMonths, 0)

	current, err := buildPeriodSummary(orgId, periodStart, periodEnd)
	if err != nil {
		return nil, fmt.Errorf("roi dashboard: current period: %w", err)
	}
	previous, err := buildPeriodSummary(orgId, prevStart, prevEnd)
	if err != nil {
		log.Warnf("roi dashboard: previous period unavailable: %v", err)
		previous = &ROIPeriodSummary{PeriodStart: prevStart, PeriodEnd: prevEnd}
	}

	deltas := computeDeltas(current, previous)
	trend := buildROITrend(orgId, periodStart, periodEnd)
	invCfg := GetROIInvestmentConfig(orgId)
	invest := computeInvestmentBreakdown(current.ProgramCost, invCfg, current.IncidentsAvoided, orgId)
	brief := generateLeadershipBrief(current, previous, &deltas)

	return &ROIDashboard{
		OrgId:           orgId,
		GeneratedAt:     time.Now().UTC(),
		CurrentPeriod:   *current,
		PreviousPeriod:  *previous,
		Deltas:          deltas,
		Trend:           trend,
		InvestmentBreak: invest,
		LeadershipBrief: brief,
	}, nil
}

// buildPeriodSummary builds an ROIPeriodSummary by aggregating data for the
// given time window. It reuses the existing GenerateROIReport function.
func buildPeriodSummary(orgId int64, start, end time.Time) (*ROIPeriodSummary, error) {
	rpt, err := GenerateROIReport(orgId, start, end)
	if err != nil {
		return nil, err
	}
	return &ROIPeriodSummary{
		PeriodStart:        start,
		PeriodEnd:          end,
		ProgramCost:        rpt.ProgramCost,
		CostAvoidance:      rpt.Metrics.CostAvoidance,
		ROIPercentage:      rpt.Metrics.ROIPercentage,
		ClickRate:          rpt.Phishing.CurrentClickRate,
		ReportRate:         rpt.Phishing.CurrentReportRate,
		TrainingCompletion: rpt.Training.CompletionRate,
		ComplianceScore:    rpt.Compliance.OverallScore,
		RiskReduction:      rpt.Metrics.OverallRiskReduction,
		IncidentsAvoided:   rpt.Metrics.EstIncidentsAvoided,
	}, nil
}

// computeDeltas calculates the difference between current and previous periods.
func computeDeltas(current, previous *ROIPeriodSummary) ROIDelta {
	d := ROIDelta{
		CostAvoidanceChange:      current.CostAvoidance - previous.CostAvoidance,
		ROIChange:                current.ROIPercentage - previous.ROIPercentage,
		ClickRateChange:          current.ClickRate - previous.ClickRate,
		ReportRateChange:         current.ReportRate - previous.ReportRate,
		TrainingCompletionChange: current.TrainingCompletion - previous.TrainingCompletion,
		ComplianceScoreChange:    current.ComplianceScore - previous.ComplianceScore,
		RiskReductionChange:      current.RiskReduction - previous.RiskReduction,
	}
	// Overall trend heuristic: sum of positive signals
	score := 0.0
	if d.ClickRateChange < 0 {
		score += 1 // lower click rate = good
	} else if d.ClickRateChange > 0 {
		score -= 1
	}
	if d.ReportRateChange > 0 {
		score += 1
	} else if d.ReportRateChange < 0 {
		score -= 1
	}
	if d.TrainingCompletionChange > 0 {
		score += 1
	}
	if d.ComplianceScoreChange > 0 {
		score += 1
	}
	if d.RiskReductionChange > 0 {
		score += 1
	}
	if score >= DeltaImprovingThreshold {
		d.OverallTrend = TrendImproving
	} else if score <= DeltaDecliningThreshold {
		d.OverallTrend = TrendDeclining
	} else {
		d.OverallTrend = TrendStable
	}
	return d
}

// buildROITrend generates monthly data points for the given period.
func buildROITrend(orgId int64, start, end time.Time) []ROITrendPoint {
	points := []ROITrendPoint{}
	cursor := time.Date(start.Year(), start.Month(), 1, 0, 0, 0, 0, time.UTC)
	for cursor.Before(end) {
		monthEnd := cursor.AddDate(0, 1, 0)
		if monthEnd.After(end) {
			monthEnd = end
		}
		rpt, err := GenerateROIReport(orgId, cursor, monthEnd)
		if err != nil {
			cursor = monthEnd
			continue
		}
		points = append(points, ROITrendPoint{
			Date:          cursor.Format("2006-01"),
			CostAvoidance: rpt.Metrics.CostAvoidance,
			ClickRate:     rpt.Phishing.CurrentClickRate,
			ReportRate:    rpt.Phishing.CurrentReportRate,
			TrainingPct:   rpt.Training.CompletionRate,
			RiskScore:     100 - rpt.Metrics.OverallRiskReduction,
		})
		cursor = monthEnd
	}
	return points
}

// computeInvestmentBreakdown allocates the total programme cost across categories.
func computeInvestmentBreakdown(totalCost float64, cfg ROIInvestmentConfig, incidents int, orgId int64) ROIInvestmentBreak {
	b := ROIInvestmentBreak{
		TotalCost:       totalCost,
		PhishingSimCost: totalCost * cfg.PhishingSimPct / 100,
		TrainingCost:    totalCost * cfg.TrainingPct / 100,
		ToolingCost:     totalCost * cfg.ToolingPct / 100,
		PersonnelCost:   totalCost * cfg.PersonnelPct / 100,
	}
	// Cost per employee
	org, err := GetOrganization(orgId)
	if err == nil && org.MaxUsers > 0 {
		b.CostPerEmployee = totalCost / float64(org.MaxUsers)
	}
	// Cost per incident avoided
	if incidents > 0 {
		b.CostPerIncident = totalCost / float64(incidents)
	}
	return b
}

// generateLeadershipBrief produces a human-readable leadership summary.
func generateLeadershipBrief(current, previous *ROIPeriodSummary, delta *ROIDelta) ROILeadershipBrief {
	brief := ROILeadershipBrief{}

	// Headline
	if current.ROIPercentage > 0 {
		brief.Headline = fmt.Sprintf("Security awareness programme delivered %.0f%% ROI with $%.0f in cost avoidance", current.ROIPercentage, current.CostAvoidance)
	} else {
		brief.Headline = "Security awareness programme is building foundational resilience"
	}

	// Key metrics
	brief.KeyMetrics = append(brief.KeyMetrics, fmt.Sprintf("Phishing click rate: %.1f%%", current.ClickRate))
	if delta.ClickRateChange < 0 {
		brief.KeyMetrics = append(brief.KeyMetrics, fmt.Sprintf("Click rate improved by %.1f percentage points vs prior period", math.Abs(delta.ClickRateChange)))
	}
	brief.KeyMetrics = append(brief.KeyMetrics, fmt.Sprintf("Training completion rate: %.1f%%", current.TrainingCompletion))
	brief.KeyMetrics = append(brief.KeyMetrics, fmt.Sprintf("Compliance score: %.1f%%", current.ComplianceScore))
	if current.IncidentsAvoided > 0 {
		brief.KeyMetrics = append(brief.KeyMetrics, fmt.Sprintf("Estimated %d security incidents avoided", current.IncidentsAvoided))
	}

	// Risks
	if current.ClickRate > 20 {
		brief.Risks = append(brief.Risks, fmt.Sprintf("Phishing click rate (%.1f%%) remains above the 20%% industry threshold", current.ClickRate))
	}
	if current.TrainingCompletion < 80 {
		brief.Risks = append(brief.Risks, fmt.Sprintf("Training completion (%.1f%%) is below the 80%% target", current.TrainingCompletion))
	}
	if current.ComplianceScore < 70 {
		brief.Risks = append(brief.Risks, fmt.Sprintf("Compliance score (%.1f%%) needs attention to meet audit requirements", current.ComplianceScore))
	}
	if len(brief.Risks) == 0 {
		brief.Risks = append(brief.Risks, "No significant risks identified in the current period")
	}

	// Recommendations
	if current.ClickRate > 15 {
		brief.Recommendations = append(brief.Recommendations, "Increase phishing simulation frequency for high-risk departments")
	}
	if current.TrainingCompletion < 90 {
		brief.Recommendations = append(brief.Recommendations, "Enable automated training reminders to improve completion rates")
	}
	if current.ComplianceScore < 80 {
		brief.Recommendations = append(brief.Recommendations, "Assign compliance-specific training modules for lagging frameworks")
	}
	if current.ROIPercentage > 100 {
		brief.Recommendations = append(brief.Recommendations, "Consider expanding the programme to cover additional risk vectors (vishing, smishing)")
	}
	if len(brief.Recommendations) == 0 {
		brief.Recommendations = append(brief.Recommendations, "Maintain current investment levels and continue monitoring trends")
	}

	// Executive summary
	brief.ExecutiveSummary = fmt.Sprintf(
		"Over the reporting period, the security awareness programme cost $%.0f and generated $%.0f in estimated cost avoidance, "+
			"resulting in a %.0f%% return on investment. The organisation's phishing click rate stands at %.1f%% "+
			"and training completion is at %.1f%%. The overall security posture trend is %s.",
		current.ProgramCost, current.CostAvoidance, current.ROIPercentage,
		current.ClickRate, current.TrainingCompletion, delta.OverallTrend,
	)

	return brief
}
