package worker

import (
	"testing"
	"time"
)

func TestBRSRecalcInterval(t *testing.T) {
	if BRSRecalcInterval != 6*time.Hour {
		t.Fatalf("expected 6h, got %v", BRSRecalcInterval)
	}
}
