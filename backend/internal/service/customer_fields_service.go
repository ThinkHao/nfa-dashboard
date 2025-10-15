package service

import (
    "encoding/json"
    "math"
    "regexp"
    "strings"
    "nfa-dashboard/internal/model"
    "nfa-dashboard/internal/repository"
    "gorm.io/datatypes"
)

// CustomerFieldsService 定义自定义字段定义的业务接口

type CustomerFieldsService interface {
    List(fieldKey, label, dataType string, enabled *bool, page, pageSize int) ([]model.RateCustomerCustomFieldDef, int64, error)
    Create(def *model.RateCustomerCustomFieldDef) (*model.RateCustomerCustomFieldDef, error)
    Update(id uint64, updates map[string]interface{}) error
    Delete(id uint64) error
}

type customerFieldsService struct{ repo repository.CustomerFieldsRepository }

func NewCustomerFieldsService(repo repository.CustomerFieldsRepository) CustomerFieldsService {
    return &customerFieldsService{repo: repo}
}

func (s *customerFieldsService) List(fieldKey, label, dataType string, enabled *bool, page, pageSize int) ([]model.RateCustomerCustomFieldDef, int64, error) {
    if page <= 0 { page = 1 }
    if pageSize <= 0 { pageSize = 10 }
    filter := map[string]interface{}{}
    if fieldKey != "" { filter["field_key"] = fieldKey }
    if label != "" { filter["label"] = label }
    if dataType != "" { filter["data_type"] = dataType }
    if enabled != nil { filter["enabled"] = *enabled }
    limit := pageSize
    offset := (page - 1) * pageSize
    return s.repo.List(filter, limit, offset)
}

func (s *customerFieldsService) Create(def *model.RateCustomerCustomFieldDef) (*model.RateCustomerCustomFieldDef, error) {
    if def == nil { return nil, NewBadRequest("nil def") }
    // 归一化
    def.FieldKey = strings.TrimSpace(def.FieldKey)
    def.DataType = strings.ToLower(strings.TrimSpace(def.DataType))
    if def.Label = strings.TrimSpace(def.Label); def.Label == "" { return nil, NewBadRequest("label is required") }
    // 基本必填
    if def.FieldKey == "" { return nil, NewBadRequest("field_key is required") }
    if def.DataType == "" { return nil, NewBadRequest("data_type is required") }
    // 命名规范与唯一性
    if !isValidFieldKey(def.FieldKey) { return nil, NewBadRequest("field_key must match ^[a-z][a-z0-9_]{1,63}$") }
    // 唯一性
    if exists, err := s.repo.ExistsByFieldKey(def.FieldKey); err != nil {
        return nil, err
    } else if exists {
        return nil, NewBadRequest("field_key already exists")
    }
    // 数据类型校验 + 组合一致性
    if err := validateFieldDefConsistency(def.DataType, def.Required, def.DefaultValue, def.ValidateRegex, def.Min, def.Max, def.Precision, def.EnumOptions); err != nil {
        if IsBadRequest(err) { return nil, err }
        return nil, err
    }
    return s.repo.Create(def)
}

