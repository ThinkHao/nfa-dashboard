package repository

import (
	"nfa-dashboard/internal/model"
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

func TestNormalizeDayBounds(t *testing.T) {
	loc := time.FixedZone("CST", 8*3600)
	start := time.Date(2026, 1, 1, 11, 22, 33, 0, loc)
	end := time.Date(2026, 3, 31, 23, 59, 58, 0, loc)

	gotStart, gotEndExclusive := normalizeDayBounds(start, end)
	if gotStart == nil || gotEndExclusive == nil {
		t.Fatalf("normalizeDayBounds should return non-nil bounds")
	}
	if gotStart.Format("2006-01-02 15:04:05") != "2026-01-01 00:00:00" {
		t.Fatalf("start bound = %s, want 2026-01-01 00:00:00", gotStart.Format("2006-01-02 15:04:05"))
	}
	if gotEndExclusive.Format("2006-01-02 15:04:05") != "2026-04-01 00:00:00" {
		t.Fatalf("end exclusive = %s, want 2026-04-01 00:00:00", gotEndExclusive.Format("2006-01-02 15:04:05"))
	}
}

func TestBuildChunkRanges(t *testing.T) {
	ranges := buildChunkRanges(1300, 500)
	if len(ranges) != 3 {
		t.Fatalf("len(ranges) = %d, want 3", len(ranges))
	}
	if ranges[0][0] != 0 || ranges[0][1] != 500 {
		t.Fatalf("first range = %+v, want [0,500)", ranges[0])
	}
	if ranges[1][0] != 500 || ranges[1][1] != 1000 {
		t.Fatalf("second range = %+v, want [500,1000)", ranges[1])
	}
	if ranges[2][0] != 1000 || ranges[2][1] != 1300 {
		t.Fatalf("third range = %+v, want [1000,1300)", ranges[2])
	}
}

func TestSettlementCustomerKey(t *testing.T) {
	d := time.Date(2026, 2, 11, 0, 0, 0, 0, time.UTC)
	key := settlementCustomerKey("湖北省", "jinshan", "武汉纺织大学", d)
	want := "湖北省|jinshan|武汉纺织大学|2026-02-11"
	if key != want {
		t.Fatalf("settlementCustomerKey()=%q, want=%q", key, want)
	}
}

func TestBuildExistingSettlementMap(t *testing.T) {
	d1 := time.Date(2026, 2, 11, 0, 0, 0, 0, time.UTC)
	d2 := time.Date(2026, 2, 12, 0, 0, 0, 0, time.UTC)
	rows := []model.SettlementCustomer{
		{ID: 101, Region: "湖北省", CP: "jinshan", SchoolName: "武汉纺织大学", ServiceDate: &d1},
		{ID: 202, Region: "湖北省", CP: "jinshan", SchoolName: "武汉纺织大学", ServiceDate: &d2},
	}
	m := buildExistingSettlementMap(rows)
	if len(m) != 2 {
		t.Fatalf("len(map)=%d, want=2", len(m))
	}
	if got := m[settlementCustomerKey("湖北省", "jinshan", "武汉纺织大学", d1)]; got != 101 {
		t.Fatalf("map[d1]=%d, want=101", got)
	}
	if got := m[settlementCustomerKey("湖北省", "jinshan", "武汉纺织大学", d2)]; got != 202 {
		t.Fatalf("map[d2]=%d, want=202", got)
	}
}
