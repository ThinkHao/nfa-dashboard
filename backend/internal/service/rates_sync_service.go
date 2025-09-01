package service

import (
    "encoding/json"
    "errors"
    "regexp"
    "time"
    "nfa-dashboard/internal/model"
    "nfa-dashboard/internal/repository"
)

// RatesSyncService 执行“客户费率同步”任务（从学校管理拉取 + 规则应用）
// 先提供未实现存根，后续补齐逻辑

type RatesSyncService interface {
    ExecuteSync() (int64, error)
}

type ratesSyncService struct{
    rulesRepo repository.SyncRulesRepository
    ratesRepo repository.RatesRepository
}

func NewRatesSyncService(rulesRepo repository.SyncRulesRepository, ratesRepo repository.RatesRepository) RatesSyncService {
    return &ratesSyncService{rulesRepo: rulesRepo, ratesRepo: ratesRepo}
}

func (s *ratesSyncService) ExecuteSync() (int64, error) {
    if s == nil || s.rulesRepo == nil || s.ratesRepo == nil {
        return 0, errors.New("service not properly initialized")
    }
    // 1) 读取启用的规则，按优先级升序
    rules, _, err := s.rulesRepo.List(map[string]interface{}{"enabled": true}, 0, 0)
    if err != nil { return 0, err }
    if len(rules) == 0 { return 0, nil }

    var totalAffected int64
    now := time.Now()

    for _, rule := range rules {
        // 2) 解析范围与字段限制、动作
        regions, _ := parseStringArray(rule.ScopeRegion)
        cps, _ := parseStringArray(rule.ScopeCP)
        whitelist, _ := parseStringArray(rule.FieldsToUpdate)
        setMap, _ := parseActionsSet(rule.Actions)

        // 如果 set 动作为空，则跳过该规则
        if len(setMap) == 0 { continue }

        // 3) 遍历 region/cp 组合（为空表示全量）
        regionList := regions
        cpList := cps
        if len(regionList) == 0 { regionList = []string{"*"} }
        if len(cpList) == 0 { cpList = []string{"*"} }

        for _, region := range regionList {
            for _, cp := range cpList {
                filter := map[string]interface{}{}
                if region != "*" { filter["region"] = region }
                if cp != "*" { filter["cp"] = cp }
                // 分页拉取，避免一次性过大（简单实现：单次拉取 500 条）
                const pageSize = 500
                page := 1
                for {
                    items, count, err := s.ratesRepo.ListCustomerRates(filter, pageSize, (page-1)*pageSize)
                    if err != nil { return totalAffected, err }
                    if len(items) == 0 { break }

                    for i := range items {
                        updated, err := s.applyRuleToCustomer(&items[i], rule, whitelist, setMap)
                        if err != nil { return totalAffected, err }
                        if updated {
                            updates := map[string]interface{}{
                                "extra":            items[i].Extra,
                                "last_sync_time":   now,
                                "last_sync_rule_id": rule.ID,
                            }
                            if err := s.ratesRepo.UpdateCustomerByID(items[i].ID, updates); err != nil { return totalAffected, err }
                            totalAffected++
                        }
                    }

                    // 翻页终止条件
                    if int64(page*pageSize) >= count { break }
                    page++
                }
            }
        }
    }

    return totalAffected, nil
}

// 将规则应用到单个客户费率，返回是否发生更新
func (s *ratesSyncService) applyRuleToCustomer(rc *model.RateCustomer, rule model.RateCustomerSyncRule, whitelist []string, setMap map[string]interface{}) (bool, error) {
    // 条件表达式暂未实现，如需后续扩展，在此处处理 rule.ConditionExpr

    // 解析现有 extra
    cur := map[string]interface{}{}
    if len(rc.Extra) > 0 {
        _ = json.Unmarshal(rc.Extra, &cur)
        if cur == nil { cur = map[string]interface{}{} }
    }

    // 生成允许更新的字段集合
    allowed := map[string]struct{}{}
    if len(whitelist) > 0 {
        for _, k := range whitelist { if isValidFieldKeyLocal(k) { allowed[k] = struct{}{} } }
    }

    // 根据覆盖策略进行变更计算
    changed := false
    for k, v := range setMap {
        if !isValidFieldKeyLocal(k) { continue }
        if len(allowed) > 0 {
            if _, ok := allowed[k]; !ok { continue }
        }
        switch rule.OverwriteStrategy {
        case "always":
            if old, ok := cur[k]; !ok || !jsonEqual(old, v) {
                cur[k] = v
                changed = true
            }
        case "if_empty":
            if old, ok := cur[k]; !ok || isEmptyValue(old) {
                cur[k] = v
                changed = true
            }
        default:
            // 未知策略则跳过该字段
        }
    }

    if !changed { return false, nil }
    bs, err := json.Marshal(cur)
    if err != nil { return false, err }
    rc.Extra = bs
    return true, nil
}

func parseStringArray(data []byte) ([]string, error) {
    if len(data) == 0 { return nil, nil }
    var arr []string
    if err := json.Unmarshal(data, &arr); err != nil { return nil, err }
    return arr, nil
}

// 目前仅支持 {"set": {"field_key": any, ...}}
func parseActionsSet(data []byte) (map[string]interface{}, error) {
    if len(data) == 0 { return map[string]interface{}{}, nil }
    var obj map[string]interface{}
    if err := json.Unmarshal(data, &obj); err != nil { return nil, err }
    raw, ok := obj["set"]
    if !ok || raw == nil { return map[string]interface{}{}, nil }
    m, ok := raw.(map[string]interface{})
    if !ok { return map[string]interface{}{}, nil }
    return m, nil
}

func isValidFieldKeyLocal(s string) bool {
    re := regexp.MustCompile(`^[a-z][a-z0-9_]{1,63}$`)
    return re.MatchString(s)
}

func isEmptyValue(v interface{}) bool {
    if v == nil { return true }
    switch t := v.(type) {
    case string:
        return t == ""
    default:
        return false
    }
}

func jsonEqual(a, b interface{}) bool {
    ba, _ := json.Marshal(a)
    bb, _ := json.Marshal(b)
    if len(ba) != len(bb) { return false }
    for i := range ba { if ba[i] != bb[i] { return false } }
    return true
}
