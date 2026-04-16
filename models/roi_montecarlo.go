package models

import (
	"math"
	"math/rand"
	"sort"
)

// ── Monte Carlo Confidence Intervals for ROI Metrics ────────────
// Provides probabilistic ranges for incidents avoided and cost avoidance
// instead of single-point estimates, making ROI claims more defensible
// to CFOs and board members.

// MonteCarloConfig controls the simulation parameters.
type MonteCarloConfig struct {
	Iterations      int     `json:"iterations"`       // Number of simulation runs (default 10,000)
	ConfidenceLevel float64 `json:"confidence_level"` // e.g., 0.90 for 90% CI
}

// DefaultMonteCarloConfig returns production-grade defaults.
func DefaultMonteCarloConfig() MonteCarloConfig {
	return MonteCarloConfig{
		Iterations:      10000,
		ConfidenceLevel: 0.90,
	}
}

// ConfidenceInterval represents a lower/upper bound at a given confidence level.
type ConfidenceInterval struct {
	Lower      float64 `json:"lower"`
	Upper      float64 `json:"upper"`
	Median     float64 `json:"median"`
	Mean       float64 `json:"mean"`
	StdDev     float64 `json:"std_dev"`
	Confidence float64 `json:"confidence"` // e.g. 0.90
}

// MonteCarloResult contains simulation results for incidents avoided and cost avoidance.
type MonteCarloResult struct {
	IncidentsAvoided    ConfidenceInterval `json:"incidents_avoided"`
	CostAvoidance       ConfidenceInterval `json:"cost_avoidance"`
	ROIPercentage       ConfidenceInterval `json:"roi_percentage"`
	BreachProbReduction ConfidenceInterval `json:"breach_prob_reduction"`
	SimulationRuns      int                `json:"simulation_runs"`
	ConfidenceLevel     float64            `json:"confidence_level"`
}

// MonteCarloInputs are the empirical parameters fed into the simulation.
type MonteCarloInputs struct {
	TotalRecipients     int64   // Total people who received simulated phish
	CurrentClickRate    float64 // Current period click rate (0-100)
	PreviousClickRate   float64 // Prior period click rate (0-100)
	ClickRateStdDev     float64 // Estimated standard deviation of click rate
	IncidentProbability float64 // Probability a click leads to an actual incident (0-1)
	IncidentProbStdDev  float64 // Std dev of incident probability
	AvgIncidentCost     float64 // Average cost per incident
	IncidentCostStdDev  float64 // Std dev of incident cost
	ProgramCost         float64 // Total programme spend
	BreachProbBaseline  float64 // Baseline annual breach probability (0-1)
}

// DefaultInputStdDevs fills in reasonable standard deviations if not provided.
func (inp *MonteCarloInputs) fillDefaults() {
	if inp.ClickRateStdDev <= 0 {
		// Use 25% of the click rate as std dev (captures natural variation)
		inp.ClickRateStdDev = math.Max(inp.CurrentClickRate*0.25, 1.0)
	}
	if inp.IncidentProbability <= 0 {
		inp.IncidentProbability = 0.10 // 10% of clicks become real incidents
	}
	if inp.IncidentProbStdDev <= 0 {
		inp.IncidentProbStdDev = inp.IncidentProbability * 0.30
	}
	if inp.IncidentCostStdDev <= 0 {
		inp.IncidentCostStdDev = inp.AvgIncidentCost * 0.40 // High variance in cost
	}
	if inp.BreachProbBaseline <= 0 {
		inp.BreachProbBaseline = 0.25 // 25% annual breach probability baseline
	}
}

