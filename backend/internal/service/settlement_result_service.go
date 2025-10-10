package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"time"

	"nfa-dashboard/internal/model"
	"nfa-dashboard/internal/repository"

	"gorm.io/datatypes"
)

type SettlementResultService interface {
	CalculateResults(filter model.SettlementResultFilter) ([]model.SettlementResultItem, int64, error)
	DeleteResult(id uint64) error
}

type settlementResultService struct {
	resultsRepo repository.SettlementResultRepository
	formulaRepo repository.SettlementFormulaRepository
}

func NewSettlementResultService(resultsRepo repository.SettlementResultRepository, formulaRepo repository.SettlementFormulaRepository) SettlementResultService {
	return &settlementResultService{resultsRepo: resultsRepo, formulaRepo: formulaRepo}
}

func (s *settlementResultService) CalculateResults(filter model.SettlementResultFilter) ([]model.SettlementResultItem, int64, error) {
	if filter.StartDate.IsZero() || filter.EndDate.IsZero() {
		return nil, 0, errors.New("必须提供开始和结束日期")
	}
	if filter.EndDate.Before(filter.StartDate) {
		return nil, 0, errors.New("结束日期不能早于开始日期")
	}

	var (
		formula *model.SettlementFormula
		err    error
	)
	if filter.FormulaID > 0 {
		formula, err = s.formulaRepo.GetByID(filter.FormulaID)
		if err != nil {
			return nil, 0, fmt.Errorf("获取公式失败: %w", err)
		}
	} else {
		formula, err = s.formulaRepo.GetFirstEnabled()
		if err != nil {
			return nil, 0, fmt.Errorf("获取默认启用公式失败: %w", err)
		}
	}
	if formula == nil {
		return nil, 0, errors.New("未找到可用的结算公式")
	}
	var tokens []model.SettlementFormulaToken
	if err := json.Unmarshal([]byte(formula.Tokens), &tokens); err != nil {
		return nil, 0, fmt.Errorf("解析公式Token失败: %w", err)
	}

	rows, _, err := s.resultsRepo.ListAggregatedFlows(filter)
	if err != nil {
		return nil, 0, err
	}

    expectedDays := int(filter.EndDate.Sub(filter.StartDate).Hours()/24) + 1
    records := make([]model.SettlementResultRecord, 0, len(rows))
    calculatedAt := time.Now()

    for _, row := range rows {
        billingDays := row.DayCount
        missingDays := 0
        if expectedDays > billingDays {
            missingDays = expectedDays - billingDays
        }

        averageFlow := 0.0
        if billingDays > 0 {
            averageFlow = row.TotalFlow / float64(billingDays)
        }

        // 将 Byte 换算为 G（GB 或 GiB），用于“元/G”口径的公式计算
        base := filter.UnitBase
        if base != 1000 && base != 1024 {
            base = 1024
        }
        denom := math.Pow(float64(base), 3) // B -> G
        avgG := averageFlow / denom
        totalG := row.TotalFlow / denom

        env := map[string]float64{
            "settlement_flow_95":    avgG,
            "settlement_flow_total": totalG,
            "customer_fee":          valueOrZero(row.CustomerFee),
            "network_line_fee":      valueOrZero(row.NetworkLineFee),
            "node_deduction_fee":    valueOrZero(row.NodeDeductionFee),
            "final_fee":             valueOrZero(row.FinalFee),
            "discount_rate":         1,
            "tax_rate":              0,
            "service_fee":           0,
        }

        amount, missingFields, evalErr := evaluateFormula(tokens, env)
        if evalErr != nil {
            return nil, 0, fmt.Errorf("公式计算失败: %w", evalErr)
        }

        // 金额四舍五入策略：HALF_UP，保留2位小数
        amountRaw := amount
        amountRounded := math.Round(amountRaw*100) / 100

        missingList := make([]string, 0, len(missingFields))
        for field := range missingFields {
            missingList = append(missingList, field)
        }
        sort.Strings(missingList)

        unitLabel := "GiB"
        if base == 1000 {
            unitLabel = "GB"
        }

        detailPayload := map[string]any{
            // 兼容原有键：保留原始 Byte 数值
            "average_95":     averageFlow,
            "total_95":       row.TotalFlow,

            // 新增：原始与换算后的并行展示
            "average_95_bytes":      averageFlow,
            "total_95_bytes":        row.TotalFlow,
            "average_95_converted":  avgG,
            "total_95_converted":    totalG,
            "converted_unit":        unitLabel,
            "unit_base":             base,

            // 费率项（原样）
            "customer_fee":          env["customer_fee"],
            "network_fee":           env["network_line_fee"],
            "node_deduction":        env["node_deduction_fee"],
            "final_fee":             env["final_fee"],

            // 金额：返回四舍五入后的 amount，同时提供原始值
            "amount":                amountRounded,
            "amount_raw":            amountRaw,
            "rounding_mode":         "HALF_UP",
            "rounding_scale":        2,
        }
        detailJSON, _ := json.Marshal(detailPayload)
        missingJSON, _ := json.Marshal(missingList)

        amountCopy := amountRounded
        record := model.SettlementResultRecord{
            FormulaID:         formula.ID,
            FormulaName:       formula.Name,
            FormulaTokens:     datatypes.JSON([]byte(formula.Tokens)),
            Region:            row.Region,
            CP:                row.CP,
            SchoolID:          row.SchoolID,
            SchoolName:        row.SchoolName,
            StartDate:         filter.StartDate,
            EndDate:           filter.EndDate,
            BillingDays:       billingDays,
            Total95Flow:       row.TotalFlow,
            Average95Flow:     averageFlow,
            CustomerFee:       pointerFromValue(row.CustomerFee),
            NetworkLineFee:    pointerFromValue(row.NetworkLineFee),
            NodeDeductionFee:  pointerFromValue(row.NodeDeductionFee),
            FinalFee:          pointerFromValue(row.FinalFee),
            Amount:            &amountCopy,
            Currency:          "CNY",
            MissingDays:       missingDays,
            MissingFields:     datatypes.JSON(missingJSON),
            CalculationDetail: datatypes.JSON(detailJSON),
            CalculatedBy:      filter.UserID,
            UpdatedAt:         calculatedAt,
        }
        records = append(records, record)
    }

    if len(records) > 0 {
        if err := s.resultsRepo.UpsertResults(records); err != nil {
            return nil, 0, err
        }
	}

	stored, total, err := s.resultsRepo.ListResults(filter)
	if err != nil {
		return nil, 0, err
	}

	items := make([]model.SettlementResultItem, 0, len(stored))
	for _, record := range stored {
		items = append(items, recordToItem(record))
	}

	return items, total, nil
}

