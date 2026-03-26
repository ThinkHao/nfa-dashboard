package repository

import (
	"testing"
	"time"
)

func TestShouldUseDailyMonthlyAggregation(t *testing.T) {
	cases := []struct {
		name   string
		filter map[string]interface{}
		want   bool
	}{
		{name: "no filter", filter: map[string]interface{}{}, want: false},
		{name: "nil owner", filter: map[string]interface{}{"channel_owner_user_id": nil}, want: false},
		{name: "zero owner int", filter: map[string]interface{}{"channel_owner_user_id": 0}, want: false},
		{name: "zero owner uint64", filter: map[string]interface{}{"channel_owner_user_id": uint64(0)}, want: false},
		{name: "zero owner string", filter: map[string]interface{}{"channel_owner_user_id": "0"}, want: false},
		{name: "empty owner string", filter: map[string]interface{}{"channel_owner_user_id": " "}, want: false},
		{name: "positive owner int", filter: map[string]interface{}{"channel_owner_user_id": 12}, want: true},
		{name: "positive owner uint64", filter: map[string]interface{}{"channel_owner_user_id": uint64(12)}, want: true},
		{name: "positive owner string", filter: map[string]interface{}{"channel_owner_user_id": "12"}, want: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := shouldUseDailyMonthlyAggregation(tc.filter)
			if got != tc.want {
				t.Fatalf("shouldUseDailyMonthlyAggregation()=%v, want=%v", got, tc.want)
			}
		})
	}
}

func TestToYearMonth(t *testing.T) {
	d, _ := time.Parse("2006-01-02", "2026-03-24")
	if got, ok := toYearMonth(d); !ok || got != "2026-03" {
		t.Fatalf("toYearMonth(time)=(%v,%v), want=(2026-03,true)", got, ok)
	}
	if got, ok := toYearMonth("2026-03-01"); !ok || got != "2026-03" {
		t.Fatalf("toYearMonth(string date)=(%v,%v), want=(2026-03,true)", got, ok)
	}
	if got, ok := toYearMonth("2026-03"); !ok || got != "2026-03" {
		t.Fatalf("toYearMonth(string month)=(%v,%v), want=(2026-03,true)", got, ok)
	}
	if _, ok := toYearMonth("bad"); ok {
		t.Fatalf("toYearMonth(bad) should fail")
	}
}
