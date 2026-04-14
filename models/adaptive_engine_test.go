package models

import (
	"testing"

	"github.com/gophish/gophish/config"
)

func setupAdaptiveEngineTest(t *testing.T) func() {
	t.Helper()
	conf := &config.Config{
		DBName:         "sqlite3",
		DBPath:         ":memory:",
		MigrationsPath: "../db/db_sqlite3/migrations/",
	}
	if err := Setup(conf); err != nil {
		t.Fatalf("Failed to set up database: %v", err)
	}
	return func() { /* in-memory DB cleaned up automatically */ }
}

func TestAdaptiveEngineConfigTableName(t *testing.T) {
	c := AdaptiveEngineConfig{}
	if c.TableName() != "adaptive_engine_configs" {
		t.Fatalf("expected 'adaptive_engine_configs', got %q", c.TableName())
	}
}

func TestGetAdaptiveEngineConfigDefaults(t *testing.T) {
	teardown := setupAdaptiveEngineTest(t)
	defer teardown()

	cfg := GetAdaptiveEngineConfig(9999)
	if !cfg.Enabled {
		t.Fatal("expected default to be enabled")
	}
	if cfg.EvalIntervalDays != DefaultEvalIntervalDays {
		t.Fatalf("expected %d eval interval days, got %d", DefaultEvalIntervalDays, cfg.EvalIntervalDays)
	}
	if cfg.BRSWeightPct != DefaultBRSWeight {
		t.Fatalf("expected BRS weight %.0f, got %.0f", DefaultBRSWeight, cfg.BRSWeightPct)
	}
	if cfg.PromoteThreshold != DefaultPromoteThreshold {
		t.Fatalf("expected promote threshold %.0f, got %.0f", DefaultPromoteThreshold, cfg.PromoteThreshold)
	}
	if cfg.DemoteThreshold != DefaultDemoteThreshold {
		t.Fatalf("expected demote threshold %.0f, got %.0f", DefaultDemoteThreshold, cfg.DemoteThreshold)
	}
}

func TestSaveAndGetAdaptiveEngineConfig(t *testing.T) {
	teardown := setupAdaptiveEngineTest(t)
	defer teardown()

	cfg := &AdaptiveEngineConfig{
		OrgId:                 1,
		Enabled:               false,
		EvalIntervalDays:      14,
		BRSWeightPct:          50,
		ClickRateWeightPct:    20,
		QuizScoreWeightPct:    20,
		TrendWeightPct:        10,
		PromoteThreshold:      80,
		DemoteThreshold:       25,
		MinSimulationsPromote: 5,
		CooldownDays:          21,
	}
	if err := SaveAdaptiveEngineConfig(cfg); err != nil {
		t.Fatalf("SaveAdaptiveEngineConfig: %v", err)
	}

	got := GetAdaptiveEngineConfig(1)
	if got.Enabled {
		t.Fatal("expected disabled")
	}
	if got.EvalIntervalDays != 14 {
		t.Fatalf("expected 14, got %d", got.EvalIntervalDays)
	}
	if got.PromoteThreshold != 80 {
		t.Fatalf("expected 80, got %.0f", got.PromoteThreshold)
	}
}

func TestAdaptiveEngineDefaultConstants(t *testing.T) {
	if DefaultEvalIntervalDays != 7 {
		t.Fatalf("expected 7, got %d", DefaultEvalIntervalDays)
	}
	if DefaultBRSWeight != 40.0 {
		t.Fatalf("expected 40, got %.0f", DefaultBRSWeight)
	}
	if DefaultClickRateWeight != 30.0 {
		t.Fatalf("expected 30, got %.0f", DefaultClickRateWeight)
	}
	if DefaultQuizScoreWeight != 20.0 {
		t.Fatalf("expected 20, got %.0f", DefaultQuizScoreWeight)
	}
	if DefaultTrendWeight != 10.0 {
		t.Fatalf("expected 10, got %.0f", DefaultTrendWeight)
	}
	if DefaultMinSimsPromote != 3 {
		t.Fatalf("expected 3, got %d", DefaultMinSimsPromote)
	}
	if DefaultCooldownDays != 14 {
		t.Fatalf("expected 14, got %d", DefaultCooldownDays)
	}
}

func TestAdaptiveEvaluationActions(t *testing.T) {
	eval := AdaptiveEvaluation{Action: "promote"}
	if eval.Action != "promote" {
		t.Fatal("expected promote action")
	}

	eval.Action = "demote"
	if eval.Action != "demote" {
		t.Fatal("expected demote action")
	}

	eval.Action = "maintain"
	if eval.Action != "maintain" {
		t.Fatal("expected maintain action")
	}
}

func TestUpdateAdaptiveEngineConfig(t *testing.T) {
	teardown := setupAdaptiveEngineTest(t)
	defer teardown()

	cfg := &AdaptiveEngineConfig{
		OrgId:   1,
		Enabled: true,
	}
	SaveAdaptiveEngineConfig(cfg)

	// Update it
	cfg.Enabled = false
	cfg.PromoteThreshold = 90
	if err := SaveAdaptiveEngineConfig(cfg); err != nil {
		t.Fatalf("SaveAdaptiveEngineConfig (update): %v", err)
	}

	got := GetAdaptiveEngineConfig(1)
	if got.Enabled {
		t.Fatal("expected disabled after update")
	}
	if got.PromoteThreshold != 90 {
		t.Fatalf("expected 90, got %.0f", got.PromoteThreshold)
	}
}
