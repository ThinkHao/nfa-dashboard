# 结算平台可靠性与代码健康度改造 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 让 NFA Dashboard 的结算链路具备"单实例调度、任务可恢复、失败可见可告警、缺数可发现"的可靠性底座，同时优化日结算性能并拆分巨型文件。

**Architecture:** 后端在 repository 层新增 MySQL 命名锁（GET_LOCK）与卡死任务清扫，scheduler 每 tick 先抢锁再调度并按 (task_type, task_date) 去重；新增 `internal/notify` 包（飞书 webhook）承接失败告警；日结算从"逐组合 N 次查询"改为"单次流式扫描分组计算 95"；新增完整性巡检服务（源表 vs 客户表逐日行数对基线）暴露 API 并接入调度器定时告警。前端补 `partial`/`interrupted` 任务状态与结算数据页缺数提示。

**Tech Stack:** Go 1.24 + Gin + GORM (MySQL 5.7)、Vue 3 + Element Plus、Vitest、go test（MySQL 集成测试用 `NFA_TEST_MYSQL_DSN` 环境变量门控，模式参考 `backend/internal/repository/settlement_slot_backfill_test.go`）。

**关键决策（已定，不要在执行时重新发明）：**
- 调度互斥用 MySQL `GET_LOCK`（锁名 `nfa:settlement:scheduler`），不引入 Redis/etcd。
- 任务防重：调度器创建前查同 `(task_type, task_date)` 是否已有 `pending/running/success/partial` 任务，**不加唯一索引**（手动重跑失败任务是合法场景）。
- 卡死判定：`status='running'` 且 `update_time` 超过 30 分钟未变（任务执行中每批都会刷 update_time）→ 标记 `interrupted`。
- 新任务状态只是 varchar 里的新字符串值（`partial`、`interrupted`），**不需要 SQL 迁移**。
- 完整性判定：对区间内非零日行数取中位数为基线，`count==0` → `missing`，`count < 80% 基线` → `low`。
- 巡检时间固定 09:00（日结算 02:00 + 客户回填之后），常量即可，不做配置。
- 所有 commit 用 `git commit -m "<type>(<scope>): <subject>" -m "Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>"` 两段式。

**执行前置检查：** 每个任务完成后运行 `cd backend && go build ./... && go test ./internal/...`（前端任务则 `cd frontend/frontend && npm run type-check && npm run test:unit`），全绿再 commit。

---

## Phase 0：基础设施

### Task 1: 配置扩展（scheduler.enabled / alert.feishu_webhook_url）

**Files:**
- Modify: `backend/config/config.go`
- Modify: `backend/config/config.yaml`（本地开发配置，追加两段）
- Test: `backend/config/config_test.go`（新建）

- [ ] **Step 1: 写失败测试**

创建 `backend/config/config_test.go`：

```go
package config

import "testing"

func TestSchedulerAndAlertGetters(t *testing.T) {
	AppConfig = Config{}
	AppConfig.Scheduler.Enabled = true
	AppConfig.Alert.FeishuWebhookURL = "https://example.com/hook"
	if !IsSchedulerEnabled() {
		t.Fatal("IsSchedulerEnabled should be true")
	}
	if GetFeishuWebhookURL() != "https://example.com/hook" {
		t.Fatal("GetFeishuWebhookURL mismatch")
	}
}
```

- [ ] **Step 2: 运行确认编译失败**

Run: `cd backend && go test ./config`
Expected: FAIL（`AppConfig.Scheduler` undefined）

- [ ] **Step 3: 实现配置结构**

在 `config.go` 的 `Config` struct 中追加两个字段：

```go
type Config struct {
	Server          ServerConfig          `mapstructure:"server"`
	Database        DatabaseConfig        `mapstructure:"database"`
	Redis           RedisConfig           `mapstructure:"redis"`
	Auth            AuthConfig            `mapstructure:"auth"`
	Binding         BindingConfig         `mapstructure:"binding"`
	RatesOwnerRoles RatesOwnerRolesConfig `mapstructure:"rates_owner_roles"`
	Scheduler       SchedulerConfig       `mapstructure:"scheduler"`
	Alert           AlertConfig           `mapstructure:"alert"`
}

// SchedulerConfig 结算调度器开关（多实例部署时只允许一个实例开启，或全部开启依赖 DB 锁互斥）
type SchedulerConfig struct {
	Enabled bool `mapstructure:"enabled"`
}

// AlertConfig 告警通道配置
type AlertConfig struct {
	FeishuWebhookURL string `mapstructure:"feishu_webhook_url"`
}
```

在 `LoadConfig()` 中 `viper.SetDefault("server.port", 8081)` 之后追加：

```go
	viper.SetDefault("scheduler.enabled", true)
```

在 BindEnv 区块（`auth.refresh_token_ttl_minutes` 那行之后）追加：

```go
	_ = viper.BindEnv("scheduler.enabled", "SCHEDULER_ENABLED")
	_ = viper.BindEnv("alert.feishu_webhook_url", "ALERT_FEISHU_WEBHOOK_URL")
```

文件末尾追加 getter：

```go
// IsSchedulerEnabled 是否在本实例启动结算调度器
func IsSchedulerEnabled() bool {
	return AppConfig.Scheduler.Enabled
}

// GetFeishuWebhookURL 飞书告警 webhook 地址，空串表示未配置
func GetFeishuWebhookURL() string {
	return AppConfig.Alert.FeishuWebhookURL
}
```

- [ ] **Step 4: config.yaml 追加默认值**

在 `backend/config/config.yaml` 末尾追加：

```yaml
scheduler:
  enabled: true

alert:
  feishu_webhook_url: ""
```

- [ ] **Step 5: 验证通过**

Run: `cd backend && go test ./config && go build ./...`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add backend/config/config.go backend/config/config.yaml backend/config/config_test.go
git commit -m "feat(config): add scheduler.enabled and alert.feishu_webhook_url" -m "Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>"
```

---

### Task 2: notify 包（飞书 webhook 通知器）

**Files:**
- Create: `backend/internal/notify/notify.go`
- Test: `backend/internal/notify/notify_test.go`

- [ ] **Step 1: 写失败测试**

创建 `backend/internal/notify/notify_test.go`：

```go
package notify

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFeishuNotifierSendsPayload(t *testing.T) {
	var got map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&got)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	n := NewFromConfig(srv.URL)
	if err := n.Send("测试标题", "测试正文"); err != nil {
		t.Fatal(err)
	}
	if got["msg_type"] != "text" {
		t.Fatalf("unexpected payload: %+v", got)
	}
	content, _ := got["content"].(map[string]interface{})
	text, _ := content["text"].(string)
	if !strings.Contains(text, "测试标题") || !strings.Contains(text, "测试正文") {
		t.Fatalf("text missing title/body: %s", text)
	}
}

func TestNoopNotifierWhenURLEmpty(t *testing.T) {
	if err := NewFromConfig("").Send("a", "b"); err != nil {
		t.Fatalf("noop notifier should never fail: %v", err)
	}
}

func TestFeishuNotifierRejectsBadStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()
	if err := NewFromConfig(srv.URL).Send("a", "b"); err == nil {
		t.Fatal("expected error on non-2xx response")
	}
}
```

- [ ] **Step 2: 运行确认失败**

Run: `cd backend && go test ./internal/notify`
Expected: FAIL（package 不存在）

- [ ] **Step 3: 实现**

创建 `backend/internal/notify/notify.go`：

```go
package notify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// Notifier 告警通知接口
type Notifier interface {
	Send(title, text string) error
}

// NewFromConfig 根据 webhook URL 创建通知器；URL 为空时返回 no-op 实现
func NewFromConfig(webhookURL string) Notifier {
	if webhookURL == "" {
		return noopNotifier{}
	}
	return &feishuNotifier{url: webhookURL, client: &http.Client{Timeout: 5 * time.Second}}
}

type noopNotifier struct{}

func (noopNotifier) Send(title, text string) error { return nil }

type feishuNotifier struct {
	url    string
	client *http.Client
}