func (s *settlementResultService) DeleteResult(id uint64) error {
	if id == 0 {
		return errors.New("无效的结算结果ID")
	}
	return s.resultsRepo.DeleteByID(id)
}

func recordToItem(record model.SettlementResultRecord) model.SettlementResultItem {
	missing := make([]string, 0)
	if len(record.MissingFields) > 0 {
		_ = json.Unmarshal([]byte(record.MissingFields), &missing)
	}

	calculationDetail := ""
	if len(record.CalculationDetail) > 0 {
		calculationDetail = string(record.CalculationDetail)
	}

	formulaTokens := ""
	if len(record.FormulaTokens) > 0 {
		formulaTokens = string(record.FormulaTokens)
	}

	return model.SettlementResultItem{
		Region:            record.Region,
		CP:                record.CP,
		SchoolID:          record.SchoolID,
		SchoolName:        record.SchoolName,
		BillingDays:       record.BillingDays,
		Average95Flow:     record.Average95Flow,
		Total95Flow:       record.Total95Flow,
		MissingDays:       record.MissingDays,
		FormulaID:         record.FormulaID,
		FormulaName:       record.FormulaName,
		FormulaTokens:     formulaTokens,
		CustomerFee:       valueOrZero(record.CustomerFee),
		NetworkLineFee:    valueOrZero(record.NetworkLineFee),
		NodeDeductionFee:  valueOrZero(record.NodeDeductionFee),
		FinalFee:          valueOrZero(record.FinalFee),
		Amount:            valueOrZero(record.Amount),
		Currency:          record.Currency,
		StartDate:         record.StartDate,
		EndDate:           record.EndDate,
		UpdatedAt:         record.UpdatedAt,
		MissingFields:     missing,
		CalculationDetail: calculationDetail,
	}
}

