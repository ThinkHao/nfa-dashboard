package scheduler

import (
	"fmt"
	"log"
	"time"

	"nfa-dashboard/internal/service"
)

// SettlementScheduler 结算调度器
type SettlementScheduler struct {
	settlementService service.SettlementService
	running           bool
	stopChan          chan struct{}
}

// NewSettlementScheduler 创建结算调度器实例
func NewSettlementScheduler(settlementService service.SettlementService) *SettlementScheduler {
	return &SettlementScheduler{
		settlementService: settlementService,
		running:           false,
		stopChan:          make(chan struct{}),
	}
}

// Start 启动调度器
func (s *SettlementScheduler) Start() {
	if s.running {
		log.Println("结算调度器已经在运行")
		return
	}

	s.running = true
	go s.run()
	log.Println("结算调度器已启动")
}

// Stop 停止调度器
func (s *SettlementScheduler) Stop() {
	if !s.running {
		log.Println("结算调度器未运行")
		return
	}

	s.stopChan <- struct{}{}
	s.running = false
	log.Println("结算调度器已停止")
}

// run 运行调度器
func (s *SettlementScheduler) run() {
	ticker := time.NewTicker(1 * time.Minute) // 每分钟检查一次
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.checkAndExecuteTasks()
		case <-s.stopChan:
			return
		}
	}
}

// checkAndExecuteTasks 检查并执行定时任务
func (s *SettlementScheduler) checkAndExecuteTasks() {
	// 获取当前时间
	now := time.Now()
	currentHour := now.Hour()
	currentMinute := now.Minute()
	currentWeekday := int(now.Weekday())
	if currentWeekday == 0 {
		currentWeekday = 7 // 将周日(0)转换为7
	}

	// 获取结算配置
	config, err := s.settlementService.GetSettlementConfig()
	if err != nil {
		log.Printf("获取结算配置失败: %v", err)
		return
	}

	// 如果未启用，则不执行任务
	if !config.Enabled {
		return
	}

	// 解析配置的时间
	dailyHour, dailyMinute, err := parseTimeString(config.DailyTime)
	if err != nil {
		log.Printf("解析每日结算时间失败: %v", err)
		return
	}

	weeklyHour, weeklyMinute, err := parseTimeString(config.WeeklyTime)
	if err != nil {
		log.Printf("解析每周结算时间失败: %v", err)
		return
	}

	// 检查是否需要执行每日结算任务
	if currentHour == dailyHour && currentMinute == dailyMinute {
		// 计算前一天的日期
		yesterday := now.AddDate(0, 0, -1)
		date := time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 0, 0, 0, 0, now.Location())

		log.Printf("开始执行每日结算任务，计算日期: %s", date.Format("2006-01-02"))
		
		// 创建并执行每日结算任务
		task, err := s.settlementService.CreateSettlementTask("daily", date)
		if err != nil {
			log.Printf("创建每日结算任务失败: %v", err)
			return
		}

		go func() {
			err := s.settlementService.ExecuteDailySettlement(task.ID, date)
			if err != nil {
				log.Printf("执行每日结算任务失败: %v", err)
			}
		}()

		// 更新上次执行时间
		config.LastExecuteTime = now
		err = s.settlementService.UpdateSettlementConfig(config)
		if err != nil {
			log.Printf("更新结算配置失败: %v", err)
		}
	}

	// 检查是否需要执行每周结算任务
	if currentWeekday == config.WeeklyDay && currentHour == weeklyHour && currentMinute == weeklyMinute {
		// 计算上一周的开始日期（上周一）
		daysToLastMonday := (int(now.Weekday()) + 6) % 7
		if daysToLastMonday == 0 {
			daysToLastMonday = 7
		}
		lastMonday := now.AddDate(0, 0, -daysToLastMonday-7)
		startDate := time.Date(lastMonday.Year(), lastMonday.Month(), lastMonday.Day(), 0, 0, 0, 0, now.Location())

		log.Printf("开始执行每周结算任务，计算开始日期: %s", startDate.Format("2006-01-02"))
		
		// 创建并执行每周结算任务
		task, err := s.settlementService.CreateSettlementTask("weekly", startDate)
		if err != nil {
			log.Printf("创建每周结算任务失败: %v", err)
			return
		}

		go func() {
			err := s.settlementService.ExecuteWeeklySettlement(task.ID, startDate)
			if err != nil {
				log.Printf("执行每周结算任务失败: %v", err)
			}
		}()

		// 更新上次执行时间
		config.LastExecuteTime = now
		err = s.settlementService.UpdateSettlementConfig(config)
		if err != nil {
			log.Printf("更新结算配置失败: %v", err)
		}
	}
}

// parseTimeString 解析时间字符串（格式：HH:MM）
func parseTimeString(timeStr string) (int, int, error) {
	var hour, minute int
	_, err := fmt.Sscanf(timeStr, "%d:%d", &hour, &minute)
	if err != nil {
		return 0, 0, err
	}

	if hour < 0 || hour > 23 || minute < 0 || minute > 59 {
		return 0, 0, fmt.Errorf("无效的时间格式: %s", timeStr)
	}

	return hour, minute, nil
}
