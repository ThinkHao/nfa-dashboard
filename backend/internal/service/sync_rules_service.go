package service

import (
    "encoding/json"
    "strings"
    "nfa-dashboard/internal/model"
    "nfa-dashboard/internal/repository"
)

// SyncRulesService 定义同步规则的业务接口

type SyncRulesService interface {
    List(name string, enabled *bool, page, pageSize int) ([]model.RateCustomerSyncRule, int64, error)
    Create(rule *model.RateCustomerSyncRule) (*model.RateCustomerSyncRule, error)
    Update(id uint64, updates map[string]interface{}) error
    Delete(id uint64) error
    UpdatePriority(id uint64, priority int) error
    SetEnabled(id uint64, enabled bool) error
}

type syncRulesService struct{ repo repository.SyncRulesRepository }

func NewSyncRulesService(repo repository.SyncRulesRepository) SyncRulesService {
    return &syncRulesService{repo: repo}
}

func (s *syncRulesService) List(name string, enabled *bool, page, pageSize int) ([]model.RateCustomerSyncRule, int64, error) {
    if page <= 0 { page = 1 }
    if pageSize <= 0 { pageSize = 10 }
    filter := map[string]interface{}{}
    if name != "" { filter["name"] = name }
    if enabled != nil { filter["enabled"] = *enabled }
    limit := pageSize
    offset := (page - 1) * pageSize
    return s.repo.List(filter, limit, offset)
}

func (s *syncRulesService) Create(rule *model.RateCustomerSyncRule) (*model.RateCustomerSyncRule, error) {
    if rule == nil { return nil, NewBadRequest("nil rule") }
    // 归一化
    rule.Name = strings.TrimSpace(rule.Name)
    rule.OverwriteStrategy = strings.ToLower(strings.TrimSpace(rule.OverwriteStrategy))
    if rule.Name == "" { return nil, NewBadRequest("name is required") }
    if !isValidOverwriteStrategy(rule.OverwriteStrategy) { return nil, NewBadRequest("invalid overwrite_strategy") }
    // JSON 字段校验
    if err := validateStringArrayJSON(rule.ScopeRegion, true); err != nil { return nil, err }
    if err := validateStringArrayJSON(rule.ScopeCP, true); err != nil { return nil, err }
    if err := validateStringArrayJSON(rule.FieldsToUpdate, true); err != nil { return nil, err }
    if len(rule.Actions) == 0 { return nil, NewBadRequest("actions is required and must be valid JSON") }
    if !json.Valid(rule.Actions) { return nil, NewBadRequest("actions must be valid JSON") }
    return s.repo.Create(rule)
}

func (s *syncRulesService) Update(id uint64, updates map[string]interface{}) error {
    if id == 0 { return NewBadRequest("invalid id") }
    if len(updates) == 0 { return NewBadRequest("no fields to update") }
    // 禁止通过通用更新修改 enabled 与 priority（仓储层已保护，这里再次拦截）
    if _, ok := updates["enabled"]; ok { return NewBadRequest("enabled cannot be updated here; use SetEnabled") }
    if _, ok := updates["priority"]; ok { return NewBadRequest("priority cannot be updated here; use UpdatePriority") }
    // 归一化与校验
    if v, ok := updates["name"]; ok {
        if v == nil { return NewBadRequest("name cannot be null") }
        name := strings.TrimSpace(v.(string))
        if name == "" { return NewBadRequest("name cannot be empty") }
        updates["name"] = name
    }
    if v, ok := updates["overwrite_strategy"]; ok {
        if v == nil { return NewBadRequest("overwrite_strategy cannot be null") }
        srt := strings.ToLower(strings.TrimSpace(v.(string)))
        if !isValidOverwriteStrategy(srt) { return NewBadRequest("invalid overwrite_strategy") }
        updates["overwrite_strategy"] = srt
    }
    if v, ok := updates["scope_region"]; ok {
        if v == nil { return NewBadRequest("scope_region cannot be null") }
        if err := validateStringArrayJSONInterface(v, true); err != nil { return err }
    }
    if v, ok := updates["scope_cp"]; ok {
        if v == nil { return NewBadRequest("scope_cp cannot be null") }
        if err := validateStringArrayJSONInterface(v, true); err != nil { return err }
    }
    if v, ok := updates["fields_to_update"]; ok {
        if v == nil { return NewBadRequest("fields_to_update cannot be null") }
        if err := validateStringArrayJSONInterface(v, true); err != nil { return err }
        // 进一步校验 field_key 形式
        if arr, ok2 := mustParseStringArrayInterface(v); ok2 {
            for _, fk := range arr {
                if !isValidFieldKey(fk) { return NewBadRequest("fields_to_update contains invalid field_key") }
            }
        }
    }
    if v, ok := updates["actions"]; ok {
        if v == nil { return NewBadRequest("actions cannot be null") }
        // 接受任意合法 JSON，但不能为空对象/数组
        var any interface{}
        bs, ok2 := toJSONBytes(v)
        if !ok2 || !json.Valid(bs) { return NewBadRequest("actions must be valid JSON") }
        if err := json.Unmarshal(bs, &any); err != nil { return NewBadRequest("actions must be valid JSON") }
        // 简单非空检查
        if isEmptyJSON(any) { return NewBadRequest("actions cannot be empty") }
    }
    return s.repo.Update(id, updates)
}

func (s *syncRulesService) Delete(id uint64) error {
    if id == 0 { return NewBadRequest("invalid id") }
    return s.repo.Delete(id)
}

func (s *syncRulesService) UpdatePriority(id uint64, priority int) error {
    if id == 0 { return NewBadRequest("invalid id") }
    if priority < 0 { return NewBadRequest("priority must be >= 0") }
    return s.repo.UpdatePriority(id, priority)
}

func (s *syncRulesService) SetEnabled(id uint64, enabled bool) error {
    if id == 0 { return NewBadRequest("invalid id") }
    return s.repo.SetEnabled(id, enabled)
}

// -------------------- 校验辅助 --------------------

func isValidOverwriteStrategy(s string) bool {
    switch s {
    case "always", "if_empty":
        return true
    default:
        return false
    }
}

// rule JSON 校验：要求是字符串数组（允许空数组，取决于 allowEmpty）
func validateStringArrayJSON(data []byte, allowEmpty bool) error {
    if len(data) == 0 { return nil }
    var arr []string
    if err := json.Unmarshal(data, &arr); err != nil { return NewBadRequest("must be JSON array of strings") }
    if !allowEmpty && len(arr) == 0 { return NewBadRequest("array cannot be empty") }
    return nil
}

func validateStringArrayJSONInterface(v interface{}, allowEmpty bool) error {
    bs, ok := toJSONBytes(v)
    if !ok { return NewBadRequest("must be valid JSON") }
    return validateStringArrayJSON(bs, allowEmpty)
}

func mustParseStringArrayInterface(v interface{}) ([]string, bool) {
    bs, ok := toJSONBytes(v)
    if !ok { return nil, false }
    var arr []string
    if err := json.Unmarshal(bs, &arr); err != nil { return nil, false }
    return arr, true
}

func toJSONBytes(v interface{}) ([]byte, bool) {
    switch t := v.(type) {
    case []byte:
        return t, true
    case json.RawMessage:
        return []byte(t), true
    case string:
        return []byte(t), true
    default:
        bs, err := json.Marshal(v)
        if err != nil { return nil, false }
        return bs, true
    }
}

func isEmptyJSON(v interface{}) bool {
    switch t := v.(type) {
    case map[string]interface{}:
        return len(t) == 0
    case []interface{}:
        return len(t) == 0
    default:
        return false
    }
}

