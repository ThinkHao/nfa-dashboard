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

func TestPreviousWeekStartReturnsLastMondayForMondayTrigger(t *testing.T) {
	now := time.Date(2026, 7, 6, 2, 0, 0, 0, time.Local)
	got := previousWeekStart(now)
	if got.Format("2006-01-02") != "2026-06-29" {
		t.Fatalf("previousWeekStart()=%s, want 2026-06-29", got.Format("2006-01-02"))
	}
}

func TestPreviousWeekStartReturnsLastMondayForMidWeekTrigger(t *testing.T) {
	now := time.Date(2026, 7, 8, 2, 0, 0, 0, time.Local)
	got := previousWeekStart(now)
	if got.Format("2006-01-02") != "2026-06-29" {
		t.Fatalf("previousWeekStart()=%s, want 2026-06-29", got.Format("2006-01-02"))
	}
}
