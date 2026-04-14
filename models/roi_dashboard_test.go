package models

import (
	"testing"
	"time"
)

func TestROIDashboardStructDefaults(t *testing.T) {
	d := ROIDashboard{}
	if d.OrgId != 0 {
		t.Errorf("expected zero OrgId, got %d", d.OrgId)
	}
	if !d.GeneratedAt.IsZero() {
		t.Error("expected zero GeneratedAt")
	}
}

func TestROIPeriodSummaryFields(t *testing.T) {
	s := ROIPeriodSummary{
		ProgramCost:   50000,
		CostAvoidance: 120000,
		ROIPercentage: 140,
		ClickRate:     12.5,
		ReportRate:    35.0,
	}
	if s.ProgramCost != 50000 {
		t.Errorf("expected ProgramCost 50000, got %f", s.ProgramCost)
	}
	if s.ROIPercentage != 140 {
		t.Errorf("expected ROI 140, got %f", s.ROIPercentage)
	}
}

func TestComputeDeltas_Improving(t *testing.T) {
	current := &ROIPeriodSummary{
		CostAvoidance:      120000,
		ROIPercentage:      200,
		ClickRate:          10,
		ReportRate:         40,
		TrainingCompletion: 95,
		ComplianceScore:    90,
		RiskReduction:      60,
	}
	previous := &ROIPeriodSummary{
		CostAvoidance:      80000,
		ROIPercentage:      100,
		ClickRate:          25,
		ReportRate:         20,
		TrainingCompletion: 70,
		ComplianceScore:    60,
		RiskReduction:      30,
	}
	d := computeDeltas(current, previous)
	if d.OverallTrend != TrendImproving {
		t.Errorf("expected trend 'improving', got %q", d.OverallTrend)
	}
	if d.CostAvoidanceChange != 40000 {
		t.Errorf("expected cost avoidance change 40000, got %f", d.CostAvoidanceChange)
	}
	if d.ClickRateChange != -15 {
		t.Errorf("expected click rate change -15, got %f", d.ClickRateChange)
	}
}

func TestComputeDeltas_Declining(t *testing.T) {
	current := &ROIPeriodSummary{
		ClickRate:          30,
		ReportRate:         10,
		TrainingCompletion: 40,
		ComplianceScore:    30,
		RiskReduction:      10,
	}
	previous := &ROIPeriodSummary{
		ClickRate:          15,
		ReportRate:         30,
		TrainingCompletion: 80,
		ComplianceScore:    75,
		RiskReduction:      50,
	}
	d := computeDeltas(current, previous)
	if d.OverallTrend != TrendDeclining {
		t.Errorf("expected trend 'declining', got %q", d.OverallTrend)
	}
}

func TestComputeDeltas_Stable(t *testing.T) {
	current := &ROIPeriodSummary{
		ClickRate:          20,
		ReportRate:         25,
		TrainingCompletion: 80,
	}
	previous := &ROIPeriodSummary{
		ClickRate:          20,
		ReportRate:         25,
		TrainingCompletion: 80,
	}
	d := computeDeltas(current, previous)
	if d.OverallTrend != TrendStable {
		t.Errorf("expected trend 'stable', got %q", d.OverallTrend)
	}
}

func TestComputeInvestmentBreakdown(t *testing.T) {
	cfg := ROIInvestmentConfig{
		PhishingSimPct: 30,
		TrainingPct:    25,
		ToolingPct:     25,
		PersonnelPct:   20,
	}
	b := computeInvestmentBreakdown(100000, cfg, 5, 0)
	if b.TotalCost != 100000 {
		t.Errorf("expected total 100000, got %f", b.TotalCost)
	}
	if b.PhishingSimCost != 30000 {
		t.Errorf("expected phishing cost 30000, got %f", b.PhishingSimCost)
	}
	if b.TrainingCost != 25000 {
		t.Errorf("expected training cost 25000, got %f", b.TrainingCost)
	}
	if b.CostPerIncident != 20000 {
		t.Errorf("expected cost/incident 20000, got %f", b.CostPerIncident)
	}
}

