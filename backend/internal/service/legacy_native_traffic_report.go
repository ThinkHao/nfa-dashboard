package service

import (
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/xuri/excelize/v2"
	"nfa-dashboard/config"
	"nfa-dashboard/internal/model"
)

const trafficReportLegacyNativeEngineVersion = "legacy-native-v1"

type legacyNativeTable struct {
	Columns []string
	Rows    [][]string
}

type legacyEDCPoint struct {
	At  time.Time
	Raw float64
}

type legacyEDCRawRow struct {
	At   time.Time
	Name string
	Raw  float64
}

type legacyNFAMeta struct {
	SchoolID    string
	SchoolName  string
	IPGroupName string
	IPGroupID   string
	NFAUUID     string
	CP          string
	SalerGroup  string
	Saler       string
	HashUUID    string
}

type legacyNFAPoint struct {
	IPGroupID string
	NFAUUID   string
	At        time.Time
	Recv      float64
	Send      float64
}

type legacyNFAGroup struct {
	Key        string
	Label      string
	SchoolID   string
	IPGroupID  string
	NFAUUID    string
	SalerGroup string
	Saler      string
	Points     map[time.Time]legacyNFAPoint
}

func legacyOriginalParams(raw []byte) map[string]interface{} {
	var all map[string]interface{}
	if err := json.Unmarshal(raw, &all); err != nil {
		return map[string]interface{}{}
	}
	if marker, ok := all["legacy_nfatool"].(map[string]interface{}); ok {
		if original, ok := marker["original_params"].(map[string]interface{}); ok {
			out := make(map[string]interface{}, len(original)+4)
			for k, v := range original {
				out[k] = v
			}
			if instance, ok := marker["data_source_instance"].(string); ok && instance != "" {
				out["data_source_instance"] = instance
			}
			if template, ok := marker["output_filename_template"].(string); ok && template != "" {
				out["output_filename_template"] = template
			}
			return out
		}
	}
	return all
}

func legacyParamString(params map[string]interface{}, key string) string {
	if value, ok := params[key]; ok && value != nil {
		return strings.TrimSpace(fmt.Sprint(value))
	}
	return ""
}

func legacyParamBool(params map[string]interface{}, key string) bool {
	value, ok := params[key]
	if !ok || value == nil {
		return false
	}
	switch v := value.(type) {
	case bool:
		return v
	case float64:
		return v != 0
	case string:
		value := strings.ToLower(strings.TrimSpace(v))
		return value == "1" || value == "true" || value == "yes" || value == "on"
	default:
		return false
	}
}

func legacyParamInt(params map[string]interface{}, key string, fallback int) int {
	value, ok := params[key]
	if !ok || value == nil {
		return fallback
	}
	var n int64
	switch v := value.(type) {
	case float64:
		n = int64(v)
	case json.Number:
		n, _ = v.Int64()
	case string:
		n, _ = strconv.ParseInt(strings.TrimSpace(v), 10, 64)
	case int:
		n = int64(v)
	case int64:
		n = v
	default:
		return fallback
	}
	if n <= 0 {
		return fallback
	}
	return int(n)
}

func legacyParamFloat(params map[string]interface{}, key string, fallback float64) float64 {
	value, ok := params[key]
	if !ok || value == nil {
		return fallback
	}
	switch v := value.(type) {
	case float64:
		return v
	case json.Number:
		if n, err := v.Float64(); err == nil {
			return n
		}
	case string:
		if n, err := strconv.ParseFloat(strings.TrimSpace(v), 64); err == nil {
			return n
		}
	}
	return fallback
}

func legacyFormatted3(value float64) string {
	return fmt.Sprintf("%.3f", value)
}

func legacyRawText(value float64) string {
	if math.Abs(value-math.Round(value)) < 1e-9 {
		return strconv.FormatInt(int64(math.Round(value)), 10)
	}
	return strconv.FormatFloat(value, 'f', -1, 64)
}

func legacyFloatText(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}

func legacyNth95(values []float64, rankIndex int) float64 {
	if len(values) == 0 {
		return 0
	}
	ordered := append([]float64(nil), values...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i] > ordered[j] })
	if rankIndex < 0 {
		rankIndex = 0
	}
	if rankIndex >= len(ordered) {
		rankIndex = len(ordered) - 1
	}
	return ordered[rankIndex]
}

func legacySelectedRaw(recv, send float64, direction string) float64 {
	switch strings.ToLower(direction) {
	case "recv":
		return recv
	case "send":
		return send
	default:
		return recv + send
	}
}

func legacyWindowLabel(task *model.TrafficReportTask, window trafficReportWindow) string {
	if task.WindowSelector != "custom" {
		return window.Label
	}
	var params map[string]interface{}
	_ = json.Unmarshal(task.WindowParams, &params)
	start := legacyParamString(params, "start_time")
	end := legacyParamString(params, "end_time")
	if len(start) >= 10 && len(end) >= 10 {
		return start[:10] + "-" + end[:10]
	}
	return window.Start.Format("2006-01-02") + "-" + window.End.Format("2006-01-02")
}

func legacyTotalDays(window trafficReportWindow) int {
	start := time.Date(window.Start.Year(), window.Start.Month(), window.Start.Day(), 0, 0, 0, 0, window.Start.Location())
	end := time.Date(window.End.Year(), window.End.Month(), window.End.Day(), 0, 0, 0, 0, window.End.Location())
	days := int(end.Sub(start).Hours()/24) + 1
	if days < 1 {
		return 1
	}
	return days
}

