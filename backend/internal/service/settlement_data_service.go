package service

import (
	"context"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"nfa-dashboard/internal/model"
	"nfa-dashboard/internal/repository"
	"strings"
	"time"
)

type SettlementDataService interface {
	List(ctx context.Context, filter SettlementCustomerFilter, page, pageSize int) ([]model.SettlementCustomer, int64, error)
	ListMonthly(ctx context.Context, filter SettlementCustomerFilter, page, pageSize int) ([]model.SettlementCustomerMonthly, int64, error)
	ListAll(ctx context.Context, filter SettlementCustomerFilter) ([]model.SettlementCustomer, error)
	Recalculate(filter SettlementCustomerFilter) (int64, error)
	RecalculateWithProgress(filter SettlementCustomerFilter, progress func(processed int64, stageMetrics map[string]int64)) (int64, error)
	EstimateRecalculateTotal(filter SettlementCustomerFilter) (int64, error)
	RebuildMonthlySnapshot(start, end *time.Time) (int64, error)
	ListUsedChannelOwners(ctx context.Context, filter SettlementCustomerFilter) ([]UsedChannelOwner, error)
	ListUsedOwnerEntities(ctx context.Context, filter SettlementCustomerFilter) ([]UsedOwnerEntity, error)
	ListUsedOwnerSubjects(ctx context.Context, filter SettlementCustomerFilter) ([]UsedOwnerSubject, error)
	BuildOwnerNameMaps(rows []model.SettlementCustomer) (map[uint64]string, map[uint64]string, error)
	CreateRecalculateTask(start, end time.Time) (int64, error)
	MarkTaskRunning(taskID int64, total int64) error
	MarkTaskProgress(taskID int64, processed int64) error
	MarkTaskStage(taskID int64, stage string, processed int64, extras map[string]interface{}) error
	MarkTaskFailed(taskID int64, errMsg string) error
	MarkTaskSuccess(taskID int64, processed int64) error
	BuildScopeHash(filter SettlementCustomerFilter) string
	AcquireScopeLock(scopeHash string, timeoutSec int) (bool, error)
	ReleaseScopeLock(scopeHash string) error
}

type SettlementCustomerFilter struct {
	Region string
	CP     string
	School string
	Start  *time.Time
	End    *time.Time
	// 费用归属业务对象ID：匹配客户费或线路费任一归属
	OwnerEntityID *uint64
	// 渠道归属系统用户ID
	ChannelOwnerUserID *uint64
}

type settlementDataService struct {
	repo           repository.SettlementDataRepository
	userRepo       repository.UserRepository
	entitiesRepo   repository.EntitiesRepository
	settlementRepo repository.SettlementRepository
}

type UsedChannelOwner struct {
	ID          uint64 `json:"id"`
	DisplayName string `json:"display_name"`
}

type UsedOwnerEntity struct {
	ID         uint64 `json:"id"`
	EntityName string `json:"entity_name"`
}

type UsedOwnerSubject struct {
	Type  string `json:"type"`
	ID    uint64 `json:"id"`
	Label string `json:"label"`
}

func NewSettlementDataService(
	repo repository.SettlementDataRepository,
	userRepo repository.UserRepository,
	entitiesRepo repository.EntitiesRepository,
	settlementRepo repository.SettlementRepository,
) SettlementDataService {
	return &settlementDataService{
		repo:           repo,
		userRepo:       userRepo,
		entitiesRepo:   entitiesRepo,
		settlementRepo: settlementRepo,
	}
}

