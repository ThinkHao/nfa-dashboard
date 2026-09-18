package service

import (
	"testing"
	"time"

	"nfa-dashboard/internal/model"
)

type trafficReportRoleRepoStub struct {
	roles []model.Role
}

func (s trafficReportRoleRepoStub) GetUserRoles(uint64) ([]model.Role, error) {
	return s.roles, nil
}

type trafficReportAccessRepoStub struct {
	listTasksOwnerID uint64
	listRunsOwnerID  uint64
}

func (s *trafficReportAccessRepoStub) CreateTask(*model.TrafficReportTask) error { return nil }

func (s *trafficReportAccessRepoStub) GetTask(id, ownerID uint64) (*model.TrafficReportTask, error) {
	return &model.TrafficReportTask{ID: id, OwnerUserID: ownerID}, nil
}

func (s *trafficReportAccessRepoStub) ListTasks(ownerID uint64, _, _ int) ([]model.TrafficReportTask, int64, error) {
	s.listTasksOwnerID = ownerID
	return []model.TrafficReportTask{}, 0, nil
}

func (s *trafficReportAccessRepoStub) UpdateTask(*model.TrafficReportTask) error { return nil }

func (s *trafficReportAccessRepoStub) ListDueTasks(time.Time, int) ([]model.TrafficReportTask, error) {
	return nil, nil
}

func (s *trafficReportAccessRepoStub) ClaimTaskRun(uint64, time.Time, time.Time) bool { return true }

func (s *trafficReportAccessRepoStub) HasActiveRun(uint64) (bool, error) { return false, nil }

func (s *trafficReportAccessRepoStub) CreateRun(*model.TrafficReportRun) error { return nil }

func (s *trafficReportAccessRepoStub) GetRun(string, uint64) (*model.TrafficReportRun, error) {
	return nil, nil
}

func (s *trafficReportAccessRepoStub) ListRuns(ownerID, _ uint64, _, _ int) ([]model.TrafficReportRun, int64, error) {
	s.listRunsOwnerID = ownerID
	return []model.TrafficReportRun{}, 0, nil
}

func (s *trafficReportAccessRepoStub) UpdateRun(*model.TrafficReportRun) error { return nil }

func (s *trafficReportAccessRepoStub) CreateArtifact(*model.TrafficReportArtifact) error {
	return nil
}

func TestTrafficReportServiceAdminCanViewAllTasksAndRuns(t *testing.T) {
	repo := &trafficReportAccessRepoStub{}
	service := NewTrafficReportService(repo, nil, nil, trafficReportRoleRepoStub{
		roles: []model.Role{{Name: " admin "}},
	})

	if _, _, err := service.ListTasks(3, 1, 50); err != nil {
		t.Fatalf("ListTasks() error = %v", err)
	}
	if repo.listTasksOwnerID != 0 {
		t.Fatalf("admin task scope = %d, want unscoped 0", repo.listTasksOwnerID)
	}
	if _, _, err := service.ListRuns(3, 0, 1, 50); err != nil {
		t.Fatalf("ListRuns() error = %v", err)
	}
	if repo.listRunsOwnerID != 0 {
		t.Fatalf("admin run scope = %d, want unscoped 0", repo.listRunsOwnerID)
	}
}

func TestTrafficReportServiceNonAdminKeepsOwnerScope(t *testing.T) {
	repo := &trafficReportAccessRepoStub{}
	service := NewTrafficReportService(repo, nil, nil, trafficReportRoleRepoStub{
		roles: []model.Role{{Name: "operator"}},
	})

	if _, _, err := service.ListTasks(3, 1, 50); err != nil {
		t.Fatalf("ListTasks() error = %v", err)
	}
	if repo.listTasksOwnerID != 3 {
		t.Fatalf("non-admin task scope = %d, want 3", repo.listTasksOwnerID)
	}
	if _, _, err := service.ListRuns(3, 0, 1, 50); err != nil {
		t.Fatalf("ListRuns() error = %v", err)
	}
	if repo.listRunsOwnerID != 3 {
		t.Fatalf("non-admin run scope = %d, want 3", repo.listRunsOwnerID)
	}
}
