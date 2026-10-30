package service

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"nfa-dashboard/config"
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
	task             *model.TrafficReportTask
	createdTask      *model.TrafficReportTask
	updatedTask      *model.TrafficReportTask
	deletedTaskID    uint64
	deletedOwnerID   uint64
}

func (s *trafficReportAccessRepoStub) CreateTask(task *model.TrafficReportTask) error {
	task.ID = 99
	s.createdTask = task
	return nil
}

func (s *trafficReportAccessRepoStub) GetTask(id, ownerID uint64) (*model.TrafficReportTask, error) {
	if s.task != nil {
		return s.task, nil
	}
	return &model.TrafficReportTask{ID: id, OwnerUserID: ownerID}, nil
}

func (s *trafficReportAccessRepoStub) ListTasks(ownerID uint64, _, _ int) ([]model.TrafficReportTask, int64, error) {
	s.listTasksOwnerID = ownerID
	return []model.TrafficReportTask{}, 0, nil
}

func (s *trafficReportAccessRepoStub) ListSuccessfulRuns(uint64) ([]model.TrafficReportRun, error) {
	return s.successfulRuns, nil
}

func (s *trafficReportAccessRepoStub) UpdateTask(task *model.TrafficReportTask) error {
	s.updatedTask = task
	return nil
}

func (s *trafficReportAccessRepoStub) DeleteTask(id, ownerID uint64) error {
	s.deletedTaskID = id
	s.deletedOwnerID = ownerID
	return nil
}

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

func TestTrafficReportServiceDuplicateTaskCopiesSettingsAndStartsPaused(t *testing.T) {
	original := &model.TrafficReportTask{
		ID: 12, OwnerUserID: 34, Name: "重点院校", Kind: model.TrafficReportKindPeriodic, Active: true,
		DataSourceType: "nfa", Timezone: "Asia/Shanghai", WindowSelector: "last_month",
		WindowParams: []byte(`{"n":7}`), Params: []byte(`{"cp":"bilibili"}`), ExportFormats: []byte(`["xlsx"]`),
	}
	repo := &trafficReportAccessRepoStub{task: original}
	svc := NewTrafficReportService(repo, nil, nil, trafficReportRoleRepoStub{})

	clone, err := svc.DuplicateTask(55, original.ID)
	if err != nil {
		t.Fatalf("DuplicateTask() error = %v", err)
	}
	if clone.ID != 99 || clone.OwnerUserID != original.OwnerUserID || clone.Name != "重点院校 - 副本" {
		t.Fatalf("unexpected duplicate identity: %+v", clone)
	}
	if clone.Active || clone.NextRunAt != nil || clone.LastRunAt != nil || clone.DeletedAt != nil {
		t.Fatalf("duplicate should start paused without run history: %+v", clone)
	}
	if string(clone.Params) != string(original.Params) || string(clone.WindowParams) != string(original.WindowParams) {
		t.Fatalf("duplicate settings changed: params=%s window=%s", clone.Params, clone.WindowParams)
	}
}

func TestTrafficReportServiceDeleteTaskUsesScopedSoftDelete(t *testing.T) {
	repo := &trafficReportAccessRepoStub{}
	svc := NewTrafficReportService(repo, nil, nil, trafficReportRoleRepoStub{})

	if err := svc.DeleteTask(55, 12); err != nil {
		t.Fatalf("DeleteTask() error = %v", err)
	}
	if repo.deletedTaskID != 12 || repo.deletedOwnerID != 55 {
		t.Fatalf("DeleteTask() scope = (%d, %d), want (12, 55)", repo.deletedTaskID, repo.deletedOwnerID)
	}
}

func TestTrafficReportServiceUpdateTaskReplacesReportSettings(t *testing.T) {
	original := &model.TrafficReportTask{
		ID: 12, OwnerUserID: 55, Name: "旧名称", Kind: model.TrafficReportKindOneOff, Active: true,
		DataSourceType: "nfa", Timezone: "Asia/Shanghai", WindowSelector: "last_month",
		WindowParams: []byte(`{}`), Params: []byte(`{"cp":"old"}`), ExportFormats: []byte(`["xlsx"]`),
	}
	repo := &trafficReportAccessRepoStub{task: original}
	svc := NewTrafficReportService(repo, nil, nil, trafficReportRoleRepoStub{})
	name, kind, source, selector, scheduleType, scheduleExpr := "新名称", model.TrafficReportKindPeriodic, "edc", "last_n_days", "daily", "03:15"
	params := map[string]interface{}{"edc_name": "nrt-edge"}
	windowParams := map[string]interface{}{"n": float64(14)}

	updated, err := svc.UpdateTask(55, original.ID, TrafficReportTaskUpdate{
		Name: &name, Kind: &kind, DataSourceType: &source, WindowSelector: &selector,
		ScheduleType: &scheduleType, ScheduleExpr: &scheduleExpr, Params: &params, WindowParams: &windowParams,
	})
	if err != nil {
		t.Fatalf("UpdateTask() error = %v", err)
	}
	if updated.Name != name || updated.Kind != kind || updated.DataSourceType != source || updated.WindowSelector != selector {
		t.Fatalf("report settings were not replaced: %+v", updated)
	}
	if repo.updatedTask != updated || updated.NextRunAt == nil || updated.ScheduleType == nil || updated.ScheduleExpr == nil || *updated.ScheduleExpr != scheduleExpr {
		t.Fatalf("periodic schedule was not saved: %+v", updated)
	}
	if string(updated.Params) != `{"edc_name":"nrt-edge"}` || string(updated.WindowParams) != `{"n":14}` {
		t.Fatalf("updated params were not serialized: params=%s window=%s", updated.Params, updated.WindowParams)
	}
}

