package service

import (
	"encoding/json"
	"errors"
	"log"
	"nfa-dashboard/internal/model"
	"nfa-dashboard/internal/repository"
	"regexp"
	"time"
)

// RatesSyncService 执行“客户费率同步”任务（从学校管理拉取 + 规则应用）
// 先提供未实现存根，后续补齐逻辑

type RatesSyncService interface {
	ExecuteSync() (int64, error)
}

type ratesSyncService struct {
	rulesRepo  repository.SyncRulesRepository
	ratesRepo  repository.RatesRepository
	schoolRepo repository.SchoolRepository
}

func NewRatesSyncService(rulesRepo repository.SyncRulesRepository, ratesRepo repository.RatesRepository, schoolRepo repository.SchoolRepository) RatesSyncService {
	return &ratesSyncService{rulesRepo: rulesRepo, ratesRepo: ratesRepo, schoolRepo: schoolRepo}
}

func (s *ratesSyncService) ExecuteSync() (int64, error) {
	if s == nil || s.rulesRepo == nil || s.ratesRepo == nil || s.schoolRepo == nil {
		return 0, errors.New("service not properly initialized")
	}
	// 1) 读取启用的规则，按优先级升序
	rules, _, err := s.rulesRepo.List(map[string]interface{}{"enabled": true}, 0, 0)
	if err != nil {
		return 0, err
	}
	log.Printf("[rates-sync] loaded %d enabled rules", len(rules))
	if len(rules) == 0 {
		return 0, nil
	}

	var totalAffected int64
	now := time.Now()

	for _, rule := range rules {
		// 2) 解析范围与字段限制、动作
		regions, _ := parseStringArray(rule.ScopeRegion)
		cps, _ := parseStringArray(rule.ScopeCP)
		whitelist, _ := parseStringArray(rule.FieldsToUpdate)
		setMap, _ := parseActionsSet(rule.Actions)
		extraSet, _ := parseFieldsToUpdateExtra(rule.FieldsToUpdate)
		// 合并：actions 优先，fields_to_update.extra 作为补充
		if len(extraSet) > 0 {
			for k, v := range extraSet {
				if _, exists := setMap[k]; !exists {
					setMap[k] = v
				}
			}
		}

		// 如果 set 动作为空，则跳过该规则
		if len(setMap) == 0 {
			log.Printf("[rates-sync] rule skipped (no actions): id=%d name=%s", rule.ID, rule.Name)
			continue
		}

		// 日志：规则关键信息
		setKeys := make([]string, 0, len(setMap))
		for k := range setMap {
			setKeys = append(setKeys, k)
		}
		log.Printf("[rates-sync] rule begin: id=%d name=%s overwrite=%s regions=%v cps=%v whitelist=%v setKeys=%v extraFromFields=%v",
			rule.ID, rule.Name, rule.OverwriteStrategy, regions, cps, whitelist, setKeys, len(extraSet))

		// 3) 遍历 region/cp 组合（为空表示全量）
		regionList := regions
		cpList := cps
		if len(regionList) == 0 {
			regionList = []string{"*"}
		}
		if len(cpList) == 0 {
			cpList = []string{"*"}
		}

		for _, region := range regionList {
			for _, cp := range cpList {
				// 从 nfa_school 按 region/cp 分页拉取学校
				schoolFilter := map[string]interface{}{}
				if region != "*" {
					schoolFilter["region"] = region
				}
				if cp != "*" {
					schoolFilter["cp"] = cp
				}
				const pageSize = 500
				page := 1
				for {
					schools, count, err := s.schoolRepo.GetAllSchools(schoolFilter, pageSize, (page-1)*pageSize)
					if err != nil {
						return totalAffected, err
					}
					if len(schools) == 0 {
						break
					}
					log.Printf("[rates-sync] fetched schools: filter=%v page=%d size=%d got=%d total=%d", schoolFilter, page, pageSize, len(schools), count)

					for i := range schools {
						sch := schools[i]
						// 尝试查找已有的 rate_customer 记录
						rcFilter := map[string]interface{}{"region": sch.Region, "cp": sch.CP, "school_name": sch.SchoolName}
						existing, _, err := s.ratesRepo.ListCustomerRates(rcFilter, 1, 0)
						if err != nil {
							return totalAffected, err
						}

						var rc model.RateCustomer
						existed := false
						if len(existing) > 0 {
							rc = existing[0]
							existed = true
						} else {
							// 预构造一条新记录（空费率、空 extra）
							name := sch.SchoolName
							rc = model.RateCustomer{Region: sch.Region, CP: sch.CP, SchoolName: &name}
						}

						updated, fieldUpdates, err := s.applyRuleToCustomer(&rc, rule, whitelist, setMap)
						if err != nil {
							return totalAffected, err
						}
						if updated {
							updKeys := make([]string, 0, len(fieldUpdates))
							for k := range fieldUpdates {
								updKeys = append(updKeys, k)
							}
							log.Printf("[rates-sync] apply %s: region=%s cp=%s school=%s keys=%v", map[bool]string{true: "update", false: "insert"}[existed], rc.Region, rc.CP, derefString(rc.SchoolName), updKeys)
							if existed {
								if fieldUpdates == nil {
									fieldUpdates = map[string]interface{}{}
								}
								fieldUpdates["last_sync_time"] = now
								fieldUpdates["last_sync_rule_id"] = rule.ID
								if err := s.ratesRepo.UpdateCustomerByID(rc.ID, fieldUpdates); err != nil {
									return totalAffected, err
								}
							} else {
								// 新建记录：将同步信息写入结构体并 Upsert
								rid := rule.ID
								rc.LastSyncTime = &now
								rc.LastSyncRuleID = &rid
								if err := s.ratesRepo.UpsertCustomerRate(&rc); err != nil {
									return totalAffected, err
								}
							}
							totalAffected++
						}
					}

					if int64(page*pageSize) >= count {
						break
					}
					page++
				}
			}
		}
		log.Printf("[rates-sync] rule end: id=%d name=%s", rule.ID, rule.Name)
	}

	log.Printf("[rates-sync] all rules finished, totalAffected=%d", totalAffected)
	return totalAffected, nil
}

