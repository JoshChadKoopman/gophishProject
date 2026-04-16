package models

import (
	"time"
)

// ── ROI Industry Benchmarks ─────────────────────────────────────
// Provides configurable industry benchmark data for ROI comparisons.
// Admins can override per-org, or use built-in defaults sourced from
// industry reports (Verizon DBIR, Ponemon, SANS, KnowBe4).

// ROIBenchmark represents a single industry benchmark metric row.
type ROIBenchmark struct {
	Id           int64     `json:"id" gorm:"primary_key"`
	OrgId        int64     `json:"org_id" gorm:"column:org_id"`
	MetricKey    string    `json:"metric_key" gorm:"column:metric_key;size:100"`
	MetricLabel  string    `json:"metric_label" gorm:"column:metric_label;size:200"`
	IndustryAvg  float64   `json:"industry_avg" gorm:"column:industry_avg"`
	IndustryP25  float64   `json:"industry_p25" gorm:"column:industry_p25"` // 25th percentile (top quartile)
	IndustryP75  float64   `json:"industry_p75" gorm:"column:industry_p75"` // 75th percentile (bottom quartile)
	Source       string    `json:"source" gorm:"column:source;size:200"`
	Year         int       `json:"year" gorm:"column:year"`
	Category     string    `json:"category" gorm:"column:category;size:100"`
	ModifiedDate time.Time `json:"modified_date" gorm:"column:modified_date"`
}

// TableName returns the GORM table name.
func (ROIBenchmark) TableName() string { return "roi_benchmarks" }

// BenchmarkComparison pairs an org's value with the industry average.
type BenchmarkComparison struct {
	MetricKey   string  `json:"metric_key"`
	MetricLabel string  `json:"metric_label"`
	OrgValue    float64 `json:"org_value"`
	IndustryAvg float64 `json:"industry_avg"`
	IndustryP25 float64 `json:"industry_p25"`
	IndustryP75 float64 `json:"industry_p75"`
	Delta       float64 `json:"delta"`      // org_value - industry_avg
	Percentile  string  `json:"percentile"` // "top_quartile", "average", "below_average"
	Favorable   bool    `json:"favorable"`  // true = org is performing better than avg
	Source      string  `json:"source"`
}

// ── Benchmark Metric Keys ──

const (
	BenchClickRate            = "click_rate"
	BenchReportRate           = "report_rate"
	BenchTrainingCompletion   = "training_completion"
	BenchAvgQuizScore         = "avg_quiz_score"
	BenchIncidentCost         = "incident_cost"
	BenchBreachCost           = "breach_cost"
	BenchTimeToDetect         = "time_to_detect_days"
	BenchPhishSusceptibility  = "phish_susceptibility"
	BenchSecuritySpendPerUser = "security_spend_per_user"
	BenchComplianceScore      = "compliance_score"
)

// ── Built-in Defaults ──

// DefaultBenchmarks returns industry-standard benchmarks (sourced from
// Verizon DBIR 2025, Ponemon 2024, KnowBe4 2025 reports).
func DefaultBenchmarks() []ROIBenchmark {
	return []ROIBenchmark{
		{MetricKey: BenchClickRate, MetricLabel: "Phishing Click Rate", IndustryAvg: 11.5, IndustryP25: 5.0, IndustryP75: 18.0, Source: "KnowBe4 Phishing Report 2025", Year: 2025, Category: "phishing"},
		{MetricKey: BenchReportRate, MetricLabel: "Phishing Report Rate", IndustryAvg: 17.0, IndustryP25: 25.0, IndustryP75: 8.0, Source: "KnowBe4 Phishing Report 2025", Year: 2025, Category: "phishing"},
		{MetricKey: BenchTrainingCompletion, MetricLabel: "Training Completion Rate", IndustryAvg: 72.0, IndustryP25: 90.0, IndustryP75: 55.0, Source: "SANS Security Awareness Report 2025", Year: 2025, Category: "training"},
		{MetricKey: BenchAvgQuizScore, MetricLabel: "Average Quiz Score", IndustryAvg: 68.0, IndustryP25: 82.0, IndustryP75: 55.0, Source: "SANS Security Awareness Report 2025", Year: 2025, Category: "training"},
		{MetricKey: BenchIncidentCost, MetricLabel: "Avg Cost Per Phishing Incident", IndustryAvg: 1500.0, IndustryP25: 800.0, IndustryP75: 3500.0, Source: "Ponemon Cost of Phishing 2024", Year: 2024, Category: "cost"},
		{MetricKey: BenchBreachCost, MetricLabel: "Avg Data Breach Cost", IndustryAvg: 4450000.0, IndustryP25: 2500000.0, IndustryP75: 6500000.0, Source: "IBM/Ponemon Cost of a Data Breach 2024", Year: 2024, Category: "cost"},
		{MetricKey: BenchTimeToDetect, MetricLabel: "Avg Time to Detect Breach (days)", IndustryAvg: 197.0, IndustryP25: 120.0, IndustryP75: 280.0, Source: "IBM/Ponemon Cost of a Data Breach 2024", Year: 2024, Category: "risk"},
		{MetricKey: BenchPhishSusceptibility, MetricLabel: "Phishing Susceptibility Rate", IndustryAvg: 34.3, IndustryP25: 18.0, IndustryP75: 45.0, Source: "KnowBe4 Phishing Report 2025", Year: 2025, Category: "phishing"},
		{MetricKey: BenchSecuritySpendPerUser, MetricLabel: "Security Awareness Spend/User", IndustryAvg: 40.0, IndustryP25: 60.0, IndustryP75: 20.0, Source: "Gartner Security Spending Survey 2025", Year: 2025, Category: "cost"},
		{MetricKey: BenchComplianceScore, MetricLabel: "Compliance Posture Score", IndustryAvg: 68.0, IndustryP25: 85.0, IndustryP75: 50.0, Source: "Nivoxis Compliance Benchmark 2025", Year: 2025, Category: "compliance"},
	}
}

