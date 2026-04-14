package worker

import (
	"testing"
	"time"
)

func TestAutopilotCheckInterval(t *testing.T) {
	if AutopilotCheckInterval != 1*time.Hour {
		t.Fatalf("expected 1h, got %v", AutopilotCheckInterval)
	}
}

func TestAutopilotCheckIntervalPositive(t *testing.T) {
	if AutopilotCheckInterval <= 0 {
		t.Fatal("interval must be positive")
	}
}
