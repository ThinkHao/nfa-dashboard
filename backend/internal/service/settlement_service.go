package service

import (
	"fmt"
	"log"
	"time"

	"nfa-dashboard/internal/model"
	"nfa-dashboard/internal/notify"
	"nfa-dashboard/internal/repository"
)

// SettlementService 结算服务接口
type SettlementService interface {
	// 获取结算配置
	GetSettlementConfig() (*model.SettlementConfig, error)
	// 更新结算配置
	UpdateSettlementConfig(config *model.SettlementConfig) error
	// 创建结算任务
	CreateSettlementTask(taskType string, taskDate time.Time) (*model.SettlementTask, error)
	// 更新结算任务状态
	UpdateSettlementTaskStatus(taskID int64, status string, errorMsg string) error
	// 删除结算任务
	DeleteSettlementTask(id int64) error
	// 获取结算任务列表
	GetSettlementTasks(taskType, status string, startDate, endDate time.Time, limit, offset int) ([]model.SettlementTaskResponse, int64, error)
	// 获取结算任务详情
	GetSettlementTaskByID(id int64) (*model.SettlementTaskResponse, error)
	// 获取结算数据列表
	GetSettlements(filter model.SettlementFilter) ([]model.SettlementResponse, int64, error)
	// 执行日结算任务
	ExecuteDailySettlement(taskID int64, date time.Time) error
	// 执行周结算任务
	ExecuteWeeklySettlement(taskID int64, weekStartDate time.Time) error
	// 执行周结算任务（支持日期范围）
	ExecuteWeeklySettlementWithDateRange(taskID int64, startDate, endDate time.Time) error
	// GetValidSchoolComboCount 获取有效院校组合数
	GetValidSchoolComboCount(userID *uint64) (int64, error)
	// TryAdvisoryLock 供调度器抢占执行权（透传 repository）
	TryAdvisoryLock(name string) (release func(), ok bool, err error)
	// HasActiveOrSuccessTask 同类型同日期是否已有 pending/running/success/partial 任务
	HasActiveOrSuccessTask(taskType string, taskDate time.Time) (bool, error)
	// MarkStaleRunningTasks 清扫卡死任务（透传 repository）
	MarkStaleRunningTasks(staleAfter time.Duration) ([]model.SettlementTask, error)
}

func (s *settlementService) GetValidSchoolComboCount(userID *uint64) (int64, error) {
	return s.repo.CountValidSchoolCombos(userID)
}

// TryAdvisoryLock 供调度器抢占执行权（透传 repository）
func (s *settlementService) TryAdvisoryLock(name string) (func(), bool, error) {
	return s.repo.TryAdvisoryLock(name)
}