func legacySafeIdentifier(value, field string) (string, error) {
	if !regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`).MatchString(value) {
		return "", fmt.Errorf("invalid SQL identifier for %s", field)
	}
	return "`" + value + "`", nil
}

func (s *trafficReportService) executeLegacyNative(ctx context.Context, task *model.TrafficReportTask, run *model.TrafficReportRun, window trafficReportWindow) (int64, map[string]interface{}, error) {
	params := legacyOriginalParams(task.Params)
	if task.DataSourceType == "edc" {
		return s.executeLegacyNativeEDC(ctx, task, run, window, params)
	}
	return s.executeLegacyNativeNFA(ctx, task, run, window, params)
}

func (s *trafficReportService) executeLegacyNativeNFA(ctx context.Context, task *model.TrafficReportTask, run *model.TrafficReportRun, window trafficReportWindow, params map[string]interface{}) (int64, map[string]interface{}, error) {
	metas, err := queryLegacyNFAMeta(ctx, params)
	if err != nil {
		return 0, nil, err
	}
	if len(metas) == 0 {
		return 0, nil, fmt.Errorf("native NFA has no matching ipgroups")
	}
	batchSize := legacyParamInt(params, "batch_size", 200)
	if batchSize < 1 {
		batchSize = 200
	}
	points, err := queryLegacyNFAPoints(ctx, metas, window, batchSize)
	if err != nil {
		return 0, nil, err
	}
	if len(points) == 0 {
		return 0, nil, fmt.Errorf("native NFA has no raw points in requested window")
	}
	unitBase := legacyParamInt(params, "unit_base", 1024)
	if unitBase != 1000 && unitBase != 1024 {
		unitBase = 1024
	}
	direction := legacyParamString(params, "direction")
	if direction == "" {
		direction = "both"
	}
	combine := legacyParamBool(params, "combine_v4_v6")
	mergeKey := legacyParamString(params, "merge_key")
	groups := groupLegacyNFAPoints(metas, points, params, combine, mergeKey)
	settlementMode := legacyParamString(params, "settlement_mode")
	if settlementMode == "" {
		settlementMode = "range_95"
	}
	windowLabel := legacyWindowLabel(task, window)
	baseName := legacyNFABaseName(params, windowLabel)
	formats := legacyExportFormats(task.ExportFormats)
	params["_legacy_total_days"] = legacyTotalDays(window)
	var table legacyNativeTable
	if legacyParamBool(params, "monthly_aggregate") {
		table = legacyNFAMonthlyTable(groups, window, direction, unitBase, settlementMode, params, combine)
		baseName += "-monthly"
	} else if legacyParamBool(params, "export_daily") {
		table = legacyNFADailyTable(groups, window, direction, unitBase, legacyParamBool(params, "aggregate_all"))
	} else {
		table = legacyNFASummaryTable(groups, window, direction, unitBase, settlementMode, combine, legacyParamBool(params, "aggregate_all"))
	}
	count, err := s.writeLegacyNativeTables(run, baseName, []legacyNativeTable{table}, formats)
	if err != nil {
		return 0, nil, err
	}
	return count, map[string]interface{}{"engine_version": trafficReportLegacyNativeEngineVersion, "source": "nfa", "direction": direction, "unit_base": unitBase, "settlement_mode": settlementMode, "matched_ipgroups": len(metas), "raw_points": len(points), "combine_v4_v6": combine}, nil
}

func queryLegacyNFAMeta(ctx context.Context, params map[string]interface{}) ([]legacyNFAMeta, error) {
	rows, err := model.DB.WithContext(ctx).Raw("SHOW COLUMNS FROM nfa.nfa_ipgroup").Rows()
	if err != nil {
		return nil, fmt.Errorf("inspect native NFA metadata schema: %w", err)
	}
	columns := map[string]bool{}
	for rows.Next() {
		var field, typ, nullable, key, defaultValue, extra sql.NullString
		if err := rows.Scan(&field, &typ, &nullable, &key, &defaultValue, &extra); err != nil {
			rows.Close()
			return nil, err
		}
		columns[field.String] = true
	}
	rows.Close()
	selects := []string{"school_id", "school_name", "ipgroup_name", "ipgroup_id", "nfa_uuid", "cp"}
	for _, optional := range []string{"saler_group", "saler", "hash_uuid"} {
		if columns[optional] {
			selects = append(selects, optional)
		}
	}
	where := []string{"region = ?", "cp = ?", "type = ?"}
	args := []interface{}{legacyParamString(params, "province"), legacyParamString(params, "cp"), "yuanxiao"}
	if school := legacyParamString(params, "school"); school != "" {
		parts := splitLegacyNames(school)
		if len(parts) > 0 {
			placeholders := make([]string, len(parts))
			for i, value := range parts {
				placeholders[i] = "?"
				args = append(args, value)
			}
			where = append(where, "school_name IN ("+strings.Join(placeholders, ",")+")")
		}
	}
	query := "SELECT DISTINCT " + strings.Join(selects, ", ") + " FROM nfa.nfa_ipgroup WHERE " + strings.Join(where, " AND ")
	result, err := model.DB.WithContext(ctx).Raw(query, args...).Rows()
	if err != nil {
		return nil, fmt.Errorf("query native NFA ipgroups: %w", err)
	}
	defer result.Close()
	seen := map[string]struct{}{}
	metas := make([]legacyNFAMeta, 0)
	for result.Next() {
		values := make([]sql.NullString, len(selects))
		dest := make([]interface{}, len(values))
		for i := range values {
			dest[i] = &values[i]
		}
		if err := result.Scan(dest...); err != nil {
			return nil, err
		}
		meta := legacyNFAMeta{
			SchoolID: values[0].String, SchoolName: values[1].String, IPGroupName: values[2].String,
			IPGroupID: values[3].String, NFAUUID: values[4].String, CP: values[5].String,
		}
		idx := 6
		if columns["saler_group"] {
			meta.SalerGroup = values[idx].String
			idx++
		}
		if columns["saler"] {
			meta.Saler = values[idx].String
			idx++
		}
		if columns["hash_uuid"] {
			meta.HashUUID = values[idx].String
		}
		key := meta.IPGroupID + "\x00" + meta.NFAUUID

		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		metas = append(metas, meta)
	}
	if err := result.Err(); err != nil {
		return nil, err
	}
	return metas, nil
}

func splitLegacyNames(value string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0)
	for _, item := range strings.FieldsFunc(value, func(r rune) bool { return r == ',' || r == '\n' }) {
		item = strings.TrimSpace(item)
		if item != "" {
			if _, ok := seen[item]; !ok {
				seen[item] = struct{}{}
				out = append(out, item)
			}
		}
	}
	return out
}

func queryLegacyNFAPoints(ctx context.Context, metas []legacyNFAMeta, window trafficReportWindow, batchSize int) ([]legacyNFAPoint, error) {
	if len(metas) == 0 {
		return nil, nil
	}
	if batchSize < 1 {
		batchSize = 200
	}
	out := make([]legacyNFAPoint, 0)
	for start := 0; start < len(metas); start += batchSize {
		end := start + batchSize
		if end > len(metas) {
			end = len(metas)
		}
		batch := metas[start:end]
		withHash := make([]string, 0, len(batch))
		withoutHash := make([]legacyNFAMeta, 0)
		for _, meta := range batch {
			if meta.HashUUID != "" {
				withHash = append(withHash, meta.HashUUID)
			} else {
				withoutHash = append(withoutHash, meta)
			}
		}
		queries := make([]struct {
			where string
			args  []interface{}
		}, 0, 2)
		if len(withHash) > 0 {
			placeholders := strings.TrimRight(strings.Repeat("?,", len(withHash)), ",")
			args := []interface{}{window.Start.Format("2006-01-02 15:04:05"), window.End.Format("2006-01-02 15:04:05")}
			for _, hash := range withHash {
				args = append(args, hash)
			}
			queries = append(queries, struct {
				where string
				args  []interface{}
			}{where: "hash_uuid IN (" + placeholders + ")", args: args})
		}
		if len(withoutHash) > 0 {
			placeholders := strings.TrimRight(strings.Repeat("(?,?),", len(withoutHash)), ",")
			args := []interface{}{window.Start.Format("2006-01-02 15:04:05"), window.End.Format("2006-01-02 15:04:05")}
			for _, meta := range withoutHash {
				args = append(args, meta.IPGroupID, meta.NFAUUID)
			}
			queries = append(queries, struct {
				where string
				args  []interface{}
			}{where: "(ipgroup_id, nfa_uuid) IN (" + placeholders + ")", args: args})
		}
		for _, queryPart := range queries {
			query := "SELECT ipgroup_id, nfa_uuid, create_time, recv, send FROM nfa.nfa_ip_group_speed_logs_5m WHERE create_time BETWEEN ? AND ? AND " + queryPart.where
			rows, err := model.DB.WithContext(ctx).Raw(query, queryPart.args...).Rows()
			if err != nil {
				return nil, fmt.Errorf("query native NFA raw points: %w", err)
			}
			for rows.Next() {
				var ipgroupID, nfaUUID sql.NullString
				var at time.Time
				var recv, send sql.NullFloat64
				if err := rows.Scan(&ipgroupID, &nfaUUID, &at, &recv, &send); err != nil {
					rows.Close()
					return nil, err
				}
				out = append(out, legacyNFAPoint{IPGroupID: ipgroupID.String, NFAUUID: nfaUUID.String, At: at, Recv: recv.Float64, Send: send.Float64})
			}
			if err := rows.Err(); err != nil {
				rows.Close()
				return nil, err
			}
			rows.Close()
		}
	}
	return out, nil
}

func groupLegacyNFAPoints(metas []legacyNFAMeta, points []legacyNFAPoint, params map[string]interface{}, combine bool, mergeKey string) []legacyNFAGroup {
	metaByPair := make(map[string]legacyNFAMeta, len(metas))
	for _, meta := range metas {
		metaByPair[meta.IPGroupID+"\x00"+meta.NFAUUID] = meta
	}
	groups := make([]legacyNFAGroup, 0)
	byKey := map[string]int{}
	aggregateAll := legacyParamBool(params, "aggregate_all")
	for _, point := range points {
		meta := metaByPair[point.IPGroupID+"\x00"+point.NFAUUID]
		key, label := legacyNFAGroupKey(meta, combine, mergeKey, aggregateAll)
		idx, ok := byKey[key]
		if !ok {
			idx = len(groups)
			byKey[key] = idx
			schoolID, ipgroupID, nfaUUID, salerGroup, saler := meta.SchoolID, meta.IPGroupID, meta.NFAUUID, meta.SalerGroup, meta.Saler
			if aggregateAll {
				schoolID, ipgroupID, nfaUUID, salerGroup, saler = "", "", "", "", ""
			}
			groups = append(groups, legacyNFAGroup{Key: key, Label: label, SchoolID: schoolID, IPGroupID: ipgroupID, NFAUUID: nfaUUID, SalerGroup: salerGroup, Saler: saler, Points: map[time.Time]legacyNFAPoint{}})
		} else {
			if groups[idx].SalerGroup == "" {
				groups[idx].SalerGroup = meta.SalerGroup
			}
			if groups[idx].Saler == "" {
				groups[idx].Saler = meta.Saler
			}
		}
		current := groups[idx].Points[point.At]
		current.At = point.At
		current.Recv += point.Recv
		current.Send += point.Send
		groups[idx].Points[point.At] = current
	}
	return groups
}

func legacyNFAGroupKey(meta legacyNFAMeta, combine bool, mergeKey string, aggregateAll bool) (string, string) {
	if aggregateAll {
		return "__all__", "全部院校汇总"
	}
	if !combine {
		return meta.IPGroupID + "\x00" + meta.NFAUUID, meta.IPGroupName
	}
	key := strings.ToLower(strings.TrimSpace(mergeKey))
	if key == "" {
		key = "ipgroup_name_base"
	}
	var value, label string
	switch key {
	case "school_id":
		value, label = meta.SchoolID, meta.IPGroupName
	case "school_name_plus_cp":
		value = strings.Trim(meta.SchoolName+"_"+meta.CP, "_")
		label = value
	case "school_name":
		value, label = meta.SchoolName, meta.SchoolName
	case "ipgroup_name":
		value, label = meta.IPGroupName, meta.IPGroupName
	default:
		value = strings.TrimSuffix(strings.TrimSuffix(meta.IPGroupName, "_V4"), "_v4")
		value = strings.TrimSuffix(strings.TrimSuffix(value, "_V6"), "_v6")
		label = value
	}
	return value, label
}

func legacyNFA95(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	idx := int(math.Ceil(float64(len(values))*0.95)) - 1
	if idx < 0 {
		idx = 0
	}
	ordered := append([]float64(nil), values...)
	sort.Float64s(ordered)
	if idx >= len(ordered) {
		idx = len(ordered) - 1
	}
	return ordered[idx]
}

func legacyNFABaseName(params map[string]interface{}, windowLabel string) string {
	province := legacyParamString(params, "province")
	cp := legacyParamString(params, "cp")
	direction := legacyParamString(params, "direction")
	if direction == "" {
		direction = "both"
	}
	return province + "-" + cp + "-" + direction + "-" + windowLabel
}

func legacyNFAValues(group legacyNFAGroup, window trafficReportWindow, direction string, unitBase int) map[string][]float64 {
	byDay := map[string][]float64{}
	for _, point := range group.Points {
		value := legacySelectedRaw(point.Recv, point.Send, direction) * 8 / 60 / float64(unitBase) / float64(unitBase)
		day := point.At.In(window.Start.Location()).Format("2006-01-02")
		byDay[day] = append(byDay[day], value)
	}
	return byDay
}

func legacyNFADailyTable(groups []legacyNFAGroup, window trafficReportWindow, direction string, unitBase int, aggregateAll bool) legacyNativeTable {
	rows := make([][]string, 0)
	for _, group := range groups {
		byDay := legacyNFAValues(group, window, direction, unitBase)
		days := make([]string, 0, len(byDay))
		for day := range byDay {
			days = append(days, day)
		}
		sort.Strings(days)
		for _, day := range days {
			mbps := legacyNFA95(byDay[day])
			raw := mbps * 60 * float64(unitBase) * float64(unitBase) / 8
			if aggregateAll {
				rows = append(rows, []string{"", group.Label, "", "", day, legacyFormatted3(raw), legacyFormatted3(mbps), direction, strconv.Itoa(len(byDay[day])), "", ""})
			} else {
				rows = append(rows, []string{group.SchoolID, group.Label, group.IPGroupID, group.NFAUUID, group.SalerGroup, group.Saler, day, legacyFormatted3(raw), legacyFormatted3(mbps), direction, strconv.Itoa(len(byDay[day]))})
			}
		}
	}
	rows = sortLegacyNFARows(rows, true, false)
	if aggregateAll {
		return legacyNativeTable{Columns: []string{"school_id", "ipgroup_name", "ipgroup_id", "nfa_uuid", "date", "daily_95th_percentile_raw", "daily_95th_percentile_mbps", "direction", "data_points_daily", "saler_group", "saler"}, Rows: rows}
	}
	return legacyNativeTable{Columns: []string{"school_id", "ipgroup_name", "ipgroup_id", "nfa_uuid", "saler_group", "saler", "date", "daily_95th_percentile_raw", "daily_95th_percentile_mbps", "direction", "data_points_daily"}, Rows: rows}
}

func legacyNFASummaryTable(groups []legacyNFAGroup, window trafficReportWindow, direction string, unitBase int, settlementMode string, combine bool, aggregateAll bool) legacyNativeTable {
	rows := make([][]string, 0, len(groups))
	for _, group := range groups {
		byDay := legacyNFAValues(group, window, direction, unitBase)
		values := make([]float64, 0)
		for _, v := range byDay {
			values = append(values, legacyNFA95(v))
		}
		mbps := 0.0
		dataPoints := 0
		if settlementMode == "daily_95_avg" {
			for _, v := range values {
				mbps += v
			}
			mbps /= float64(legacyTotalDays(window))
			dataPoints = len(values)
		} else {
			all := make([]float64, 0)
			for _, point := range group.Points {
				all = append(all, legacySelectedRaw(point.Recv, point.Send, direction)*8/60/float64(unitBase)/float64(unitBase))
			}
			mbps = legacyNFA95(all)
			dataPoints = len(all)
		}
		raw := mbps * 60 * float64(unitBase) * float64(unitBase) / 8
		if aggregateAll {
			rows = append(rows, []string{"", group.Label, "", "", legacyFormatted3(raw), legacyFormatted3(mbps), direction, "", ""})
		} else {
			rows = append(rows, []string{group.SchoolID, group.Label, group.IPGroupID, group.NFAUUID, group.SalerGroup, group.Saler, legacyFormatted3(raw), legacyFormatted3(mbps), strconv.Itoa(dataPoints), direction})
		}
	}
	if !combine {
		rows = sortLegacyNFARows(rows, false, false)
	}
	if aggregateAll {
		return legacyNativeTable{Columns: []string{"school_id", "ipgroup_name", "ipgroup_id", "nfa_uuid", "95th_percentile_raw", "95th_percentile_mbps", "direction", "saler_group", "saler"}, Rows: rows}
	}
	return legacyNativeTable{Columns: []string{"school_id", "ipgroup_name", "ipgroup_id", "nfa_uuid", "saler_group", "saler", "95th_percentile_raw", "95th_percentile_mbps", "data_points", "direction"}, Rows: rows}
}

func legacyNFAMonthlyTable(groups []legacyNFAGroup, window trafficReportWindow, direction string, unitBase int, settlementMode string, params map[string]interface{}, combine bool) legacyNativeTable {
	rows := make([][]string, 0)
	for _, group := range groups {
		months := map[string][]legacyNFAPoint{}
		for at, point := range group.Points {
			month := at.In(window.Start.Location()).Format("2006-01")
			months[month] = append(months[month], point)
		}
		monthNames := make([]string, 0, len(months))
		for month := range months {
			monthNames = append(monthNames, month)
		}
		sort.Strings(monthNames)
		for _, month := range monthNames {
			monthPoints := months[month]
			// nfatool's NFA monthly path calls process_schools_batched with
			// export_daily=false for each calendar month. Even when the
			// settlement mode is daily_95_avg, that path therefore computes a
			// single range 95 over the month's 5-minute points.
			values := make([]float64, 0, len(monthPoints))
			for _, point := range monthPoints {
				values = append(values, legacySelectedRaw(point.Recv, point.Send, direction)*8/60/float64(unitBase)/float64(unitBase))
			}
			mbps := legacyNFA95(values)
			dataPoints := len(values)
			raw := mbps * 60 * float64(unitBase) * float64(unitBase) / 8
			schoolID, ipgroupID, nfaUUID := group.SchoolID, group.IPGroupID, group.NFAUUID
			if combine {
				schoolID, ipgroupID, nfaUUID = "", "", ""
			}
			rows = append(rows, []string{schoolID, group.Label, ipgroupID, nfaUUID, group.SalerGroup, group.Saler, legacyFormatted3(raw), legacyFormatted3(mbps), strconv.Itoa(dataPoints), direction, month})
		}
	}
	if combine {
		sort.SliceStable(rows, func(i, j int) bool {
			if rows[i][10] != rows[j][10] {
				return rows[i][10] < rows[j][10]
			}
			return rows[i][1] < rows[j][1]
		})
	} else {
		rows = sortLegacyNFARows(rows, false, true)
	}
	return legacyNativeTable{Columns: []string{"school_id", "ipgroup_name", "ipgroup_id", "nfa_uuid", "saler_group", "saler", "95th_percentile_raw", "95th_percentile_mbps", "data_points", "direction", "month"}, Rows: rows}
}

func sortLegacyNFARows(rows [][]string, daily, monthly bool) [][]string {
	sort.SliceStable(rows, func(i, j int) bool {
		nameI, nameJ := rows[i][1], rows[j][1]
		baseI, rankI := legacyVariantSort(nameI)
		baseJ, rankJ := legacyVariantSort(nameJ)
		if baseI != baseJ {
			return baseI < baseJ
		}
		if rankI != rankJ {
			return rankI < rankJ
		}
		if nameI != nameJ {
			return nameI < nameJ
		}
		if daily && len(rows[i]) > 6 && len(rows[j]) > 6 {
			return rows[i][6] < rows[j][6]
		}
		if monthly && len(rows[i]) > 6 && len(rows[j]) > 6 {
			return rows[i][6] < rows[j][6]
		}
		return rows[i][0] < rows[j][0]
	})
	return rows
}

func legacyVariantSort(name string) (string, int) {
	lower := strings.ToLower(name)
	if strings.HasSuffix(lower, "_v4") {
		return name[:len(name)-3], 0
	}
	if strings.HasSuffix(lower, "_v6") {
		return name[:len(name)-3], 1
	}
	return name, 2
}

func openLegacyEDC(ctx context.Context) (*sql.DB, error) {
	cfg := config.AppConfig.TrafficReport.EDC
	if strings.TrimSpace(cfg.Host) == "" || strings.TrimSpace(cfg.User) == "" || strings.TrimSpace(cfg.Password) == "" || strings.TrimSpace(cfg.DBName) == "" {
		return nil, errors.New("native EDC source config is incomplete")
	}
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=true&loc=Local", cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("open native EDC source: %w", err)
	}
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping native EDC source: %w", err)
	}
	return db, nil
}

func legacyEDCNamePredicate(column, requested, mode string) (string, []interface{}, error) {
	names := make([]string, 0)
	seen := map[string]struct{}{}
	for _, value := range strings.FieldsFunc(requested, func(r rune) bool { return r == ',' || r == '\n' }) {
		value = strings.TrimSpace(value)
		if value != "" {
			if _, ok := seen[value]; !ok {
				seen[value] = struct{}{}
				names = append(names, value)
			}
		}
	}
	if len(names) == 0 {
		return "", nil, errors.New("edc_name is required")
	}
	parts := make([]string, 0, len(names))
	args := make([]interface{}, 0, len(names))
	for _, name := range names {
		if strings.ContainsAny(name, "*?") {
			parts = append(parts, column+" LIKE ?")
			args = append(args, strings.NewReplacer("*", "%", "?", "_").Replace(name))
			continue
		}
		if strings.EqualFold(mode, "exact") {
			parts = append(parts, column+" = ?")
			args = append(args, name)
		} else {
			parts = append(parts, column+" LIKE ?")
			args = append(args, name+"%")
		}
	}
	if len(parts) == 1 {
		return parts[0], args, nil
	}
	return "(" + strings.Join(parts, " OR ") + ")", args, nil
}

func queryLegacyEDCRaw(ctx context.Context, db *sql.DB, params map[string]interface{}, window trafficReportWindow, aggregate bool) ([]legacyEDCPoint, []legacyEDCRawRow, error) {
	cfg := config.AppConfig.TrafficReport.EDC
	table, err := legacySafeIdentifier(cfg.Table, "edc table")
	if err != nil {
		return nil, nil, err
	}
	timeCol, err := legacySafeIdentifier(cfg.TimeColumn, "edc time column")
	if err != nil {
		return nil, nil, err
	}
	nameCol, err := legacySafeIdentifier(cfg.NameColumn, "edc name column")
	if err != nil {
		return nil, nil, err
	}
	valueCol, err := legacySafeIdentifier(cfg.ValueColumn, "edc value column")
	if err != nil {
		return nil, nil, err
	}
	predicate, args, err := legacyEDCNamePredicate(nameCol, legacyParamString(params, "edc_name"), legacyParamString(params, "edc_match_mode"))
	if err != nil {
		return nil, nil, err
	}
	where := []string{predicate, timeCol + " >= ?", timeCol + " <= ?"}
	args = append(args, window.Start, window.End)
	if strings.TrimSpace(cfg.ExcludeLike) != "" {
		where = append(where, nameCol+" NOT LIKE ?")
		args = append(args, cfg.ExcludeLike)
	}
	if aggregate {
		query := fmt.Sprintf("SELECT %s AS create_time, SUM(%s) AS total_service_size FROM %s WHERE %s GROUP BY %s ORDER BY %s", timeCol, valueCol, table, strings.Join(where, " AND "), timeCol, timeCol)
		rows, err := db.QueryContext(ctx, query, args...)
		if err != nil {
			return nil, nil, fmt.Errorf("query native EDC aggregate: %w", err)
		}
		defer rows.Close()
		out := make([]legacyEDCPoint, 0)
		for rows.Next() {
			var at time.Time
			var raw sql.NullFloat64
			if err := rows.Scan(&at, &raw); err != nil {
				return nil, nil, err
			}
			out = append(out, legacyEDCPoint{At: at, Raw: raw.Float64})
		}
		return out, nil, rows.Err()
	}
	query := fmt.Sprintf("SELECT %s AS create_time, %s AS edc_name, %s AS service_size FROM %s WHERE %s ORDER BY %s, %s", timeCol, nameCol, valueCol, table, strings.Join(where, " AND "), timeCol, nameCol)
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, nil, fmt.Errorf("query native EDC raw rows: %w", err)
	}
	defer rows.Close()
	out := make([]legacyEDCRawRow, 0)
	for rows.Next() {
		var at time.Time
		var name string
		var raw sql.NullFloat64
		if err := rows.Scan(&at, &name, &raw); err != nil {
			return nil, nil, err
		}
		out = append(out, legacyEDCRawRow{At: at, Name: name, Raw: raw.Float64})
	}
	return nil, out, rows.Err()
}

func (s *trafficReportService) executeLegacyNativeEDC(ctx context.Context, task *model.TrafficReportTask, run *model.TrafficReportRun, window trafficReportWindow, params map[string]interface{}) (int64, map[string]interface{}, error) {
	db, err := openLegacyEDC(ctx)
	if err != nil {
		return 0, nil, err
	}
	defer db.Close()
	instance := legacyParamString(params, "data_source_instance")
	if instance == "" {
		instance = "ali"
	}
	unitBase := legacyParamInt(params, "unit_base", 1024)
	if unitBase != 1000 && unitBase != 1024 {
		unitBase = 1024
	}
	direction := legacyParamString(params, "direction")
	if direction == "" {
		direction = "both"
	}
	baseName := legacyEDCBaseName(params, instance, legacyWindowLabel(task, window))
	formats := legacyExportFormats(task.ExportFormats)
	if legacyParamBool(params, "export_raw") {
		_, rawRows, err := queryLegacyEDCRaw(ctx, db, params, window, false)
		if err != nil {
			return 0, nil, err
		}
		table := legacyEDCRawTable(rawRows, params, instance, unitBase)
		count, err := s.writeLegacyNativeTables(run, baseName+"-raw", []legacyNativeTable{table}, formats)
		if err != nil {
			return 0, nil, err
		}
		return count, map[string]interface{}{"engine_version": trafficReportLegacyNativeEngineVersion, "source": "edc", "raw_export": true, "raw_rows": len(rawRows), "instance": instance}, nil
	}
	points, _, err := queryLegacyEDCRaw(ctx, db, params, window, true)
	if err != nil {
		return 0, nil, err
	}
	rankIndex := config.AppConfig.TrafficReport.EDC.DailyRankIndex
	if rankIndex < 0 {
		rankIndex = 14
	}
	daily := legacyEDCDaily(points, window, rankIndex, unitBase, params, instance)
	settlementMode := legacyParamString(params, "settlement_mode")
	if settlementMode == "" {
		settlementMode = "range_95"
	}
	var tables []legacyNativeTable
	if legacyParamBool(params, "monthly_aggregate") {
		tables = []legacyNativeTable{legacyEDCMonthlyTable(daily, points, window, rankIndex, unitBase, settlementMode, params, instance)}
		baseName += "-monthly"
	} else if legacyParamBool(params, "export_daily") {
		tables = []legacyNativeTable{legacyEDCDailyTable(daily)}
	} else {
		tables = []legacyNativeTable{legacyEDCSummaryTable(daily, points, rankIndex, unitBase, settlementMode, params, instance)}
	}
	count, err := s.writeLegacyNativeTables(run, baseName, tables, formats)
	if err != nil {
		return 0, nil, err
	}
	summary := map[string]interface{}{"engine_version": trafficReportLegacyNativeEngineVersion, "source": "edc", "instance": instance, "settlement_mode": settlementMode, "daily_days": len(daily), "range_points": len(points), "direction": direction, "unit_base": unitBase}
	for key, value := range legacyEDCBudgetSummary(points, daily, window, rankIndex, params) {
		summary[key] = value
	}
	return count, summary, nil
}

// legacyEDCBudgetSummary keeps the budget values visible in the run metadata,
// matching the two unit-base results that nfatool displayed on its task card.
// The raw daily average includes every requested day, including days without
// source rows, just like the legacy daily export path.
func legacyEDCBudgetSummary(points []legacyEDCPoint, daily [][]string, window trafficReportWindow, rankIndex int, params map[string]interface{}) map[string]interface{} {
	enabled := legacyParamBool(params, "data_budget_enabled")
	mul := legacyParamFloat(params, "data_budget_mul", 8)
	div := legacyParamFloat(params, "data_budget_div", 300)
	if div == 0 {
		div = 300
	}
	rawDailyAvg := 0.0
	for _, row := range daily {
		if len(row) < 4 {
			continue
		}
		value, _ := strconv.ParseFloat(row[3], 64)
		rawDailyAvg += value
	}
	if totalDays := legacyTotalDays(window); totalDays > 0 {
		rawDailyAvg /= float64(totalDays)
	}
	rawValues := make([]float64, 0, len(points))
	for _, point := range points {
		rawValues = append(rawValues, point.Raw)
	}
	rawRange := legacyNth95(rawValues, rankIndex)
	step1 := func(raw float64) float64 { return raw * mul / div }
	toBudget := func(step float64, base float64) float64 { return step / base / base }
	dailyStep1, rangeStep1 := step1(rawDailyAvg), step1(rawRange)
	return map[string]interface{}{
		"budget_enabled":         enabled,
		"year_month":             window.End.Format("2006-01"),
		"budget_formula":         fmt.Sprintf("raw*%g/%g/base/base", mul, div),
		"budget_mul":             mul,
		"budget_div":             div,
		"raw_daily_95_avg":       rawDailyAvg,
		"raw_range_95":           rawRange,
		"raw_daily_95_avg_step1": dailyStep1,
		"raw_range_95_step1":     rangeStep1,
		"daily_95_avg_1000":      toBudget(dailyStep1, 1000),
		"range_95_1000":          toBudget(rangeStep1, 1000),
		"daily_95_avg_1024":      toBudget(dailyStep1, 1024),
		"range_95_1024":          toBudget(rangeStep1, 1024),
		"budget_daily_days":      len(daily),
		"budget_range_points":    len(points),
	}
}

func legacyEDCBaseName(params map[string]interface{}, instance, windowLabel string) string {
	name := legacyParamString(params, "edc_name")
	if name == "" {
		name = "edc"
	}
	return name + "-" + instance + "-" + windowLabel
}

func legacyExportFormats(raw []byte) []string {
	var formats []string
	_ = json.Unmarshal(raw, &formats)
	if len(formats) == 0 {
		return []string{"csv"}
	}
	return formats
}

func legacyEDCRawTable(rows []legacyEDCRawRow, params map[string]interface{}, instance string, unitBase int) legacyNativeTable {
	columns := []string{"create_time", "edc_name", "service_size", "data_source_type", "data_source_instance", "requested_edc_name", "unit_base", "saler_group", "saler", "service_size_mbps_1000", "service_size_mbps_1024"}
	out := make([][]string, 0, len(rows))
	requested := legacyParamString(params, "edc_name")
	for _, row := range rows {
		out = append(out, []string{row.At.Format("2006-01-02 15:04:05"), row.Name, legacyRawText(row.Raw), "edc", instance, requested, strconv.Itoa(unitBase), "", "", legacyFloatText(row.Raw * 8 / 300 / 1000 / 1000), legacyFloatText(row.Raw * 8 / 300 / 1024 / 1024)})
	}
	return legacyNativeTable{Columns: columns, Rows: out}
}

func legacyEDCDaily(points []legacyEDCPoint, window trafficReportWindow, rankIndex, unitBase int, params map[string]interface{}, instance string) [][]string {
	byDay := map[string][]float64{}
	for _, point := range points {
		day := point.At.In(window.Start.Location()).Format("2006-01-02")
		byDay[day] = append(byDay[day], point.Raw)
	}
	days := make([]string, 0, legacyTotalDays(window))
	for day := time.Date(window.Start.Year(), window.Start.Month(), window.Start.Day(), 0, 0, 0, 0, window.Start.Location()); !day.After(window.End); day = day.AddDate(0, 0, 1) {
		days = append(days, day.Format("2006-01-02"))
	}
	rows := make([][]string, 0, len(days))
	name := legacyParamString(params, "edc_name")
	for _, day := range days {
		raw := legacyNth95(byDay[day], rankIndex)
		rows = append(rows, []string{day, name, instance, legacyFormatted3(raw), legacyFormatted3(raw * 8 / 300 / float64(unitBase) / float64(unitBase)), strconv.Itoa(len(byDay[day])), "", ""})
	}
	return rows
}

func legacyEDCDailyTable(rows [][]string) legacyNativeTable {
	return legacyNativeTable{Columns: []string{"date", "edc_name", "data_source_instance", "daily_95th_percentile_raw", "daily_95th_percentile_mbps", "data_points_daily", "saler_group", "saler"}, Rows: rows}
}

func legacyEDCSummaryTable(daily [][]string, points []legacyEDCPoint, rankIndex, unitBase int, settlementMode string, params map[string]interface{}, instance string) legacyNativeTable {
	name := legacyParamString(params, "edc_name")
	raw := 0.0
	dataPoints := len(points)
	if settlementMode == "daily_95_avg" {
		for _, row := range daily {
			v, _ := strconv.ParseFloat(row[3], 64)
			raw += v
		}
		if len(daily) > 0 {
			raw /= float64(legacyTotalDaysFromRows(daily, params))
		}
		dataPoints = len(daily)
	} else {
		values := make([]float64, 0, len(points))
		for _, point := range points {
			values = append(values, point.Raw)
		}
		raw = legacyNth95(values, rankIndex)
	}
	columns := []string{"edc_name", "data_source_instance", "95th_percentile_raw", "95th_percentile_mbps", "settlement_mode", "data_points"}
	row := []string{name, instance, legacyFormatted3(raw), legacyFormatted3(raw * 8 / 300 / float64(unitBase) / float64(unitBase)), settlementMode, strconv.Itoa(dataPoints)}
	return legacyNativeTable{Columns: columns, Rows: [][]string{row}}
}

func legacyTotalDaysFromRows(rows [][]string, params map[string]interface{}) int {
	if len(rows) == 0 {
		return 1
	}
	// The row set contains one row per day. The old implementation divides by
	// the full requested window, including days with no data; callers pass the
	// exact row count only for the common complete daily export path.
	if n := legacyParamInt(params, "_legacy_total_days", 0); n > 0 {
		return n
	}
	return len(rows)
}

func legacyEDCMonthlyTable(daily [][]string, points []legacyEDCPoint, window trafficReportWindow, rankIndex, unitBase int, settlementMode string, params map[string]interface{}, instance string) legacyNativeTable {
	name := legacyParamString(params, "edc_name")
	months := map[string][]legacyEDCPoint{}
	for _, point := range points {
		months[point.At.In(window.Start.Location()).Format("2006-01")] = append(months[point.At.In(window.Start.Location()).Format("2006-01")], point)
	}
	monthNames := make([]string, 0, len(months))
	for month := range months {
		monthNames = append(monthNames, month)
	}
	sort.Strings(monthNames)
	rows := make([][]string, 0, len(monthNames))
	for _, month := range monthNames {
		monthPoints := months[month]
		raw := 0.0
		dataPoints := len(monthPoints)
		if settlementMode == "daily_95_avg" {
			vals := make([]float64, 0)
			for _, row := range daily {
				if strings.HasPrefix(row[0], month) {
					v, _ := strconv.ParseFloat(row[3], 64)
					vals = append(vals, v)
				}
			}
			for _, v := range vals {
				raw += v
			}
			if len(vals) > 0 {
				raw /= float64(len(vals))
			}
			dataPoints = len(vals)
		} else {
			vals := make([]float64, 0, len(monthPoints))
			for _, point := range monthPoints {
				vals = append(vals, point.Raw)
			}
			raw = legacyNth95(vals, rankIndex)
		}
		rows = append(rows, []string{month, name, instance, legacyFormatted3(raw), legacyFormatted3(raw * 8 / 300 / float64(unitBase) / float64(unitBase)), settlementMode, strconv.Itoa(dataPoints)})
	}
	return legacyNativeTable{Columns: []string{"month", "edc_name", "data_source_instance", "95th_percentile_raw", "95th_percentile_mbps", "settlement_mode", "data_points"}, Rows: rows}
}

func (s *trafficReportService) writeLegacyNativeTables(run *model.TrafficReportRun, baseName string, tables []legacyNativeTable, formats []string) (int64, error) {
	baseName = legacySafeArtifactName(baseName)
	root := filepath.Join(config.GetTrafficReportStorageDir(), run.ID)
	if err := os.MkdirAll(root, 0o750); err != nil {
		return 0, err
	}
	var rowCount int64
	for _, table := range tables {
		rowCount += int64(len(table.Rows))
		for _, format := range formats {
			var path string
			var mediaType string
			switch strings.ToLower(format) {
			case "xlsx":
				path = filepath.Join(root, baseName+".xlsx")
				mediaType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
				if err := writeLegacyNativeXLSX(path, table); err != nil {
					return 0, err
				}
			default:
				path = filepath.Join(root, baseName+".csv")
				mediaType = "text/csv; charset=utf-8"
				if err := writeLegacyNativeCSV(path, table); err != nil {
					return 0, err
				}
			}
			st, err := os.Stat(path)
			if err != nil {
				return 0, err
			}
			if err := s.repo.CreateArtifact(&model.TrafficReportArtifact{RunID: run.ID, FileName: filepath.Base(path), MediaType: mediaType, StoragePath: path, FileSize: st.Size()}); err != nil {
				return 0, err
			}
		}
	}
	return rowCount, nil
}

func writeLegacyNativeCSV(path string, table legacyNativeTable) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o640)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.Write([]byte{0xef, 0xbb, 0xbf}); err != nil {
		return err
	}
	w := csv.NewWriter(f)
	if err := w.Write(table.Columns); err != nil {
		return err
	}
	for _, row := range table.Rows {
		if err := w.Write(row); err != nil {
			return err
		}
	}
	w.Flush()
	return w.Error()
}

func writeLegacyNativeXLSX(path string, table legacyNativeTable) error {
	f := excelize.NewFile()
	defer f.Close()
	for c, column := range table.Columns {
		cell, _ := excelize.CoordinatesToCellName(c+1, 1)
		_ = f.SetCellValue("Sheet1", cell, column)
	}
	for r, row := range table.Rows {
		for c, value := range row {
			cell, _ := excelize.CoordinatesToCellName(c+1, r+2)
			if value == "" {
				continue
			}
			if table.Columns[c] == "create_time" {
				if parsed, err := time.ParseInLocation("2006-01-02 15:04:05", value, time.Local); err == nil {
					_ = f.SetCellValue("Sheet1", cell, parsed)
					continue
				}
			}
			if legacyXLSXNumericColumn(table.Columns[c]) {
				if n, err := strconv.ParseFloat(value, 64); err == nil {
					if strings.Contains(table.Columns[c], "mbps") {
						n, _ = strconv.ParseFloat(strconv.FormatFloat(n, 'g', 16, 64), 64)
					}
					_ = f.SetCellValue("Sheet1", cell, n)
					continue
				}
			}
			_ = f.SetCellValue("Sheet1", cell, value)
		}
	}
	return f.SaveAs(path)
}

func legacyXLSXNumericColumn(column string) bool {
	return strings.Contains(column, "service_size") || column == "data_points" || column == "data_points_daily" || column == "unit_base"
}

func legacySafeArtifactName(value string) string {
	return strings.TrimRight(strings.Map(func(r rune) rune {
		switch r {
		case '<', '>', ':', '"', '/', '\\', '|', '?', '*':
			return '_'
		default:
			if r < 0x20 {
				return '_'
			}
			return r
		}
	}, value), " .")
}
