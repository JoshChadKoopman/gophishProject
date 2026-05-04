package logger

import (
	"testing"

	"github.com/sirupsen/logrus"
)

func TestJSONFormat(t *testing.T) {
	if err := Setup(&Config{Format: "json"}); err != nil {
		t.Fatal(err)
	}
	if _, ok := Logger.Formatter.(*logrus.JSONFormatter); !ok {
		t.Fatalf("expected JSONFormatter, got %T", Logger.Formatter)
	}
	// Default (empty) format should restore text formatter.
	if err := Setup(&Config{Format: ""}); err != nil {
		t.Fatal(err)
	}
	if _, ok := Logger.Formatter.(*logrus.TextFormatter); !ok {
		t.Fatalf("expected TextFormatter, got %T", Logger.Formatter)
	}
}

func TestLogLevel(t *testing.T) {
	tests := map[string]logrus.Level{
		"":      logrus.InfoLevel,
		"debug": logrus.DebugLevel,
		"info":  logrus.InfoLevel,
		"error": logrus.ErrorLevel,
		"fatal": logrus.FatalLevel,
	}
	config := &Config{}
	for level, expected := range tests {
		config.Level = level
		err := Setup(config)
		if err != nil {
			t.Fatalf("error setting logging level %v", err)
		}
		if Logger.Level != expected {
			t.Fatalf("invalid logging level. expected %v got %v", expected, Logger.Level)
		}
	}
}