// RunMonteCarloSimulation performs a Monte Carlo simulation to generate
// confidence intervals for incidents avoided, cost avoidance, and ROI.
func RunMonteCarloSimulation(inputs MonteCarloInputs, cfg MonteCarloConfig) MonteCarloResult {
	if cfg.Iterations <= 0 {
		cfg = DefaultMonteCarloConfig()
	}
	inputs.fillDefaults()

	rng := rand.New(rand.NewSource(42)) // Deterministic for reproducibility

	incidentsResults := make([]float64, cfg.Iterations)
	costResults := make([]float64, cfg.Iterations)
	roiResults := make([]float64, cfg.Iterations)
	breachResults := make([]float64, cfg.Iterations)

	clickRateDelta := inputs.PreviousClickRate - inputs.CurrentClickRate
	if clickRateDelta < 0 {
		clickRateDelta = 0
	}

	for i := 0; i < cfg.Iterations; i++ {
		// Sample click rate reduction with noise
		sampledDelta := sampleNormalClamped(rng, clickRateDelta, inputs.ClickRateStdDev, 0, 100)

		// Sample incident probability
		sampledIncidentProb := sampleNormalClamped(rng, inputs.IncidentProbability, inputs.IncidentProbStdDev, 0.01, 0.5)

		// Sample cost per incident (log-normal-like: clamp at 50% of mean minimum)
		sampledCost := sampleNormalClamped(rng, inputs.AvgIncidentCost, inputs.IncidentCostStdDev, inputs.AvgIncidentCost*0.2, inputs.AvgIncidentCost*5.0)

		// Compute incidents avoided in this simulation
		reductionFraction := sampledDelta / 100.0
		incidentsAvoided := float64(inputs.TotalRecipients) * reductionFraction * sampledIncidentProb
		if incidentsAvoided < 0 {
			incidentsAvoided = 0
		}

		incidentsResults[i] = incidentsAvoided
		costResults[i] = incidentsAvoided * sampledCost

		// ROI percentage for this run
		if inputs.ProgramCost > 0 {
			roiResults[i] = (costResults[i] - inputs.ProgramCost) / inputs.ProgramCost * 100
		}

		// Breach probability reduction
		breachReduction := (sampledDelta / math.Max(inputs.PreviousClickRate, 1)) * inputs.BreachProbBaseline
		if breachReduction < 0 {
			breachReduction = 0
		}
		breachResults[i] = breachReduction * 100 // as percentage
	}

	return MonteCarloResult{
		IncidentsAvoided:    computeCI(incidentsResults, cfg.ConfidenceLevel),
		CostAvoidance:       computeCI(costResults, cfg.ConfidenceLevel),
		ROIPercentage:       computeCI(roiResults, cfg.ConfidenceLevel),
		BreachProbReduction: computeCI(breachResults, cfg.ConfidenceLevel),
		SimulationRuns:      cfg.Iterations,
		ConfidenceLevel:     cfg.ConfidenceLevel,
	}
}

// sampleNormalClamped draws from a normal distribution clamped to [lo, hi].
func sampleNormalClamped(rng *rand.Rand, mean, stddev, lo, hi float64) float64 {
	val := rng.NormFloat64()*stddev + mean
	if val < lo {
		return lo
	}
	if val > hi {
		return hi
	}
	return val
}

// computeCI computes a confidence interval from sorted samples.
func computeCI(samples []float64, confidence float64) ConfidenceInterval {
	n := len(samples)
	if n == 0 {
		return ConfidenceInterval{Confidence: confidence}
	}

	sort.Float64s(samples)

	alpha := (1 - confidence) / 2
	lowerIdx := int(math.Floor(alpha * float64(n)))
	upperIdx := int(math.Ceil((1-alpha)*float64(n))) - 1
	medianIdx := n / 2

	if lowerIdx < 0 {
		lowerIdx = 0
	}
	if upperIdx >= n {
		upperIdx = n - 1
	}

	// Compute mean and std dev
	sum := 0.0
	for _, v := range samples {
		sum += v
	}
	mean := sum / float64(n)

	sumSq := 0.0
	for _, v := range samples {
		d := v - mean
		sumSq += d * d
	}
	stdDev := math.Sqrt(sumSq / float64(n))

	return ConfidenceInterval{
		Lower:      math.Round(samples[lowerIdx]*100) / 100,
		Upper:      math.Round(samples[upperIdx]*100) / 100,
		Median:     math.Round(samples[medianIdx]*100) / 100,
		Mean:       math.Round(mean*100) / 100,
		StdDev:     math.Round(stdDev*100) / 100,
		Confidence: confidence,
	}
}

// RunROIMonteCarlo is a convenience wrapper that extracts simulation inputs
// from an existing ROI report and runs the Monte Carlo analysis.
func RunROIMonteCarlo(rpt *ROIReport) MonteCarloResult {
	var totalRecipients int64
	scope := OrgScope{OrgId: rpt.OrgId}
	overview, err := GetReportOverview(scope)
	if err == nil {
		totalRecipients = overview.TotalRecipients
	}

	inputs := MonteCarloInputs{
		TotalRecipients:   totalRecipients,
		CurrentClickRate:  rpt.Phishing.CurrentClickRate,
		PreviousClickRate: rpt.Phishing.PreviousClickRate,
		AvgIncidentCost:   rpt.AvgIncidentCost,
		ProgramCost:       rpt.ProgramCost,
	}

	return RunMonteCarloSimulation(inputs, DefaultMonteCarloConfig())
}
