package controller

import (
	"testing"
	"time"
)

func TestParseTrafficTimeParam(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{name: "rfc3339", input: "2026-03-31T12:00:00Z", wantErr: false},
		{name: "datetime", input: "2026-03-31 20:00:00", wantErr: false},
		{name: "invalid", input: "2026/03/31", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseTrafficTimeParam(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseTrafficTimeParam(%q) error = %v, wantErr=%v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestValidateTrafficTimeRange(t *testing.T) {
	start := time.Date(2026, 3, 30, 10, 0, 0, 0, time.UTC)
	end := time.Date(2026, 3, 30, 11, 0, 0, 0, time.UTC)

	if err := validateTrafficTimeRange(start, end); err != nil {
		t.Fatalf("expected valid range, got error: %v", err)
	}

	if err := validateTrafficTimeRange(end, start); err == nil {
		t.Fatal("expected error when start >= end, got nil")
	}

	if err := validateTrafficTimeRange(start, start); err == nil {
		t.Fatal("expected error when start == end, got nil")
	}
}
