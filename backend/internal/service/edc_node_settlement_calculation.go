package service

import (
	"fmt"
	"math"
	"sort"
	"time"

	"nfa-dashboard/internal/model"
)

const (
	EDCSettlementModeDaily95Avg = "daily_95_avg"
	EDCSettlementModeRange95    = "range_95"
)

var edcSettlementUnitBases = []int{1000, 1024}

type edcNodeKey struct {
	EntityID    uint64
	Region      string
	CP          string
	DisplayName string
}

func normalizeEDCSettlementMode(mode string) string {
	switch mode {
	case EDCSettlementModeRange95, "monthly95":
		return EDCSettlementModeRange95
	default:
		return EDCSettlementModeDaily95Avg
	}
}

func buildEDCNodeMonthlyTrafficSettlement(entity model.EDCEntity, month time.Time, mode string, unitBase int, raw95 float64, mbps95 float64) model.SettlementNodeMonthly95 {
	serviceMonth := month.In(time.Local).Format("2006-01")
	settlementTime := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, month.Location())
	return model.SettlementNodeMonthly95{
		EntityID:        entity.ID,
		DisplayName:     entity.DisplayName,
		Region:          entity.Region,
		CP:              entity.CP,
		ServiceMonth:    serviceMonth,
		SettlementMode:  normalizeEDCSettlementMode(mode),
		UnitBase:        normalizeEDCUnitBase(unitBase),
		Raw95:           raw95,
		Mbps95:          mbps95,
		SettlementValue: mbps95,
		SettlementTime:  settlementTime,
	}
}

func buildEDCNodeDailyTrafficSettlement(entity model.EDCEntity, day time.Time, mode string, unitBase int, raw95 float64, mbps95 float64) model.SettlementNodeDaily95 {
	settlementTime := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, day.Location())
	return model.SettlementNodeDaily95{
		EntityID:        entity.ID,
		DisplayName:     entity.DisplayName,
		Region:          entity.Region,
		CP:              entity.CP,
		ServiceMonth:    day.In(time.Local).Format("2006-01"),
		SettlementMode:  normalizeEDCSettlementMode(mode),
		UnitBase:        normalizeEDCUnitBase(unitBase),
		Raw95:           raw95,
		Mbps95:          mbps95,
		SettlementValue: mbps95,
		SettlementTime:  settlementTime,
	}
}

func normalizeEDCUnitBase(base int) int {
	if base == 1000 {
		return 1000
	}
	return 1024
}

func edcRawToMbps(raw int64, base int) float64 {
	b := float64(normalizeEDCUnitBase(base))
	return float64(raw) * 8 / 300 / b / b
}

func percentile95Raw(values []int64) (float64, bool) {
	if len(values) == 0 {
		return 0, false
	}
	sorted := append([]int64(nil), values...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] > sorted[j] })
	excludeCount := int(math.Ceil(float64(len(sorted)) * 0.05))
	if excludeCount >= len(sorted) {
		excludeCount = len(sorted) - 1
	}
	return float64(sorted[excludeCount]), true
}

func computeRange95Raw(points []model.EDCTraffic5m) (float64, bool) {
	values := make([]int64, 0, len(points))
	for _, point := range points {
		values = append(values, point.ServiceSize)
	}
	return percentile95Raw(values)
}

func computeNodeRange95Raw(points []model.EDCNodeTrafficPoint) (float64, bool) {
	values := make([]int64, 0, len(points))
	for _, point := range points {
		values = append(values, point.ServiceSize)
	}
	return percentile95Raw(values)
}

func computeDaily95AvgRaw(points []model.EDCTraffic5m) (float64, bool) {
	byDay := map[string][]int64{}
	for _, point := range points {
		day := point.Bucket5m.In(time.Local).Format("2006-01-02")
		byDay[day] = append(byDay[day], point.ServiceSize)
	}
	if len(byDay) == 0 {
		return 0, false
	}
	total := 0.0
	count := 0
	for _, values := range byDay {
		raw, ok := percentile95Raw(values)
		if !ok {
			continue
		}
		total += raw
		count++
	}
	if count == 0 {
		return 0, false
	}
	return total / float64(count), true
}

