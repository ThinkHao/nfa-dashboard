package service

import (
	"fmt"
	"time"

	"nfa-dashboard/internal/model"
	"nfa-dashboard/internal/repository"
)

type EDCNodeSettlementService interface {
	HasSettlementTraffic(start, end time.Time) (bool, error)
	ExecuteDailyTask(taskID int64, day time.Time) error
	ExecuteDailyRangeTask(taskID int64, start, end time.Time) error
	ExecuteMonthlyTask(taskID int64, month time.Time) error
	ExecuteMonthlyRangeTask(taskID int64, start, end time.Time) error
	ListDailySettlements(filter map[string]interface{}, page, pageSize int) ([]model.SettlementNodeDaily95, int64, error)
	ListMonthlySettlements(filter map[string]interface{}, page, pageSize int) ([]model.SettlementNodeMonthly95, int64, error)
}

type edcNodeSettlementService struct {
	repo           repository.EDCNodeSettlementRepository
	ratesRepo      repository.RatesRepository
	settlementRepo repository.SettlementRepository
}

func NewEDCNodeSettlementService(repo repository.EDCNodeSettlementRepository, ratesRepo repository.RatesRepository, settlementRepo repository.SettlementRepository) EDCNodeSettlementService {
	return &edcNodeSettlementService{repo: repo, ratesRepo: ratesRepo, settlementRepo: settlementRepo}
}

func (s *edcNodeSettlementService) HasSettlementTraffic(start, end time.Time) (bool, error) {
	ok, err := s.repo.ExistsTrafficPointByDisplayNode(start, end)
	if err != nil {
		return false, err
	}
	return ok, nil
}

func (s *edcNodeSettlementService) ExecuteDailyTask(taskID int64, day time.Time) error {
	return s.ExecuteDailyRangeTask(taskID, day, day)
}

func (s *edcNodeSettlementService) ExecuteDailyRangeTask(taskID int64, start, end time.Time) error {
	if err := s.updateTask(taskID, "running", 0, ""); err != nil {
		return err
	}
	totalProcessed := 0
	multiDay := !startOfDay(start).Equal(startOfDay(end))
	for day := startOfDay(start); !day.After(startOfDay(end)); day = day.AddDate(0, 0, 1) {
		if multiDay {
			hasTraffic, err := s.repo.ExistsTrafficPointByDisplayNode(day, day.AddDate(0, 0, 1))
			if err != nil {
				_ = s.updateTask(taskID, "failed", totalProcessed, err.Error())
				return err
			}
			if !hasTraffic {
				continue
			}
		}
		rows, processed, err := s.calculateDaily(day)
		if err != nil {
			_ = s.updateTask(taskID, "failed", totalProcessed+processed, err.Error())
			return err
		}
		if err := s.repo.DeleteDailySettlements(day, day.AddDate(0, 0, 1)); err != nil {
			_ = s.updateTask(taskID, "failed", totalProcessed+processed, err.Error())
			return err
		}
		if err := s.repo.UpsertDailySettlements(rows); err != nil {
			_ = s.updateTask(taskID, "failed", totalProcessed+processed, err.Error())
			return err
		}
		totalProcessed += processed
		_ = s.updateTask(taskID, "running", totalProcessed, "")
	}
	return s.updateTask(taskID, "success", totalProcessed, "")
}

func (s *edcNodeSettlementService) ExecuteMonthlyTask(taskID int64, month time.Time) error {
	return s.ExecuteMonthlyRangeTask(taskID, month, month)
}

func (s *edcNodeSettlementService) ExecuteMonthlyRangeTask(taskID int64, start, end time.Time) error {
	if err := s.updateTask(taskID, "running", 0, ""); err != nil {
		return err
	}
	totalProcessed := 0
	multiMonth := !startOfMonth(start).Equal(startOfMonth(end))
	for month := startOfMonth(start); !month.After(startOfMonth(end)); month = month.AddDate(0, 1, 0) {
		if multiMonth {
			hasTraffic, err := s.repo.ExistsTrafficPointByDisplayNode(month, month.AddDate(0, 1, 0))
			if err != nil {
				_ = s.updateTask(taskID, "failed", totalProcessed, err.Error())
				return err
			}
			if !hasTraffic {
				continue
			}
		}
		monthly, processed, err := s.calculateMonthly(month)
		if err != nil {
			_ = s.updateTask(taskID, "failed", totalProcessed+processed, err.Error())
			return err
		}
		if err := s.repo.DeleteMonthlySettlements(month.Format("2006-01")); err != nil {
			_ = s.updateTask(taskID, "failed", totalProcessed+processed, err.Error())
			return err
		}
		if err := s.repo.UpsertMonthlySettlements(monthly); err != nil {
			_ = s.updateTask(taskID, "failed", totalProcessed+processed, err.Error())
			return err
		}
		totalProcessed += processed
		_ = s.updateTask(taskID, "running", totalProcessed, "")
	}
	return s.updateTask(taskID, "success", totalProcessed, "")
}

