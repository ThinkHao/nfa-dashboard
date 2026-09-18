package repository

import (
	"time"

	"nfa-dashboard/internal/model"
)

type TrafficReportRepository interface {
	CreateTask(task *model.TrafficReportTask) error
	GetTask(id, ownerID uint64) (*model.TrafficReportTask, error)
	ListTasks(ownerID uint64, page, pageSize int) ([]model.TrafficReportTask, int64, error)
	UpdateTask(task *model.TrafficReportTask) error
	ListDueTasks(now time.Time, limit int) ([]model.TrafficReportTask, error)
	ClaimTaskRun(taskID uint64, now, nextRun time.Time) bool
	HasActiveRun(taskID uint64) (bool, error)
	CreateRun(run *model.TrafficReportRun) error
	GetRun(id string, ownerID uint64) (*model.TrafficReportRun, error)
	ListRuns(ownerID uint64, taskID uint64, page, pageSize int) ([]model.TrafficReportRun, int64, error)
	UpdateRun(run *model.TrafficReportRun) error
	CreateArtifact(artifact *model.TrafficReportArtifact) error
}

type trafficReportRepository struct{}

func NewTrafficReportRepository() TrafficReportRepository { return &trafficReportRepository{} }

func (r *trafficReportRepository) CreateTask(task *model.TrafficReportTask) error {
	return model.DB.Create(task).Error
}

func (r *trafficReportRepository) GetTask(id, ownerID uint64) (*model.TrafficReportTask, error) {
	var task model.TrafficReportTask
	q := model.DB.Where("id = ?", id)
	if ownerID > 0 {
		q = q.Where("owner_user_id = ?", ownerID)
	}
	if err := q.First(&task).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

func (r *trafficReportRepository) ListTasks(ownerID uint64, page, pageSize int) ([]model.TrafficReportTask, int64, error) {
	var items []model.TrafficReportTask
	var total int64
	q := model.DB.Model(&model.TrafficReportTask{}).Where("owner_user_id = ?", ownerID)
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 50
	}
	if err := q.Order("updated_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *trafficReportRepository) UpdateTask(task *model.TrafficReportTask) error {
	return model.DB.Save(task).Error
}

func (r *trafficReportRepository) ListDueTasks(now time.Time, limit int) ([]model.TrafficReportTask, error) {
	if limit < 1 || limit > 100 {
		limit = 20
	}
	var tasks []model.TrafficReportTask
	err := model.DB.Where("active = ? AND kind = ? AND next_run_at IS NOT NULL AND next_run_at <= ?", true, model.TrafficReportKindPeriodic, now).
		Order("next_run_at ASC").Limit(limit).Find(&tasks).Error
	return tasks, err
}

// ClaimTaskRun advances the next due time atomically. Only the instance that
// changed the row is allowed to create the run.
func (r *trafficReportRepository) ClaimTaskRun(taskID uint64, now, nextRun time.Time) bool {
	result := model.DB.Model(&model.TrafficReportTask{}).
		Where("id = ? AND active = ? AND kind = ? AND next_run_at IS NOT NULL AND next_run_at <= ?", taskID, true, model.TrafficReportKindPeriodic, now).
		Updates(map[string]interface{}{"next_run_at": nextRun, "last_run_at": now})
	return result.Error == nil && result.RowsAffected == 1
}

func (r *trafficReportRepository) HasActiveRun(taskID uint64) (bool, error) {
	var count int64
	err := model.DB.Model(&model.TrafficReportRun{}).
		Where("task_id = ? AND status IN ?", taskID, []string{model.TrafficReportStatusPending, model.TrafficReportStatusRunning}).
		Count(&count).Error
	return count > 0, err
}

func (r *trafficReportRepository) CreateRun(run *model.TrafficReportRun) error {
	return model.DB.Create(run).Error
}

func (r *trafficReportRepository) GetRun(id string, ownerID uint64) (*model.TrafficReportRun, error) {
	var run model.TrafficReportRun
	q := model.DB.Preload("Artifacts").
		Joins("JOIN traffic_report_tasks ON traffic_report_tasks.id = traffic_report_runs.task_id").
		Where("traffic_report_runs.id = ?", id)
	if ownerID > 0 {
		q = q.Where("traffic_report_tasks.owner_user_id = ?", ownerID)
	}
	if err := q.First(&run).Error; err != nil {
		return nil, err
	}
	return &run, nil
}

func (r *trafficReportRepository) ListRuns(ownerID uint64, taskID uint64, page, pageSize int) ([]model.TrafficReportRun, int64, error) {
	var items []model.TrafficReportRun
	var total int64
	q := model.DB.Model(&model.TrafficReportRun{}).
		Joins("JOIN traffic_report_tasks ON traffic_report_tasks.id = traffic_report_runs.task_id").
		Where("traffic_report_tasks.owner_user_id = ?", ownerID)
	if taskID > 0 {
		q = q.Where("traffic_report_runs.task_id = ?", taskID)
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 50
	}
	if err := q.Preload("Artifacts").Order("traffic_report_runs.created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *trafficReportRepository) UpdateRun(run *model.TrafficReportRun) error {
	return model.DB.Save(run).Error
}

func (r *trafficReportRepository) CreateArtifact(artifact *model.TrafficReportArtifact) error {
	return model.DB.Create(artifact).Error
}
