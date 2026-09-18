package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"nfa-dashboard/internal/model"
)

func cloneJSONMap(raw []byte) map[string]interface{} {
	out := map[string]interface{}{}
	if len(raw) == 0 {
		return out
	}
	if err := json.Unmarshal(raw, &out); err != nil || out == nil {
		return map[string]interface{}{}
	}
	return out
}

func migrateLegacyParamsToGoV1(source string, raw map[string]interface{}) (map[string]interface{}, []string, error) {
	params := make(map[string]interface{})
	warnings := make([]string, 0, 3)
	copyParam := func(key string) {
		if value, ok := raw[key]; ok && value != nil {
			params[key] = value
		}
	}
	copyAlias := func(target string, keys ...string) {
		for _, key := range keys {
			if value, ok := raw[key]; ok && value != nil && strings.TrimSpace(fmt.Sprint(value)) != "" {
				params[target] = value
				return
			}
		}
	}

	for _, key := range []string{"direction", "unit_base", "settlement_mode", "data_budget_enabled", "data_budget_mul", "data_budget_div"} {
		copyParam(key)
	}

	switch source {
	case "nfa":
		copyParam("school_name")
		copyParam("school_names")
		copyAlias("school_name", "school")
		copyAlias("region", "region", "province")
		copyParam("cp")
		if legacyParamBool(raw, "export_daily") || legacyParamBool(raw, "monthly_aggregate") {
			return nil, nil, errors.New("该 NFA 任务使用日汇总或月汇总导出，当前不能自动迁移为 go-v1")
		}
		if strings.TrimSpace(legacyParamString(raw, "merge_key")) != "" {
			return nil, nil, errors.New("该 NFA 任务使用 merge_key，当前不能自动迁移为 go-v1")
		}
		if legacyParamBool(raw, "aggregate_all") {
			return nil, nil, errors.New("该 NFA 任务使用 aggregate_all，当前不能自动迁移为 go-v1")
		}
		if legacyParamBool(raw, "combine_v4_v6") {
			warnings = append(warnings, "原任务启用了 V4/V6 合并；请重点核对 go-v1 的学校汇总结果")
		}
	case "edc":
		for _, key := range []string{"entity_ids", "entity_type", "src_region", "dst_region", "region", "cp"} {
			copyParam(key)
		}
		if legacyParamBool(raw, "export_raw") || legacyParamBool(raw, "export_daily") || legacyParamBool(raw, "monthly_aggregate") {
			return nil, nil, errors.New("该 EDC 任务使用原始或日/月汇总导出，当前不能自动迁移为 go-v1")
		}
		if legacyParamBool(raw, "aggregate_all") {
			return nil, nil, errors.New("该 EDC 任务使用 aggregate_all，当前不能自动迁移为 go-v1")
		}
		if len(asUint64Slice(params["entity_ids"])) == 0 {
			requested := splitMigrationNames(legacyParamString(raw, "edc_name"))
			mode := strings.ToLower(strings.TrimSpace(legacyParamString(raw, "edc_match_mode")))
			if len(requested) == 0 {
				return nil, nil, errors.New("该 EDC 任务缺少 entity_ids 或 edc_name，无法安全迁移")
			}
			if mode != "exact" {
				return nil, nil, errors.New("该 EDC 任务不是精确名称匹配，无法安全转换为 entity_ids")
			}
			ids, err := resolveMigrationEDCEntityIDs(requested)
			if err != nil {
				return nil, nil, err
			}
			params["entity_ids"] = ids
			warnings = append(warnings, "原任务的 EDC 名称筛选已转换为当前平台 entity_ids")
		}
		if instance := legacyParamString(raw, "data_source_instance"); instance != "" {
			warnings = append(warnings, "原任务的数据源实例参数不会带入 go-v1，迁移任务使用当前平台标准化数据")
		}
	default:
		return nil, nil, errors.New("不支持的数据源类型")
	}
	return params, warnings, nil
}

func splitMigrationNames(value string) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0)
	for _, item := range strings.FieldsFunc(value, func(r rune) bool { return r == ',' || r == '\n' }) {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		result = append(result, item)
	}
	return result
}

func resolveMigrationEDCEntityIDs(names []string) ([]uint64, error) {
	var entities []model.EDCEntity
	if err := model.DB.Where("enabled = ? AND is_backup = ?", true, false).Where("edc_name IN ?", names).Find(&entities).Error; err != nil {
		return nil, fmt.Errorf("查询当前平台 EDC 实体失败: %w", err)
	}
	found := map[string]bool{}
	ids := make([]uint64, 0, len(entities))
	for _, entity := range entities {
		found[entity.EDCName] = true
		ids = append(ids, entity.ID)
	}
	missing := make([]string, 0)
	for _, name := range names {
		if !found[name] {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("当前平台找不到启用且非备份的 EDC 实体: %s", strings.Join(missing, ", "))
	}
	if len(ids) == 0 {
		return nil, errors.New("未解析出可迁移的 EDC 实体")
	}
	return ids, nil
}

func asUint64Slice(value interface{}) []uint64 {
	items, ok := value.([]interface{})
	if !ok {
		if typed, ok := value.([]uint64); ok {
			return typed
		}
		return nil
	}
	result := make([]uint64, 0, len(items))
	for _, item := range items {
		switch v := item.(type) {
		case float64:
			if v > 0 && v == float64(uint64(v)) {
				result = append(result, uint64(v))
			}
		case json.Number:
			if n, err := v.Int64(); err == nil && n > 0 {
				result = append(result, uint64(n))
			}
		case uint64:
			if v > 0 {
				result = append(result, v)
			}
		}
	}
	return result
}