func (s *edcNodeSettlementService) calculateDaily(day time.Time) ([]model.SettlementNodeDaily95, int, error) {
	start := startOfDay(day)
	end := start.AddDate(0, 0, 1)
	entities, err := s.repo.ListEnabledEntities()
	if err != nil {
		return nil, 0, err
	}
	if len(entities) == 0 {
		return nil, 0, fmt.Errorf("没有启用的 EDC 节点，请先在 EDC 节点映射中启用节点")
	}
	rates, err := s.ratesRepo.ListAllFinalNodeRates()
	if err != nil {
		return nil, 0, err
	}
	points, err := s.repo.ListTrafficPointsByDisplayNode(start, end)
	if err != nil {
		return nil, 0, err
	}
	groups, err := s.ratesRepo.ListEnabledNodeSettlementGroups()
	if err != nil {
		return nil, 0, err
	}
	memberIDs := nodeSettlementGroupMemberIDs(groups)
	groupPoints := []model.EDCNodeTrafficPoint(nil)
	if len(memberIDs) > 0 {
		groupPoints, err = s.repo.ListTrafficPointsByEntities(memberIDs, start, end)
		if err != nil {
			return nil, 0, err
		}
	}
	grouped := groupEDCNodeTrafficPoints(filterGroupedEDCNodePoints(points, memberIDs))
	groupedSettlementPoints := groupEDCNodeSettlementPoints(groupPoints, groups)
	keys := sortedEDCNodeKeys(grouped)
	rows := make([]model.SettlementNodeDaily95, 0, len(keys)+len(groupedSettlementPoints))
	processed := 0
	for _, group := range groups {
		groupPoints := groupedSettlementPoints[group.ID]
		if len(groupPoints) == 0 {
			continue
		}
		raw, ok := computeNodeRange95Raw(aggregateEDCNodeTrafficPoints(groupPoints))
		if !ok {
			continue
		}
		rate, ok := selectFinalGroupRateForSettlement(group, rates, EDCSettlementModeDaily95Avg)
		if !ok {
			continue
		}
		for _, base := range edcSettlementUnitBases {
			mbps95 := raw * 8 / 300 / float64(base) / float64(base)
			rows = append(rows, buildEDCGroupDailySettlement(group, rate, start, raw, mbps95, base))
			processed++
		}
	}
	for _, key := range keys {
		raw, ok := computeNodeRange95Raw(grouped[key])
		if !ok {
			continue
		}
		entity := edcNodeEntityFromKey(key)
		rate, ok := selectFinalNodeRateForSettlement(entity, rates, EDCSettlementModeDaily95Avg)
		if !ok {
			continue
		}
		builtRows := buildEDCNodeDailySettlementRows(entity, rate, start, raw)
		rows = append(rows, builtRows...)
		processed += len(builtRows)
	}
	if len(rows) == 0 {
		return rows, processed, buildEmptyEDCNodeSettlementError(len(keys), "日", len(points) > 0)
	}
	return rows, processed, nil
}

func (s *edcNodeSettlementService) calculateMonthly(month time.Time) ([]model.SettlementNodeMonthly95, int, error) {
	start := startOfMonth(month)
	end := start.AddDate(0, 1, 0)
	entities, err := s.repo.ListEnabledEntities()
	if err != nil {
		return nil, 0, err
	}
	if len(entities) == 0 {
		return nil, 0, fmt.Errorf("没有启用的 EDC 节点，请先在 EDC 节点映射中启用节点")
	}
	rates, err := s.ratesRepo.ListAllFinalNodeRates()
	if err != nil {
		return nil, 0, err
	}
	points, err := s.repo.ListTrafficPointsByDisplayNode(start, end)
	if err != nil {
		return nil, 0, err
	}
	groups, err := s.ratesRepo.ListEnabledNodeSettlementGroups()
	if err != nil {
		return nil, 0, err
	}
	memberIDs := nodeSettlementGroupMemberIDs(groups)
	groupPoints := []model.EDCNodeTrafficPoint(nil)
	if len(memberIDs) > 0 {
		groupPoints, err = s.repo.ListTrafficPointsByEntities(memberIDs, start, end)
		if err != nil {
			return nil, 0, err
		}
	}
	grouped := groupEDCNodeTrafficPoints(filterGroupedEDCNodePoints(points, memberIDs))
	groupedSettlementPoints := groupEDCNodeSettlementPoints(groupPoints, groups)
	keys := sortedEDCNodeKeys(grouped)
	monthlyRows := make([]model.SettlementNodeMonthly95, 0, len(keys)+len(groupedSettlementPoints))
	processed := 0
	for _, group := range groups {
		groupPoints := groupedSettlementPoints[group.ID]
		if len(groupPoints) == 0 {
			continue
		}
		raw95, ok := computeNodeRange95Raw(aggregateEDCNodeTrafficPoints(groupPoints))
		if !ok {
			continue
		}
		rate, ok := selectFinalGroupRateForSettlement(group, rates, EDCSettlementModeRange95)
		if !ok {
			continue
		}
		for _, base := range edcSettlementUnitBases {
			mbps95 := raw95 * 8 / 300 / float64(base) / float64(base)
			monthlyRows = append(monthlyRows, buildEDCGroupMonthlySettlement(group, rate, start, raw95, mbps95, base))
			processed++
		}
	}
	for _, key := range keys {
		nodePoints := grouped[key]
		entity := edcNodeEntityFromKey(key)
		raw95, ok := computeNodeRange95Raw(nodePoints)
		if !ok {
			continue
		}
		rate, ok := selectFinalNodeRateForSettlement(entity, rates, EDCSettlementModeRange95)
		if !ok {
			continue
		}
		builtRows := buildEDCNodeMonthlySettlementRows(entity, rate, start, raw95)
		monthlyRows = append(monthlyRows, builtRows...)
		processed += len(builtRows)
	}
	if len(monthlyRows) == 0 {
		return monthlyRows, processed, buildEmptyEDCNodeSettlementError(len(keys), "月", len(points) > 0)
	}
	return monthlyRows, processed, nil
}