func (n *feishuNotifier) Send(title, text string) error {
	payload := map[string]interface{}{
		"msg_type": "text",
		"content":  map[string]string{"text": fmt.Sprintf("【NFA结算告警】%s\n%s", title, text)},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	resp, err := n.client.Post(n.url, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("feishu webhook 返回状态码 %d", resp.StatusCode)
	}
	return nil
}

// SendAsync 异步发送并记录错误，避免阻塞结算主流程
func SendAsync(n Notifier, title, text string) {
	if n == nil {
		return
	}
	go func() {
		if err := n.Send(title, text); err != nil {
			log.Printf("发送告警失败: title=%s err=%v", title, err)
		}
	}()
}
```

- [ ] **Step 4: 验证通过**

Run: `cd backend && go test ./internal/notify`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add backend/internal/notify
git commit -m "feat(notify): add feishu webhook notifier with noop fallback" -m "Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>"
```

---

### Task 3: SettlementService 依赖注入修复（dataRepo + notifier 进构造函数）

背景：`settlement_service.go:336` 和 `:428` 在 goroutine 里直接 `repository.NewSettlementDataRepository()`，破坏手动 DI 模式。本任务只改构造与字段，不改行为。

**Files:**
- Modify: `backend/internal/service/settlement_service.go`
- Modify: `backend/internal/bootstrap/app.go:34`

- [ ] **Step 1: 修改 service 结构与构造函数**

`settlement_service.go` 中：

```go
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
```

import 块加 `"nfa-dashboard/internal/notify"`。

- [ ] **Step 2: 替换两处内联构造**

`ExecuteDailySettlement`（约 336 行）与 `ExecuteWeeklySettlementWithDateRange`（约 428 行）goroutine 中的：

```go
dataRepo := repository.NewSettlementDataRepository()
```

删掉该行，后续 `dataRepo.` 调用改为 `s.dataRepo.`。

- [ ] **Step 3: 更新 app.go 装配**

`bootstrap/app.go` 中原第 33-34 行改为（`settlementDataRepo` 的声明从原 41 行**上移**到这里，原 41 行删除）：

```go
	settlementRepo := repository.NewSettlementRepository()
	settlementDataRepo := repository.NewSettlementDataRepository()
	notifier := notify.NewFromConfig(config.GetFeishuWebhookURL())
	settlementService := service.NewSettlementService(settlementRepo, settlementDataRepo, notifier)
```

import 块加 `"nfa-dashboard/config"` 与 `"nfa-dashboard/internal/notify"`。

- [ ] **Step 4: 全量编译 + 测试**

Run: `cd backend && go build ./... && go test ./internal/...`
Expected: PASS（若有测试直接调 `NewSettlementService`，同步补 `nil, notify.NewFromConfig("")` 参数；当前 grep 确认仅 app.go 调用）

- [ ] **Step 5: Commit**

```bash
git add backend/internal/service/settlement_service.go backend/internal/bootstrap/app.go
git commit -m "refactor(settlement): inject data repo and notifier into settlement service" -m "Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>"
```

---

## Phase 1：调度与任务可靠性

### Task 4: 仓库层 TryAdvisoryLock + MarkStaleRunningTasks

**Files:**
- Modify: `backend/internal/repository/settlement_repository.go`（接口 + 实现）
- Modify: `backend/internal/controller/customer_rate_import_task_controller_test.go`（stub 兼容）
- Test: `backend/internal/repository/settlement_repository_lock_test.go`（新建，env-gated）

- [ ] **Step 1: 写 env-gated 集成测试**

创建 `backend/internal/repository/settlement_repository_lock_test.go`：

```go
package repository

import (
	"os"
	"testing"
	"time"

	"nfa-dashboard/internal/model"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func openLockTestDB(t *testing.T) {
	t.Helper()
	dsn := os.Getenv("NFA_TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("NFA_TEST_MYSQL_DSN not set")
	}
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("connect test db: %v", err)
	}
	model.DB = db
}

func TestTryAdvisoryLockMutualExclusion(t *testing.T) {
	openLockTestDB(t)
	repo := NewSettlementRepository()

	release, ok, err := repo.TryAdvisoryLock("nfa:test:lock")
	if err != nil || !ok {
		t.Fatalf("first lock should succeed: ok=%v err=%v", ok, err)
	}
	_, ok2, err := repo.TryAdvisoryLock("nfa:test:lock")
	if err != nil {
		t.Fatal(err)
	}
	if ok2 {
		t.Fatal("second lock should fail while held")
	}
	release()
	release2, ok3, err := repo.TryAdvisoryLock("nfa:test:lock")
	if err != nil || !ok3 {
		t.Fatalf("lock after release should succeed: ok=%v err=%v", ok3, err)
	}
	release2()
}

func TestMarkStaleRunningTasks(t *testing.T) {
	openLockTestDB(t)
	repo := NewSettlementRepository()

	old := time.Now().Add(-2 * time.Hour)
	task := &model.SettlementTask{TaskType: "daily", TaskDate: time.Now(), Status: "running", CreateTime: old, UpdateTime: old}
	if err := repo.CreateSettlementTask(task); err != nil {
		t.Fatal(err)
	}
	defer model.DB.Delete(&model.SettlementTask{}, task.ID)
	// autoUpdateTime 会覆盖 update_time，插入后手动改回旧时间
	model.DB.Model(&model.SettlementTask{}).Where("id = ?", task.ID).UpdateColumn("update_time", old)

	stale, err := repo.MarkStaleRunningTasks(30 * time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, s := range stale {
		if s.ID == task.ID {
			found = true
		}
	}
	if !found {
		t.Fatal("应将该任务识别为 stale")
	}
	got, err := repo.GetSettlementTaskByID(task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "interrupted" {
		t.Fatalf("status=%s, want interrupted", got.Status)
	}
}
```

- [ ] **Step 2: 运行确认编译失败**

Run: `cd backend && go test ./internal/repository -run 'TestTryAdvisoryLock|TestMarkStale'`
Expected: FAIL（方法未定义）

- [ ] **Step 3: 接口追加两个方法**

`SettlementRepository` 接口（settlement_repository.go:18-53）末尾追加：

```go
	// TryAdvisoryLock 尝试获取 MySQL 命名锁（不等待）；成功时返回释放函数
	TryAdvisoryLock(name string) (release func(), ok bool, err error)
	// MarkStaleRunningTasks 将超过 staleAfter 无进度更新的 running 任务标记为 interrupted，返回被标记的任务
	MarkStaleRunningTasks(staleAfter time.Duration) ([]model.SettlementTask, error)
```

- [ ] **Step 4: 实现**

settlement_repository.go 追加（import 加 `"context"`、`"database/sql"`）：

```go
// TryAdvisoryLock 使用独占连接获取 MySQL GET_LOCK；同名锁被任何连接持有时立即返回 ok=false
func (r *settlementRepository) TryAdvisoryLock(name string) (func(), bool, error) {
	sqlDB, err := model.DB.DB()
	if err != nil {
		return nil, false, err
	}
	conn, err := sqlDB.Conn(context.Background())
	if err != nil {
		return nil, false, err
	}
	var got sql.NullInt64
	if err := conn.QueryRowContext(context.Background(), "SELECT GET_LOCK(?, 0)", name).Scan(&got); err != nil {
		_ = conn.Close()
		return nil, false, err
	}
	if !got.Valid || got.Int64 != 1 {
		_ = conn.Close()
		return nil, false, nil
	}
	release := func() {
		_, _ = conn.ExecContext(context.Background(), "SELECT RELEASE_LOCK(?)", name)
		_ = conn.Close()
	}
	return release, true, nil
}

// MarkStaleRunningTasks 清扫因进程重启等原因永久卡在 running 的任务
func (r *settlementRepository) MarkStaleRunningTasks(staleAfter time.Duration) ([]model.SettlementTask, error) {
	cutoff := time.Now().Add(-staleAfter)
	var stale []model.SettlementTask
	if err := model.DB.Where("status = ? AND update_time < ?", "running", cutoff).Find(&stale).Error; err != nil {
		return nil, err
	}
	if len(stale) == 0 {
		return nil, nil
	}
	ids := make([]int64, 0, len(stale))
	for _, t := range stale {
		ids = append(ids, t.ID)
	}
	now := time.Now()
	err := model.DB.Model(&model.SettlementTask{}).
		Where("id IN (?) AND status = ?", ids, "running").
		Updates(map[string]interface{}{
			"status":        "interrupted",
			"task_stage":    "interrupted",
			"end_time":      now,
			"error_message": fmt.Sprintf("任务超过 %.0f 分钟无进度更新，已自动标记为中断（可能因进程重启或执行异常），请确认后重新发起", staleAfter.Minutes()),
			"update_time":   now,
		}).Error
	if err != nil {
		return nil, err
	}
	return stale, nil
}
```

- [ ] **Step 5: 修复 stub 编译**

`customer_rate_import_task_controller_test.go` 中的 `settlementRepoImportTaskStub` struct 定义第一行嵌入接口以自动满足新方法（若已嵌入则跳过）：

```go
type settlementRepoImportTaskStub struct {
	repository.SettlementRepository
	// ...原有字段保持不变
}
```

- [ ] **Step 6: 验证**

Run: `cd backend && go build ./... && go test ./internal/... `
Expected: PASS（无 DSN 时 lock 测试 SKIP）。若本机有测试库：`NFA_TEST_MYSQL_DSN='user:pass@tcp(host:3306)/nfa_test?charset=utf8mb4&parseTime=True&loc=Local' go test ./internal/repository -run 'TestTryAdvisoryLock|TestMarkStale' -v` → PASS

- [ ] **Step 7: Commit**

```bash
git add backend/internal/repository/settlement_repository.go backend/internal/repository/settlement_repository_lock_test.go backend/internal/controller/customer_rate_import_task_controller_test.go
git commit -m "feat(repo): add advisory lock and stale running task sweeper" -m "Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>"
```

---

### Task 5: 调度器 leader 锁 + 同日任务去重 + 启停开关

**Files:**
- Modify: `backend/internal/service/settlement_service.go`（接口 + service 包装方法）
- Modify: `backend/internal/scheduler/settlement_scheduler.go`
- Modify: `backend/internal/bootstrap/app.go:113-114`
- Modify: `backend/internal/controller/edc_node_settlement_controller_test.go`（stub 兼容）
- Test: `backend/internal/scheduler/settlement_scheduler_test.go`（新建）

- [ ] **Step 1: SettlementService 接口追加三个方法**

`settlement_service.go` 接口末尾追加：

```go
	// TryAdvisoryLock 供调度器抢占执行权（透传 repository）
	TryAdvisoryLock(name string) (release func(), ok bool, err error)
	// HasActiveOrSuccessTask 同类型同日期是否已有 pending/running/success/partial 任务
	HasActiveOrSuccessTask(taskType string, taskDate time.Time) (bool, error)
	// MarkStaleRunningTasks 清扫卡死任务（透传 repository）
	MarkStaleRunningTasks(staleAfter time.Duration) ([]model.SettlementTask, error)
```

实现：

```go
func (s *settlementService) TryAdvisoryLock(name string) (func(), bool, error) {
	return s.repo.TryAdvisoryLock(name)
}

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

func (s *settlementService) MarkStaleRunningTasks(staleAfter time.Duration) ([]model.SettlementTask, error) {
	return s.repo.MarkStaleRunningTasks(staleAfter)
}
```

`edc_node_settlement_controller_test.go` 的 `settlementServiceForNodeTaskTest` struct 第一行嵌入 `service.SettlementService`（若未嵌入），以满足新接口方法。

- [ ] **Step 2: 写调度器去重测试（红）**

创建 `backend/internal/scheduler/settlement_scheduler_test.go`：

```go
package scheduler

import (
	"testing"
	"time"

	"nfa-dashboard/internal/model"
	"nfa-dashboard/internal/notify"
	"nfa-dashboard/internal/service"
)

type schedSvcStub struct {
	service.SettlementService
	exists  bool
	created []string
}

func (s *schedSvcStub) HasActiveOrSuccessTask(taskType string, taskDate time.Time) (bool, error) {
	return s.exists, nil
}

func (s *schedSvcStub) CreateSettlementTask(taskType string, taskDate time.Time) (*model.SettlementTask, error) {
	s.created = append(s.created, taskType)
	return &model.SettlementTask{ID: 1, TaskType: taskType, TaskDate: taskDate}, nil
}

func TestCreateAndRunSkipsWhenTaskExists(t *testing.T) {
	stub := &schedSvcStub{exists: true}
	sch := NewSettlementScheduler(stub, nil, notify.NewFromConfig(""))
	ran := make(chan int64, 1)
	sch.createAndRun("daily", time.Now(), func(id int64) { ran <- id })
	if len(stub.created) != 0 {
		t.Fatal("同日期任务已存在时不应重复创建")
	}
	select {
	case <-ran:
		t.Fatal("不应执行任务")
	case <-time.After(50 * time.Millisecond):
	}
}

func TestCreateAndRunCreatesAndRuns(t *testing.T) {
	stub := &schedSvcStub{exists: false}
	sch := NewSettlementScheduler(stub, nil, notify.NewFromConfig(""))
	ran := make(chan int64, 1)
	sch.createAndRun("daily", time.Now(), func(id int64) { ran <- id })
	if len(stub.created) != 1 {
		t.Fatalf("应创建 1 个任务, got %d", len(stub.created))
	}
	select {
	case id := <-ran:
		if id != 1 {
			t.Fatalf("taskID want 1 got %d", id)
		}
	case <-time.After(time.Second):
		t.Fatal("任务未被执行")
	}
}
```

Run: `cd backend && go test ./internal/scheduler`
Expected: FAIL（`createAndRun`、新构造签名不存在）

- [ ] **Step 3: 改造调度器**

`settlement_scheduler.go`：

结构体与构造函数增加 notifier：

```go
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
```

import 块加 `"nfa-dashboard/internal/notify"`。

`checkAndExecuteTasks()` 开头（获取当前时间之前）插入抢锁与清扫：

```go
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

	// ……以下为原有逻辑
```

新增两个方法：

```go
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
```

四个创建分支改为调用 `createAndRun`（原 `CreateSettlementTask` + `go func` 各 ~12 行替换为一个调用）：

daily 分支（原 110-124 行）：

```go
		log.Printf("开始执行每日结算任务，计算日期: %s", date.Format("2006-01-02"))
		s.createAndRun("daily", date, func(taskID int64) {
			if err := s.settlementService.ExecuteDailySettlement(taskID, date); err != nil {
				log.Printf("执行每日结算任务失败: %v", err)
			}
		})
```

weekly 分支（原 146-158 行）：

```go
		log.Printf("开始执行每周结算任务，计算开始日期: %s", startDate.Format("2006-01-02"))
		s.createAndRun("weekly", startDate, func(taskID int64) {
			if err := s.settlementService.ExecuteWeeklySettlement(taskID, startDate); err != nil {
				log.Printf("执行每周结算任务失败: %v", err)
			}
		})
```

node_daily95 分支（原 182-191 行）：

```go
			log.Printf("开始执行EDC节点每日结算任务，计算日期: %s", date.Format("2006-01-02"))
			s.createAndRun("node_daily95", date, func(taskID int64) {
				if err := s.nodeService.ExecuteDailyTask(taskID, date); err != nil {
					log.Printf("执行EDC节点每日结算任务失败: %v", err)
				}
			})
```

node_monthly95 分支（原 215-224 行）：

```go
		log.Printf("开始执行EDC节点月结算任务，计算月份: %s", month.Format("2006-01"))
		s.createAndRun("node_monthly95", month, func(taskID int64) {
			if err := s.nodeService.ExecuteMonthlyTask(taskID, month); err != nil {
				log.Printf("执行EDC节点月结算任务失败: %v", err)
			}
		})
```

各分支尾部原有的 `config.LastExecuteTime = now; UpdateSettlementConfig` 保留不动。

- [ ] **Step 4: app.go 接入开关**

原 113-114 行改为：

```go
	settlementScheduler := scheduler.NewSettlementScheduler(settlementService, edcNodeSettlementSvc, notifier)
	if config.IsSchedulerEnabled() {
		settlementScheduler.Start()
	} else {
		log.Println("scheduler.enabled=false，本实例不启动结算调度器")
	}
```

import 块加 `"log"`（若未引入）。

- [ ] **Step 5: 验证**

Run: `cd backend && go build ./... && go test ./internal/scheduler ./internal/service ./internal/controller`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add backend/internal/scheduler backend/internal/service/settlement_service.go backend/internal/bootstrap/app.go backend/internal/controller/edc_node_settlement_controller_test.go
git commit -m "feat(scheduler): leader lock, per-date task dedupe, stale sweeper and enable switch" -m "Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>"
```

---

### Task 6: 修复 CreateSettlementTask 的 ID 回查 bug

GORM `Create` 会回填自增 ID；原实现按 `(task_type, task_date)` 查"最新一条"在并发/同日多任务时会拿错行。

**Files:**
- Modify: `backend/internal/service/settlement_service.go:67-97`
- Test: `backend/internal/service/settlement_service_test.go`（追加用例）

- [ ] **Step 1: 写失败测试**

在 `settlement_service_test.go` 追加（import 需含 `time`、`nfa-dashboard/internal/model`、`nfa-dashboard/internal/repository`、`nfa-dashboard/internal/notify`）：

```go
type createTaskRepoStub struct {
	repository.SettlementRepository
	listCalled bool
}

func (s *createTaskRepoStub) CreateSettlementTask(task *model.SettlementTask) error {
	task.ID = 42 // 模拟 GORM 回填自增 ID
	return nil
}

func (s *createTaskRepoStub) GetSettlementTasks(filter map[string]interface{}, limit, offset int) ([]model.SettlementTask, int64, error) {
	s.listCalled = true
	return nil, 0, nil
}

func TestCreateSettlementTaskReturnsInsertedID(t *testing.T) {
	stub := &createTaskRepoStub{}
	svc := NewSettlementService(stub, nil, notify.NewFromConfig(""))
	task, err := svc.CreateSettlementTask("daily", time.Date(2026, 7, 1, 0, 0, 0, 0, time.Local))
	if err != nil {
		t.Fatal(err)
	}
	if task.ID != 42 {
		t.Fatalf("应返回插入后回填的 ID 42, got %d", task.ID)
	}
	if stub.listCalled {
		t.Fatal("不应再按 type+date 反查任务列表")
	}
}
```

Run: `cd backend && go test ./internal/service -run TestCreateSettlementTaskReturnsInsertedID`
Expected: FAIL（listCalled 为 true）

- [ ] **Step 2: 实现**

`CreateSettlementTask` 整体替换为：

```go
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
```

- [ ] **Step 3: 验证通过并回归**

Run: `cd backend && go test ./internal/service`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add backend/internal/service/settlement_service.go backend/internal/service/settlement_service_test.go
git commit -m "fix(settlement): return inserted task directly instead of racy query-back" -m "Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>"
```

---

### Task 7: customer_init 回填逻辑合并（修 EndTime bug + 失败告警）

`ExecuteDailySettlement` 与 `ExecuteWeeklySettlementWithDateRange` 尾部两段 goroutine 几乎相同，且 `end := time.Now()` 在 backfill 执行**前**取值导致 EndTime 失真。合并为一个方法并顺带接入失败告警。

**Files:**
- Modify: `backend/internal/service/settlement_service.go`
- Test: `backend/internal/service/settlement_service_test.go`

- [ ] **Step 1: 新增合并方法**

在 `settlement_service.go` 追加（import 需含 `"nfa-dashboard/internal/notify"`）：

```go
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
```

- [ ] **Step 2: 替换两处调用**

`ExecuteDailySettlement` 尾部整段 `go func(runDate time.Time) { ... }(date)` 替换为：

```go
	s.triggerCustomerInitAfter("daily", date, date)
```

`ExecuteWeeklySettlementWithDateRange` 尾部整段 `go func(sdate, edate time.Time) { ... }(startDate, endDate)` 替换为：

```go
	s.triggerCustomerInitAfter("weekly", startDate, endDate)
```

- [ ] **Step 3: 验证**

Run: `cd backend && go build ./... && go test ./internal/service`
Expected: PASS（`shouldFailCustomerInitOnZeroAffected` 既有测试继续绿）

- [ ] **Step 4: Commit**

```bash
git add backend/internal/service/settlement_service.go
git commit -m "fix(settlement): consolidate customer_init backfill, fix end_time and add failure alert" -m "Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>"
```

---

### Task 8: 任务失败统一告警

**Files:**
- Modify: `backend/internal/service/settlement_service.go:100-120`（UpdateSettlementTaskStatus）
- Test: `backend/internal/service/settlement_service_test.go`

- [ ] **Step 1: 写失败测试**

`settlement_service_test.go` 追加（import 加 `sync`、`strings`）：

```go
type recordingNotifier struct {
	mu     sync.Mutex
	titles []string
}

func (r *recordingNotifier) Send(title, text string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.titles = append(r.titles, title)
	return nil
}

func (r *recordingNotifier) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.titles)
}

type statusRepoStub struct {
	repository.SettlementRepository
	task *model.SettlementTask
}

func (s *statusRepoStub) GetSettlementTaskByID(id int64) (*model.SettlementTask, error) {
	return s.task, nil
}

func (s *statusRepoStub) UpdateSettlementTask(task *model.SettlementTask) error {
	s.task = task
	return nil
}

func TestUpdateTaskStatusFailedTriggersAlert(t *testing.T) {
	rec := &recordingNotifier{}
	stub := &statusRepoStub{task: &model.SettlementTask{ID: 7, TaskType: "daily", TaskDate: time.Now(), Status: "running"}}
	svc := NewSettlementService(stub, nil, rec)
	if err := svc.UpdateSettlementTaskStatus(7, "failed", "mock 错误"); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(time.Second)
	for rec.count() == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if rec.count() != 1 {
		t.Fatalf("failed 状态应触发一次告警, got %d", rec.count())
	}
	if !strings.Contains(stub.task.ErrorMessage, "mock 错误") {
		t.Fatalf("error message 未落库: %s", stub.task.ErrorMessage)
	}
}
```

Run: `cd backend && go test ./internal/service -run TestUpdateTaskStatusFailedTriggersAlert`
Expected: FAIL（未发告警）

- [ ] **Step 2: 实现**

`UpdateSettlementTaskStatus` 末尾的 `return s.repo.UpdateSettlementTask(task)` 改为：

```go
	if err := s.repo.UpdateSettlementTask(task); err != nil {
		return err
	}
	if status == "failed" {
		notify.SendAsync(s.notifier, "结算任务失败",
			fmt.Sprintf("任务 #%d (%s, %s) 执行失败：%s", task.ID, task.TaskType, task.TaskDate.Format("2006-01-02"), errorMsg))
	}
	return nil
```

- [ ] **Step 3: 验证 + Commit**

Run: `cd backend && go test ./internal/service`
Expected: PASS

```bash
git add backend/internal/service/settlement_service.go backend/internal/service/settlement_service_test.go
git commit -m "feat(settlement): send alert when task transitions to failed" -m "Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>"
```

---

## Phase 2：日结算性能 + 部分失败可见

### Task 9: 聚合日95计算 CalculateDaily95ForCombos

替代"每组合 1 次学校查询 + 1 次全天流量查询"的 N+1 模式：单次流式扫描当天 `nfa_school_traffic`，按 (school_id, region, cp) 分组计算 95。95 口径与 `CalculateDaily95WithRegionAndCP`（settlement_repository.go:850-954）完全一致：值降序排序后取 `settlement95.DescendingIndex(n)`。

**Files:**
- Modify: `backend/internal/repository/settlement_repository.go`（接口 + 实现）
- Test: `backend/internal/repository/settlement_repository_daily95_test.go`（新建）

- [ ] **Step 1: 写纯函数失败测试**

创建 `backend/internal/repository/settlement_repository_daily95_test.go`：

```go
package repository

import (
	"sort"
	"testing"
	"time"

	"nfa-dashboard/internal/settlement95"
)

func TestPick95PointMatchesDescendingIndex(t *testing.T) {
	values := []int64{10, 50, 30, 90, 70, 20, 60, 40, 80, 100}
	times := make([]time.Time, len(values))
	base := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	for i := range times {
		times[i] = base.Add(time.Duration(i) * time.Minute)
	}

	sorted := append([]int64(nil), values...)
	sort.Slice(sorted, func(a, b int) bool { return sorted[a] > sorted[b] })
	wantVal := sorted[settlement95.DescendingIndex(len(sorted))]

	gotVal, gotTime := pick95Point(values, times)
	if gotVal != wantVal {
		t.Fatalf("95 值不一致: got %d want %d", gotVal, wantVal)
	}
	found := false
	for i, v := range values {
		if v == gotVal && times[i].Equal(gotTime) {
			found = true
		}
	}
	if !found {
		t.Fatal("95 时间点必须来自值等于 95 值的采样点")
	}
}

func TestPick95PointSinglePoint(t *testing.T) {
	v, at := pick95Point([]int64{7}, []time.Time{time.Date(2026, 7, 1, 8, 0, 0, 0, time.UTC)})
	if v != 7 || at.Hour() != 8 {
		t.Fatalf("单点应原样返回: v=%d at=%v", v, at)
	}
}
```

Run: `cd backend && go test ./internal/repository -run TestPick95Point`
Expected: FAIL（函数不存在）

- [ ] **Step 2: 实现**

接口追加：

```go
	// CalculateDaily95ForCombos 单次扫描当天流量数据，按组合分组计算日95（仅返回 combos 中命中的组合）
	CalculateDaily95ForCombos(date time.Time, combos []model.SchoolRegionCP) ([]model.SchoolSettlement, error)
```

实现追加到 settlement_repository.go：

```go
// pick95Point 与 CalculateDaily95WithRegionAndCP 口径一致：按值降序取 DescendingIndex 处的值与时间
func pick95Point(values []int64, times []time.Time) (int64, time.Time) {
	idx := make([]int, len(values))
	for i := range idx {
		idx[i] = i
	}
	sort.SliceStable(idx, func(a, b int) bool { return values[idx[a]] > values[idx[b]] })
	i95 := settlement95.DescendingIndex(len(values))
	return values[idx[i95]], times[idx[i95]]
}

func comboKey(schoolID, region, cp string) string {
	return schoolID + "\x00" + region + "\x00" + cp
}

// CalculateDaily95ForCombos 流式分组计算：一次查询代替逐组合 N+1 查询
func (r *settlementRepository) CalculateDaily95ForCombos(date time.Time, combos []model.SchoolRegionCP) ([]model.SchoolSettlement, error) {
	startTime := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endTime := time.Date(date.Year(), date.Month(), date.Day(), 23, 59, 59, 999999999, date.Location())

	valid := make(map[string]model.SchoolRegionCP, len(combos))
	for _, c := range combos {
		if c.SchoolID == "" || c.Region == "" || c.CP == "" {
			continue
		}
		valid[comboKey(c.SchoolID, c.Region, c.CP)] = c
	}

	rows, err := model.DB.Model(&model.SchoolTraffic{}).
		Select("school_id, region, cp, total_recv, create_time").
		Where("create_time BETWEEN ? AND ?", startTime, endTime).
		Order("school_id, region, cp").
		Rows()
	if err != nil {
		return nil, fmt.Errorf("查询流量数据失败: %v", err)
	}
	defer rows.Close()

	var (
		settlements []model.SchoolSettlement
		curKey      string
		curValues   []int64
		curTimes    []time.Time
	)
	flush := func() {
		if curKey == "" || len(curValues) == 0 {
			return
		}
		combo, ok := valid[curKey]
		if !ok {
			return
		}
		value, at := pick95Point(curValues, curTimes)
		settlements = append(settlements, model.SchoolSettlement{
			SchoolID:        combo.SchoolID,
			SchoolName:      combo.SchoolName,
			Region:          combo.Region,
			CP:              combo.CP,
			SettlementValue: value,
			SettlementTime:  at,
			SettlementDate:  date,
		})
	}
	for rows.Next() {
		var (
			schoolID, region, cp string
			totalRecv            int64
			createTime           time.Time
		)
		if err := rows.Scan(&schoolID, &region, &cp, &totalRecv, &createTime); err != nil {
			return nil, fmt.Errorf("读取流量数据失败: %v", err)
		}
		key := comboKey(schoolID, region, cp)
		if key != curKey {
			flush()
			curKey = key
			curValues = curValues[:0]
			curTimes = curTimes[:0]
		}
		curValues = append(curValues, totalRecv)
		curTimes = append(curTimes, createTime)
	}
	flush()
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历流量数据失败: %v", err)
	}
	return settlements, nil
}
```

- [ ] **Step 3: 补 env-gated 新旧一致性测试**

同测试文件追加（`openLockTestDB` 复用 Task 4 定义）：

```go
func TestCalculateDaily95ForCombosMatchesLegacy(t *testing.T) {
	openLockTestDB(t)
	repo := NewSettlementRepository()
	date := time.Date(2026, 6, 1, 0, 0, 0, 0, time.Local)

	// 插入 3 个测试采样点（school 不需要真实存在于 nfa_school 表，聚合方法只依赖流量行）
	points := []model.SchoolTraffic{
		{CreateTime: date.Add(1 * time.Minute), SchoolID: "TEST95", SchoolName: "测试校", Region: "浙江", CP: "CT", HashUUID: "t95-1", TotalRecv: 100},
		{CreateTime: date.Add(2 * time.Minute), SchoolID: "TEST95", SchoolName: "测试校", Region: "浙江", CP: "CT", HashUUID: "t95-2", TotalRecv: 300},
		{CreateTime: date.Add(3 * time.Minute), SchoolID: "TEST95", SchoolName: "测试校", Region: "浙江", CP: "CT", HashUUID: "t95-3", TotalRecv: 200},
	}
	if err := model.DB.Create(&points).Error; err != nil {
		t.Fatal(err)
	}
	defer model.DB.Where("school_id = ?", "TEST95").Delete(&model.SchoolTraffic{})

	combos := []model.SchoolRegionCP{{SchoolID: "TEST95", SchoolName: "测试校", Region: "浙江", CP: "CT"}}
	got, err := repo.CalculateDaily95ForCombos(date, combos)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("应命中 1 个组合, got %d", len(got))
	}
	// n=3 时 DescendingIndex 的取值与旧口径一致（旧方法依赖 nfa_school 表存在，这里直接对齐 pick95Point）
	sorted := []int64{300, 200, 100}
	want := sorted[settlement95.DescendingIndex(3)]
	if got[0].SettlementValue != want {
		t.Fatalf("95 值: got %d want %d", got[0].SettlementValue, want)
	}
}
```

- [ ] **Step 4: 验证 + Commit**

Run: `cd backend && go test ./internal/repository`
Expected: PASS（env-gated 测试无 DSN 时 SKIP）

```bash
git add backend/internal/repository/settlement_repository.go backend/internal/repository/settlement_repository_daily95_test.go
git commit -m "perf(settlement): single-scan grouped daily95 calculation" -m "Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>"
```

---

### Task 10: 服务层切换聚合计算 + 合并重复实现 + weekly partial 状态

删除 `executeDailySettlementInternal` 与 `ExecuteDailySettlement` 的重复循环；weekly 部分天失败不再静默，落 `partial` 状态。

**Files:**
- Modify: `backend/internal/service/settlement_service.go`
- Test: `backend/internal/service/settlement_service_test.go`

- [ ] **Step 1: 写失败测试**

`settlement_service_test.go` 追加（import 加 `encoding/json`、`fmt`）：

```go
type weeklyRepoStub struct {
	repository.SettlementRepository
	tasks     map[int64]*model.SettlementTask
	failDates map[string]bool
}

func (s *weeklyRepoStub) ListValidSchoolCombos(userID *uint64) ([]model.SchoolRegionCP, error) {
	return []model.SchoolRegionCP{{SchoolID: "s1", SchoolName: "一中", Region: "浙江", CP: "CT"}}, nil
}

func (s *weeklyRepoStub) CalculateDaily95ForCombos(date time.Time, combos []model.SchoolRegionCP) ([]model.SchoolSettlement, error) {
	if s.failDates[date.Format("2006-01-02")] {
		return nil, fmt.Errorf("mock 计算失败")
	}
	return []model.SchoolSettlement{{SchoolID: "s1", SettlementDate: date}}, nil
}

func (s *weeklyRepoStub) GetSettlementTaskByID(id int64) (*model.SettlementTask, error) {
	return s.tasks[id], nil
}

func (s *weeklyRepoStub) UpdateSettlementTask(task *model.SettlementTask) error {
	s.tasks[task.ID] = task
	return nil
}

func (s *weeklyRepoStub) BatchCreateSettlements(settlements []model.SchoolSettlement) error {
	return nil
}

func (s *weeklyRepoStub) GetSettlementConfig() (*model.SettlementConfig, error) {
	return &model.SettlementConfig{Enabled: false}, nil // 阻断 customer_init goroutine
}

func TestWeeklyPartialStatusOnSomeDayFailure(t *testing.T) {
	stub := &weeklyRepoStub{
		tasks:     map[int64]*model.SettlementTask{1: {ID: 1, TaskType: "weekly", Status: "pending"}},
		failDates: map[string]bool{"2026-07-02": true},
	}
	svc := NewSettlementService(stub, nil, notify.NewFromConfig(""))
	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.Local)
	end := time.Date(2026, 7, 3, 0, 0, 0, 0, time.Local)
	if err := svc.ExecuteWeeklySettlementWithDateRange(1, start, end); err != nil {
		t.Fatal(err)
	}
	task := stub.tasks[1]
	if task.Status != "partial" {
		t.Fatalf("部分天失败应落 partial, got %s", task.Status)
	}
	if !strings.Contains(task.ErrorMessage, "2026-07-02") {
		t.Fatalf("错误信息应包含失败日期: %s", task.ErrorMessage)
	}
}

func TestMergeTaskMeta(t *testing.T) {
	out := mergeTaskMeta(`{"a":1}`, map[string]interface{}{"b": 2})
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(out), &m); err != nil {
		t.Fatal(err)
	}
	if m["a"] == nil || m["b"] == nil {
		t.Fatalf("合并后缺 key: %s", out)
	}
	if got := mergeTaskMeta("", map[string]interface{}{"x": 1}); !strings.Contains(got, `"x":1`) {
		t.Fatalf("空 meta 合并异常: %s", got)
	}
}
```

Run: `cd backend && go test ./internal/service -run 'TestWeeklyPartial|TestMergeTaskMeta'`
Expected: FAIL

- [ ] **Step 2: 实现**

`settlement_service.go`（import 加 `"encoding/json"`、`"strings"`）：

**删除** `executeDailySettlementInternal`（231-270 行）。新增共享计算与 meta 工具：

```go
// calculateDailySettlements 计算某日全部有效组合的结算行
// 返回：结算行、尝试的组合数、命中（有流量）的组合数
func (s *settlementService) calculateDailySettlements(date time.Time) ([]model.SchoolSettlement, int, int, error) {
	combos, err := s.repo.ListValidSchoolCombos(nil)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("获取有效学校组合失败: %v", err)
	}
	settlements, err := s.repo.CalculateDaily95ForCombos(date, combos)
	if err != nil {
		return nil, 0, 0, err
	}
	return settlements, len(combos), len(settlements), nil
}

// mergeTaskMeta 在已有 task_meta JSON 上合并新键（解析失败时保留原值）
func mergeTaskMeta(existing string, extra map[string]interface{}) string {
	m := map[string]interface{}{}
	if strings.TrimSpace(existing) != "" {
		_ = json.Unmarshal([]byte(existing), &m)
	}
	for k, v := range extra {
		m[k] = v
	}
	b, err := json.Marshal(m)
	if err != nil {
		return existing
	}
	return string(b)
}
```

`ExecuteDailySettlement` 整体替换为：

```go
// ExecuteDailySettlement 执行日结算任务（单次聚合查询计算全部组合）
func (s *settlementService) ExecuteDailySettlement(taskID int64, date time.Time) error {
	if err := s.UpdateSettlementTaskStatus(taskID, "running", ""); err != nil {
		return fmt.Errorf("更新任务状态失败: %v", err)
	}
	settlements, attempted, hit, err := s.calculateDailySettlements(date)
	if err != nil {
		_ = s.UpdateSettlementTaskStatus(taskID, "failed", err.Error())
		return err
	}
	if len(settlements) > 0 {
		if err := s.repo.BatchCreateSettlements(settlements); err != nil {
			_ = s.UpdateSettlementTaskStatus(taskID, "failed", fmt.Sprintf("保存结算数据失败: %v", err))
			return fmt.Errorf("保存结算数据失败: %v", err)
		}
	}
	task, err := s.repo.GetSettlementTaskByID(taskID)
	if err != nil {
		return fmt.Errorf("获取任务信息失败: %v", err)
	}
	now := time.Now()
	task.Status = "success"
	task.EndTime = &now
	task.ProcessedCount = attempted
	task.TotalCount = attempted
	task.TaskMeta = mergeTaskMeta(task.TaskMeta, map[string]interface{}{
		"combos_total":     attempted,
		"combos_with_data": hit,
		"combos_no_data":   attempted - hit,
	})
	if err := s.repo.UpdateSettlementTask(task); err != nil {
		return fmt.Errorf("更新任务状态失败: %v", err)
	}
	s.triggerCustomerInitAfter("daily", date, date)
	return nil
}
```

`ExecuteWeeklySettlementWithDateRange` 主体替换为（尾部 `triggerCustomerInitAfter` 调用保持 Task 7 的样子）：

```go
// ExecuteWeeklySettlementWithDateRange 执行周结算任务（支持自定义日期范围；部分天失败落 partial）
func (s *settlementService) ExecuteWeeklySettlementWithDateRange(taskID int64, startDate, endDate time.Time) error {
	if err := s.UpdateSettlementTaskStatus(taskID, "running", ""); err != nil {
		return fmt.Errorf("更新任务状态失败: %v", err)
	}
	var (
		all        []model.SchoolSettlement
		total      int
		totalDays  int
		failedDays []string
	)
	for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
		totalDays++
		ds, attempted, _, err := s.calculateDailySettlements(d)
		if err != nil {
			log.Printf("计算 %s 日结算失败: %v", d.Format("2006-01-02"), err)
			failedDays = append(failedDays, d.Format("2006-01-02"))
			continue
		}
		all = append(all, ds...)
		total += attempted
		// 每完成一天更新进度，便于前端显示与卡死清扫判活
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
	now := time.Now()
	task.EndTime = &now
	task.ProcessedCount = total
	switch {
	case len(failedDays) == 0:
		task.Status = "success"
	case len(failedDays) == totalDays:
		task.Status = "failed"
		task.ErrorMessage = fmt.Sprintf("全部 %d 天计算失败: %s", totalDays, strings.Join(failedDays, ", "))
	default:
		task.Status = "partial"
		task.ErrorMessage = fmt.Sprintf("%d/%d 天计算失败: %s", len(failedDays), totalDays, strings.Join(failedDays, ", "))
	}
	if len(failedDays) > 0 {
		task.TaskMeta = mergeTaskMeta(task.TaskMeta, map[string]interface{}{"failed_days": failedDays})
	}
	if err := s.repo.UpdateSettlementTask(task); err != nil {
		return fmt.Errorf("更新任务状态失败: %v", err)
	}
	if task.Status == "partial" || task.Status == "failed" {
		notify.SendAsync(s.notifier, "周结算任务异常",
			fmt.Sprintf("任务 #%d (%s ~ %s) 状态 %s：%s", taskID, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"), task.Status, task.ErrorMessage))
	}
	s.triggerCustomerInitAfter("weekly", startDate, endDate)
	return nil
}
```

- [ ] **Step 3: 验证 + Commit**

Run: `cd backend && go build ./... && go test ./internal/...`
Expected: PASS

```bash
git add backend/internal/service/settlement_service.go backend/internal/service/settlement_service_test.go
git commit -m "feat(settlement): aggregated daily95 in service, dedupe daily loop, weekly partial status" -m "Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>"
```

---

### Task 11: 前端任务状态支持 partial / interrupted

**Files:**
- Modify: `frontend/frontend/src/components/settlement/SettlementTasksTab.vue`

- [ ] **Step 1: 筛选下拉加选项**

第 17-22 行状态下拉中 `失败` 选项之后追加：

```html
            <el-option label="部分成功" value="partial" />
            <el-option label="已中断" value="interrupted" />
```

- [ ] **Step 2: 标签映射**

约 668 行 `statusTagType`（函数名以实际为准，即包含 `case 'failed': return 'danger'` 的 switch）中 `failed` 分支后追加：

```ts
    case 'partial': return 'warning'
    case 'interrupted': return 'danger'
```

约 679 行状态文案 switch 中 `failed` 分支后追加：

```ts
    case 'partial': return '部分成功'
    case 'interrupted': return '已中断'
```

- [ ] **Step 3: 验证 + Commit**

Run: `cd frontend/frontend && npm run type-check && npm run test:unit`
Expected: PASS

```bash
git add frontend/frontend/src/components/settlement/SettlementTasksTab.vue
git commit -m "feat(web): render partial and interrupted settlement task statuses" -m "Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>"
```

---

## Phase 3：完整性巡检与缺数可视化

### Task 12: 完整性统计（repo + 纯逻辑 + 单测）

**Files:**
- Create: `backend/internal/model/settlement_integrity.go`
- Create: `backend/internal/repository/settlement_data_integrity.go`
- Create: `backend/internal/service/settlement_integrity_service.go`
- Modify: `backend/internal/repository/settlement_data_repository.go`（接口追加两个方法）
- Test: `backend/internal/service/settlement_integrity_service_test.go`

- [ ] **Step 1: 写失败测试（报告纯逻辑）**

创建 `backend/internal/service/settlement_integrity_service_test.go`：

```go
package service

import (
	"testing"
	"time"
)

func TestBuildIntegrityReportFlagsAlternatingGaps(t *testing.T) {
	start := time.Date(2026, 6, 8, 0, 0, 0, 0, time.Local)
	end := time.Date(2026, 6, 11, 0, 0, 0, 0, time.Local)
	src := map[string]int64{"2026-06-08": 100, "2026-06-09": 100, "2026-06-10": 100, "2026-06-11": 100}
	cust := map[string]int64{"2026-06-08": 90, "2026-06-10": 90} // 隔日缺失（复现 6 月事故形态）

	report := buildIntegrityReport(src, cust, start, end)
	if len(report.Days) != 4 {
		t.Fatalf("应有 4 天, got %d", len(report.Days))
	}
	if report.Days[1].CustomerStatus != "missing" || report.Days[3].CustomerStatus != "missing" {
		t.Fatalf("应标记 06-09/06-11 客户数据缺失: %+v", report.Days)
	}
	if report.Days[0].CustomerStatus != "ok" || report.Days[0].SourceStatus != "ok" {
		t.Fatalf("正常日不应误报: %+v", report.Days[0])
	}
}

func TestIntegrityStatusLowThreshold(t *testing.T) {
	if got := integrityStatus(79, 100); got != "low" {
		t.Fatalf("79/100 应为 low, got %s", got)
	}
	if got := integrityStatus(80, 100); got != "ok" {
		t.Fatalf("80/100 应为 ok, got %s", got)
	}
	if got := integrityStatus(0, 100); got != "missing" {
		t.Fatalf("0 应为 missing, got %s", got)
	}
}
```

Run: `cd backend && go test ./internal/service -run 'TestBuildIntegrity|TestIntegrityStatus'`
Expected: FAIL

- [ ] **Step 2: model**

创建 `backend/internal/model/settlement_integrity.go`：

```go
package model

// SettlementIntegrityDay 单日完整性统计（NFA 源表 vs 客户日结算表）
type SettlementIntegrityDay struct {
	Date           string `json:"date"`
	SourceCount    int64  `json:"source_count"`
	CustomerCount  int64  `json:"customer_count"`
	SourceStatus   string `json:"source_status"`   // ok / low / missing
	CustomerStatus string `json:"customer_status"` // ok / low / missing
}

// SettlementIntegrityReport 区间完整性报告；Baseline 为非零日行数中位数
type SettlementIntegrityReport struct {
	SourceBaseline   int64                    `json:"source_baseline"`
	CustomerBaseline int64                    `json:"customer_baseline"`
	Days             []SettlementIntegrityDay `json:"days"`
}
```

- [ ] **Step 3: repository**

`SettlementDataRepository` 接口（settlement_data_repository.go 顶部接口定义处）追加：

```go
	// DailySchoolSettlementCounts 按日统计 NFA 源表行数
	DailySchoolSettlementCounts(region, cp, school string, start, end time.Time) (map[string]int64, error)
	// DailyCustomerSettlementCounts 按日统计客户日结算（活动槽）行数
	DailyCustomerSettlementCounts(region, cp, school string, start, end time.Time) (map[string]int64, error)
```

（若有 `SettlementDataRepository` 的测试 stub 未编译，用嵌入接口的方式修复，同 Task 4 Step 5。）

创建 `backend/internal/repository/settlement_data_integrity.go`：

```go
package repository

import (
	"time"

	"nfa-dashboard/internal/model"
)

type dailyCountRow struct {
	Day string `gorm:"column:day"`
	Cnt int64  `gorm:"column:cnt"`
}

// DailySchoolSettlementCounts 按日统计 nfa_school_settlement 行数
func (r *settlementDataRepository) DailySchoolSettlementCounts(region, cp, school string, start, end time.Time) (map[string]int64, error) {
	qb := model.DB.Table("nfa_school_settlement").
		Select("DATE_FORMAT(settlement_date, '%Y-%m-%d') AS day, COUNT(*) AS cnt").
		Where("settlement_date >= ? AND settlement_date < ?", integrityDayStart(start), integrityDayStart(end).AddDate(0, 0, 1))
	if region != "" {
		qb = qb.Where("region = ?", region)
	}
	if cp != "" {
		qb = qb.Where("cp = ?", cp)
	}
	if school != "" {
		qb = qb.Where("school_id = ?", school)
	}
	var rows []dailyCountRow
	if err := qb.Group("day").Scan(&rows).Error; err != nil {
		return nil, err
	}
	return countRowsToMap(rows), nil
}

// DailyCustomerSettlementCounts 按日统计客户日结算行数（有槽表走活动槽，否则走旧表）
func (r *settlementDataRepository) DailyCustomerSettlementCounts(region, cp, school string, start, end time.Time) (map[string]int64, error) {
	useSlot := isSlotTableSupported()
	table := (model.SettlementCustomer{}).TableName() + " AS sc"
	if useSlot {
		table = "settlement_customer_v AS sc"
	}
	qb := model.DB.Table(table).
		Select("DATE_FORMAT(sc.service_date, '%Y-%m-%d') AS day, COUNT(*) AS cnt").
		Where("sc.service_date >= ? AND sc.service_date < ?", integrityDayStart(start), integrityDayStart(end).AddDate(0, 0, 1))
	if useSlot {
		qb = withActiveSlot(qb, "sc")
	}
	if region != "" {
		qb = qb.Where("sc.region = ?", region)
	}
	if cp != "" {
		qb = qb.Where("sc.cp = ?", cp)
	}
	if school != "" {
		qb = qb.Where("sc.school_id = ?", school)
	}
	var rows []dailyCountRow
	if err := qb.Group("day").Scan(&rows).Error; err != nil {
		return nil, err
	}
	return countRowsToMap(rows), nil
}

func integrityDayStart(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func countRowsToMap(rows []dailyCountRow) map[string]int64 {
	out := make(map[string]int64, len(rows))
	for _, r := range rows {
		out[r.Day] = r.Cnt
	}
	return out
}
```

注：若 `model.SettlementCustomer` 的 TableName 方法签名不是值接收者，按实际调整该表达式（目的只是取旧表名字符串）。

- [ ] **Step 4: service**

创建 `backend/internal/service/settlement_integrity_service.go`：

```go
package service

import (
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"nfa-dashboard/internal/model"
	"nfa-dashboard/internal/notify"
	"nfa-dashboard/internal/repository"
)

// SettlementIntegrityService 结算数据完整性巡检
type SettlementIntegrityService interface {
	// CheckRange 检查日期区间内 NFA 源表与客户日结算表的每日行数
	CheckRange(region, cp, school string, start, end time.Time) (*model.SettlementIntegrityReport, error)
	// CheckYesterdayAndAlert 巡检昨日数据（对比近14天基线），异常时发送告警
	CheckYesterdayAndAlert()
}

type settlementIntegrityService struct {
	dataRepo repository.SettlementDataRepository
	notifier notify.Notifier
}

// NewSettlementIntegrityService 创建完整性巡检服务
func NewSettlementIntegrityService(dataRepo repository.SettlementDataRepository, notifier notify.Notifier) SettlementIntegrityService {
	return &settlementIntegrityService{dataRepo: dataRepo, notifier: notifier}
}

func (s *settlementIntegrityService) CheckRange(region, cp, school string, start, end time.Time) (*model.SettlementIntegrityReport, error) {
	srcCounts, err := s.dataRepo.DailySchoolSettlementCounts(region, cp, school, start, end)
	if err != nil {
		return nil, fmt.Errorf("统计源表行数失败: %v", err)
	}
	custCounts, err := s.dataRepo.DailyCustomerSettlementCounts(region, cp, school, start, end)
	if err != nil {
		return nil, fmt.Errorf("统计客户结算行数失败: %v", err)
	}
	return buildIntegrityReport(srcCounts, custCounts, start, end), nil
}

// buildIntegrityReport 纯函数：按非零日行数中位数为基线判定 missing/low/ok
func buildIntegrityReport(srcCounts, custCounts map[string]int64, start, end time.Time) *model.SettlementIntegrityReport {
	report := &model.SettlementIntegrityReport{}
	var srcNonZero, custNonZero []int64
	for d := integrityDayOf(start); !d.After(integrityDayOf(end)); d = d.AddDate(0, 0, 1) {
		key := d.Format("2006-01-02")
		day := model.SettlementIntegrityDay{
			Date:          key,
			SourceCount:   srcCounts[key],
			CustomerCount: custCounts[key],
		}
		if day.SourceCount > 0 {
			srcNonZero = append(srcNonZero, day.SourceCount)
		}
		if day.CustomerCount > 0 {
			custNonZero = append(custNonZero, day.CustomerCount)
		}
		report.Days = append(report.Days, day)
	}
	report.SourceBaseline = medianInt64(srcNonZero)
	report.CustomerBaseline = medianInt64(custNonZero)
	for i := range report.Days {
		report.Days[i].SourceStatus = integrityStatus(report.Days[i].SourceCount, report.SourceBaseline)
		report.Days[i].CustomerStatus = integrityStatus(report.Days[i].CustomerCount, report.CustomerBaseline)
	}
	return report
}

func integrityStatus(count, baseline int64) string {
	switch {
	case count == 0:
		return "missing"
	case baseline > 0 && count*100 < baseline*80:
		return "low"
	default:
		return "ok"
	}
}

func integrityDayOf(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func medianInt64(v []int64) int64 {
	if len(v) == 0 {
		return 0
	}
	s := append([]int64(nil), v...)
	sort.Slice(s, func(a, b int) bool { return s[a] < s[b] })
	return s[len(s)/2]
}

func (s *settlementIntegrityService) CheckYesterdayAndAlert() {
	yesterday := time.Now().AddDate(0, 0, -1)
	start := yesterday.AddDate(0, 0, -13)
	report, err := s.CheckRange("", "", "", start, yesterday)
	if err != nil {
		log.Printf("结算数据完整性巡检失败: %v", err)
		notify.SendAsync(s.notifier, "结算完整性巡检失败", err.Error())
		return
	}
	if len(report.Days) == 0 {
		return
	}
	last := report.Days[len(report.Days)-1]
	var problems []string
	if last.SourceStatus != "ok" {
		problems = append(problems, fmt.Sprintf("NFA 源表 %s（行数 %d，基线 %d）", last.SourceStatus, last.SourceCount, report.SourceBaseline))
	}
	if last.CustomerStatus != "ok" {
		problems = append(problems, fmt.Sprintf("客户日结算 %s（行数 %d，基线 %d）", last.CustomerStatus, last.CustomerCount, report.CustomerBaseline))
	}
	if len(problems) == 0 {
		log.Printf("结算数据完整性巡检通过: %s", last.Date)
		return
	}
	notify.SendAsync(s.notifier, "结算数据缺失告警",
		fmt.Sprintf("日期 %s 数据异常：%s。请检查提取器与结算任务。", last.Date, strings.Join(problems, "；")))
}
```

- [ ] **Step 5: 验证 + Commit**

Run: `cd backend && go build ./... && go test ./internal/service ./internal/repository`
Expected: PASS

```bash
git add backend/internal/model/settlement_integrity.go backend/internal/repository/settlement_data_integrity.go backend/internal/repository/settlement_data_repository.go backend/internal/service/settlement_integrity_service.go backend/internal/service/settlement_integrity_service_test.go
git commit -m "feat(settlement): daily row-count integrity check with median baseline" -m "Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>"
```

---

### Task 13: 巡检接入调度器 + 完整性 API

**Files:**
- Modify: `backend/internal/scheduler/settlement_scheduler.go`
- Modify: `backend/internal/controller/settlement_data_controller.go`
- Modify: `backend/internal/bootstrap/app.go`

- [ ] **Step 1: 调度器接入**

`SettlementScheduler` struct 加字段 `integrityService service.SettlementIntegrityService`，构造函数签名改为：

```go
func NewSettlementScheduler(settlementService service.SettlementService, nodeService service.EDCNodeSettlementService, integrityService service.SettlementIntegrityService, notifier notify.Notifier) *SettlementScheduler {
```

（同步更新 Task 5 的调度器测试构造调用：加 `nil` 参数。）

常量区加 `integrityCheckHour = 9`。`checkAndExecuteTasks` 在 `if !config.Enabled { return }` 之后插入：

```go
	// 每天固定时间巡检昨日数据完整性（在日结算与客户回填完成之后）
	if s.integrityService != nil && currentHour == integrityCheckHour && currentMinute == 0 {
		go s.integrityService.CheckYesterdayAndAlert()
	}
```

- [ ] **Step 2: 控制器与路由**

`SettlementDataController` struct 加字段 `integritySvc service.SettlementIntegrityService`，构造函数追加对应参数（查看该文件现有 `NewSettlementDataController` 定义，追加参数与字段赋值）。新增 handler：

```go
// GetIntegrity 结算数据完整性检查：按日返回源表/客户表行数及状态
func (c *SettlementDataController) GetIntegrity(ctx *gin.Context) {
	start, err1 := time.ParseInLocation("2006-01-02", ctx.Query("start_date"), time.Local)
	end, err2 := time.ParseInLocation("2006-01-02", ctx.Query("end_date"), time.Local)
	if err1 != nil || err2 != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "start_date / end_date 必填，格式 YYYY-MM-DD"})
		return
	}
	if end.Before(start) || end.Sub(start) > 62*24*time.Hour {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "日期区间无效（最长 62 天）"})
		return
	}
	report, err := c.integritySvc.CheckRange(ctx.Query("region"), ctx.Query("cp"), ctx.Query("school_id"), start, end)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "完整性检查失败", "error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"code": 200, "message": "OK", "data": report})
}
```

- [ ] **Step 3: app.go 装配与路由**

装配（settlementDataService 附近）：

```go
	settlementIntegritySvc := service.NewSettlementIntegrityService(settlementDataRepo, notifier)
	settlementDataController := controller.NewSettlementDataController(settlementDataService, settlementIntegritySvc)
```

调度器构造更新：

```go
	settlementScheduler := scheduler.NewSettlementScheduler(settlementService, edcNodeSettlementSvc, settlementIntegritySvc, notifier)
```

路由（`settlement.GET("/data/node/monthly", ...)` 之后，约 175 行）：

```go
			settlement.GET("/data/integrity", authMW.PermissionRequired("settlement.data.read"), settlementDataController.GetIntegrity)
```

- [ ] **Step 4: 验证 + Commit**

Run: `cd backend && go build ./... && go test ./internal/...`
Expected: PASS

```bash
git add backend/internal/scheduler/settlement_scheduler.go backend/internal/scheduler/settlement_scheduler_test.go backend/internal/controller/settlement_data_controller.go backend/internal/bootstrap/app.go
git commit -m "feat(settlement): integrity API endpoint and scheduled daily inspection" -m "Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>"
```

---

### Task 14: 前端缺数提示（结算数据 Tab）

**Files:**
- Create: `frontend/frontend/src/components/settlement/settlement-integrity-utils.ts`
- Test: `frontend/frontend/src/components/settlement/__tests__/settlement-integrity-utils.spec.ts`
- Modify: `frontend/frontend/src/api/index.ts`（settlementData 段）
- Modify: `frontend/frontend/src/components/settlement/SettlementDataTab.vue`

- [ ] **Step 1: 写失败测试**

创建 `frontend/frontend/src/components/settlement/__tests__/settlement-integrity-utils.spec.ts`：

```ts
import { describe, it, expect } from 'vitest'
import { pickProblemDays, integritySummaryText, type IntegrityDay } from '../settlement-integrity-utils'

const day = (date: string, srcStatus: string, custStatus: string): IntegrityDay => ({
  date,
  source_count: 100,
  customer_count: 90,
  source_status: srcStatus as IntegrityDay['source_status'],
  customer_status: custStatus as IntegrityDay['customer_status'],
})

describe('settlement-integrity-utils', () => {
  it('过滤出异常日', () => {
    const days = [day('2026-06-08', 'ok', 'ok'), day('2026-06-09', 'ok', 'missing'), day('2026-06-10', 'low', 'ok')]
    expect(pickProblemDays(days).map((d) => d.date)).toEqual(['2026-06-09', '2026-06-10'])
  })

  it('非数组输入返回空', () => {
    expect(pickProblemDays(null)).toEqual([])
    expect(pickProblemDays(undefined)).toEqual([])
  })

  it('摘要文本包含日期与缺失类型', () => {
    const text = integritySummaryText([day('2026-06-09', 'ok', 'missing')])
    expect(text).toContain('2026-06-09')
    expect(text).toContain('客户结算缺失')
  })

  it('超过 10 天折叠展示', () => {
    const many = Array.from({ length: 12 }, (_, i) => day(`2026-06-${String(i + 1).padStart(2, '0')}`, 'missing', 'ok'))
    expect(integritySummaryText(many)).toContain('共 12 天')
  })
})
```

Run: `cd frontend/frontend && npm run test:unit -- src/components/settlement/__tests__/settlement-integrity-utils.spec.ts`
Expected: FAIL（模块不存在）

- [ ] **Step 2: 实现工具模块**

创建 `frontend/frontend/src/components/settlement/settlement-integrity-utils.ts`：

```ts
export interface IntegrityDay {
  date: string
  source_count: number
  customer_count: number
  source_status: 'ok' | 'low' | 'missing'
  customer_status: 'ok' | 'low' | 'missing'
}

export function pickProblemDays(days: IntegrityDay[] | null | undefined): IntegrityDay[] {
  if (!Array.isArray(days)) return []
  return days.filter((d) => d.source_status !== 'ok' || d.customer_status !== 'ok')
}

export function integritySummaryText(problems: IntegrityDay[]): string {
  if (!problems.length) return ''
  const label = (status: string, name: string) =>
    status === 'ok' ? '' : `${name}${status === 'missing' ? '缺失' : '偏低'}`
  const parts = problems.slice(0, 10).map((d) => {
    const tags = [label(d.source_status, '源表'), label(d.customer_status, '客户结算')].filter(Boolean)
    return `${d.date}（${tags.join('、')}）`
  })
  const more = problems.length > 10 ? ` 等共 ${problems.length} 天` : ''
  return parts.join('、') + more
}
```

Run 同上 → PASS

- [ ] **Step 3: API 客户端**

`frontend/frontend/src/api/index.ts` 的 `settlementData` 对象中，`nodeMonthlyList` 之后追加：

```ts
    // 数据完整性检查（NFA 日粒度）
    integrity(params?: any, config?: AxiosRequestConfig) {
      return api
        .get('/api/v1/settlement/data/integrity', { params, ...(config || {}) })
        .then((d: any) => (d && typeof d === 'object' && 'data' in d ? (d as any).data : d))
    },
```

- [ ] **Step 4: SettlementDataTab 集成**

script 部分（`const settlementData = ref<...>` 附近）追加，`settlementData` API 客户端的 import 名以文件中 `fetchData` 内调用 `list(...)` 的对象为准（复用同一个导入）：

```ts
import { pickProblemDays, integritySummaryText, type IntegrityDay } from './settlement-integrity-utils'

const integrityDays = ref<IntegrityDay[]>([])
const integrityProblems = computed(() => pickProblemDays(integrityDays.value))
const integrityText = computed(() => integritySummaryText(integrityProblems.value))

async function checkIntegrity() {
  integrityDays.value = []
  if (dataSource.value !== 'nfa' || granularity.value !== 'daily' || !dateRange.value) return
  try {
    const res: any = await settlementDataApi.integrity({
      start_date: dateRange.value[0].slice(0, 10),
      end_date: dateRange.value[1].slice(0, 10),
      region: filterForm.region || undefined,
      cp: filterForm.cp || undefined,
      school_id: filterForm.school_id || undefined,
    })
    integrityDays.value = res?.days || []
  } catch {
    // 完整性检查失败不阻塞主查询
  }
}
```

（`settlementDataApi` 为占位名，替换为该文件里实际调用 `.list(...)` 的对象。）

在 `fetchData` 成功路径末尾（items 赋值之后）追加一行：

```ts
    void checkIntegrity()
```

template 中 `<div class="table-header">...</div>` 闭合后、`<el-table` 之前插入：

```html
      <el-alert
        v-if="integrityProblems.length"
        type="warning"
        show-icon
        :closable="false"
        style="margin-bottom: 12px"
      >
        <template #title>检测到 {{ integrityProblems.length }} 天结算数据可能缺失或偏低</template>
        {{ integrityText }}
      </el-alert>
```

- [ ] **Step 5: 验证 + Commit**

Run: `cd frontend/frontend && npm run type-check && npm run test:unit`
Expected: PASS

```bash
git add frontend/frontend/src/components/settlement/settlement-integrity-utils.ts frontend/frontend/src/components/settlement/__tests__/settlement-integrity-utils.spec.ts frontend/frontend/src/api/index.ts frontend/frontend/src/components/settlement/SettlementDataTab.vue
git commit -m "feat(web): show missing-day integrity warning on settlement data tab" -m "Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>"
```

---

## Phase 4：大文件拆分（纯机械移动，无行为变更）

### Task 15: settlement_data_repository.go 拆成 4 个文件

同包内函数移动，import 随移。移动后 `settlement_data_repository.go` 只留：接口定义、struct、`NewSettlementDataRepository`、`normalizeDayBounds`、unit-base 三个helper（`normalizeSettlementResultUnitBase`/`loadSettlementResultUnitBaseFromSystemSettings`/`settlementValueToGbps`）、`ListSettlementCustomer`、`applySettlementCustomerFilters`、`shouldApplySettlementFilterRules`、`applyRateFilterRulesIfEnabled`、`extractServiceMonthRange`、`applyServiceMonthRangeToDailyQB`。

**Files:**
- Modify: `backend/internal/repository/settlement_data_repository.go`
- Create: `backend/internal/repository/settlement_data_slots.go`
- Create: `backend/internal/repository/settlement_data_backfill.go`
- Create: `backend/internal/repository/settlement_data_monthly.go`

- [ ] **Step 1: 创建 settlement_data_slots.go 并移入槽位函数**

移入（原行号供定位）：`slotTableOnce`/`slotTableSupported` 包级变量、`isSlotTableSupported`(38)、`withActiveSlot`(47)、`resolveScopeHash`(53)、`buildMonthList`(1219)、`activeSlotForMonth`(1235)、`publishSlotForMonth`(1256)、`copyFullMonthFromActiveSlot`(1375)、`countMonthSlotRows`(1413)。

- [ ] **Step 2: 创建 settlement_data_backfill.go 并移入回填管线**

移入：`buildChunkRanges`(95)、`settlementCustomerKey`(113)、`buildExistingSettlementMap`(117)、`CountSchoolSettlementRows`(822)、`BackfillFromSchoolSettlement`(852)、`buildSettlementCustomerUpsertAssignments`(1158)、`updateRateCustomerIncrementBatch`(1188)、`buildTmpSourceWhere`(1273)、`prepareTmpTables`(1297)、`countTempRows`(1313)、`createTmpSourceAndKeys`(1321)、`runTempPipelineForMonth`(1425)、`cloneStageMetrics`(1732)、`backfillFromSchoolSettlementWithSlot`(1740)。

- [ ] **Step 3: 创建 settlement_data_monthly.go 并移入月度逻辑**

移入：`listSettlementCustomerMonthlyFromDaily`(247)、`ListSettlementCustomerMonthly`(362)、`applyMonthlySnapshotFilters`(430)、`toYearMonth`(457)、`isMonthlySnapshotStale`(470)、`shouldUseDailyMonthlyAggregation`(531)、`RebuildSettlementCustomerMonthly`(557)、`UpdateRecalculated`(708)、`calcServiceYearIndex`(735)、`findMatchedDiscountRule`(751)、`findDiscountRatioByYear`(777)、`discountRuleFieldSet`(800)。

每个新文件头部 `package repository` + 按需 import（`go build` 报错驱动补齐即可）。Task 12 创建的 `settlement_data_integrity.go` 保持不动。

- [ ] **Step 4: 验证零行为变更**

Run: `cd backend && gofmt -l ./internal/repository && go build ./... && go test ./internal/repository ./internal/service`
Expected: gofmt 无输出，测试 PASS；`git diff --stat` 确认只有文件间移动、无逻辑改动

- [ ] **Step 5: Commit**

```bash
git add backend/internal/repository
git commit -m "refactor(repo): split settlement_data_repository into slots/backfill/monthly files" -m "Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>"
```

---

### Task 16: TrafficView.vue 抽离时间工具函数

**Files:**
- Create: `frontend/frontend/src/views/traffic-view-time-utils.ts`
- Test: `frontend/frontend/src/views/__tests__/traffic-view-time-utils.spec.ts`
- Modify: `frontend/frontend/src/views/TrafficView.vue`

- [ ] **Step 1: 写失败测试**

创建 `frontend/frontend/src/views/__tests__/traffic-view-time-utils.spec.ts`：

```ts
import { describe, it, expect } from 'vitest'
import {
  toMinuteKeyStr,
  parseQueryTimeInput,
  normalizeTimeRangeForRequest,
  toFiveMinuteKeyISO,
} from '../traffic-view-time-utils'

describe('traffic-view-time-utils', () => {
  it('toMinuteKeyStr 解析 ISO / 斜杠格式到分钟键', () => {
    expect(toMinuteKeyStr('2025-11-05T15:45:30Z')).toBe('2025-11-05 15:45')
    expect(toMinuteKeyStr('2025/11/05 15:45')).toBe('2025-11-05 15:45')
  })

  it('parseQueryTimeInput 支持 YYYY-MM-DD HH:mm:ss', () => {
    const d = parseQueryTimeInput('2026-07-01 08:30:00')
    expect(d).not.toBeNull()
    expect(d!.getHours()).toBe(8)
    expect(parseQueryTimeInput('')).toBeNull()
  })

  it('normalizeTimeRangeForRequest 拒绝倒序区间', () => {
    expect(() => normalizeTimeRangeForRequest('2026-07-02 00:00:00', '2026-07-01 00:00:00')).toThrow()
  })

  it('normalizeTimeRangeForRequest 返回 RFC3339', () => {
    const r = normalizeTimeRangeForRequest('2026-07-01 00:00:00', '2026-07-02 00:00:00')
    expect(r.startRFC3339.endsWith('Z')).toBe(true)
  })

  it('toFiveMinuteKeyISO 归一化到 5 分钟桶', () => {
    expect(toFiveMinuteKeyISO(new Date('2026-07-01T08:03:20Z'))).toBe('2026-07-01T08:00:00Z')
    expect(toFiveMinuteKeyISO(new Date('2026-07-01T08:07:00Z'))).toBe('2026-07-01T08:05:00Z')
  })
})
```

Run: `cd frontend/frontend && npm run test:unit -- src/views/__tests__/traffic-view-time-utils.spec.ts`
Expected: FAIL（模块不存在）

- [ ] **Step 2: 移动函数**

创建 `frontend/frontend/src/views/traffic-view-time-utils.ts`，从 `TrafficView.vue` 第 37-160 行**原样剪切**以下 7 个函数并加 `export`：`formatLabel`、`toMinuteKeyStr`、`toRFC3339Seconds`、`parseQueryTimeInput`、`normalizeTimeRangeForRequest`、`toFiveMinuteKeyISO`、`parseTime`（函数体一字不改）。

`TrafficView.vue` script 顶部追加：

```ts
import {
  formatLabel,
  toMinuteKeyStr,
  toRFC3339Seconds,
  parseQueryTimeInput,
  normalizeTimeRangeForRequest,
  toFiveMinuteKeyISO,
  parseTime,
} from './traffic-view-time-utils'
```

（若 type-check 报某函数未使用，从 import 中移除该项而不是删除导出。）

- [ ] **Step 3: 验证 + Commit**

Run: `cd frontend/frontend && npm run type-check && npm run test:unit`
Expected: PASS

```bash
git add frontend/frontend/src/views/traffic-view-time-utils.ts frontend/frontend/src/views/__tests__/traffic-view-time-utils.spec.ts frontend/frontend/src/views/TrafficView.vue
git commit -m "refactor(web): extract TrafficView time helpers into tested module" -m "Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>"
```

---

### Task 17: SettlementDataTab.vue 抽离金额明细纯函数

**Files:**
- Create: `frontend/frontend/src/components/settlement/settlement-amount-utils.ts`
- Test: `frontend/frontend/src/components/settlement/__tests__/settlement-amount-utils.spec.ts`
- Modify: `frontend/frontend/src/components/settlement/SettlementDataTab.vue`

- [ ] **Step 1: 移动纯函数**

创建 `settlement-amount-utils.ts`，从 `SettlementDataTab.vue` **原样剪切**以下 7 个模块级纯函数（原行号：327/346/350/360/369/381/396）并加 `export`：`daysInMonthFrom`、`toFixedNum`、`formatMoney`、`parseDateOnlyToDate`、`calcServiceYearIndexFront`、`findDiscountRatioByYearFront`、`ruleAffectsField`（函数体一字不改；这些函数不引用组件状态，若剪切时发现某函数引用了 ref/computed 则留在原文件并从本任务移出）。

`SettlementDataTab.vue` 增加 import：

```ts
import {
  daysInMonthFrom,
  toFixedNum,
  formatMoney,
  parseDateOnlyToDate,
  calcServiceYearIndexFront,
  findDiscountRatioByYearFront,
  ruleAffectsField,
} from './settlement-amount-utils'
```

- [ ] **Step 2: 写特征测试（characterization）**

创建 `__tests__/settlement-amount-utils.spec.ts`：

```ts
import { describe, it, expect } from 'vitest'
import { daysInMonthFrom, parseDateOnlyToDate, toFixedNum } from '../settlement-amount-utils'

describe('settlement-amount-utils', () => {
  it('parseDateOnlyToDate 解析 YYYY-MM-DD', () => {
    const d = parseDateOnlyToDate('2026-07-03')
    expect(d).not.toBeNull()
    expect(d!.getFullYear()).toBe(2026)
    expect(d!.getMonth()).toBe(6)
    expect(d!.getDate()).toBe(3)
  })

  it('parseDateOnlyToDate 非法输入返回 null', () => {
    expect(parseDateOnlyToDate('not-a-date')).toBeNull()
    expect(parseDateOnlyToDate(null)).toBeNull()
  })

  it('daysInMonthFrom 返回所在月天数', () => {
    expect(daysInMonthFrom('2026-02-15')).toBe(28)
    expect(daysInMonthFrom('2024-02-15')).toBe(29)
    expect(daysInMonthFrom('2026-07-01')).toBe(31)
  })

  it('toFixedNum 保留两位小数', () => {
    expect(toFixedNum(1.005, 2)).toMatch(/^1\.0[01]$/)
    expect(toFixedNum(2, 2)).toBe('2.00')
  })
})
```

注：这是移动后的**特征测试**——若断言与实际行为不符（如 `daysInMonthFrom` 对无效输入有 fallback），以移动前后行为一致为准修正断言，不改函数实现。

- [ ] **Step 3: 验证 + Commit**

Run: `cd frontend/frontend && npm run type-check && npm run test:unit`
Expected: PASS

```bash
git add frontend/frontend/src/components/settlement/settlement-amount-utils.ts frontend/frontend/src/components/settlement/__tests__/settlement-amount-utils.spec.ts frontend/frontend/src/components/settlement/SettlementDataTab.vue
git commit -m "refactor(web): extract settlement amount pure helpers with characterization tests" -m "Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>"
```

---

### Task 18: 文档收尾

**Files:**
- Modify: `CLAUDE.md`

- [ ] **Step 1: 在 CLAUDE.md 的 Business Semantics → Settlement 段落末尾追加**

```markdown
- 结算调度器多实例安全：每 tick 先抢 MySQL `GET_LOCK('nfa:settlement:scheduler')`，创建前按 `(task_type, task_date)` 去重；`scheduler.enabled`（env `SCHEDULER_ENABLED`）可关停本实例调度。
- 任务状态含 `partial`（周结算部分天失败）与 `interrupted`（30 分钟无进度更新被清扫）；失败/中断/缺数通过 `alert.feishu_webhook_url`（env `ALERT_FEISHU_WEBHOOK_URL`）告警。
- 完整性巡检：`GET /api/v1/settlement/data/integrity`（源表 vs 客户表逐日行数，非零中位数为基线，<80% 为 low，0 为 missing）；调度器每天 09:00 自动巡检昨日并告警。
```

- [ ] **Step 2: Commit**

```bash
git add CLAUDE.md
git commit -m "docs: document scheduler lock, task statuses, alerting and integrity check" -m "Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>"
```

---

## 明确不在本计划内（后续单独立项）

1. **费率生效日期版本化**与**账期封账**（用户指定暂缓）。
2. **SettlementDataTab 导出对话框组件化**（约 300 行，与组件状态深度耦合，值得做但需要单独的拆分设计）。
3. **TrafficView chartOption 抽成 composable**（约 260 行，依赖十余个响应式绑定，同上）。
4. **EDC 节点任务（`edc_node_settlement_service`）的 partial 状态**：本计划的卡死清扫与失败告警已覆盖它（走 `UpdateSettlementTaskStatus` / 清扫器），逐期 partial 粒度另行评估。