func TestComputeInvestmentBreakdownZeroIncidents(t *testing.T) {
	cfg := DefaultInvestmentSplit
	b := computeInvestmentBreakdown(50000, cfg, 0, 0)
	if b.CostPerIncident != 0 {
		t.Errorf("expected zero cost/incident, got %f", b.CostPerIncident)
	}
}

func TestGenerateLeadershipBrief_PositiveROI(t *testing.T) {
	current := &ROIPeriodSummary{
		ROIPercentage:      250,
		CostAvoidance:      100000,
		ProgramCost:        40000,
		ClickRate:          12,
		TrainingCompletion: 92,
		ComplianceScore:    85,
		IncidentsAvoided:   5,
	}
	previous := &ROIPeriodSummary{
		ClickRate:          25,
		TrainingCompletion: 70,
	}
	delta := computeDeltas(current, previous)
	brief := generateLeadershipBrief(current, previous, &delta)

	if brief.Headline == "" {
		t.Error("expected non-empty headline")
	}
	if len(brief.KeyMetrics) == 0 {
		t.Error("expected at least one key metric")
	}
	if brief.ExecutiveSummary == "" {
		t.Error("expected non-empty executive summary")
	}
}

func TestGenerateLeadershipBrief_ZeroROI(t *testing.T) {
	current := &ROIPeriodSummary{ROIPercentage: 0}
	previous := &ROIPeriodSummary{}
	delta := computeDeltas(current, previous)
	brief := generateLeadershipBrief(current, previous, &delta)
	if brief.Headline != "Security awareness programme is building foundational resilience" {
		t.Errorf("unexpected headline for zero ROI: %s", brief.Headline)
	}
}

func TestGenerateLeadershipBrief_HighClickRate(t *testing.T) {
	current := &ROIPeriodSummary{ClickRate: 25, TrainingCompletion: 50, ComplianceScore: 60}
	previous := &ROIPeriodSummary{}
	delta := computeDeltas(current, previous)
	brief := generateLeadershipBrief(current, previous, &delta)
	found := false
	for _, r := range brief.Risks {
		if len(r) > 0 {
			found = true
		}
	}
	if !found {
		t.Error("expected at least one risk for high click rate")
	}
}

func TestROIInvestmentConfigTableName(t *testing.T) {
	cfg := ROIInvestmentConfig{}
	if cfg.TableName() != "roi_investment_configs" {
		t.Errorf("expected table name 'roi_investment_configs', got %s", cfg.TableName())
	}
}

func TestDefaultInvestmentSplit(t *testing.T) {
	total := DefaultInvestmentSplit.PhishingSimPct +
		DefaultInvestmentSplit.TrainingPct +
		DefaultInvestmentSplit.ToolingPct +
		DefaultInvestmentSplit.PersonnelPct
	if total != 100 {
		t.Errorf("default investment split should total 100, got %f", total)
	}
}

func TestTrendConstants(t *testing.T) {
	if TrendImproving != "improving" {
		t.Errorf("expected 'improving', got %q", TrendImproving)
	}
	if TrendStable != "stable" {
		t.Errorf("expected 'stable', got %q", TrendStable)
	}
	if TrendDeclining != "declining" {
		t.Errorf("expected 'declining', got %q", TrendDeclining)
	}
}

func TestDeltaThresholds(t *testing.T) {
	if DeltaImprovingThreshold != 2.0 {
		t.Errorf("expected improving threshold 2.0, got %f", DeltaImprovingThreshold)
	}
	if DeltaDecliningThreshold != -2.0 {
		t.Errorf("expected declining threshold -2.0, got %f", DeltaDecliningThreshold)
	}
}

func TestROIDashboardTimestamp(t *testing.T) {
	now := time.Now().UTC()
	d := ROIDashboard{GeneratedAt: now}
	if d.GeneratedAt.IsZero() {
		t.Error("expected non-zero GeneratedAt")
	}
	if d.GeneratedAt != now {
		t.Errorf("expected %v, got %v", now, d.GeneratedAt)
	}
}