func (s *customerFieldsService) Update(id uint64, updates map[string]interface{}) error {
    if id == 0 { return NewBadRequest("invalid id") }
    if len(updates) == 0 { return NewBadRequest("no fields to update") }
    // 拉取原记录，合并新值后做一致性校验
    old, err := s.repo.GetByID(id)
    if err != nil { return err }
    eff := *old // 拷贝一份
    // 不允许修改 field_key（仓储已保护，服务层亦拦截）
    if _, ok := updates["field_key"]; ok { return NewBadRequest("field_key cannot be updated") }
    // 应用更新（仅允许存在字段）
    if v, ok := updates["label"]; ok {
        if v == nil { return NewBadRequest("label cannot be null") }
        s := strings.TrimSpace(v.(string))
        if s == "" { return NewBadRequest("label cannot be empty") }
        eff.Label = s
    }
    if v, ok := updates["data_type"]; ok {
        if v == nil { return NewBadRequest("data_type cannot be null") }
        sdt := strings.ToLower(strings.TrimSpace(v.(string)))
        eff.DataType = sdt
    }
    if v, ok := updates["required"]; ok {
        if v == nil { return NewBadRequest("required cannot be null") }
        eff.Required = v.(bool)
    }
    if v, ok := updates["default_value"]; ok {
        if v == nil {
            eff.DefaultValue = nil
        } else if b, ok2 := v.([]byte); ok2 {
            eff.DefaultValue = datatypes.JSON(b)
        } else if js, ok2 := v.(json.RawMessage); ok2 {
            eff.DefaultValue = datatypes.JSON(js)
        } else {
            // 最后尝试将 interface{} 序列化
            buf, mErr := json.Marshal(v)
            if mErr != nil { return NewBadRequest("default_value invalid json") }
            eff.DefaultValue = datatypes.JSON(buf)
        }
    }
    if v, ok := updates["validate_regex"]; ok {
        if v == nil { eff.ValidateRegex = nil } else { s := v.(string); eff.ValidateRegex = &s }
    }
    if v, ok := updates["min"]; ok {
        if v == nil { eff.Min = nil } else { f := v.(float64); eff.Min = &f }
    }
    if v, ok := updates["max"]; ok {
        if v == nil { eff.Max = nil } else { f := v.(float64); eff.Max = &f }
    }
    if v, ok := updates["precision"]; ok {
        if v == nil {
            eff.Precision = nil
        } else if f, ok2 := v.(float64); ok2 {
            p := int(f)
            eff.Precision = &p
        } else if i, ok2 := v.(int); ok2 {
            p := int(i)
            eff.Precision = &p
        } else {
            return NewBadRequest("precision invalid type")
        }
    }
    if v, ok := updates["enum_options"]; ok {
        if v == nil {
            eff.EnumOptions = nil
        } else if b, ok2 := v.([]byte); ok2 {
            eff.EnumOptions = datatypes.JSON(b)
        } else if js, ok2 := v.(json.RawMessage); ok2 {
            eff.EnumOptions = datatypes.JSON(js)
        } else {
            buf, mErr := json.Marshal(v)
            if mErr != nil { return NewBadRequest("enum_options invalid json") }
            eff.EnumOptions = datatypes.JSON(buf)
        }
    }
    if v, ok := updates["usable_in_rules"]; ok {
        if v == nil { return NewBadRequest("usable_in_rules cannot be null") }
        eff.UsableInRules = v.(bool)
    }
    if v, ok := updates["enabled"]; ok {
        if v == nil { return NewBadRequest("enabled cannot be null") }
        eff.Enabled = v.(bool)
    }
    // 归一化
    eff.DataType = strings.ToLower(strings.TrimSpace(eff.DataType))
    // 一致性校验
    if !isValidDataType(eff.DataType) { return NewBadRequest("invalid data_type") }
    if err := validateFieldDefConsistency(eff.DataType, eff.Required, eff.DefaultValue, eff.ValidateRegex, eff.Min, eff.Max, eff.Precision, eff.EnumOptions); err != nil {
        if IsBadRequest(err) { return err }
        return err
    }
    return s.repo.Update(id, updates)
}

func (s *customerFieldsService) Delete(id uint64) error {
    if id == 0 { return NewBadRequest("invalid id") }
    return s.repo.Delete(id)
}

// -------------------- 校验辅助 --------------------

var (
    fieldKeyRe = regexp.MustCompile(`^[a-z][a-z0-9_]{1,63}$`)
)

func isValidFieldKey(s string) bool { return fieldKeyRe.MatchString(s) }

func isValidDataType(dt string) bool {
    switch dt {
    case "string", "number", "integer", "boolean":
        return true
    default:
        return false
    }
}