func groupEDCNodeTrafficPoints(points []model.EDCNodeTrafficPoint) map[edcNodeKey][]model.EDCNodeTrafficPoint {
	grouped := make(map[edcNodeKey][]model.EDCNodeTrafficPoint)
	for _, point := range points {
		key := edcNodeKey{
			EntityID:    point.EntityID,
			Region:      point.Region,
			CP:          point.CP,
			DisplayName: point.DisplayName,
		}
		grouped[key] = append(grouped[key], point)
	}
	return grouped
}

func sortedEDCNodeKeys(grouped map[edcNodeKey][]model.EDCNodeTrafficPoint) []edcNodeKey {
	keys := make([]edcNodeKey, 0, len(grouped))
	for key := range grouped {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		a, b := keys[i], keys[j]
		if a.Region != b.Region {
			return a.Region < b.Region
		}
		if a.CP != b.CP {
			return a.CP < b.CP
		}
		if a.DisplayName != b.DisplayName {
			return a.DisplayName < b.DisplayName
		}
		return a.EntityID < b.EntityID
	})
	return keys
}

func filterEDCNodePoints(points []model.EDCNodeTrafficPoint, start, end time.Time) []model.EDCNodeTrafficPoint {
	out := make([]model.EDCNodeTrafficPoint, 0)
	for _, point := range points {
		if !point.Bucket5m.Before(start) && point.Bucket5m.Before(end) {
			out = append(out, point)
		}
	}
	return out
}

func edcNodeEntityFromKey(key edcNodeKey) model.EDCEntity {
	return model.EDCEntity{
		ID:          key.EntityID,
		DisplayName: key.DisplayName,
		Region:      key.Region,
		CP:          key.CP,
	}
}

func edcNodeGroupLabel(key edcNodeKey) string {
	return fmt.Sprintf("%s/%s/%s", key.Region, key.CP, key.DisplayName)
}

func selectFinalNodeRate(entity model.EDCEntity, rates []model.RateFinalNode) (model.RateFinalNode, bool) {
	var fallback *model.RateFinalNode
	for i := range rates {
		rate := rates[i]
		if rate.EntityID != nil && *rate.EntityID == entity.ID {
			return rate, true
		}
		if rate.EntityID == nil && rate.Region == entity.Region && rate.CP == entity.CP && fallback == nil {
			fallback = &rates[i]
		}
	}
	if fallback != nil {
		return *fallback, true
	}
	return model.RateFinalNode{}, false
}

func selectFinalNodeRateForSettlement(entity model.EDCEntity, rates []model.RateFinalNode, mode string) (model.RateFinalNode, bool) {
	targetMode := normalizeEDCSettlementMode(mode)
	var fallback *model.RateFinalNode
	for i := range rates {
		rate := rates[i]
		if normalizeEDCSettlementMode(rate.SettlementMode) != targetMode || rate.FinalFee == nil {
			continue
		}
		if rate.EntityID != nil && *rate.EntityID == entity.ID {
			return rate, true
		}
		if rate.EntityID == nil && rate.Region == entity.Region && rate.CP == entity.CP && fallback == nil {
			fallback = &rates[i]
		}
	}
	if fallback != nil {
		return *fallback, true
	}
	return model.RateFinalNode{}, false
}

func floatPtrValue(v *float64) float64 {
	if v == nil {
		return 0
	}
	return *v
}

func billFromFee(fee *float64) *float64 {
	if fee == nil {
		return nil
	}
	value := *fee
	return &value
}

func billValue(v *float64) float64 {
	if v == nil {
		return 0
	}
	return *v
}

func buildEDCNodeMonthlySettlementRows(entity model.EDCEntity, rate model.RateFinalNode, month time.Time, raw95 float64) []model.SettlementNodeMonthly95 {
	rows := make([]model.SettlementNodeMonthly95, 0, len(edcSettlementUnitBases))
	for _, base := range edcSettlementUnitBases {
		mbps95 := raw95 * 8 / 300 / float64(base) / float64(base)
		rows = append(rows, buildEDCNodeMonthlySettlement(entity, rate, month, raw95, mbps95, base))
	}
	return rows
}

func buildEDCNodeDailySettlementRows(entity model.EDCEntity, rate model.RateFinalNode, day time.Time, raw95 float64) []model.SettlementNodeDaily95 {
	rows := make([]model.SettlementNodeDaily95, 0, len(edcSettlementUnitBases))
	for _, base := range edcSettlementUnitBases {
		mbps95 := raw95 * 8 / 300 / float64(base) / float64(base)
		rows = append(rows, buildEDCNodeDailySettlement(entity, rate, day, raw95, mbps95, base))
	}
	return rows
}