func nodeSettlementGroupMemberIDs(groups []model.EDCNodeSettlementGroup) []uint64 {
	ids := make([]uint64, 0)
	seen := make(map[uint64]struct{})
	for _, group := range groups {
		for _, member := range group.Members {
			if member.EntityID == 0 {
				continue
			}
			if _, ok := seen[member.EntityID]; ok {
				continue
			}
			seen[member.EntityID] = struct{}{}
			ids = append(ids, member.EntityID)
		}
	}
	return ids
}

func filterGroupedEDCNodePoints(points []model.EDCNodeTrafficPoint, memberIDs []uint64) []model.EDCNodeTrafficPoint {
	if len(memberIDs) == 0 {
		return points
	}
	members := make(map[uint64]struct{}, len(memberIDs))
	for _, id := range memberIDs {
		members[id] = struct{}{}
	}
	out := make([]model.EDCNodeTrafficPoint, 0, len(points))
	for _, point := range points {
		if _, ok := members[point.EntityID]; ok {
			continue
		}
		out = append(out, point)
	}
	return out
}

func groupEDCNodeSettlementPoints(points []model.EDCNodeTrafficPoint, groups []model.EDCNodeSettlementGroup) map[uint64][]model.EDCNodeTrafficPoint {
	memberGroupIDs := make(map[uint64]uint64)
	for _, group := range groups {
		for _, member := range group.Members {
			memberGroupIDs[member.EntityID] = group.ID
		}
	}
	grouped := make(map[uint64][]model.EDCNodeTrafficPoint)
	for _, point := range points {
		if groupID, ok := memberGroupIDs[point.EntityID]; ok {
			grouped[groupID] = append(grouped[groupID], point)
		}
	}
	return grouped
}

func buildEmptyEDCNodeSettlementError(entitiesWithTraffic int, periodLabel string, hasTrafficPoints bool) error {
	if entitiesWithTraffic == 0 {
		return fmt.Errorf("任务周期内没有可结算的 EDC 节点%s95流量数据", periodLabel)
	}
	if hasTrafficPoints {
		return fmt.Errorf("没有可用的 EDC 节点%s95费率或流量单价", periodLabel)
	}
	return fmt.Errorf("没有生成任何 EDC 节点%s95结算数据", periodLabel)
}

func filterEDCPoints(points []model.EDCTraffic5m, start, end time.Time) []model.EDCTraffic5m {
	out := make([]model.EDCTraffic5m, 0)
	for _, point := range points {
		if !point.Bucket5m.Before(start) && point.Bucket5m.Before(end) {
			out = append(out, point)
		}
	}
	return out
}

func startOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func startOfMonth(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
}

func (s *edcNodeSettlementService) updateTask(taskID int64, status string, processed int, errorMsg string) error {
	task, err := s.settlementRepo.GetSettlementTaskByID(taskID)
	if err != nil {
		return err
	}
	now := time.Now()
	task.Status = status
	task.ProcessedCount = processed
	if status == "running" && task.StartTime == nil {
		task.StartTime = &now
	}
	if status == "success" || status == "failed" {
		task.EndTime = &now
	}
	task.ErrorMessage = errorMsg
	return s.settlementRepo.UpdateSettlementTask(task)
}

func (s *edcNodeSettlementService) ListDailySettlements(filter map[string]interface{}, page, pageSize int) ([]model.SettlementNodeDaily95, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	return s.repo.ListDailySettlements(filter, pageSize, (page-1)*pageSize)
}

func (s *edcNodeSettlementService) ListMonthlySettlements(filter map[string]interface{}, page, pageSize int) ([]model.SettlementNodeMonthly95, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	return s.repo.ListMonthlySettlements(filter, pageSize, (page-1)*pageSize)
}
