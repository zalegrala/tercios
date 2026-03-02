package config

import (
	"testing"
	"time"
)

func TestValidateAllowsZeroMaxRequests(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Requests.PerExporter = 0
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected valid config with max-requests=0, got %v", err)
	}
}

func TestValidateRejectsNegativeMaxRequests(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Requests.PerExporter = -1
	if err := cfg.Validate(); err == nil {
		t.Fatalf("expected error for negative max-requests")
	}
}

func TestValidateRejectsNegativeRampUp(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Requests.RampUp = Duration{Duration: -1 * time.Second}
	if err := cfg.Validate(); err == nil {
		t.Fatalf("expected error for negative ramp-up")
	}
}

func TestValidateRejectsNegativeExportTimeout(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Requests.ExportTimeout = Duration{Duration: -1 * time.Second}
	if err := cfg.Validate(); err == nil {
		t.Fatalf("expected error for negative export-timeout")
	}
}
