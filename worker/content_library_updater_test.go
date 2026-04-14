package worker

import (
	"testing"
	"time"
)

func TestContentLibraryUpdateInterval(t *testing.T) {
	if ContentLibraryUpdateInterval != 24*time.Hour {
		t.Fatalf("expected 24h, got %v", ContentLibraryUpdateInterval)
	}
}

func TestContentLibraryUpdateIntervalPositive(t *testing.T) {
	if ContentLibraryUpdateInterval <= 0 {
		t.Fatal("interval must be positive")
	}
}
