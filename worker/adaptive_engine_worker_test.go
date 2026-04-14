package worker

import (
	"testing"
	"time"
)

// ─── Unit tests for adaptive_engine_worker.go ───

func TestAdaptiveEngineCheckInterval(t *testing.T) {
	if AdaptiveEngineCheckInterval != 6*time.Hour {
		t.Fatalf("expected 6h, got %v", AdaptiveEngineCheckInterval)
	}
}

func TestAdaptiveEngineCheckIntervalPositive(t *testing.T) {
	if AdaptiveEngineCheckInterval <= 0 {
		t.Fatal("interval must be positive")
	}
}

func TestAdaptiveEngineCheckIntervalMultipleOfHour(t *testing.T) {
	if AdaptiveEngineCheckInterval%time.Hour != 0 {
		t.Fatal("interval should be a whole number of hours")
	}
}