// HasActiveOrSuccessTask 同类型同日期是否已有 pending/running/success/partial 任务
func (s *settlementService) HasActiveOrSuccessTask(taskType string, taskDate time.Time) (bool, error) {
	filter := map[string]interface{}{
		"task_type":     taskType,
		"task_date":     taskDate,
		"status IN (?)": []string{"pending", "running", "success", "partial"},
	}
	_, count, err := s.repo.GetSettlementTasks(filter, 1, 0)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// MarkStaleRunningTasks 清扫卡死任务（透传 repository）
func (s *settlementService) MarkStaleRunningTasks(staleAfter time.Duration) ([]model.SettlementTask, error) {
	return s.repo.MarkStaleRunningTasks(staleAfter)
}

// settlementService 结算服务实现
type settlementService struct {
	repo     repository.SettlementRepository
	dataRepo repository.SettlementDataRepository
	notifier notify.Notifier
}

// NewSettlementService 创建结算服务实例
func NewSettlementService(repo repository.SettlementRepository, dataRepo repository.SettlementDataRepository, notifier notify.Notifier) SettlementService {
	return &settlementService{repo: repo, dataRepo: dataRepo, notifier: notifier}
}

// GetSettlementConfig 获取结算配置
func (s *settlementService) GetSettlementConfig() (*model.SettlementConfig, error) {
	return s.repo.GetSettlementConfig()
}

// UpdateSettlementConfig 更新结算配置
func (s *settlementService) UpdateSettlementConfig(config *model.SettlementConfig) error {
	return s.repo.UpdateSettlementConfig(config)
}

// CreateSettlementTask 创建结算任务（GORM Create 会回填自增 ID）
func (s *settlementService) CreateSettlementTask(taskType string, taskDate time.Time) (*model.SettlementTask, error) {
	now := time.Now()
	task := &model.SettlementTask{
		TaskType:       taskType,
		TaskDate:       taskDate,
		Status:         "pending",
		ProcessedCount: 0,
		CreateTime:     now,
		UpdateTime:     now,
	}
	if err := s.repo.CreateSettlementTask(task); err != nil {
		return nil, err
	}
	return task, nil
}

// UpdateSettlementTaskStatus 更新结算任务状态
func (s *settlementService) UpdateSettlementTaskStatus(taskID int64, status string, errorMsg string) error {
	task, err := s.repo.GetSettlementTaskByID(taskID)
	if err != nil {
		return err
	}

	task.Status = status
	if status == "running" {
		now := time.Now()
		task.StartTime = &now
	} else if status == "success" || status == "failed" {
		now := time.Now()
		task.EndTime = &now
	}

	if errorMsg != "" {
		task.ErrorMessage = errorMsg
	}

	return s.repo.UpdateSettlementTask(task)
}

// DeleteSettlementTask 删除结算任务
func (s *settlementService) DeleteSettlementTask(id int64) error {
	// 首先检查任务是否存在
	task, err := s.repo.GetSettlementTaskByID(id)
	if err != nil {
		return err
	}

	// 检查任务状态，不允许删除正在运行的任务
	if task.Status == "running" {
		return fmt.Errorf("不能删除正在运行的任务")
	}

	// 删除任务
	return s.repo.DeleteSettlementTask(id)
}

// GetSettlementTasks 获取结算任务列表
func (s *settlementService) GetSettlementTasks(taskType, status string, startDate, endDate time.Time, limit, offset int) ([]model.SettlementTaskResponse, int64, error) {
	filter := make(map[string]interface{})

	if taskType != "" {
		filter["task_type"] = taskType
	}

	if status != "" {
		filter["status"] = status
	}

	if !startDate.IsZero() {
		filter["task_date >= ?"] = startDate
	}

	if !endDate.IsZero() {
		filter["task_date <= ?"] = endDate
	}

	tasks, count, err := s.repo.GetSettlementTasks(filter, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	var responses []model.SettlementTaskResponse
	for _, task := range tasks {
		var st, et time.Time
		if task.StartTime != nil {
			st = *task.StartTime
		}
		if task.EndTime != nil {
			et = *task.EndTime
		}
		responses = append(responses, model.SettlementTaskResponse{
			ID:             task.ID,
			TaskType:       task.TaskType,
			TaskDate:       task.TaskDate,
			Status:         task.Status,
			TaskStage:      task.TaskStage,
			StartTime:      st,
			EndTime:        et,
			ProcessedCount: task.ProcessedCount,
			TotalCount:     task.TotalCount,
			ErrorMessage:   task.ErrorMessage,
			TaskMeta:       task.TaskMeta,
			CreateTime:     task.CreateTime,
			UpdateTime:     task.UpdateTime,
		})
	}

	return responses, count, nil
}

// GetSettlementTaskByID 获取结算任务详情
func (s *settlementService) GetSettlementTaskByID(id int64) (*model.SettlementTaskResponse, error) {
	task, err := s.repo.GetSettlementTaskByID(id)
	if err != nil {
		return nil, err
	}

	var st, et time.Time
	if task.StartTime != nil {
		st = *task.StartTime
	}
	if task.EndTime != nil {
		et = *task.EndTime
	}
	response := &model.SettlementTaskResponse{
		ID:             task.ID,
		TaskType:       task.TaskType,
		TaskDate:       task.TaskDate,
		Status:         task.Status,
		TaskStage:      task.TaskStage,
		StartTime:      st,
		EndTime:        et,
		ProcessedCount: task.ProcessedCount,
		TotalCount:     task.TotalCount,
		ErrorMessage:   task.ErrorMessage,
		TaskMeta:       task.TaskMeta,
		CreateTime:     task.CreateTime,
		UpdateTime:     task.UpdateTime,
	}

	return response, nil
}

// GetSettlements 获取结算数据列表
func (s *settlementService) GetSettlements(filter model.SettlementFilter) ([]model.SettlementResponse, int64, error) {
	return s.repo.GetSettlements(filter)
}

// executeDailySettlementInternal 内部方法，执行日结算的实际计算逻辑
// 返回结算数据、处理记录数和错误
func (s *settlementService) executeDailySettlementInternal(date time.Time) ([]model.SchoolSettlement, int, error) {
	log.Printf("开始计算 %s 的日结算数据", date.Format("2006-01-02"))

	processedCount := 0
	var settlements []model.SchoolSettlement

	validCombinations, err := s.repo.ListValidSchoolCombos(nil)
	if err != nil {
		return nil, 0, fmt.Errorf("获取有效学校组合失败: %v", err)
	}

	log.Printf("找到 %d 个有效的学校、地区、运营商组合", len(validCombinations))

	// 为每个有效组合计算95值
	for _, combo := range validCombinations {
		// 跳过字段为 NULL 或空字符串的无效院校组合（双重保证）
		if combo.SchoolID == "" || combo.Region == "" || combo.CP == "" {
			log.Printf("跳过无效院校组合: schoolID=%s, region=%s, cp=%s", combo.SchoolID, combo.Region, combo.CP)
			continue
		}
		// 计算95值，传入学校ID、地区和运营商
		settlement, err := s.repo.CalculateDaily95WithRegionAndCP(date, combo.SchoolID, combo.Region, combo.CP)
		if err != nil {
			log.Printf("计算学校 %s 在地区 %s 运营商 %s 的日95值失败: %v",
				combo.SchoolName, combo.Region, combo.CP, err)
			continue
		}

		if settlement != nil {
			settlements = append(settlements, *settlement)
		}
		// 处理计数统一为“已尝试处理的组合数”（与 total_count 口径一致）
		processedCount++
	}

	log.Printf("完成 %s 的日结算计算，共生成 %d 条数据", date.Format("2006-01-02"), processedCount)
	return settlements, processedCount, nil
}

// ExecuteDailySettlement 执行日结算任务
func (s *settlementService) ExecuteDailySettlement(taskID int64, date time.Time) error {
	// 标记运行中
	if err := s.UpdateSettlementTaskStatus(taskID, "running", ""); err != nil {
		return fmt.Errorf("更新任务状态失败: %v", err)
	}

	// 分段执行并上报进度
	processedCount := 0
	var settlements []model.SchoolSettlement
	validCombinations, err := s.repo.ListValidSchoolCombos(nil)
	if err != nil {
		_ = s.UpdateSettlementTaskStatus(taskID, "failed", fmt.Sprintf("获取有效学校组合失败: %v", err))
		return fmt.Errorf("获取有效学校组合失败: %v", err)
	}
	batch := 20
	for i, combo := range validCombinations {
		if combo.SchoolID == "" || combo.Region == "" || combo.CP == "" {
			continue
		}
		settlement, calErr := s.repo.CalculateDaily95WithRegionAndCP(date, combo.SchoolID, combo.Region, combo.CP)
		if calErr == nil && settlement != nil {
			settlements = append(settlements, *settlement)
		}
		processedCount++
		// 每批上报一次处理进度
		if processedCount%batch == 0 || i == len(validCombinations)-1 {
			if task, e := s.repo.GetSettlementTaskByID(taskID); e == nil {
				task.ProcessedCount = processedCount
				_ = s.repo.UpdateSettlementTask(task)
			}
		}
	}

	// 保存
	if len(settlements) > 0 {
		if err := s.repo.BatchCreateSettlements(settlements); err != nil {
			_ = s.UpdateSettlementTaskStatus(taskID, "failed", fmt.Sprintf("保存结算数据失败: %v", err))
			return fmt.Errorf("保存结算数据失败: %v", err)
		}
	}

	// 标记成功
	task, err := s.repo.GetSettlementTaskByID(taskID)
	if err != nil {
		return fmt.Errorf("获取任务信息失败: %v", err)
	}
	task.Status = "success"
	now := time.Now()
	task.EndTime = &now
	task.ProcessedCount = processedCount
	if err := s.repo.UpdateSettlementTask(task); err != nil {
		return fmt.Errorf("更新任务状态失败: %v", err)
	}

	s.triggerCustomerInitAfter("daily", date, date)

	return nil
}

// ExecuteWeeklySettlement 执行周结算任务（以开始日期+6天为结束）
func (s *settlementService) ExecuteWeeklySettlement(taskID int64, weekStartDate time.Time) error {
	weekEndDate := weekStartDate.AddDate(0, 0, 6)
	return s.ExecuteWeeklySettlementWithDateRange(taskID, weekStartDate, weekEndDate)
}

// ExecuteWeeklySettlementWithDateRange 执行周结算任务（支持自定义日期范围）
func (s *settlementService) ExecuteWeeklySettlementWithDateRange(taskID int64, startDate, endDate time.Time) error {
	if err := s.UpdateSettlementTaskStatus(taskID, "running", ""); err != nil {
		return fmt.Errorf("更新任务状态失败: %v", err)
	}

	var all []model.SchoolSettlement
	total := 0
	for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
		ds, cnt, err := s.executeDailySettlementInternal(d)
		if err != nil {
			continue
		}
		all = append(all, ds...)
		total += cnt
		// 每完成一天后，更新一次已处理数量，便于前端精准显示进度与 ETA
		if task, e := s.repo.GetSettlementTaskByID(taskID); e == nil {
			task.ProcessedCount = total
			_ = s.repo.UpdateSettlementTask(task)
		}
	}
	if len(all) > 0 {
		if err := s.repo.BatchCreateSettlements(all); err != nil {
			_ = s.UpdateSettlementTaskStatus(taskID, "failed", fmt.Sprintf("保存结算数据失败: %v", err))
			return fmt.Errorf("保存结算数据失败: %v", err)
		}
	}
	task, err := s.repo.GetSettlementTaskByID(taskID)
	if err != nil {
		return fmt.Errorf("获取任务信息失败: %v", err)
	}
	task.Status = "success"
	now := time.Now()
	task.EndTime = &now
	task.ProcessedCount = total
	if err := s.repo.UpdateSettlementTask(task); err != nil {
		return fmt.Errorf("更新任务状态失败: %v", err)
	}

	// 周结算完成后按配置触发初算（不标记复算）
	s.triggerCustomerInitAfter("weekly", startDate, endDate)
	return nil
}

// triggerCustomerInitAfter 按配置在结算完成后异步触发客户侧初算回填
// source: "daily" | "weekly"，分别受 RecalcAfterDaily / RecalcAfterWeekly 控制
func (s *settlementService) triggerCustomerInitAfter(source string, start, end time.Time) {
	go func() {
		cfg, err := s.repo.GetSettlementConfig()
		if err != nil || !cfg.Enabled {
			return
		}
		if source == "daily" && !cfg.RecalcAfterDaily {
			return
		}
		if source == "weekly" && !cfg.RecalcAfterWeekly {
			return
		}
		now := time.Now()
		init := &model.SettlementTask{TaskType: "customer_init", TaskDate: start, Status: "running", StartTime: &now, CreateTime: now, UpdateTime: now}
		if err := s.repo.CreateSettlementTask(init); err != nil {
			return
		}
		rangeLabel := fmt.Sprintf("%s ~ %s", start.Format("2006-01-02"), end.Format("2006-01-02"))
		fail := func(msg string) {
			endAt := time.Now()
			init.Status = "failed"
			init.EndTime = &endAt
			init.ErrorMessage = msg
			_ = s.repo.UpdateSettlementTask(init)
			log.Printf("customer_init task failed: task_id=%d range=%s: %s", init.ID, rangeLabel, msg)
			notify.SendAsync(s.notifier, "客户结算回填失败", fmt.Sprintf("任务 #%d (%s)：%s", init.ID, rangeLabel, msg))
		}
		srcCount, err := s.dataRepo.CountSchoolSettlementRows("", "", "", start, end)
		if err != nil {
			fail(fmt.Sprintf("统计源数据失败: %v", err))
			return
		}
		affected, err := s.dataRepo.BackfillFromSchoolSettlement("", "", "", start, end, false, nil)
		if err != nil {
			fail(err.Error())
			return
		}
		if shouldFailCustomerInitOnZeroAffected(srcCount, affected) {
			fail(fmt.Sprintf("源表有数据但回填0条（疑似日期边界异常）: source=%d, affected=%d", srcCount, affected))
			return
		}
		endAt := time.Now()
		init.Status = "success"
		init.EndTime = &endAt
		init.ProcessedCount = int(affected)
		_ = s.repo.UpdateSettlementTask(init)
	}()
}

func shouldFailCustomerInitOnZeroAffected(srcCount, affected int64) bool {
	return srcCount > 0 && affected == 0
}