func buildEDCNodeMonthlySettlement(entity model.EDCEntity, rate model.RateFinalNode, month time.Time, raw95 float64, mbps95 float64, unitBase int) model.SettlementNodeMonthly95 {
	trafficBill := mbps95 * floatPtrValue(rate.FinalFee)
	cpBill := billFromFee(rate.CPFee)
	rackBill := billFromFee(rate.RackFee)
	otherBill := billFromFee(rate.OtherFee)
	totalBill := trafficBill + billValue(cpBill) + billValue(rackBill) + billValue(otherBill)
	base := normalizeEDCUnitBase(unitBase)
	mode := normalizeEDCSettlementMode(rate.SettlementMode)
	serviceMonth := month.In(time.Local).Format("2006-01")
	settlementTime := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, month.Location())
	fee := floatPtrValue(rate.FinalFee)
	return model.SettlementNodeMonthly95{
		EntityID:                   entity.ID,
		DisplayName:                entity.DisplayName,
		Region:                     entity.Region,
		CP:                         entity.CP,
		ServiceMonth:               serviceMonth,
		SettlementMode:             mode,
		UnitBase:                   base,
		Raw95:                      raw95,
		Mbps95:                     mbps95,
		SettlementValue:            mbps95,
		SettlementTime:             settlementTime,
		Monthly95Fee:               &fee,
		Monthly95Bill:              &trafficBill,
		TrafficBill:                &trafficBill,
		CPFee:                      rate.CPFee,
		CPBill:                     cpBill,
		CPFeeOwnerID:               rate.CPFeeOwnerID,
		RackFee:                    rate.RackFee,
		RackBill:                   rackBill,
		TotalBill:                  &totalBill,
		NodeConstructionFee:        rate.NodeConstructionFee,
		NodeConstructionBill:       &trafficBill,
		NodeConstructionFeeOwnerID: rate.NodeConstructionFeeOwnerID,
		RackFeeOwnerID:             rate.RackFeeOwnerID,
		OtherFee:                   rate.OtherFee,
		OtherBill:                  otherBill,
		OtherFeeOwnerID:            rate.OtherFeeOwnerID,
	}
}

func buildEDCNodeDailySettlement(entity model.EDCEntity, rate model.RateFinalNode, day time.Time, raw95 float64, mbps95 float64, unitBase int) model.SettlementNodeDaily95 {
	trafficBill := mbps95 * floatPtrValue(rate.FinalFee)
	cpBill := billFromFee(rate.CPFee)
	rackBill := billFromFee(rate.RackFee)
	otherBill := billFromFee(rate.OtherFee)
	totalBill := trafficBill + billValue(cpBill) + billValue(rackBill) + billValue(otherBill)
	base := normalizeEDCUnitBase(unitBase)
	mode := normalizeEDCSettlementMode(rate.SettlementMode)
	settlementTime := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, day.Location())
	fee := floatPtrValue(rate.FinalFee)
	return model.SettlementNodeDaily95{
		EntityID:                   entity.ID,
		DisplayName:                entity.DisplayName,
		Region:                     entity.Region,
		CP:                         entity.CP,
		ServiceMonth:               day.In(time.Local).Format("2006-01"),
		SettlementMode:             mode,
		UnitBase:                   base,
		Raw95:                      raw95,
		Mbps95:                     mbps95,
		SettlementValue:            mbps95,
		SettlementTime:             settlementTime,
		Daily95Fee:                 &fee,
		Daily95Bill:                &trafficBill,
		TrafficBill:                &trafficBill,
		CPFee:                      rate.CPFee,
		CPBill:                     cpBill,
		CPFeeOwnerID:               rate.CPFeeOwnerID,
		RackBill:                   rackBill,
		TotalBill:                  &totalBill,
		NodeConstructionFee:        rate.NodeConstructionFee,
		NodeConstructionBill:       &trafficBill,
		NodeConstructionFeeOwnerID: rate.NodeConstructionFeeOwnerID,
		RackFee:                    rate.RackFee,
		RackFeeOwnerID:             rate.RackFeeOwnerID,
		OtherFee:                   rate.OtherFee,
		OtherBill:                  otherBill,
		OtherFeeOwnerID:            rate.OtherFeeOwnerID,
	}
}