// ── CRUD ──

// GetBenchmarks returns all benchmarks for an org; if none exist, returns defaults.
func GetBenchmarks(orgId int64) []ROIBenchmark {
	var benchmarks []ROIBenchmark
	db.Where(queryWhereOrgID, orgId).Order("category, metric_key").Find(&benchmarks)
	if len(benchmarks) == 0 {
		return DefaultBenchmarks()
	}
	return benchmarks
}

// GetBenchmarkByKey returns a specific benchmark for an org.
func GetBenchmarkByKey(orgId int64, key string) ROIBenchmark {
	var b ROIBenchmark
	err := db.Where("org_id = ? AND metric_key = ?", orgId, key).First(&b).Error
	if err != nil {
		// Fallback to defaults
		for _, d := range DefaultBenchmarks() {
			if d.MetricKey == key {
				return d
			}
		}
	}
	return b
}

// SaveBenchmark upserts a benchmark for an org.
func SaveBenchmark(b *ROIBenchmark) error {
	var existing ROIBenchmark
	err := db.Where("org_id = ? AND metric_key = ?", b.OrgId, b.MetricKey).First(&existing).Error
	if err == nil {
		b.Id = existing.Id
	}
	b.ModifiedDate = time.Now().UTC()
	return db.Save(b).Error
}

// SeedBenchmarks inserts default benchmarks for an org if none exist.
func SeedBenchmarks(orgId int64) error {
	var count int64
	db.Model(&ROIBenchmark{}).Where(queryWhereOrgID, orgId).Count(&count)
	if count > 0 {
		return nil // Already seeded
	}
	for _, b := range DefaultBenchmarks() {
		b.OrgId = orgId
		b.ModifiedDate = time.Now().UTC()
		if err := db.Create(&b).Error; err != nil {
			return err
		}
	}
	return nil
}

// DeleteBenchmark removes a benchmark by ID.
func DeleteBenchmark(id int64, orgId int64) error {
	return db.Where("id = ? AND org_id = ?", id, orgId).Delete(&ROIBenchmark{}).Error
}

// ── Comparison Engine ──

// lowerIsBetter defines which metrics are favorable when the org's value is below the industry average.
var lowerIsBetter = map[string]bool{
	BenchClickRate:           true,
	BenchIncidentCost:        true,
	BenchBreachCost:          true,
	BenchTimeToDetect:        true,
	BenchPhishSusceptibility: true,
}

// CompareOrgToBenchmarks builds comparison rows for the given org's ROI report.
func CompareOrgToBenchmarks(orgId int64, rpt *ROIReport) []BenchmarkComparison {
	benchmarks := GetBenchmarks(orgId)
	comparisons := []BenchmarkComparison{}

	orgValues := map[string]float64{
		BenchClickRate:           rpt.Phishing.CurrentClickRate,
		BenchReportRate:          rpt.Phishing.CurrentReportRate,
		BenchTrainingCompletion:  rpt.Training.CompletionRate,
		BenchAvgQuizScore:        rpt.Training.AvgQuizScore,
		BenchIncidentCost:        rpt.AvgIncidentCost,
		BenchBreachCost:          rpt.AvgBreachCost,
		BenchPhishSusceptibility: rpt.Phishing.CurrentClickRate, // proxy
		BenchComplianceScore:     rpt.Compliance.OverallScore,
	}

	cfg := GetROIConfig(orgId)
	empCount := cfg.EmployeeCount
	if empCount <= 0 {
		empCount = DefaultEmployeeCount
	}
	if cfg.ProgramCost > 0 {
		orgValues[BenchSecuritySpendPerUser] = cfg.ProgramCost / float64(empCount)
	}

	for _, b := range benchmarks {
		orgVal, ok := orgValues[b.MetricKey]
		if !ok {
			continue
		}
		comp := BenchmarkComparison{
			MetricKey:   b.MetricKey,
			MetricLabel: b.MetricLabel,
			OrgValue:    orgVal,
			IndustryAvg: b.IndustryAvg,
			IndustryP25: b.IndustryP25,
			IndustryP75: b.IndustryP75,
			Delta:       orgVal - b.IndustryAvg,
			Source:      b.Source,
		}

		// Determine if favorable
		if lowerIsBetter[b.MetricKey] {
			comp.Favorable = orgVal <= b.IndustryAvg
		} else {
			comp.Favorable = orgVal >= b.IndustryAvg
		}

		// Determine percentile bracket
		if lowerIsBetter[b.MetricKey] {
			if orgVal <= b.IndustryP25 {
				comp.Percentile = "top_quartile"
			} else if orgVal >= b.IndustryP75 {
				comp.Percentile = "below_average"
			} else {
				comp.Percentile = "average"
			}
		} else {
			if orgVal >= b.IndustryP25 {
				comp.Percentile = "top_quartile"
			} else if orgVal <= b.IndustryP75 {
				comp.Percentile = "below_average"
			} else {
				comp.Percentile = "average"
			}
		}

		comparisons = append(comparisons, comp)
	}

	return comparisons
}
