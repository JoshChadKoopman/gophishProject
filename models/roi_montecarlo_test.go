package models

import (
	"math"
	"testing"
)

func TestDefaultMonteCarloConfig(t *testing.T) {
	cfg := DefaultMonteCarloConfig()
	if cfg.Iterations != 10000 {
		t.Errorf("Expected 10000 iterations, got %d", cfg.Iterations)
	}
	if cfg.ConfidenceLevel != 0.90 {
		t.Errorf("Expected 0.90 confidence, got %f", cfg.ConfidenceLevel)
	}
}

func TestMonteCarloInputsFillDefaults(t *testing.T) {
	inp := &MonteCarloInputs{
		CurrentClickRate: 20.0,
		AvgIncidentCost:  50000,
	}
	inp.fillDefaults()

	if inp.ClickRateStdDev != 5.0 { // 25% of 20
		t.Errorf("Expected ClickRateStdDev=5.0, got %f", inp.ClickRateStdDev)
	}
	if inp.IncidentProbability != 0.10 {
		t.Errorf("Expected IncidentProbability=0.10, got %f", inp.IncidentProbability)
	}
	if inp.IncidentProbStdDev != 0.03 {
		t.Errorf("Expected IncidentProbStdDev=0.03, got %f", inp.IncidentProbStdDev)
	}
	if inp.IncidentCostStdDev != 20000 {
		t.Errorf("Expected IncidentCostStdDev=20000, got %f", inp.IncidentCostStdDev)
	}
	if inp.BreachProbBaseline != 0.25 {
		t.Errorf("Expected BreachProbBaseline=0.25, got %f", inp.BreachProbBaseline)
	}
}

func TestRunMonteCarloSimulation(t *testing.T) {
	inp := MonteCarloInputs{
		TotalRecipients:   1000,
		CurrentClickRate:  10.0,
		PreviousClickRate: 25.0,
		AvgIncidentCost:   50000,
		ProgramCost:       100000,
	}
	cfg := MonteCarloConfig{
		Iterations:      1000, // Fewer iterations for speed in test
		ConfidenceLevel: 0.90,
	}

	result := RunMonteCarloSimulation(inp, cfg)

	if result.SimulationRuns != 1000 {
		t.Errorf("Expected 1000 runs, got %d", result.SimulationRuns)
	}
	if result.ConfidenceLevel != 0.90 {
		t.Errorf("Expected 0.90 confidence, got %f", result.ConfidenceLevel)
	}
	// With a 15% click rate improvement on 1000 people, we expect positive incidents avoided
	if result.IncidentsAvoided.Mean <= 0 {
		t.Errorf("Expected positive mean incidents avoided, got %f", result.IncidentsAvoided.Mean)
	}
	// CI lower should be <= median <= upper
	if result.IncidentsAvoided.Lower > result.IncidentsAvoided.Median {
		t.Errorf("Lower bound (%f) > median (%f)", result.IncidentsAvoided.Lower, result.IncidentsAvoided.Median)
	}
	if result.IncidentsAvoided.Median > result.IncidentsAvoided.Upper {
		t.Errorf("Median (%f) > upper bound (%f)", result.IncidentsAvoided.Median, result.IncidentsAvoided.Upper)
	}
	// StdDev should be non-negative
	if result.CostAvoidance.StdDev < 0 {
		t.Errorf("StdDev should be non-negative, got %f", result.CostAvoidance.StdDev)
	}
}

func TestRunMonteCarloZeroImprovement(t *testing.T) {
	inp := MonteCarloInputs{
		TotalRecipients:   500,
		CurrentClickRate:  20.0,
		PreviousClickRate: 20.0, // No improvement
		AvgIncidentCost:   50000,
		ProgramCost:       100000,
	}
	cfg := MonteCarloConfig{
		Iterations:      500,
		ConfidenceLevel: 0.90,
	}

	result := RunMonteCarloSimulation(inp, cfg)

	// With no improvement, mean incidents avoided should be near zero
	if math.Abs(result.IncidentsAvoided.Mean) > 10 {
		t.Errorf("Expected near-zero incidents avoided with no improvement, got %f", result.IncidentsAvoided.Mean)
	}
}