// 将规则应用到单个客户费率，返回是否发生更新以及需要持久化到 DB 的字段集合
func (s *ratesSyncService) applyRuleToCustomer(rc *model.RateCustomer, rule model.RateCustomerSyncRule, whitelist []string, setMap map[string]interface{}) (bool, map[string]interface{}, error) {
	// 条件表达式暂未实现，如需后续扩展，在此处处理 rule.ConditionExpr

	// 解析现有 extra
	cur := map[string]interface{}{}
	if len(rc.Extra) > 0 {
		_ = json.Unmarshal(rc.Extra, &cur)
		if cur == nil {
			cur = map[string]interface{}{}
		}
	}

	// 生成允许更新的字段集合
	allowed := map[string]struct{}{}
	if len(whitelist) > 0 {
		for _, k := range whitelist {
			if isValidFieldKeyLocal(k) {
				allowed[k] = struct{}{}
			}
		}
	}

	// 根据覆盖策略进行变更计算
	changed := false
	updates := map[string]interface{}{}
	// 顶层支持的费率字段（与前端模板字段一致）
	topFields := map[string]struct{}{"customer_fee": {}, "network_line_fee": {}, "general_fee": {}}

	for k, v := range setMap {
		if !isValidFieldKeyLocal(k) {
			continue
		}
		if len(allowed) > 0 {
			if _, ok := allowed[k]; !ok {
				continue
			}
		}
		// 如果是顶层费率字段，且该行处于手工模式，则跳过更新
		if _, isTop := topFields[k]; isTop {
			if rc.FeeMode == "configed" {
				// 保留人工配置的价格字段
				continue
			}
			// 仅接受数值；字符串尝试解析成 float64
			var f *float64
			switch t := v.(type) {
			case float64:
				f = &t
			case float32:
				x := float64(t)
				f = &x
			case int:
				x := float64(t)
				f = &x
			case int64:
				x := float64(t)
				f = &x
			case json.Number:
				if fv, err := t.Float64(); err == nil {
					f = &fv
				}
			case string:
				// 简单尝试解析
				if num, err := json.Number(t).Float64(); err == nil {
					f = &num
				}
			}
			// 计算覆盖策略
			var curPtr **float64
			switch k {
			case "customer_fee":
				curPtr = &rc.CustomerFee
			case "network_line_fee":
				curPtr = &rc.NetworkLineFee
			case "general_fee":
				curPtr = &rc.GeneralFee
			}
			if f != nil && curPtr != nil {
				switch rule.OverwriteStrategy {
				case "always":
					// 比较是否不同（nil 或 值不同）
					if *curPtr == nil || **curPtr != *f {
						*curPtr = f
						updates[k] = *f
						changed = true
						log.Printf("[rates-sync] field changed (always): id=%d key=%s new=%v", rc.ID, k, *f)
					}
				case "if_empty":
					if *curPtr == nil {
						*curPtr = f
						updates[k] = *f
						changed = true
						log.Printf("[rates-sync] field changed (if_empty): id=%d key=%s new=%v", rc.ID, k, *f)
					}
				}
			}
			continue
		}
		// 否则更新到 extra JSON
		switch rule.OverwriteStrategy {
		case "always":
			if old, ok := cur[k]; !ok || !jsonEqual(old, v) {
				cur[k] = v
				changed = true
				log.Printf("[rates-sync] extra changed (always): id=%d key=%s", rc.ID, k)
			}
		case "if_empty":
			if old, ok := cur[k]; !ok || isEmptyValue(old) {
				cur[k] = v
				changed = true
				log.Printf("[rates-sync] extra changed (if_empty): id=%d key=%s", rc.ID, k)
			}
		default:
			// 未知策略则跳过该字段
		}
	}

	if !changed {
		return false, nil, nil
	}
	bs, err := json.Marshal(cur)
	if err != nil {
		return false, nil, err
	}
	rc.Extra = bs
	updates["extra"] = bs
	return true, updates, nil
}