func valueOrZero(v *float64) float64 {
	if v == nil {
		return 0
	}
	return *v
}

func pointerFromValue(v *float64) *float64 {
	if v == nil {
		return nil
	}
	value := *v
	return &value
}

func evaluateFormula(tokens []model.SettlementFormulaToken, env map[string]float64) (float64, map[string]struct{}, error) {
	rpn, err := toRPN(tokens)
	if err != nil {
		return 0, nil, err
	}
	stack := make([]float64, 0, len(rpn))
	missing := make(map[string]struct{})

	for _, token := range rpn {
		switch token.Type {
		case "number":
			val, err := parseNumber(token.Value)
			if err != nil {
				return 0, nil, fmt.Errorf("解析常量 %s 失败: %w", token.Value, err)
			}
			stack = append(stack, val)
		case "field":
			val, ok := env[token.Value]
			if !ok {
				missing[token.Value] = struct{}{}
				stack = append(stack, 0)
			} else {
				stack = append(stack, val)
			}
		case "operator":
			if len(stack) < 2 {
				return 0, nil, errors.New("表达式不合法，操作数不足")
			}
			b := stack[len(stack)-1]
			a := stack[len(stack)-2]
			stack = stack[:len(stack)-2]
			var res float64
			switch token.Value {
			case "+":
				res = a + b
			case "-":
				res = a - b
			case "*":
				res = a * b
			case "/":
				if b == 0 {
					res = 0
				} else {
					res = a / b
				}
			default:
				return 0, nil, fmt.Errorf("不支持的运算符: %s", token.Value)
			}
			stack = append(stack, res)
		}
	}

	if len(stack) != 1 {
		return 0, nil, errors.New("表达式不合法，无法完成计算")
	}
	return stack[0], missing, nil
}

func toRPN(tokens []model.SettlementFormulaToken) ([]model.SettlementFormulaToken, error) {
	output := make([]model.SettlementFormulaToken, 0, len(tokens))
	stack := make([]model.SettlementFormulaToken, 0)
	prec := map[string]int{
		"+": 1,
		"-": 1,
		"*": 2,
		"/": 2,
	}

	for _, token := range tokens {
		switch token.Type {
		case "number", "field":
			output = append(output, token)
		case "operator":
			switch token.Value {
			case "(":
				stack = append(stack, token)
			case ")":
				found := false
				for len(stack) > 0 {
					top := stack[len(stack)-1]
					stack = stack[:len(stack)-1]
					if top.Type == "operator" && top.Value == "(" {
						found = true
						break
					}
					output = append(output, top)
				}
				if !found {
					return nil, errors.New("括号不匹配")
				}
			default:
				for len(stack) > 0 {
					top := stack[len(stack)-1]
					if top.Type != "operator" {
						break
					}
					topPrec, okTop := prec[top.Value]
					curPrec, okCur := prec[token.Value]
					if !okTop || !okCur || topPrec < curPrec {
						break
					}
					stack = stack[:len(stack)-1]
					output = append(output, top)
				}
				stack = append(stack, token)
			}
		default:
			return nil, fmt.Errorf("未知的 token 类型: %s", token.Type)
		}
	}

	for len(stack) > 0 {
		top := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if top.Type == "operator" && (top.Value == "(" || top.Value == ")") {
			return nil, errors.New("括号不匹配")
		}
		output = append(output, top)
	}

	return output, nil
}

func parseNumber(val string) (float64, error) {
	switch val {
	case "pi":
		return math.Pi, nil
	default:
		return strconv.ParseFloat(val, 64)
	}
}
