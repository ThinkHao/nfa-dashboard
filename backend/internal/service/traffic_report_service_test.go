package service

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"nfa-dashboard/internal/model"
)

func TestNextTrafficReportTimeDailyUsesConfiguredTimezone(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	now := time.Date(2026, 9, 17, 1, 0, 0, 0, loc)
	got, err := nextTrafficReportTime(now, "daily", "02:00", "Asia/Shanghai")
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, 9, 17, 2, 0, 0, 0, loc)
	if !got.Equal(want) {
		t.Fatalf("got %s, want %s", got, want)
	}
}

func TestResolveTrafficReportWindowCustomPreservesTimezoneAndLabel(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	task := &model.TrafficReportTask{Timezone: "Asia/Shanghai", WindowSelector: "custom", WindowParams: []byte(`{"start_time":"2026-09-01 00:00:00","end_time":"2026-09-02 00:00:00"}`)}
	window, err := resolveTrafficReportWindow(task)
	if err != nil {
		t.Fatal(err)
	}
	if window.Start.Hour() != 0 || window.Start.Location().String() != loc.String() || window.Label != "20260901-20260902" {
		t.Fatalf("unexpected window: %+v", window)
	}
}

func TestBuildReportSummaryUsesNfatool95Rank(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	params := trafficReportParams{Direction: "both", UnitBase: 1000}
	points := map[time.Time]float64{}
	for i := 0; i < 20; i++ {
		points[time.Date(2026, 9, 1, 0, i, 0, 0, loc)] = float64(i + 1)
	}
	summary := buildReportSummary("nfa", params, points, trafficReportWindow{Start: time.Date(2026, 9, 1, 0, 0, 0, 0, loc), End: time.Date(2026, 9, 2, 0, 0, 0, 0, loc)}, 20)
	if summary["range_points"] != 20 {
		t.Fatalf("range_points=%v", summary["range_points"])
	}
	if summary["daily_days"] != 1 {
		t.Fatalf("daily_days=%v", summary["daily_days"])
	}
	if summary["range_95_mbps"].(float64) <= 0 {
		t.Fatalf("range95=%v", summary["range_95_mbps"])
	}
}

func TestLegacyTaskIDAcceptsNumericAndStringMarkers(t *testing.T) {
	tests := []struct {
		name   string
		params string
		want   uint64
		ok     bool
	}{
		{name: "number", params: `{"legacy_nfatool":{"source_task_id":170}}`, want: 170, ok: true},
		{name: "string", params: `{"legacy_nfatool":{"source_task_id":"295"}}`, want: 295, ok: true},
		{name: "missing", params: `{}`, ok: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := legacyTaskID([]byte(tt.params))
			if got != tt.want || ok != tt.ok {
				t.Fatalf("legacyTaskID() = (%d, %v), want (%d, %v)", got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestCountCSVRowsExcludesHeader(t *testing.T) {
	path := filepath.Join(t.TempDir(), "report.csv")
	if err := os.WriteFile(path, []byte("name,value\na,1\nb,2\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := countCSVRows(path)
	if err != nil {
		t.Fatal(err)
	}
	if got != 2 {
		t.Fatalf("countCSVRows() = %d, want 2", got)
	}
}

func TestLegacyMediaTypeUsesArtifactExtension(t *testing.T) {
	if got := legacyMediaType("result.XLSX"); got != "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" {
		t.Fatalf("unexpected xlsx media type: %s", got)
	}
	if got := legacyMediaType("result.csv"); got != "text/csv; charset=utf-8" {
		t.Fatalf("unexpected csv media type: %s", got)
	}
}

func TestLegacyNFA95UsesAscending95thRank(t *testing.T) {
	values := make([]float64, 0, 20)
	for i := 1; i <= 20; i++ {
		values = append(values, float64(i))
	}
	if got := legacyNFA95(values); got != 19 {
		t.Fatalf("legacyNFA95() = %v, want 19", got)
	}
}

func TestLegacyEDCNamePredicateMapsPrefixAndGlob(t *testing.T) {
	fragment, args, err := legacyEDCNamePredicate("`edc_name`", "BJ-jinshan-*", "prefix")
	if err != nil {
		t.Fatal(err)
	}
	if fragment != "`edc_name` LIKE ?" || len(args) != 1 || args[0] != "BJ-jinshan-%" {
		t.Fatalf("unexpected predicate: %s %#v", fragment, args)
	}
}

func TestMigrateLegacyNFAParamsMapsAliases(t *testing.T) {
	params, warnings, err := migrateLegacyParamsToGoV1("nfa", map[string]interface{}{
		"school":              "示例大学",
		"province":            "北京",
		"cp":                  "aliyun",
		"direction":           "recv",
		"unit_base":           float64(1000),
		"data_budget_enabled": true,
		"combine_v4_v6":       true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := params["school_name"]; got != "示例大学" {
		t.Fatalf("school_name = %#v", got)
	}
	if got := params["region"]; got != "北京" {
		t.Fatalf("region = %#v", got)
	}
	if len(warnings) != 1 {
		t.Fatalf("warnings = %#v", warnings)
	}
}

func TestMigrateLegacyNFAParamsRejectsAggregateMode(t *testing.T) {
	if _, _, err := migrateLegacyParamsToGoV1("nfa", map[string]interface{}{"aggregate_all": true}); err == nil {
		t.Fatal("expected aggregate_all migration to be rejected")
	}
}

func TestMigrateLegacyEDCParamsRejectsPrefixMatch(t *testing.T) {
	_, _, err := migrateLegacyParamsToGoV1("edc", map[string]interface{}{
		"edc_name":       "BJ-jinshan-*",
		"edc_match_mode": "prefix",
	})
	if err == nil {
		t.Fatal("expected non-exact EDC match to be rejected")
	}
}

func TestLegacySafeArtifactNameReplacesWindowsWildcards(t *testing.T) {
	if got := legacySafeArtifactName("BJ-jinshan-*-raw"); got != "BJ-jinshan-_-raw" {
		t.Fatalf("legacySafeArtifactName() = %q", got)
	}
}
