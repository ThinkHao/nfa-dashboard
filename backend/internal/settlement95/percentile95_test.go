package settlement95

import "testing"

func TestDescendingIndexUsesNfatoolRank(t *testing.T) {
	tests := []struct {
		name        string
		totalPoints int
		want        int
	}{
		{name: "full day 288 points", totalPoints: 288, want: 14},
		{name: "partial day 286 points", totalPoints: 286, want: 14},
		{name: "partial day 281 points", totalPoints: 281, want: 14},
		{name: "single point", totalPoints: 1, want: 0},
		{name: "empty", totalPoints: 0, want: -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DescendingIndex(tt.totalPoints); got != tt.want {
				t.Fatalf("DescendingIndex(%d)=%d, want %d", tt.totalPoints, got, tt.want)
			}
		})
	}
}
