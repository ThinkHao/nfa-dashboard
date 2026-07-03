package scheduler

import (
	"fmt"
	"log"
	"time"

	"nfa-dashboard/internal/notify"
	"nfa-dashboard/internal/service"
)

const (
	schedulerLockName  = "nfa:settlement:scheduler"
	staleTaskThreshold = 30 * time.Minute
)

// SettlementScheduler 结算调度器
type SettlementScheduler struct {
	settlementService service.SettlementService
	nodeService       service.EDCNodeSettlementService
	notifier          notify.Notifier
	running           bool
	stopChan          chan struct{}
}

// NewSettlementScheduler 创建结算调度器实例
func NewSettlementScheduler(settlementService service.SettlementService, nodeService service.EDCNodeSettlementService, notifier notify.Notifier) *SettlementScheduler {
	return &SettlementScheduler{
		settlementService: settlementService,
		nodeService:       nodeService,
		notifier:          notifier,
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
	release, ok, err := s.settlementService.TryAdvisoryLock(schedulerLockName)
	if err != nil {
		log.Printf("获取调度器锁失败: %v", err)
		return
	}
	if !ok {
		// 其它实例正在调度，本实例跳过本轮
		return
	}
	defer release()

	s.sweepStaleTasks()

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

	// 如果总开关未启用，则不执行任务
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

	// 检查是否需要执行每日结算任务（需 daily_enabled=true）
	if config.DailyEnabled && currentHour == dailyHour && currentMinute == dailyMinute {
		// 计算前一天的日期
		yesterday := now.AddDate(0, 0, -1)
		date := time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 0, 0, 0, 0, now.Location())

		log.Printf("开始执行每日结算任务，计算日期: %s", date.Format("2006-01-02"))
		s.createAndRun("daily", date, func(taskID int64) {
			if err := s.settlementService.ExecuteDailySettlement(taskID, date); err != nil {
				log.Printf("执行每日结算任务失败: %v", err)
			}
		})

		// 更新上次执行时间
		config.LastExecuteTime = now
		err = s.settlementService.UpdateSettlementConfig(config)
		if err != nil {
			log.Printf("更新结算配置失败: %v", err)
		}
	}

	// 检查是否需要执行每周结算任务（需 weekly_enabled=true）
	if config.WeeklyEnabled && currentWeekday == config.WeeklyDay && currentHour == weeklyHour && currentMinute == weeklyMinute {
		// 计算上一周的开始日期（上周一）
		startDate := previousWeekStart(now)

		log.Printf("开始执行每周结算任务，计算开始日期: %s", startDate.Format("2006-01-02"))
		s.createAndRun("weekly", startDate, func(taskID int64) {
			if err := s.settlementService.ExecuteWeeklySettlement(taskID, startDate); err != nil {
				log.Printf("执行每周结算任务失败: %v", err)
			}
		})

		// 更新上次执行时间
		config.LastExecuteTime = now
		err = s.settlementService.UpdateSettlementConfig(config)
		if err != nil {
			log.Printf("更新结算配置失败: %v", err)
		}
	}

	if s.nodeService == nil {
		return
	}

	if config.NodeDailyEnabled {
		nodeDailyHour, nodeDailyMinute, err := parseTimeString(defaultScheduleTime(config.NodeDailyTime, "03:00"))
		if err != nil {
			log.Printf("解析EDC节点每日结算时间失败: %v", err)
			return
		}
		if currentHour == nodeDailyHour && currentMinute == nodeDailyMinute {
			yesterday := now.AddDate(0, 0, -1)
			date := time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 0, 0, 0, 0, now.Location())
			log.Printf("开始执行EDC节点每日结算任务，计算日期: %s", date.Format("2006-01-02"))
			s.createAndRun("node_daily95", date, func(taskID int64) {
				if err := s.nodeService.ExecuteDailyTask(taskID, date); err != nil {
					log.Printf("执行EDC节点每日结算任务失败: %v", err)
				}
			})
			config.LastExecuteTime = now
			if err := s.settlementService.UpdateSettlementConfig(config); err != nil {
				log.Printf("更新结算配置失败: %v", err)
			}
		}
	}

	if config.NodeMonthlyEnabled {
		nodeMonthlyHour, nodeMonthlyMinute, err := parseTimeString(defaultScheduleTime(config.NodeMonthlyTime, "04:00"))
		if err != nil {
			log.Printf("解析EDC节点月结算时间失败: %v", err)
			return
		}
		monthlyDay := config.NodeMonthlyDay
		if monthlyDay <= 0 {
			monthlyDay = 1
		}
		if now.Day() != monthlyDay || currentHour != nodeMonthlyHour || currentMinute != nodeMonthlyMinute {
			return
		}
		lastMonth := now.AddDate(0, -1, 0)
		month := time.Date(lastMonth.Year(), lastMonth.Month(), 1, 0, 0, 0, 0, now.Location())
		log.Printf("开始执行EDC节点月结算任务，计算月份: %s", month.Format("2006-01"))
		s.createAndRun("node_monthly95", month, func(taskID int64) {
			if err := s.nodeService.ExecuteMonthlyTask(taskID, month); err != nil {
				log.Printf("执行EDC节点月结算任务失败: %v", err)
			}
		})
		config.LastExecuteTime = now
		if err := s.settlementService.UpdateSettlementConfig(config); err != nil {
			log.Printf("更新结算配置失败: %v", err)
		}
	}
}

// sweepStaleTasks 每 10 分钟清扫一次卡死任务并告警
func (s *SettlementScheduler) sweepStaleTasks() {
	if time.Now().Minute()%10 != 0 {
		return
	}
	stale, err := s.settlementService.MarkStaleRunningTasks(staleTaskThreshold)
	if err != nil {
		log.Printf("清扫卡死任务失败: %v", err)
		return
	}
	for _, t := range stale {
		log.Printf("任务 #%d (%s) 无进度更新，已标记为中断", t.ID, t.TaskType)
		notify.SendAsync(s.notifier, "结算任务中断",
			fmt.Sprintf("任务 #%d (%s, %s) 超过 30 分钟无进度更新，已标记为中断，请检查后重新发起。", t.ID, t.TaskType, t.TaskDate.Format("2006-01-02")))
	}
}

// createAndRun 创建任务并异步执行；同类型同日期已有活跃/成功任务时跳过（防多实例或重复触发）
func (s *SettlementScheduler) createAndRun(taskType string, taskDate time.Time, run func(taskID int64)) {
	exists, err := s.settlementService.HasActiveOrSuccessTask(taskType, taskDate)
	if err != nil {
		log.Printf("检查已有任务失败: type=%s date=%s err=%v", taskType, taskDate.Format("2006-01-02"), err)
		return
	}
	if exists {
		log.Printf("已存在同日期任务，跳过自动创建: type=%s date=%s", taskType, taskDate.Format("2006-01-02"))
		return
	}
	task, err := s.settlementService.CreateSettlementTask(taskType, taskDate)
	if err != nil {
		log.Printf("创建任务失败: type=%s err=%v", taskType, err)
		return
	}
	go run(task.ID)
}

func defaultScheduleTime(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func previousWeekStart(now time.Time) time.Time {
	daysSinceMonday := (int(now.Weekday()) + 6) % 7
	thisMonday := now.AddDate(0, 0, -daysSinceMonday)
	lastMonday := thisMonday.AddDate(0, 0, -7)
	return time.Date(lastMonday.Year(), lastMonday.Month(), lastMonday.Day(), 0, 0, 0, 0, now.Location())
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
