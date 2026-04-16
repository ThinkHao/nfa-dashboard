package controller

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"nfa-dashboard/internal/model"
	"nfa-dashboard/internal/repository"

	"github.com/gin-gonic/gin"
)

type settlementRepoImportTaskStub struct {
	repository.SettlementRepository
	mu               sync.Mutex
	tasks            map[int64]*model.SettlementTask
	markFailedCalled int
}

func newSettlementRepoImportTaskStub() *settlementRepoImportTaskStub {
	return &settlementRepoImportTaskStub{tasks: make(map[int64]*model.SettlementTask)}
}

func (s *settlementRepoImportTaskStub) GetSettlementTaskByID(id int64) (*model.SettlementTask, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.tasks[id]
	if !ok {
		return nil, errors.New("not found")
	}
	cp := *t
	return &cp, nil
}

func (s *settlementRepoImportTaskStub) UpdateSettlementTask(task *model.SettlementTask) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := *task
	s.tasks[task.ID] = &cp
	return nil
}

func (s *settlementRepoImportTaskStub) MarkSettlementTaskFailedByID(id int64, message string, updateTime time.Time, taskMeta string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.tasks[id]
	if !ok {
		return nil
	}
	t.Status = "failed"
	t.TaskStage = "failed"
	t.ErrorMessage = message
	t.UpdateTime = updateTime
	t.TaskMeta = taskMeta
	end := updateTime
	t.EndTime = &end
	s.markFailedCalled++
	return nil
}

func TestContinueCustomerImportTask_IdempotentWhenRunning(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := newSettlementRepoImportTaskStub()
	repo.tasks[101] = &model.SettlementTask{
		ID:       101,
		TaskType: customerRateImportTaskType,
		Status:   "running",
	}
	ctl := NewSettlementRatesController(&settlementRatesServiceStub{}, repo)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/settlement/rates/customer/import/tasks/101/continue", nil)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = req
	ctx.Params = gin.Params{{Key: "id", Value: "101"}}

	ctl.ContinueCustomerImportTask(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got := resp["status"]; got != "running" {
		t.Fatalf("expected status=running, got %v", got)
	}
}

func TestContinueCustomerImportTask_IdempotentWhenSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := newSettlementRepoImportTaskStub()
	repo.tasks[102] = &model.SettlementTask{
		ID:       102,
		TaskType: customerRateImportTaskType,
		Status:   "success",
	}
	ctl := NewSettlementRatesController(&settlementRatesServiceStub{}, repo)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/settlement/rates/customer/import/tasks/102/continue", nil)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = req
	ctx.Params = gin.Params{{Key: "id", Value: "102"}}

	ctl.ContinueCustomerImportTask(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got := resp["status"]; got != "success" {
		t.Fatalf("expected status=success, got %v", got)
	}
}

func TestContinueCustomerImportTask_ExpiredWaitingConfirmMarksFailed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := newSettlementRepoImportTaskStub()
	meta := customerRateImportTaskMeta{
		Message:                 "检测到未匹配用户，等待确认自动创建后继续",
		WaitingConfirmExpiresAt: time.Now().Add(-time.Minute).Unix(),
	}
	repo.tasks[103] = &model.SettlementTask{
		ID:        103,
		TaskType:  customerRateImportTaskType,
		Status:    "waiting_user_confirm",
		TaskStage: "waiting_user_confirm",
		TaskMeta:  marshalCustomerRateImportTaskMeta(meta),
	}
	ctl := NewSettlementRatesController(&settlementRatesServiceStub{}, repo)
	ctl.importTaskManager.set(103, customerRateImportTaskRuntime{
		Filename:     "x.csv",
		Content:      []byte("a,b\n1,2"),
		ValidateOnly: false,
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/settlement/rates/customer/import/tasks/103/continue", nil)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = req
	ctx.Params = gin.Params{{Key: "id", Value: "103"}}

	ctl.ContinueCustomerImportTask(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
	task, _ := repo.GetSettlementTaskByID(103)
	if task.Status != "failed" {
		t.Fatalf("expected task status=failed, got %s", task.Status)
	}
}
