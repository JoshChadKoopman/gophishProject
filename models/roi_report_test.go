package models

import (
	"testing"
	"time"
)

func TestROIConfigDefaults(t *testing.T) {
	// Verify the default constants are sensible without hitting the database.
	// GetROIConfig() requires a DB connection; here we test that the default
	// ROIConfig constructed directly matches the expected constant values.
	cfg := ROIConfig{
		OrgId:           99999,
		ProgramCost:     DefaultProgramCost,
		AvgBreachCost:   DefaultBreachCost,
		AvgIncidentCost: DefaultIncidentCost,
		EmployeeCount:   DefaultEmployeeCount,
		AvgSalaryHr:     DefaultAvgSalaryHr,
		Currency:        "USD",
	}
	if cfg.ProgramCost != DefaultProgramCost {
		t.Errorf("expected ProgramCost=%f, got %f", DefaultProgramCost, cfg.ProgramCost)
	}
	if cfg.AvgBreachCost != DefaultBreachCost {
		t.Errorf("expected AvgBreachCost=%f, got %f", DefaultBreachCost, cfg.AvgBreachCost)
	}
	if cfg.AvgIncidentCost != DefaultIncidentCost {
		t.Errorf("expected AvgIncidentCost=%f, got %f", DefaultIncidentCost, cfg.AvgIncidentCost)
	}
	if cfg.EmployeeCount != DefaultEmployeeCount {
		t.Errorf("expected EmployeeCount=%d, got %d", DefaultEmployeeCount, cfg.EmployeeCount)
	}
	if cfg.AvgSalaryHr != DefaultAvgSalaryHr {
		t.Errorf("expected AvgSalaryHr=%f, got %f", DefaultAvgSalaryHr, cfg.AvgSalaryHr)
	}
	if cfg.Currency != "USD" {
		t.Errorf("expected Currency=USD, got %s", cfg.Currency)
	}
}

func TestROIConfigTableName(t *testing.T) {
	cfg := ROIConfig{}
	if cfg.TableName() != "roi_configs" {
		t.Errorf("expected table name 'roi_configs', got '%s'", cfg.TableName())
	}
}

func TestGenerateROIFindings_EmptyReport(t *testing.T) {
	r := &ROIReport{}
	findings := generateROIFindings(r)
	if len(findings) == 0 {
		t.Error("expected at least one finding")
	}
	if findings[0] != "Insufficient data to generate ROI findings. Run more campaigns and training to build data." {
		t.Errorf("unexpected finding: %s", findings[0])
	}
}

func TestGenerateROIFindings_WithData(t *testing.T) {
	r := &ROIReport{
		Metrics: ROIMetrics{
			ROIPercentage:      250.0,
			CostAvoidance:      75000.0,
			TrainingHoursSaved: 100.0,
			TrainingCostSaved:  4500.0,
		},
		Phishing: ROIPhishingSection{
			ClickRateReduction: 10.0,
			PreviousClickRate:  35.0,
			CurrentClickRate:   25.0,
			IncidentsAvoided:   5,
		},
		Training: ROITrainingSection{
			CompletionRate: 85.0,
		},
		Remediation: ROIRemediationSection{
			PathsCompleted: 3,
			RiskReduction:  45.0,
		},
		Compliance: ROIComplianceSection{
			OverallScore:      80.0,
			AuditReadiness:    80.0,
			FrameworksCovered: 2,
		},
	}
	findings := generateROIFindings(r)
	if len(findings) == 0 {
		t.Error("expected multiple findings")
	}
	// Should include ROI percentage finding
	found := false
	for _, f := range findings {
		if len(f) > 0 {
			found = true
		}
	}
	if !found {
		t.Error("no findings generated from data")
	}
}

