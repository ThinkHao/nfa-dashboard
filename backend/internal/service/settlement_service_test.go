package service

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"nfa-dashboard/internal/model"
	"nfa-dashboard/internal/notify"
	"nfa-dashboard/internal/repository"
)

func TestShouldFailCustomerInitOnZeroAffected(t *testing.T) {
	cases := []struct {
		name     string
		srcCount int64
		affected int64
		want     bool
	}{
		{name: "source positive affected zero", srcCount: 10, affected: 0, want: true},
		{name: "source zero affected zero", srcCount: 0, affected: 0, want: false},
		{name: "source positive affected positive", srcCount: 10, affected: 5, want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := shouldFailCustomerInitOnZeroAffected(tc.srcCount, tc.affected)
			if got != tc.want {
				t.Fatalf("shouldFailCustomerInitOnZeroAffected(%d,%d)=%v, want=%v", tc.srcCount, tc.affected, got, tc.want)
			}
		})
	}
}

type createTaskRepoStub struct {
	repository.SettlementRepository
	listCalled bool
}

func (s *createTaskRepoStub) CreateSettlementTask(task *model.SettlementTask) error {
	task.ID = 42
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
