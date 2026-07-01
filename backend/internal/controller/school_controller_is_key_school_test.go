package controller

import "testing"

func TestNormalizeIsKeySchoolFilter(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		want   string
		wantOk bool
	}{
		{name: "empty", input: "", want: "", wantOk: false},
		{name: "one", input: "1", want: "1", wantOk: true},
		{name: "zero", input: "0", want: "0", wantOk: true},
		{name: "true", input: "true", want: "1", wantOk: true},
		{name: "false", input: "false", want: "0", wantOk: true},
		{name: "spaces", input: " 1 ", want: "1", wantOk: true},
		{name: "invalid", input: "2", want: "", wantOk: false},
		{name: "garbage", input: "yes", want: "", wantOk: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := normalizeIsKeySchoolFilter(tt.input)
			if got != tt.want || ok != tt.wantOk {
				t.Fatalf("normalizeIsKeySchoolFilter(%q) = (%q,%v), want (%q,%v)", tt.input, got, ok, tt.want, tt.wantOk)
			}
		})
	}
}