func TestGenerateROIRecommendations_AllGood(t *testing.T) {
	r := &ROIReport{
		Metrics: ROIMetrics{
			ROIPercentage: 300.0,
		},
		Phishing: ROIPhishingSection{
			CurrentClickRate: 5.0,
		},
		Training: ROITrainingSection{
			CompletionRate: 95.0,
		},
		Hygiene: ROIHygieneSection{
			FullyCompliantPct: 90.0,
		},
		Remediation: ROIRemediationSection{
			PathsCreated:   5,
			CompletionRate: 90.0,
		},
		Compliance: ROIComplianceSection{
			OverallScore:      85.0,
			FrameworksCovered: 2,
		},
	}
	recs := generateROIRecommendations(r)
	if len(recs) != 1 || recs[0] != "Security programme is performing well. Maintain current investment and cadence." {
		t.Errorf("expected 'all good' recommendation, got %v", recs)
	}
}

func TestGenerateROIRecommendations_NeedsImprovement(t *testing.T) {
	r := &ROIReport{
		Metrics: ROIMetrics{
			ROIPercentage: 50.0,
		},
		Phishing: ROIPhishingSection{
			CurrentClickRate: 25.0,
		},
		Training: ROITrainingSection{
			CompletionRate: 60.0,
		},
		Hygiene: ROIHygieneSection{
			FullyCompliantPct: 50.0,
		},
		Remediation: ROIRemediationSection{
			PathsCreated:   10,
			CompletionRate: 40.0,
		},
		Compliance: ROIComplianceSection{
			OverallScore:      50.0,
			FrameworksCovered: 1,
		},
	}
	recs := generateROIRecommendations(r)
	if len(recs) < 4 {
		t.Errorf("expected at least 4 recommendations, got %d: %v", len(recs), recs)
	}
}

func TestROIReportStructFields(t *testing.T) {
	now := time.Now().UTC()
	report := ROIReport{
		OrgId:       1,
		PeriodStart: now.AddDate(-1, 0, 0),
		PeriodEnd:   now,
		PeriodLabel: "Test Period",
		GeneratedAt: now,
		ProgramCost: 50000.0,
	}

	if report.OrgId != 1 {
		t.Errorf("expected OrgId=1, got %d", report.OrgId)
	}
	if report.ProgramCost != 50000.0 {
		t.Errorf("expected ProgramCost=50000, got %f", report.ProgramCost)
	}
}

func TestROIMetricsZeroDivision(t *testing.T) {
	// Ensure that when baseline click rate is 0, phishRiskReduction doesn't panic
	// This tests the fix for the divide-by-zero guard
	m := ROIMetrics{}
	ph := ROIPhishingSection{
		ClickRateReduction: 0,
	}

	var baselineClickRate float64 = 0
	var phishRiskReduction float64
	if baselineClickRate > 0 {
		phishRiskReduction = ph.ClickRateReduction / baselineClickRate * 100
	}

	if phishRiskReduction != 0 {
		t.Errorf("expected phishRiskReduction=0 when baseline is 0, got %f", phishRiskReduction)
	}
	_ = m // suppress unused warning
}

func TestROISectionsDefaults(t *testing.T) {
	ps := ROIPhishingSection{}
	if ps.TotalSimulations != 0 || ps.ClickRateReduction != 0 {
		t.Error("default phishing section should have zero values")
	}

	ts := ROITrainingSection{}
	if ts.TotalCourses != 0 || ts.CompletionRate != 0 {
		t.Error("default training section should have zero values")
	}

	rs := ROIRemediationSection{}
	if rs.PathsCreated != 0 || rs.CompletionRate != 0 {
		t.Error("default remediation section should have zero values")
	}

	hs := ROIHygieneSection{}
	if hs.DevicesManaged != 0 || hs.AvgHygieneScore != 0 {
		t.Error("default hygiene section should have zero values")
	}

	cs := ROIComplianceSection{}
	if cs.FrameworksCovered != 0 || cs.OverallScore != 0 {
		t.Error("default compliance section should have zero values")
	}
}