func TestTrafficReportServiceGroupsDownloadMonthsByFinishedTime(t *testing.T) {
	sepOne := time.Date(2026, 9, 3, 1, 0, 0, 0, time.UTC)
	sepTwo := time.Date(2026, 9, 8, 1, 0, 0, 0, time.UTC)
	aug := time.Date(2026, 8, 31, 15, 59, 0, 0, time.UTC)
	repo := &trafficReportAccessRepoStub{successfulRuns: []model.TrafficReportRun{
		{ID: "sep-1", FinishedAt: &sepOne, Artifacts: []model.TrafficReportArtifact{{FileName: "report-1.xlsx", FileSize: 10}, {FileName: "report-1.csv", FileSize: 5}}},
		{ID: "sep-2", FinishedAt: &sepTwo, Artifacts: []model.TrafficReportArtifact{{FileName: "report-2.XLSX", FileSize: 7}}},
		{ID: "aug-1", FinishedAt: &aug, Artifacts: []model.TrafficReportArtifact{{FileName: "report-3.xlsx", FileSize: 3}}},
	}}
	service := NewTrafficReportService(repo, nil, nil, trafficReportRoleRepoStub{roles: []model.Role{{Name: "admin"}}})

	months, err := service.ListDownloadMonths(3)
	if err != nil {
		t.Fatalf("ListDownloadMonths() error = %v", err)
	}
	if len(months) != 2 || months[0].Month != "2026-09" || months[1].Month != "2026-08" {
		t.Fatalf("months = %#v, want 2026-09 then 2026-08", months)
	}
	if months[0].RunCount != 2 || months[0].ArtifactCount != 2 || months[0].TotalSize != 17 {
		t.Fatalf("September summary = %#v", months[0])
	}
}

func TestCreateMonthlyArchiveIncludesOnlyXLSXArtifacts(t *testing.T) {
	root := t.TempDir()
	previousStorageDir := config.AppConfig.TrafficReport.StorageDir
	config.AppConfig.TrafficReport.StorageDir = root
	t.Cleanup(func() { config.AppConfig.TrafficReport.StorageDir = previousStorageDir })

	runID := "run-1"
	runDir := filepath.Join(root, runID)
	if err := os.MkdirAll(runDir, 0o700); err != nil {
		t.Fatal(err)
	}
	artifacts := []model.TrafficReportArtifact{
		{ID: 1, FileName: "report.xlsx"},
		{ID: 2, FileName: "report.csv"},
		{ID: 3, FileName: "report-metadata.json"},
	}
	for i := range artifacts {
		path := filepath.Join(runDir, artifacts[i].FileName)
		if err := os.WriteFile(path, []byte(artifacts[i].FileName), 0o600); err != nil {
			t.Fatal(err)
		}
		artifacts[i].StoragePath = path
	}
	finishedAt := time.Date(2026, 9, 8, 10, 0, 0, 0, time.FixedZone("Asia/Shanghai", 8*60*60))
	repo := &trafficReportAccessRepoStub{successfulRuns: []model.TrafficReportRun{{ID: runID, FinishedAt: &finishedAt, Artifacts: artifacts}}}
	service := NewTrafficReportService(repo, nil, nil, nil)
	archive, err := service.CreateMonthlyArchive(3, "2026-09", nil)
	if err != nil {
		t.Fatalf("CreateMonthlyArchive() error = %v", err)
	}
	if archive.ArtifactCount != 1 {
		t.Fatalf("archive artifact count = %d, want 1", archive.ArtifactCount)
	}
	reader, err := zip.OpenReader(archive.Path)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	if len(reader.File) != 1 || !strings.HasSuffix(reader.File[0].Name, ".xlsx") {
		t.Fatalf("archive files = %#v, want one XLSX", reader.File)
	}
}

func TestSafeTrafficReportArchiveNameRemovesPathSegments(t *testing.T) {
	if got := safeTrafficReportArchiveName(`..\nested/report.csv`); got != "report.csv" {
		t.Fatalf("safeTrafficReportArchiveName() = %q, want report.csv", got)
	}
}