func validateFieldDefConsistency(dataType string, required bool, defaultJSON datatypes.JSON, validateRegex *string, min, max *float64, precision *int, enumJSON datatypes.JSON) error {
    // data_type 合法
    if !isValidDataType(dataType) { return NewBadRequest("invalid data_type") }
    // precision 校验（仅 number 有意义，integer 只能为0 或空）
    if precision != nil {
        if *precision < 0 || *precision > 10 { return NewBadRequest("precision must be between 0 and 10") }
        if dataType == "integer" && *precision != 0 { return NewBadRequest("integer precision must be 0") }
    }
    // min/max 仅对 number/integer 有意义
    if (min != nil || max != nil) && !(dataType == "number" || dataType == "integer") {
        return NewBadRequest("min/max only allowed for number/integer")
    }
    if min != nil && max != nil && *min > *max { return NewBadRequest("min cannot be greater than max") }
    // 校验 regex（仅 string）
    if validateRegex != nil {
        if dataType != "string" { return NewBadRequest("validate_regex only allowed for string type") }
        if _, err := regexp.Compile(*validateRegex); err != nil { return NewBadRequest("validate_regex is invalid") }
    }
    // 解析 enum_options（如提供）
    var enumVals []interface{}
    if len(enumJSON) > 0 {
        if err := json.Unmarshal(enumJSON, &enumVals); err != nil {
            return NewBadRequest("enum_options must be a JSON array")
        }
        // 元素类型一致性
        for _, v := range enumVals {
            if !isValueTypeMatches(v, dataType) { return NewBadRequest("enum_options elements type mismatch with data_type") }
        }
        // 去重
        if hasDup(enumVals) { return NewBadRequest("enum_options has duplicate values") }
    }
    // 默认值校验
    if len(defaultJSON) > 0 {
        var dv interface{}
        if err := json.Unmarshal(defaultJSON, &dv); err != nil { return NewBadRequest("default_value is not valid JSON") }
        if !isValueTypeMatches(dv, dataType) { return NewBadRequest("default_value type mismatch with data_type") }
        // 正则
        if dataType == "string" && validateRegex != nil {
            re, _ := regexp.Compile(*validateRegex)
            if s, ok := dv.(string); ok {
                if !re.MatchString(s) { return NewBadRequest("default_value does not match validate_regex") }
            }
        }
        // 范围
        if dataType == "number" || dataType == "integer" {
            f, _ := dv.(float64)
            if min != nil && f < *min { return NewBadRequest("default_value less than min") }
            if max != nil && f > *max { return NewBadRequest("default_value greater than max") }
            if dataType == "integer" && math.Trunc(f) != f { return NewBadRequest("default_value must be integer") }
        }
        // 枚举
        if len(enumVals) > 0 && !inArray(enumVals, dv) { return NewBadRequest("default_value not in enum_options") }
    }
    // required 本身无额外限制（是否提供默认值由业务决定，允许 required=true 且无默认值）
    return nil
}

func isValueTypeMatches(v interface{}, dataType string) bool {
    switch dataType {
    case "string":
        _, ok := v.(string); return ok
    case "boolean":
        _, ok := v.(bool); return ok
    case "integer":
        // JSON 数字默认是 float64，需进一步判断是否整数
        f, ok := v.(float64); if !ok { return false }
        return math.Trunc(f) == f
    case "number":
        _, ok := v.(float64); return ok
    default:
        return false
    }
}

func hasDup(arr []interface{}) bool {
    seen := map[string]struct{}{}
    for _, v := range arr {
        b, _ := json.Marshal(v)
        k := string(b)
        if _, ok := seen[k]; ok { return true }
        seen[k] = struct{}{}
    }
    return false
}

func inArray(arr []interface{}, target interface{}) bool {
    tb, _ := json.Marshal(target)
    tk := string(tb)
    for _, v := range arr {
        vb, _ := json.Marshal(v)
        if string(vb) == tk { return true }
    }
    return false
}