func parseStringArray(data []byte) ([]string, error) {
	if len(data) == 0 {
		return nil, nil
	}
	var arr []string
	if err := json.Unmarshal(data, &arr); err != nil {
		return nil, err
	}
	return arr, nil
}

// 目前仅支持 {"set": {"field_key": any, ...}}
func parseActionsSet(data []byte) (map[string]interface{}, error) {
	if len(data) == 0 {
		return map[string]interface{}{}, nil
	}
	var obj map[string]interface{}
	if err := json.Unmarshal(data, &obj); err != nil {
		return nil, err
	}
	// 兼容前端 template 格式：{"type":"template","values":{...}}
	if t, ok := obj["type"].(string); ok && t == "template" {
		if vals, ok := obj["values"].(map[string]interface{}); ok {
			return vals, nil
		}
		return map[string]interface{}{}, nil
	}
	// 默认识别 {"set": {...}}
	raw, ok := obj["set"]
	if !ok || raw == nil {
		return map[string]interface{}{}, nil
	}
	m, ok := raw.(map[string]interface{})
	if !ok {
		return map[string]interface{}{}, nil
	}
	return m, nil
}

// 解析 fields_to_update 中的 extra 字段集合，形如 {"extra": {"remark": "批量"}}
func parseFieldsToUpdateExtra(data []byte) (map[string]interface{}, error) {
	if len(data) == 0 {
		return map[string]interface{}{}, nil
	}
	var obj map[string]interface{}
	if err := json.Unmarshal(data, &obj); err != nil {
		return map[string]interface{}{}, nil
	}
	raw, ok := obj["extra"]
	if !ok || raw == nil {
		return map[string]interface{}{}, nil
	}
	m, ok := raw.(map[string]interface{})
	if !ok {
		return map[string]interface{}{}, nil
	}
	return m, nil
}

func isValidFieldKeyLocal(s string) bool {
	re := regexp.MustCompile(`^[a-z][a-z0-9_]{1,63}$`)
	return re.MatchString(s)
}

func isEmptyValue(v interface{}) bool {
	if v == nil {
		return true
	}
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
	if len(ba) != len(bb) {
		return false
	}
	for i := range ba {
		if ba[i] != bb[i] {
			return false
		}
	}
	return true
}

// derefString returns pointer value or empty string
func derefString(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
