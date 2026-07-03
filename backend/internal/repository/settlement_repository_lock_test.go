package repository

import (
	"os"
	"testing"
	"time"

	"nfa-dashboard/internal/model"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// 警告：NFA_TEST_MYSQL_DSN 必须指向可随意写入的一次性测试库，
// 切勿指向共享/生产库（本测试会清扫整表 running 任务）。
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
	prevDB := model.DB
	model.DB = db
	t.Cleanup(func() { model.DB = prevDB })
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
	if got.TaskStage != "interrupted" {
		t.Fatalf("task_stage=%s, want interrupted", got.TaskStage)
	}
	if got.EndTime == nil {
		t.Fatal("end_time should be set")
	}
	if got.ErrorMessage == "" {
		t.Fatal("error_message should be set")
	}
}
