package service

import (
	"fmt"
	"log"
	"time"

	"nfa-dashboard/internal/model"
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
	// GetDailySettlementDetails 获取日95明细数据列表
	GetDailySettlementDetails(filter model.SettlementFilter) ([]model.DailySettlementDetail, int64, error) // 假设 model.DailySettlementDetail 存在
}

// GetDailySettlementDetails 获取日95明细数据列表
func (s *settlementService) GetDailySettlementDetails(filter model.SettlementFilter) ([]model.DailySettlementDetail, int64, error) {
	return s.repo.GetDailySettlementDetails(filter)
}

// settlementService 结算服务实现
type settlementService struct {
	repo repository.SettlementRepository
}

// NewSettlementService 创建结算服务实例
func NewSettlementService(repo repository.SettlementRepository) SettlementService {
	return &settlementService{
		repo: repo,
	}
}

// GetSettlementConfig 获取结算配置
func (s *settlementService) GetSettlementConfig() (*model.SettlementConfig, error) {
	return s.repo.GetSettlementConfig()
}

// UpdateSettlementConfig 更新结算配置
func (s *settlementService) UpdateSettlementConfig(config *model.SettlementConfig) error {
	return s.repo.UpdateSettlementConfig(config)
}

// CreateSettlementTask 创建结算任务
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

	err := s.repo.CreateSettlementTask(task)
	if err != nil {
		return nil, err
	}

	// 获取创建后的任务，确保有正确的ID
	var tasks []model.SettlementTask
	filter := map[string]interface{}{
		"task_type": taskType,
		"task_date": taskDate,
	}
	tasks, _, err = s.repo.GetSettlementTasks(filter, 1, 0)
	if err != nil || len(tasks) == 0 {
		// 如果查询失败，返回原始任务
		return task, nil
	}

	// 返回最新创建的任务
	return &tasks[0], nil
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
			StartTime:      st,
			EndTime:        et,
			ProcessedCount: task.ProcessedCount,
			ErrorMessage:   task.ErrorMessage,
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
		StartTime:      st,
		EndTime:        et,
		ProcessedCount: task.ProcessedCount,
		ErrorMessage:   task.ErrorMessage,
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

	// 直接获取所有存在的学校、地区、运营商的有效组合
	// 构建SQL，获取所有唯一的学校ID、地区、运营商组合
	type SchoolRegionCP struct {
		SchoolID   string
		SchoolName string
		Region     string
		CP         string
	}

	var validCombinations []SchoolRegionCP
	query := `
SELECT DISTINCT school_id, school_name, region, cp
FROM nfa_school
WHERE school_id IS NOT NULL AND school_id <> ''
  AND school_name IS NOT NULL AND school_name <> ''
  AND region IS NOT NULL AND region <> ''
  AND cp IS NOT NULL AND cp <> ''`
	err := model.DB.Raw(query).Scan(&validCombinations).Error
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
			processedCount++
		}
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

	// 计算
	settlements, processedCount, err := s.executeDailySettlementInternal(date)
	if err != nil {
		_ = s.UpdateSettlementTaskStatus(taskID, "failed", fmt.Sprintf("执行日结算失败: %v", err))
		return fmt.Errorf("执行日结算失败: %v", err)
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

	go func(runDate time.Time) {
		cfg, cfgErr := s.repo.GetSettlementConfig()
		if cfgErr != nil || !cfg.Enabled || !cfg.RecalcAfterDaily {
			return
		}
		init := &model.SettlementTask{TaskType: "customer_init", TaskDate: runDate, Status: "running", StartTime: ptrTime(time.Now()), CreateTime: time.Now(), UpdateTime: time.Now()}
		if err := model.DB.Create(init).Error; err != nil {
			return
		}
		dataRepo := repository.NewSettlementDataRepository()
		affected, recErr := dataRepo.BackfillFromSchoolSettlement("", "", "", runDate, runDate, false)
		if recErr != nil {
			_ = model.DB.Model(&model.SettlementTask{}).Where("id = ?", init.ID).Updates(map[string]interface{}{"status": "failed", "end_time": time.Now(), "error_message": recErr.Error()}).Error
			return
		}
		_ = model.DB.Model(&model.SettlementTask{}).Where("id = ?", init.ID).Updates(map[string]interface{}{"status": "success", "end_time": time.Now(), "processed_count": affected}).Error
	}(date)

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
	go func(sdate, edate time.Time) {
		cfg, e := s.repo.GetSettlementConfig()
		if e != nil || !cfg.Enabled || !cfg.RecalcAfterWeekly {
			return
		}
		init := &model.SettlementTask{TaskType: "customer_init", TaskDate: sdate, Status: "running", StartTime: ptrTime(time.Now()), CreateTime: time.Now(), UpdateTime: time.Now()}
		if err := model.DB.Create(init).Error; err != nil {
			return
		}
		dataRepo := repository.NewSettlementDataRepository()
		affected, recErr := dataRepo.BackfillFromSchoolSettlement("", "", "", sdate, edate, false)
		if recErr != nil {
			_ = model.DB.Model(&model.SettlementTask{}).Where("id = ?", init.ID).Updates(map[string]interface{}{"status": "failed", "end_time": time.Now(), "error_message": recErr.Error()}).Error
			return
		}
		_ = model.DB.Model(&model.SettlementTask{}).Where("id = ?", init.ID).Updates(map[string]interface{}{"status": "success", "end_time": time.Now(), "processed_count": affected}).Error
	}(startDate, endDate)
	return nil
}

// 辅助：取指针
func ptrTime(t time.Time) *time.Time { return &t }
