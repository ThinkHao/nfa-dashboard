package service

import (
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