func (s *settlementDataService) List(ctx context.Context, filter SettlementCustomerFilter, page, pageSize int) ([]model.SettlementCustomer, int64, error) {
	m := map[string]interface{}{}
	if filter.Region != "" {
		m["region"] = filter.Region
	}
	if filter.CP != "" {
		m["cp"] = filter.CP
	}
	if filter.School != "" {
		m["school_name"] = filter.School
	}
	if filter.Start != nil {
		m["start_service_date"] = *filter.Start
	}
	if filter.End != nil {
		m["end_service_date"] = *filter.End
	}
	if filter.OwnerEntityID != nil && *filter.OwnerEntityID > 0 {
		m["owner_entity_id"] = *filter.OwnerEntityID
	}
	if filter.ChannelOwnerUserID != nil && *filter.ChannelOwnerUserID > 0 {
		m["channel_owner_user_id"] = *filter.ChannelOwnerUserID
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	limit := pageSize
	offset := (page - 1) * pageSize
	return s.repo.ListSettlementCustomer(ctx, m, limit, offset)
}

func (s *settlementDataService) ListMonthly(ctx context.Context, filter SettlementCustomerFilter, page, pageSize int) ([]model.SettlementCustomerMonthly, int64, error) {
	m := map[string]interface{}{}
	if filter.Region != "" {
		m["region"] = filter.Region
	}
	if filter.CP != "" {
		m["cp"] = filter.CP
	}
	if filter.School != "" {
		m["school_name"] = filter.School
	}
	if filter.Start != nil {
		m["start_service_date"] = *filter.Start
	}
	if filter.End != nil {
		m["end_service_date"] = *filter.End
	}
	if filter.OwnerEntityID != nil && *filter.OwnerEntityID > 0 {
		m["owner_entity_id"] = *filter.OwnerEntityID
	}
	if filter.ChannelOwnerUserID != nil && *filter.ChannelOwnerUserID > 0 {
		m["channel_owner_user_id"] = *filter.ChannelOwnerUserID
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	limit := pageSize
	offset := (page - 1) * pageSize
	return s.repo.ListSettlementCustomerMonthly(ctx, m, limit, offset)
}

func (s *settlementDataService) ListAll(ctx context.Context, filter SettlementCustomerFilter) ([]model.SettlementCustomer, error) {
	m := map[string]interface{}{}
	if filter.Region != "" {
		m["region"] = filter.Region
	}
	if filter.CP != "" {
		m["cp"] = filter.CP
	}
	if filter.School != "" {
		m["school_name"] = filter.School
	}
	if filter.Start != nil {
		m["start_service_date"] = *filter.Start
	}
	if filter.End != nil {
		m["end_service_date"] = *filter.End
	}
	if filter.OwnerEntityID != nil && *filter.OwnerEntityID > 0 {
		m["owner_entity_id"] = *filter.OwnerEntityID
	}
	if filter.ChannelOwnerUserID != nil && *filter.ChannelOwnerUserID > 0 {
		m["channel_owner_user_id"] = *filter.ChannelOwnerUserID
	}
	rows, _, err := s.repo.ListSettlementCustomer(ctx, m, 100000, 0)
	return rows, err
}

// Recalculate 轻量实现：按筛选范围从 nfa_school_settlement 回填/覆盖 settlement_customer 基础字段
func (s *settlementDataService) Recalculate(filter SettlementCustomerFilter) (int64, error) {
	return s.RecalculateWithProgress(filter, nil)
}

func (s *settlementDataService) RecalculateWithProgress(filter SettlementCustomerFilter, progress func(processed int64, stageMetrics map[string]int64)) (int64, error) {
	var start, end time.Time
	if filter.Start != nil {
		start = *filter.Start
	}
	if filter.End != nil {
		end = *filter.End
	}
	return s.repo.BackfillFromSchoolSettlement(filter.Region, filter.CP, filter.School, start, end, true, progress)
}

func (s *settlementDataService) EstimateRecalculateTotal(filter SettlementCustomerFilter) (int64, error) {
	var start, end time.Time
	if filter.Start != nil {
		start = *filter.Start
	}
	if filter.End != nil {
		end = *filter.End
	}
	return s.repo.CountSchoolSettlementRows(filter.Region, filter.CP, filter.School, start, end)
}

func (s *settlementDataService) RebuildMonthlySnapshot(start, end *time.Time) (int64, error) {
	var sTime, eTime time.Time
	if start != nil {
		sTime = *start
	}
	if end != nil {
		eTime = *end
	}
	return s.repo.RebuildSettlementCustomerMonthly(sTime, eTime)
}

func (s *settlementDataService) ListUsedChannelOwners(ctx context.Context, filter SettlementCustomerFilter) ([]UsedChannelOwner, error) {
	rows, err := s.ListAll(ctx, filter)
	if err != nil {
		return nil, err
	}
	ids := make([]uint64, 0)
	set := map[uint64]struct{}{}
	for _, row := range rows {
		if row.ChannelOwnerUserID != nil && *row.ChannelOwnerUserID > 0 {
			if _, ok := set[*row.ChannelOwnerUserID]; !ok {
				set[*row.ChannelOwnerUserID] = struct{}{}
				ids = append(ids, *row.ChannelOwnerUserID)
			}
		}
	}
	users, err := s.userRepo.FindByIDs(ids)
	if err != nil {
		return nil, err
	}
	items := make([]UsedChannelOwner, 0, len(users))
	for _, u := range users {
		items = append(items, UsedChannelOwner{ID: u.ID, DisplayName: displayUserName(u)})
	}
	return items, nil
}

func (s *settlementDataService) ListUsedOwnerEntities(ctx context.Context, filter SettlementCustomerFilter) ([]UsedOwnerEntity, error) {
	rows, err := s.ListAll(ctx, filter)
	if err != nil {
		return nil, err
	}
	ids := make([]uint64, 0)
	set := map[uint64]struct{}{}
	for _, row := range rows {
		for _, id := range []*uint64{row.CustomerFeeOwnerID, row.NetworkLineFeeOwnerID, row.NodeDeductionFeeOwnerID} {
			if id != nil && *id > 0 {
				if _, ok := set[*id]; !ok {
					set[*id] = struct{}{}
					ids = append(ids, *id)
				}
			}
		}
	}
	if len(ids) == 0 {
		return []UsedOwnerEntity{}, nil
	}
	ents, _, err := s.entitiesRepo.List(map[string]interface{}{"ids": ids}, len(ids), 0)
	if err != nil {
		return nil, err
	}
	items := make([]UsedOwnerEntity, 0, len(ents))
	for _, e := range ents {
		items = append(items, UsedOwnerEntity{ID: e.ID, EntityName: strings.TrimSpace(e.EntityName)})
	}
	return items, nil
}

func (s *settlementDataService) ListUsedOwnerSubjects(ctx context.Context, filter SettlementCustomerFilter) ([]UsedOwnerSubject, error) {
	rows, err := s.ListAll(ctx, filter)
	if err != nil {
		return nil, err
	}
	ids := make([]uint64, 0)
	set := map[uint64]struct{}{}
	for _, row := range rows {
		for _, id := range []*uint64{row.CustomerFeeOwnerID, row.NetworkLineFeeOwnerID, row.NodeDeductionFeeOwnerID, row.ChannelOwnerUserID} {
			if id != nil && *id > 0 {
				if _, ok := set[*id]; !ok {
					set[*id] = struct{}{}
					ids = append(ids, *id)
				}
			}
		}
	}
	users, err := s.userRepo.FindByIDs(ids)
	if err != nil {
		return nil, err
	}
	items := make([]UsedOwnerSubject, 0, len(users))
	for _, u := range users {
		items = append(items, UsedOwnerSubject{Type: "user", ID: u.ID, Label: displayUserName(u)})
	}
	return items, nil
}

func (s *settlementDataService) BuildOwnerNameMaps(rows []model.SettlementCustomer) (map[uint64]string, map[uint64]string, error) {
	entityIDs := make([]uint64, 0)
	entitySet := map[uint64]struct{}{}
	userIDs := make([]uint64, 0)
	userSet := map[uint64]struct{}{}

	for _, row := range rows {
		for _, id := range []*uint64{row.CustomerFeeOwnerID, row.NetworkLineFeeOwnerID} {
			if id != nil && *id > 0 {
				if _, ok := entitySet[*id]; !ok {
					entitySet[*id] = struct{}{}
					entityIDs = append(entityIDs, *id)
				}
			}
		}
		if row.ChannelOwnerUserID != nil && *row.ChannelOwnerUserID > 0 {
			if _, ok := userSet[*row.ChannelOwnerUserID]; !ok {
				userSet[*row.ChannelOwnerUserID] = struct{}{}
				userIDs = append(userIDs, *row.ChannelOwnerUserID)
			}
		}
	}

	entityMap := map[uint64]string{}
	if len(entityIDs) > 0 {
		ents, _, err := s.entitiesRepo.List(map[string]interface{}{"ids": entityIDs}, len(entityIDs), 0)
		if err != nil {
			return nil, nil, err
		}
		for _, e := range ents {
			entityMap[e.ID] = strings.TrimSpace(e.EntityName)
		}
	}

	userMap := map[uint64]string{}
	if len(userIDs) > 0 {
		users, err := s.userRepo.FindByIDs(userIDs)
		if err != nil {
			return nil, nil, err
		}
		for _, u := range users {
			userMap[u.ID] = displayUserName(u)
		}
	}
	return entityMap, userMap, nil
}

func (s *settlementDataService) CreateRecalculateTask(start, end time.Time) (int64, error) {
	metaBytes, _ := json.Marshal(map[string]interface{}{
		"range": map[string]string{
			"start": start.Format("2006-01-02"),
			"end":   end.Format("2006-01-02"),
		},
		"stage": "queued",
	})
	task := &model.SettlementTask{
		TaskType:       "customer_recalc",
		TaskDate:       start,
		Status:         "pending",
		TaskStage:      "queued",
		ProcessedCount: 0,
		CreateTime:     time.Now(),
		UpdateTime:     time.Now(),
		ErrorMessage:   start.Format("2006-01-02") + "," + end.Format("2006-01-02"), // 兼容历史字段
		TaskMeta:       string(metaBytes),
	}
	if err := s.settlementRepo.CreateSettlementTask(task); err != nil {
		return 0, err
	}
	return task.ID, nil
}

func (s *settlementDataService) MarkTaskRunning(taskID int64, total int64) error {
	task, err := s.settlementRepo.GetSettlementTaskByID(taskID)
	if err != nil {
		return err
	}
	now := time.Now()
	task.Status = "running"
	task.TaskStage = "computing"
	task.StartTime = &now
	if total > 0 {
		task.TotalCount = int(total)
	}
	task.ProcessedCount = 0
	task.UpdateTime = now
	enrichTaskMeta(task, map[string]interface{}{
		"stage":     "computing",
		"processed": 0,
		"total":     total,
	})
	return s.settlementRepo.UpdateSettlementTask(task)
}

func (s *settlementDataService) MarkTaskProgress(taskID int64, processed int64) error {
	task, err := s.settlementRepo.GetSettlementTaskByID(taskID)
	if err != nil {
		return err
	}
	now := time.Now()
	task.ProcessedCount = int(processed)
	task.UpdateTime = now
	enrichTaskMeta(task, map[string]interface{}{
		"processed": processed,
	})
	return s.settlementRepo.UpdateSettlementTask(task)
}

func (s *settlementDataService) MarkTaskStage(taskID int64, stage string, processed int64, extras map[string]interface{}) error {
	task, err := s.settlementRepo.GetSettlementTaskByID(taskID)
	if err != nil {
		return err
	}
	now := time.Now()
	task.TaskStage = stage
	task.UpdateTime = now
	if processed >= 0 {
		task.ProcessedCount = int(processed)
	}
	patch := map[string]interface{}{
		"stage": stage,
	}
	if processed >= 0 {
		patch["processed"] = processed
	}
	for k, v := range extras {
		patch[k] = v
	}
	enrichTaskMeta(task, patch)
	return s.settlementRepo.UpdateSettlementTask(task)
}

func (s *settlementDataService) MarkTaskFailed(taskID int64, errMsg string) error {
	task, err := s.settlementRepo.GetSettlementTaskByID(taskID)
	if err != nil {
		return err
	}
	now := time.Now()
	task.Status = "failed"
	task.TaskStage = "failed"
	task.EndTime = &now
	task.ErrorMessage = errMsg
	enrichTaskMeta(task, map[string]interface{}{
		"stage": "failed",
		"error": errMsg,
	})
	return s.settlementRepo.UpdateSettlementTask(task)
}

func (s *settlementDataService) MarkTaskSuccess(taskID int64, processed int64) error {
	task, err := s.settlementRepo.GetSettlementTaskByID(taskID)
	if err != nil {
		return err
	}
	now := time.Now()
	task.Status = "success"
	task.TaskStage = "completed"
	task.EndTime = &now
	task.ProcessedCount = int(processed)
	if task.TotalCount <= 0 {
		task.TotalCount = int(processed)
	}
	task.UpdateTime = now
	enrichTaskMeta(task, map[string]interface{}{
		"stage":     "completed",
		"processed": processed,
		"total":     task.TotalCount,
	})
	return s.settlementRepo.UpdateSettlementTask(task)
}

func (s *settlementDataService) BuildScopeHash(filter SettlementCustomerFilter) string {
	var start, end string
	if filter.Start != nil {
		start = filter.Start.Format("2006-01-02")
	}
	if filter.End != nil {
		end = filter.End.Format("2006-01-02")
	}
	payload := fmt.Sprintf("%s|%s|%s|%s|%s", filter.Region, filter.CP, filter.School, start, end)
	sum := sha1.Sum([]byte(payload))
	return fmt.Sprintf("%x", sum)
}

func (s *settlementDataService) AcquireScopeLock(scopeHash string, timeoutSec int) (bool, error) {
	if strings.TrimSpace(scopeHash) == "" {
		return false, nil
	}
	if timeoutSec < 0 {
		timeoutSec = 0
	}
	lockName := "customer_recalc_scope_" + scopeHash
	var got int
	if err := model.DB.Raw("SELECT GET_LOCK(?, ?)", lockName, timeoutSec).Scan(&got).Error; err != nil {
		return false, err
	}
	return got == 1, nil
}

func (s *settlementDataService) ReleaseScopeLock(scopeHash string) error {
	if strings.TrimSpace(scopeHash) == "" {
		return nil
	}
	lockName := "customer_recalc_scope_" + scopeHash
	var released int
	return model.DB.Raw("SELECT RELEASE_LOCK(?)", lockName).Scan(&released).Error
}

func enrichTaskMeta(task *model.SettlementTask, patch map[string]interface{}) {
	if task == nil || len(patch) == 0 {
		return
	}
	meta := map[string]interface{}{}
	if strings.TrimSpace(task.TaskMeta) != "" {
		_ = json.Unmarshal([]byte(task.TaskMeta), &meta)
	}
	for k, v := range patch {
		meta[k] = v
	}
	b, err := json.Marshal(meta)
	if err != nil {
		return
	}
	task.TaskMeta = string(b)
}

func displayUserName(u model.User) string {
	if u.Alias != nil && strings.TrimSpace(*u.Alias) != "" {
		return strings.TrimSpace(*u.Alias)
	}
	if strings.TrimSpace(u.Username) != "" {
		return strings.TrimSpace(u.Username)
	}
	return fmt.Sprintf("用户#%d", u.ID)
}
