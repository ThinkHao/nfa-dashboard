package service

import (
	"os"
	"path/filepath"
	"reflect"
	"strconv"
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

func TestResolveTrafficReportWindowUsesMonthLabelForFullMonth(t *testing.T) {
	task := &model.TrafficReportTask{Timezone: "Asia/Shanghai", WindowSelector: "custom", WindowParams: []byte(`{"start_time":"2026-09-01 00:00:00","end_time":"2026-09-30 23:59:59"}`)}
	window, err := resolveTrafficReportWindow(task)
	if err != nil {
		t.Fatal(err)
	}
	if window.Label != "202609" {
		t.Fatalf("window label = %q, want 202609", window.Label)
	}
}

func TestBuildEDCDailyReportRowsMatchesLegacyExportColumns(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	start := time.Date(2026, 9, 1, 0, 0, 0, 0, loc)
	window := trafficReportWindow{Start: start, End: time.Date(2026, 9, 2, 23, 59, 59, 0, loc)}
	rows := make([]trafficReportEDCRow, 0, 40)
	for i := 0; i < 20; i++ {
		rows = append(rows, trafficReportEDCRow{CreateTime: start.Add(time.Duration(i) * 5 * time.Minute), EntityID: 1, EntityName: "BJ-Bilibili", ServiceSize: int64(i + 1), CacheSize: 1000})
		rows = append(rows, trafficReportEDCRow{CreateTime: start.Add(time.Duration(i) * 5 * time.Minute), EntityID: 2, EntityName: "BJ-Bilibili-Backup", ServiceSize: 10, CacheSize: 2000})
	}
	// Matched EDC names are combined by timestamp before calculating the daily percentile.
	params := trafficReportParams{Direction: "both", UnitBase: 1024, DataSourceInstance: "ali", EDCName: "BJ-Bilibili,BJ-Bilibili-Backup", SettlementMode: "daily_95_avg"}
	got := buildEDCDailyReportRows(rows, window, params, 14)
	wantColumns := []string{"date", "edc_name", "data_source_instance", "daily_95th_percentile_raw", "daily_95th_percentile_mbps", "data_points_daily", "saler_group", "saler"}
	if columns := reportKeysForRows("edc", got); !reflect.DeepEqual(columns, wantColumns) {
		t.Fatalf("columns = %#v, want %#v", columns, wantColumns)
	}
	if len(got) != 2 {
		t.Fatalf("row count = %d, want 2 daily rows", len(got))
	}
	first := got[0]
	if first["date"] != "2026-09-01" || first["edc_name"] != "BJ-Bilibili,BJ-Bilibili-Backup" || first["data_source_instance"] != "ali" {
		t.Fatalf("unexpected first row metadata: %#v", first)
	}
	if first["daily_95th_percentile_raw"] != float64(16) || first["data_points_daily"] != 20 {
		t.Fatalf("unexpected daily 95 row: %#v", first)
	}
	wantMbps := float64(16*8) / 300 / 1024 / 1024
	if first["daily_95th_percentile_mbps"] != wantMbps {
		t.Fatalf("daily Mbps = %v, want %v", first["daily_95th_percentile_mbps"], wantMbps)
	}
	if summary := buildEDCSummaryReportRows(rows, got, params, 14, window); len(summary) != 1 || summary[0]["edc_name"] != params.EDCName || summary[0]["95th_percentile_raw"] != float64(8) {
		t.Fatalf("summary should aggregate matched names using service_size only: %#v", summary)
	}
	monthly := buildEDCMonthlyReportRows(rows, got, window, params, 14)
	if len(monthly) != 1 || monthly[0]["edc_name"] != params.EDCName || monthly[0]["95th_percentile_raw"] != float64(8) || monthly[0]["data_points"] != 2 {
		t.Fatalf("monthly row should aggregate matched names using service_size only: %#v", monthly)
	}
	second := got[1]
	if second["date"] != "2026-09-02" || second["daily_95th_percentile_raw"] != float64(0) || second["data_points_daily"] != 0 {
		t.Fatalf("expected zero-data day, got %#v", second)
	}
}

func TestBuildEDCReportRowsUsesSelectedExportMode(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	start := time.Date(2026, 9, 1, 0, 0, 0, 0, loc)
	window := trafficReportWindow{Start: start, End: start.Add(23*time.Hour + 59*time.Minute + 59*time.Second)}
	input := []trafficReportEDCRow{{CreateTime: start, EntityID: 1, EntityName: "BJ-Bilibili", ServiceSize: 120}}

	daily := buildEDCReportRows(input, window, trafficReportParams{Direction: "both", UnitBase: 1024, ExportDaily: true}, 14)
	if len(daily) != 1 || daily[0]["date"] != "2026-09-01" {
		t.Fatalf("daily mode rows = %#v", daily)
	}

	raw := buildEDCReportRows(input, window, trafficReportParams{Direction: "both", UnitBase: 1024, ExportRaw: true}, 14)
	if len(raw) != 1 || raw[0]["create_time"] != start || raw[0]["selected_bytes"] != float64(120) {
		t.Fatalf("raw mode rows = %#v", raw)
	}

	preferRaw := buildEDCReportRows(input, window, trafficReportParams{Direction: "both", UnitBase: 1024, ExportRaw: true, ExportDaily: true}, 14)
	if _, ok := preferRaw[0]["create_time"]; !ok {
		t.Fatalf("raw mode should take precedence when both flags are set: %#v", preferRaw)
	}

	monthly := buildEDCReportRows(input, window, trafficReportParams{Direction: "both", UnitBase: 1024, MonthlyAggregate: true, ExportDaily: true}, 14)
	if len(monthly) != 1 || monthly[0]["month"] != "2026-09" {
		t.Fatalf("monthly mode should take precedence over daily mode: %#v", monthly)
	}
}

func TestBuildNFAReportRowsDailyCanMergeV4V6(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	at := time.Date(2026, 9, 1, 0, 0, 0, 0, loc)
	input := []trafficReportNFARow{
		{CreateTime: at, SchoolID: "school-1", SchoolName: "Example_V4", Region: "四川省", CP: "bilibili", Recv: 120},
		{CreateTime: at, SchoolID: "school-1", SchoolName: "Example_V6", Region: "四川省", CP: "bilibili", Recv: 80},
	}
	got := buildNFAReportRows(input, trafficReportWindow{Start: at, End: at.Add(23*time.Hour + 59*time.Minute + 59*time.Second)}, trafficReportParams{
		Direction: "both", UnitBase: 1000, ExportDaily: true, CombineV4V6: true, MergeKey: "ipgroup_name_base",
	})
	if len(got) != 1 {
		t.Fatalf("merged daily rows = %#v, want one row", got)
	}
	row := got[0]
	if row["school_name"] != "Example" || row["data_points_daily"] != 1 {
		t.Fatalf("unexpected merged daily row: %#v", row)
	}
	if raw := row["daily_95th_percentile_raw"].(float64); raw < 199.999 || raw > 200.001 {
		t.Fatalf("daily raw 95 = %v, want 200", raw)
	}
}

func TestNormalizeTrafficReportParamsMatchesExportOptionDependencies(t *testing.T) {
	got := normalizeTrafficReportParams(trafficReportParams{ExportRaw: true, ExportDaily: true, MonthlyAggregate: true, CombineV4V6: false, MergeKey: "school_id"})
	if got.ExportDaily || got.MonthlyAggregate || got.MergeKey != "" || got.SettlementMode != "range_95" || got.SortOrder != "desc" || got.BatchSize != 200 {
		t.Fatalf("normalized options = %#v", got)
	}
}

func TestLegacyNFARawTableIncludesNFAIntervalConversions(t *testing.T) {
	at := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	table := legacyNFARawTable([]legacyNFAGroup{{
		SchoolID: "school-1", Label: "Example", IPGroupID: "group-1", NFAUUID: "uuid-1",
		Points: map[time.Time]legacyNFAPoint{at: {At: at, Recv: 60, Send: 40}},
	}}, "both", 1000, "default")
	if len(table.Rows) != 1 || len(table.Rows[0]) != len(table.Columns) {
		t.Fatalf("unexpected NFA raw table: %#v", table)
	}
	columns := make(map[string]int, len(table.Columns))
	for i, column := range table.Columns {
		columns[column] = i
	}
	if table.Rows[0][columns["selected_bytes"]] != "100" || table.Rows[0][columns["data_source_instance"]] != "default" {
		t.Fatalf("unexpected NFA raw row: %#v", table.Rows[0])
	}
	recvMbps, err := strconv.ParseFloat(table.Rows[0][columns["recv_mbps_1000"]], 64)
	if err != nil || recvMbps != 0.000008 {
		t.Fatalf("NFA recv Mbps = %q (%v), want 0.000008", table.Rows[0][columns["recv_mbps_1000"]], err)
	}
}

func TestFilterTrafficReportXLSXArtifacts(t *testing.T) {
	artifacts := []model.TrafficReportArtifact{
		{ID: 1, FileName: "report.xlsx"},
		{ID: 2, FileName: "report.csv"},
		{ID: 3, FileName: "report-metadata.json"},
		{ID: 4, FileName: "another.XLSX"},
	}
	got := filterTrafficReportXLSXArtifacts(artifacts)
	if len(got) != 2 || got[0].ID != 1 || got[1].ID != 4 {
		t.Fatalf("filtered artifacts = %#v, want only XLSX", got)
	}
}

func TestSanitizeReportNamePreservesChineseAndReplacesPathSeparators(t *testing.T) {
	if got := sanitizeReportName("重点报表/北京"); got != "重点报表_北京" {
		t.Fatalf("sanitizeReportName() = %q", got)
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
