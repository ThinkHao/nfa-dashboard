package repository

import (
	"strings"
	"testing"
)

func TestDailyTrafficVolumeGrouping(t *testing.T) {
	tests := []struct {
		name          string
		cp            string
		wantSelect    string
		wantGroupByCP bool
	}{
		{name: "all CPs", cp: "", wantSelect: "'' AS cp", wantGroupByCP: false},
		{name: "single CP", cp: "bilibili", wantSelect: "cp", wantGroupByCP: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cpSelect, groupBy := dailyTrafficVolumeGrouping(tt.cp)
			if cpSelect != tt.wantSelect {
				t.Fatalf("cpSelect = %q, want %q", cpSelect, tt.wantSelect)
			}
			if got := strings.HasSuffix(groupBy, ", cp"); got != tt.wantGroupByCP {
				t.Fatalf("groupBy = %q, group by CP = %v, want %v", groupBy, got, tt.wantGroupByCP)
			}
		})
	}
}
