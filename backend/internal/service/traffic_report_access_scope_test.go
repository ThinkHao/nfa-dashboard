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
	successfulRuns   []model.TrafficReportRun
}

func (s *trafficReportAccessRepoStub) CreateTask(*model.TrafficReportTask) error { return nil }

func (s *trafficReportAccessRepoStub) GetTask(id, ownerID uint64) (*model.TrafficReportTask, error) {
	return &model.TrafficReportTask{ID: id, OwnerUserID: ownerID}, nil
}

func (s *trafficReportAccessRepoStub) ListTasks(ownerID uint64, _, _ int) ([]model.TrafficReportTask, int64, error) {
	s.listTasksOwnerID = ownerID
	return []model.TrafficReportTask{}, 0, nil
}

func (s *trafficReportAccessRepoStub) ListSuccessfulRuns(uint64) ([]model.TrafficReportRun, error) {
	return s.successfulRuns, nil
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

func TestTrafficReportServiceGroupsDownloadMonthsByFinishedTime(t *testing.T) {
	sepOne := time.Date(2026, 9, 3, 1, 0, 0, 0, time.UTC)
	sepTwo := time.Date(2026, 9, 8, 1, 0, 0, 0, time.UTC)
	aug := time.Date(2026, 8, 31, 15, 59, 0, 0, time.UTC)
	repo := &trafficReportAccessRepoStub{successfulRuns: []model.TrafficReportRun{
		{ID: "sep-1", FinishedAt: &sepOne, Artifacts: []model.TrafficReportArtifact{{FileSize: 10}, {FileSize: 5}}},
		{ID: "sep-2", FinishedAt: &sepTwo, Artifacts: []model.TrafficReportArtifact{{FileSize: 7}}},
		{ID: "aug-1", FinishedAt: &aug, Artifacts: []model.TrafficReportArtifact{{FileSize: 3}}},
	}}
	service := NewTrafficReportService(repo, nil, nil, trafficReportRoleRepoStub{roles: []model.Role{{Name: "admin"}}})

	months, err := service.ListDownloadMonths(3)
	if err != nil {
		t.Fatalf("ListDownloadMonths() error = %v", err)
	}
	if len(months) != 2 || months[0].Month != "2026-09" || months[1].Month != "2026-08" {
		t.Fatalf("months = %#v, want 2026-09 then 2026-08", months)
	}
	if months[0].RunCount != 2 || months[0].ArtifactCount != 3 || months[0].TotalSize != 22 {
		t.Fatalf("September summary = %#v", months[0])
	}
}

func TestSafeTrafficReportArchiveNameRemovesPathSegments(t *testing.T) {
	if got := safeTrafficReportArchiveName(`..\nested/report.csv`); got != "report.csv" {
		t.Fatalf("safeTrafficReportArchiveName() = %q, want report.csv", got)
	}
}
