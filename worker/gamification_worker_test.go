package worker

import (
	"testing"
	"time"
)

func TestGamificationInterval(t *testing.T) {
	if GamificationInterval != 24*time.Hour {
		t.Fatalf("expected 24h, got %v", GamificationInterval)
	}
}

func TestGamificationIntervalPositive(t *testing.T) {
	if GamificationInterval <= 0 {
		t.Fatal("interval must be positive")
	}
}
