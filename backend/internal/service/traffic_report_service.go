package service

import (
	"context"
	"crypto/rand"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
	"nfa-dashboard/config"
	"nfa-dashboard/internal/model"
	"nfa-dashboard/internal/repository"
	"nfa-dashboard/internal/settlement95"

	"gorm.io/datatypes"
)

const trafficReportEngineVersion = "go-v1"

type TrafficReportTaskInput struct {
	Name           string                 `json:"name" binding:"required"`
	Kind           string                 `json:"kind"`
	Active         *bool                  `json:"active"`
	DataSourceType string                 `json:"data_source_type" binding:"required"`
	ScheduleType   string                 `json:"schedule_type"`
	ScheduleExpr   string                 `json:"schedule_expr"`
	Timezone       string                 `json:"timezone"`
	WindowSelector string                 `json:"window_selector"`
	WindowParams   map[string]interface{} `json:"window_params"`
	Params         map[string]interface{} `json:"params"`
	ExportFormats  []string               `json:"export_formats"`
}

type TrafficReportTaskUpdate struct {
	Name         *string `json:"name"`
	Active       *bool   `json:"active"`
	ScheduleType *string `json:"schedule_type"`
	ScheduleExpr *string `json:"schedule_expr"`
}

type TrafficReportTaskMigrationResult struct {
	Task     *model.TrafficReportTask `json:"task"`
	Warnings []string                 `json:"warnings,omitempty"`
}

type trafficReportParams struct {
	SchoolName     string   `json:"school_name"`
	SchoolNames    []string `json:"school_names"`
	Region         string   `json:"region"`
	CP             string   `json:"cp"`
	EntityIDs      []uint64 `json:"entity_ids"`
	EntityType     string   `json:"entity_type"`
	SrcRegion      string   `json:"src_region"`
	DstRegion      string   `json:"dst_region"`
	Direction      string   `json:"direction"`
	UnitBase       int      `json:"unit_base"`
	SettlementMode string   `json:"settlement_mode"`
	BudgetEnabled  bool     `json:"data_budget_enabled"`
	BudgetMul      float64  `json:"data_budget_mul"`
	BudgetDiv      float64  `json:"data_budget_div"`
}

type trafficReportNFARow struct {
	CreateTime time.Time `gorm:"column:create_time"`
	SchoolID   string    `gorm:"column:school_id"`
	SchoolName string    `gorm:"column:school_name"`
	Region     string    `gorm:"column:region"`
	CP         string    `gorm:"column:cp"`
	Recv       int64     `gorm:"column:recv_bytes"`
	Send       int64     `gorm:"column:send_bytes"`
}

type trafficReportEDCRow struct {
	CreateTime  time.Time `gorm:"column:create_time"`
	EntityID    uint64    `gorm:"column:entity_id"`
	EntityName  string    `gorm:"column:entity_name"`
	Alias       string    `gorm:"column:alias"`
	Region      string    `gorm:"column:region"`
	CP          string    `gorm:"column:cp"`
	EntityType  string    `gorm:"column:entity_type"`
	SrcRegion   string    `gorm:"column:src_region"`
	DstRegion   string    `gorm:"column:dst_region"`
	ServiceSize int64     `gorm:"column:service_bytes"`
	CacheSize   int64     `gorm:"column:cache_bytes"`
}

type TrafficReportService interface {
	CreateTask(ownerID uint64, input TrafficReportTaskInput) (*model.TrafficReportTask, error)
	GetTask(ownerID, id uint64) (*model.TrafficReportTask, error)
	ListTasks(ownerID uint64, page, pageSize int) ([]model.TrafficReportTask, int64, error)
	UpdateTask(ownerID, id uint64, input TrafficReportTaskUpdate) (*model.TrafficReportTask, error)
	MigrateTaskToGoV1(ownerID, id uint64) (*TrafficReportTaskMigrationResult, error)
	StartRun(ownerID, taskID uint64) (string, error)
	GetRun(ownerID uint64, runID string) (*model.TrafficReportRun, error)
	ListRuns(ownerID, taskID uint64, page, pageSize int) ([]model.TrafficReportRun, int64, error)
	DownloadArtifact(ownerID uint64, artifactID uint64) (*model.TrafficReportArtifact, error)
	RunDueTasks(ctx context.Context)
}

type trafficReportService struct {
	repo         repository.TrafficReportRepository
	trafficScope TrafficScopeService
	edcScope     EDCTrafficScopeService
	roleRepo     trafficReportRoleRepository
}

type trafficReportRoleRepository interface {
	GetUserRoles(userID uint64) ([]model.Role, error)
}

func NewTrafficReportService(repo repository.TrafficReportRepository, trafficScope TrafficScopeService, edcScope EDCTrafficScopeService, roleRepo trafficReportRoleRepository) TrafficReportService {
	return &trafficReportService{repo: repo, trafficScope: trafficScope, edcScope: edcScope, roleRepo: roleRepo}
}

func (s *trafficReportService) CreateTask(ownerID uint64, input TrafficReportTaskInput) (*model.TrafficReportTask, error) {
	if ownerID == 0 {
		return nil, NewBadRequest("invalid owner")
	}
	if strings.TrimSpace(input.Name) == "" {
		return nil, NewBadRequest("name is required")
	}
	if input.DataSourceType != "nfa" && input.DataSourceType != "edc" {
		return nil, NewBadRequest("data_source_type must be nfa or edc")
	}
	if input.Kind == "" {
		input.Kind = model.TrafficReportKindOneOff
	}
	if input.Kind != model.TrafficReportKindOneOff && input.Kind != model.TrafficReportKindPeriodic {
		return nil, NewBadRequest("kind must be one_off or periodic")
	}
	if input.Timezone == "" {
		input.Timezone = "Asia/Shanghai"
	}
	if _, err := time.LoadLocation(input.Timezone); err != nil {
		return nil, NewBadRequest("invalid timezone")
	}
	if input.WindowSelector == "" {
		input.WindowSelector = "custom"
	}
	if input.WindowSelector != "custom" && input.WindowSelector != "last_week" && input.WindowSelector != "last_month" && input.WindowSelector != "last_n_days" {
		return nil, NewBadRequest("unsupported window_selector")
	}
	if input.Kind == model.TrafficReportKindPeriodic {
		if input.ScheduleType == "" {
			input.ScheduleType = "daily"
		}
		if _, err := nextTrafficReportTime(time.Now(), input.ScheduleType, input.ScheduleExpr, input.Timezone); err != nil {
			return nil, NewBadRequest(err.Error())
		}
	}
	if len(input.ExportFormats) == 0 {
		input.ExportFormats = []string{"csv"}
	}
	for _, f := range input.ExportFormats {
		if f != "csv" && f != "xlsx" {
			return nil, NewBadRequest("export_formats only supports csv and xlsx")
		}
	}
	if input.Params == nil {
		input.Params = map[string]interface{}{}
	}
	if input.WindowParams == nil {
		input.WindowParams = map[string]interface{}{}
	}
	active := true
	if input.Active != nil {
		active = *input.Active
	}
	windowParams, _ := json.Marshal(input.WindowParams)
	params, _ := json.Marshal(input.Params)
	formats, _ := json.Marshal(input.ExportFormats)
	task := &model.TrafficReportTask{
		OwnerUserID: ownerID, Name: strings.TrimSpace(input.Name), Kind: input.Kind, Active: active,
		DataSourceType: input.DataSourceType, Timezone: input.Timezone, WindowSelector: input.WindowSelector,
		WindowParams: datatypes.JSON(windowParams), Params: datatypes.JSON(params), ExportFormats: datatypes.JSON(formats),
	}
	if input.ScheduleType != "" {
		task.ScheduleType = &input.ScheduleType
	}
	if input.ScheduleExpr != "" {
		task.ScheduleExpr = &input.ScheduleExpr
	}
	if input.Kind == model.TrafficReportKindPeriodic && active {
		next, _ := nextTrafficReportTime(time.Now(), input.ScheduleType, input.ScheduleExpr, input.Timezone)
		task.NextRunAt = &next
	}
	if err := s.repo.CreateTask(task); err != nil {
		return nil, err
	}
	return task, nil
}

func (s *trafficReportService) GetTask(ownerID, id uint64) (*model.TrafficReportTask, error) {
	accessOwnerID, err := s.accessOwnerID(ownerID)
	if err != nil {
		return nil, err
	}
	return s.repo.GetTask(id, accessOwnerID)
}

func (s *trafficReportService) ListTasks(ownerID uint64, page, pageSize int) ([]model.TrafficReportTask, int64, error) {
	accessOwnerID, err := s.accessOwnerID(ownerID)
	if err != nil {
		return nil, 0, err
	}
	return s.repo.ListTasks(accessOwnerID, page, pageSize)
}

func (s *trafficReportService) UpdateTask(ownerID, id uint64, input TrafficReportTaskUpdate) (*model.TrafficReportTask, error) {
	accessOwnerID, err := s.accessOwnerID(ownerID)
	if err != nil {
		return nil, err
	}
	task, err := s.repo.GetTask(id, accessOwnerID)
	if err != nil {
		return nil, err
	}
	if input.Name != nil && strings.TrimSpace(*input.Name) != "" {
		task.Name = strings.TrimSpace(*input.Name)
	}
	if input.ScheduleType != nil {
		task.ScheduleType = input.ScheduleType
	}
	if input.ScheduleExpr != nil {
		task.ScheduleExpr = input.ScheduleExpr
	}
	if input.Active != nil {
		task.Active = *input.Active
	}
	if task.Kind == model.TrafficReportKindPeriodic && task.Active {
		typ, expr := "daily", ""
		if task.ScheduleType != nil && *task.ScheduleType != "" {
			typ = *task.ScheduleType
		}
		if task.ScheduleExpr != nil {
			expr = *task.ScheduleExpr
		}
		next, e := nextTrafficReportTime(time.Now(), typ, expr, task.Timezone)
		if e != nil {
			return nil, NewBadRequest(e.Error())
		}
		task.NextRunAt = &next
	} else {
		task.NextRunAt = nil
	}
	if err := s.repo.UpdateTask(task); err != nil {
		return nil, err
	}
	return task, nil
}

func (s *trafficReportService) MigrateTaskToGoV1(ownerID, id uint64) (*TrafficReportTaskMigrationResult, error) {
	accessOwnerID, err := s.accessOwnerID(ownerID)
	if err != nil {
		return nil, err
	}
	task, err := s.repo.GetTask(id, accessOwnerID)
	if err != nil {
		return nil, err
	}
	if _, ok := legacyTaskID(task.Params); !ok {
		return nil, NewBadRequest("仅支持将导入的 nfatool 任务迁移到 go-v1")
	}

	legacyParams := legacyOriginalParams(task.Params)
	params, warnings, err := migrateLegacyParamsToGoV1(task.DataSourceType, legacyParams)
	if err != nil {
		return nil, NewBadRequest(err.Error())
	}
	params["_traffic_report_migration"] = map[string]interface{}{
		"source_task_id": id,
		"source_engine":  trafficReportLegacyNativeEngineVersion,
		"created_at":     time.Now().UTC().Format(time.RFC3339),
	}
	paramBytes, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}

	windowParams := cloneJSONMap(task.WindowParams)
	if task.WindowSelector == "custom" {
		if _, ok := windowParams["start_time"]; !ok {
			if value, exists := legacyParams["start_time"]; exists {
				windowParams["start_time"] = value
			}
		}
		if _, ok := windowParams["end_time"]; !ok {
			if value, exists := legacyParams["end_time"]; exists {
				windowParams["end_time"] = value
			}
		}
	}
	windowBytes, err := json.Marshal(windowParams)
	if err != nil {
		return nil, err
	}

	name := strings.TrimSpace(task.Name) + "（go-v1迁移）"
	if len([]rune(name)) > 200 {
		name = string([]rune(name)[:200])
	}
	clone := &model.TrafficReportTask{
		OwnerUserID:    ownerID,
		Name:           name,
		Kind:           task.Kind,
		Active:         false,
		DataSourceType: task.DataSourceType,
		Timezone:       task.Timezone,
		WindowSelector: task.WindowSelector,
		WindowParams:   windowBytes,
		Params:         paramBytes,
		ExportFormats:  append([]byte(nil), task.ExportFormats...),
	}
	if task.ScheduleType != nil {
		value := *task.ScheduleType
		clone.ScheduleType = &value
	}
	if task.ScheduleExpr != nil {
		value := *task.ScheduleExpr
		clone.ScheduleExpr = &value
	}
	warnings = append(warnings, "新任务默认暂停；完成与原任务的结果比对后再启用周期计划")
	if err := s.repo.CreateTask(clone); err != nil {
		return nil, err
	}
	return &TrafficReportTaskMigrationResult{Task: clone, Warnings: warnings}, nil
}

func (s *trafficReportService) StartRun(ownerID, taskID uint64) (string, error) {
	accessOwnerID, err := s.accessOwnerID(ownerID)
	if err != nil {
		return "", err
	}
	task, err := s.repo.GetTask(taskID, accessOwnerID)
	if err != nil {
		return "", err
	}
	active, err := s.repo.HasActiveRun(taskID)
	if err != nil {
		return "", err
	}
	if active {
		return "", NewBadRequest("该报表已有运行中的任务")
	}
	id := newTrafficReportID()
	now := time.Now()
	run := &model.TrafficReportRun{ID: id, TaskID: task.ID, Status: model.TrafficReportStatusPending, EngineVersion: trafficReportEngineVersion, CreatedAt: now}
	if err := s.repo.CreateRun(run); err != nil {
		return "", err
	}
	go s.executeRun(context.Background(), task, run)
	return id, nil
}

func (s *trafficReportService) GetRun(ownerID uint64, runID string) (*model.TrafficReportRun, error) {
	accessOwnerID, err := s.accessOwnerID(ownerID)
	if err != nil {
		return nil, err
	}
	return s.repo.GetRun(runID, accessOwnerID)
}

func (s *trafficReportService) ListRuns(ownerID, taskID uint64, page, pageSize int) ([]model.TrafficReportRun, int64, error) {
	accessOwnerID, err := s.accessOwnerID(ownerID)
	if err != nil {
		return nil, 0, err
	}
	return s.repo.ListRuns(accessOwnerID, taskID, page, pageSize)
}

func (s *trafficReportService) DownloadArtifact(ownerID, artifactID uint64) (*model.TrafficReportArtifact, error) {
	accessOwnerID, err := s.accessOwnerID(ownerID)
	if err != nil {
		return nil, err
	}
	var artifact model.TrafficReportArtifact
	q := model.DB.Table("traffic_report_artifacts AS a").
		Joins("JOIN traffic_report_runs r ON r.id = a.run_id").
		Joins("JOIN traffic_report_tasks t ON t.id = r.task_id").
		Where("a.id = ?", artifactID)
	if accessOwnerID > 0 {
		q = q.Where("t.owner_user_id = ?", accessOwnerID)
	}
	if err := q.First(&artifact).Error; err != nil {
		return nil, err
	}
	return &artifact, nil
}

func (s *trafficReportService) accessOwnerID(userID uint64) (uint64, error) {
	if userID == 0 {
		return 0, NewBadRequest("invalid user id")
	}
	if s.roleRepo == nil {
		return userID, nil
	}
	roles, err := s.roleRepo.GetUserRoles(userID)
	if err != nil {
		return 0, err
	}
	for _, role := range roles {
		if strings.EqualFold(strings.TrimSpace(role.Name), "admin") {
			return 0, nil
		}
	}
	return userID, nil
}

func (s *trafficReportService) RunDueTasks(ctx context.Context) {
	now := time.Now()
	tasks, err := s.repo.ListDueTasks(now, 20)
	if err != nil {
		return
	}
	for _, task := range tasks {
		typ, expr := "daily", ""
		if task.ScheduleType != nil && *task.ScheduleType != "" {
			typ = *task.ScheduleType
		}
		if task.ScheduleExpr != nil {
			expr = *task.ScheduleExpr
		}
		next, e := nextTrafficReportTime(now, typ, expr, task.Timezone)
		if e != nil || !s.repo.ClaimTaskRun(task.ID, now, next) {
			continue
		}
		id := newTrafficReportID()
		run := &model.TrafficReportRun{ID: id, TaskID: task.ID, Status: model.TrafficReportStatusPending, EngineVersion: trafficReportEngineVersion, CreatedAt: now}
		if s.repo.CreateRun(run) == nil {
			go s.executeRun(ctx, &task, run)
		}
	}
}

func (s *trafficReportService) executeRun(ctx context.Context, task *model.TrafficReportTask, run *model.TrafficReportRun) {
	start := time.Now()
	run.Status = model.TrafficReportStatusRunning
	run.StartedAt = &start
	stage := "读取配置"
	run.ProgressStage = &stage
	run.ProgressPct = 5
	_ = s.repo.UpdateRun(run)
	window, err := resolveTrafficReportWindow(task)
	if err != nil {
		s.failRun(run, err)
		return
	}
	resolvedWindow, _ := json.Marshal(window)
	run.ResolvedWindow = datatypes.JSON(resolvedWindow)
	var params trafficReportParams
	if err := json.Unmarshal(task.Params, &params); err != nil {
		s.failRun(run, err)
		return
	}
	params = normalizeTrafficReportParams(params)
	resolvedParams, _ := json.Marshal(params)
	run.ResolvedParams = datatypes.JSON(resolvedParams)
	stage = "查询原始数据"
	run.ProgressStage = &stage
	run.ProgressPct = 25
	_ = s.repo.UpdateRun(run)

	var rows []map[string]interface{}
	var summary map[string]interface{}
	legacyProxy := false
	legacyNative := false
	if _, ok := legacyTaskID(task.Params); ok && config.AppConfig.TrafficReport.LegacyNativeEnabled {
		legacyNative = true
		run.EngineVersion = trafficReportLegacyNativeEngineVersion
		rowCount, nativeSummary, nativeErr := s.executeLegacyNative(ctx, task, run, window)
		if nativeErr != nil {
			s.failRun(run, nativeErr)
			return
		}
		run.RowCount = rowCount
		summary = nativeSummary
	} else if _, ok := legacyTaskID(task.Params); ok && strings.TrimSpace(config.AppConfig.TrafficReport.LegacyBaseURL) != "" {
		legacyProxy = true
		run.EngineVersion = trafficReportLegacyEngineVersion
		rowCount, legacySummary, proxyErr := s.executeLegacyProxy(ctx, task, run)
		if proxyErr != nil {
			s.failRun(run, proxyErr)
			return
		}
		run.RowCount = rowCount
		summary = legacySummary
	} else {
		rows, summary, err = s.queryReportRows(ctx, task, params, window)
		if err != nil {
			s.failRun(run, err)
			return
		}
		run.RowCount = int64(len(rows))
	}
	stage = "生成文件"
	run.ProgressStage = &stage
	run.ProgressPct = 60
	_ = s.repo.UpdateRun(run)
	if !legacyProxy && !legacyNative {
		if err := os.MkdirAll(filepath.Join(config.GetTrafficReportStorageDir(), run.ID), 0o750); err != nil {
			s.failRun(run, err)
			return
		}
		formats := []string{}
		_ = json.Unmarshal(task.ExportFormats, &formats)
		for _, format := range formats {
			artifact, err := writeTrafficReportArtifact(run.ID, task, rows, params, window, summary, format)
			if err != nil {
				s.failRun(run, err)
				return
			}
			if err := s.repo.CreateArtifact(artifact); err != nil {
				s.failRun(run, err)
				return
			}
		}
	}
	metaPath := filepath.Join(config.GetTrafficReportStorageDir(), run.ID, "report-metadata.json")
	meta := map[string]interface{}{"engine_version": run.EngineVersion, "window": window, "params": params, "summary": summary}
	metaBytes, _ := json.MarshalIndent(meta, "", "  ")
	if err := os.WriteFile(metaPath, metaBytes, 0o640); err != nil {
		s.failRun(run, err)
		return
	}
	if err := s.repo.CreateArtifact(&model.TrafficReportArtifact{RunID: run.ID, FileName: "report-metadata.json", MediaType: "application/json", StoragePath: metaPath, FileSize: int64(len(metaBytes))}); err != nil {
		s.failRun(run, err)
		return
	}

	finished := time.Now()
	run.Status = model.TrafficReportStatusSuccess
	run.ProgressPct = 100
	stage = "完成"
	run.ProgressStage = &stage
	run.FinishedAt = &finished
	run.Summary, _ = json.Marshal(summary)
	_ = s.repo.UpdateRun(run)
}

func (s *trafficReportService) failRun(run *model.TrafficReportRun, err error) {
	message := err.Error()
	now := time.Now()
	run.Status = model.TrafficReportStatusFailed
	run.ErrorMessage = &message
	run.FinishedAt = &now
	_ = s.repo.UpdateRun(run)
}

type trafficReportWindow struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
	Label string    `json:"label"`
}

func resolveTrafficReportWindow(task *model.TrafficReportTask) (trafficReportWindow, error) {
	loc, err := time.LoadLocation(task.Timezone)
	if err != nil {
		return trafficReportWindow{}, err
	}
	params := map[string]interface{}{}
	_ = json.Unmarshal(task.WindowParams, &params)
	now := time.Now().In(loc)
	var start, end time.Time
	switch task.WindowSelector {
	case "custom":
		var ok bool
		start, ok = parseReportTime(params["start_time"], loc)
		if !ok {
			return trafficReportWindow{}, errors.New("custom window requires start_time")
		}
		end, ok = parseReportTime(params["end_time"], loc)
		if !ok {
			return trafficReportWindow{}, errors.New("custom window requires end_time")
		}
	case "last_week":
		monday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc).AddDate(0, 0, -int(now.Weekday()+6)%7)
		start = monday.AddDate(0, 0, -7)
		end = monday.Add(-time.Second)
	case "last_month":
		thisMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc)
		start = thisMonth.AddDate(0, -1, 0)
		end = thisMonth.Add(-time.Second)
	case "last_n_days":
		n := 7
		if v, ok := params["n"].(float64); ok && int(v) > 0 {
			n = int(v)
		}
		end = time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, loc)
		first := end.AddDate(0, 0, -(n - 1))
		start = time.Date(first.Year(), first.Month(), first.Day(), 0, 0, 0, 0, loc)
	default:
		return trafficReportWindow{}, errors.New("unsupported window_selector")
	}
	if !start.Before(end) {
		return trafficReportWindow{}, errors.New("window start must be earlier than end")
	}
	return trafficReportWindow{Start: start, End: end, Label: start.Format("20060102") + "-" + end.Format("20060102")}, nil
}

func parseReportTime(value interface{}, loc *time.Location) (time.Time, bool) {
	s, ok := value.(string)
	if !ok || strings.TrimSpace(s) == "" {
		return time.Time{}, false
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
		if t, err := time.Parse(layout, s); err == nil {
			return t.In(loc), true
		}
	}
	if t, err := time.ParseInLocation("2006-01-02 15:04:05", s, loc); err == nil {
		return t, true
	}
	return time.Time{}, false
}

func (s *trafficReportService) queryReportRows(ctx context.Context, task *model.TrafficReportTask, params trafficReportParams, window trafficReportWindow) ([]map[string]interface{}, map[string]interface{}, error) {
	params = normalizeTrafficReportParams(params)
	if params.Direction != "recv" && params.Direction != "send" && params.Direction != "both" {
		return nil, nil, errors.New("direction must be recv, send or both")
	}
	if task.DataSourceType == "nfa" {
		rows, err := queryNFAReportRows(ctx, params, window, s.trafficScope, task.OwnerUserID)
		if err != nil {
			return nil, nil, err
		}
		out := make([]map[string]interface{}, 0, len(rows))
		points := map[time.Time]float64{}
		for _, row := range rows {
			value := reportDirectionValue(row.Recv, row.Send, params.Direction)
			points[row.CreateTime] += value
			out = append(out, map[string]interface{}{"create_time": row.CreateTime, "school_id": row.SchoolID, "school_name": row.SchoolName, "region": row.Region, "cp": row.CP, "recv_bytes": row.Recv, "send_bytes": row.Send, "total_bytes": row.Recv + row.Send, "selected_bytes": value})
		}
		return out, buildReportSummary(task.DataSourceType, params, points, window, len(out)), nil
	}
	rows, err := queryEDCReportRows(ctx, params, window, s.edcScope, task.OwnerUserID)
	if err != nil {
		return nil, nil, err
	}
	out := make([]map[string]interface{}, 0, len(rows))
	points := map[time.Time]float64{}
	for _, row := range rows {
		value := reportDirectionValue(row.ServiceSize, row.CacheSize, params.Direction)
		points[row.CreateTime] += value
		out = append(out, map[string]interface{}{"create_time": row.CreateTime, "entity_id": row.EntityID, "entity_name": row.EntityName, "alias": row.Alias, "region": row.Region, "cp": row.CP, "entity_type": row.EntityType, "src_region": row.SrcRegion, "dst_region": row.DstRegion, "service_bytes": row.ServiceSize, "cache_bytes": row.CacheSize, "total_bytes": row.ServiceSize + row.CacheSize, "selected_bytes": value})
	}
	return out, buildReportSummary(task.DataSourceType, params, points, window, len(out)), nil
}

func normalizeTrafficReportParams(params trafficReportParams) trafficReportParams {
	if params.Direction == "" {
		params.Direction = "both"
	}
	if params.UnitBase != 1000 && params.UnitBase != 1024 {
		params.UnitBase = 1024
	}
	if params.BudgetMul == 0 {
		params.BudgetMul = 8
	}
	if params.BudgetDiv == 0 {
		params.BudgetDiv = 300
	}
	return params
}

func reportDirectionValue(recv, send int64, direction string) float64 {
	switch direction {
	case "recv":
		return float64(recv)
	case "send":
		return float64(send)
	default:
		return float64(recv + send)
	}
}

func buildReportSummary(source string, params trafficReportParams, points map[time.Time]float64, window trafficReportWindow, rowCount int) map[string]interface{} {
	interval := 60
	if source == "edc" {
		interval = 300
	}
	values := make([]float64, 0, len(points))
	budgetValues := make([]float64, 0, len(points))
	rawValues := make([]float64, 0, len(points))
	days := map[string][]float64{}
	budgetDays := map[string][]float64{}
	rawDays := map[string][]float64{}
	for t, raw := range points {
		mbps := raw * 8 / float64(interval) / float64(params.UnitBase) / float64(params.UnitBase)
		budget := mbps
		if params.BudgetEnabled {
			budget = raw * params.BudgetMul / params.BudgetDiv / float64(params.UnitBase) / float64(params.UnitBase)
		}
		values = append(values, mbps)
		budgetValues = append(budgetValues, budget)
		rawValues = append(rawValues, raw)
		key := t.In(window.Start.Location()).Format("2006-01-02")
		days[key] = append(days[key], mbps)
		budgetDays[key] = append(budgetDays[key], budget)
		rawDays[key] = append(rawDays[key], raw)
	}
	percentile := func(v []float64) float64 {
		if len(v) == 0 {
			return 0
		}
		sort.Slice(v, func(i, j int) bool { return v[i] > v[j] })
		idx := settlement95.DescendingIndex(len(v))
		return v[idx]
	}
	range95 := percentile(values)
	daily := make([]float64, 0, len(days))
	for _, v := range days {
		daily = append(daily, percentile(v))
	}
	dailyAvg := 0.0
	for _, v := range daily {
		dailyAvg += v
	}
	if len(daily) > 0 {
		dailyAvg /= float64(len(daily))
	}
	budgetDaily := make([]float64, 0, len(budgetDays))
	for _, v := range budgetDays {
		budgetDaily = append(budgetDaily, percentile(v))
	}
	budgetDailyAvg := 0.0
	for _, v := range budgetDaily {
		budgetDailyAvg += v
	}
	if len(budgetDaily) > 0 {
		budgetDailyAvg /= float64(len(budgetDaily))
	}
	budgetRange := percentile(budgetValues)
	rawRange := percentile(rawValues)
	rawDaily := make([]float64, 0, len(rawDays))
	for _, v := range rawDays {
		rawDaily = append(rawDaily, percentile(v))
	}
	rawDailyAvg := 0.0
	for _, v := range rawDaily {
		rawDailyAvg += v
	}
	if totalDays := len(rawDaily); totalDays > 0 {
		rawDailyAvg /= float64(totalDays)
	}
	formula := fmt.Sprintf("raw*8/%d/%d/%d", interval, params.UnitBase, params.UnitBase)
	if params.BudgetEnabled {
		formula = fmt.Sprintf("raw*%g/%g/%d/%d", params.BudgetMul, params.BudgetDiv, params.UnitBase, params.UnitBase)
	}
	budgetStep := func(raw float64) float64 { return raw * params.BudgetMul / params.BudgetDiv }
	return map[string]interface{}{"data_source_type": source, "raw_value_unit": "bytes_per_interval", "raw_interval_seconds": interval, "unit_base": params.UnitBase, "direction": params.Direction, "formula": formula, "window_start": window.Start, "window_end": window.End, "row_count": rowCount, "range_95_mbps": range95, "daily_95_avg_mbps": dailyAvg, "budget_enabled": params.BudgetEnabled, "budget_mul": params.BudgetMul, "budget_div": params.BudgetDiv, "budget_formula": fmt.Sprintf("raw*%g/%g/base/base", params.BudgetMul, params.BudgetDiv), "budget_range_value": budgetRange, "budget_daily_avg_value": budgetDailyAvg, "raw_range_95": rawRange, "raw_daily_95_avg": rawDailyAvg, "range_95_1000": budgetStep(rawRange) / 1000 / 1000, "daily_95_avg_1000": budgetStep(rawDailyAvg) / 1000 / 1000, "range_95_1024": budgetStep(rawRange) / 1024 / 1024, "daily_95_avg_1024": budgetStep(rawDailyAvg) / 1024 / 1024, "daily_days": len(days), "range_points": len(points), "settlement_mode": params.SettlementMode}
}

func queryNFAReportRows(ctx context.Context, params trafficReportParams, window trafficReportWindow, scopeService TrafficScopeService, ownerID uint64) ([]trafficReportNFARow, error) {
	var rows []trafficReportNFARow
	q := model.DB.WithContext(ctx).Table("nfa_school_traffic").Select("create_time, school_id, school_name, region, cp, total_recv AS recv_bytes, total_send AS send_bytes").Where("create_time >= ? AND create_time < ?", window.Start, window.End.Add(time.Second))
	if params.SchoolName != "" {
		q = q.Where("school_name = ?", params.SchoolName)
	}
	if len(params.SchoolNames) > 0 {
		q = q.Where("school_name IN ?", params.SchoolNames)
	}
	if params.Region != "" {
		q = q.Where("region = ?", params.Region)
	}
	if cps := splitReportCSV(params.CP); len(cps) > 0 {
		q = q.Where("cp IN ?", cps)
	}
	if scopeService != nil {
		scope, err := scopeService.ResolveEffectiveScope(ownerID)
		if err != nil {
			return nil, err
		}
		if scope.Source == model.TrafficScopeSourceNone {
			return rows, nil
		}
		if scope.Source != model.TrafficScopeSourceDefaultAdminRole && len(scope.AllowedSchoolKeys) == 0 {
			return rows, nil
		}
		if len(scope.AllowedSchoolKeys) > 0 {
			parts := make([]string, 0, len(scope.AllowedSchoolKeys))
			args := make([]interface{}, 0, len(scope.AllowedSchoolKeys)*3)
			for _, k := range scope.AllowedSchoolKeys {
				parts = append(parts, "(school_id = ? AND region = ? AND cp = ?)")
				args = append(args, k.SchoolID, k.Region, k.CP)
			}
			q = q.Where(strings.Join(parts, " OR "), args...)
		}
	}
	return rows, q.Order("create_time ASC, school_name ASC, region ASC, cp ASC").Scan(&rows).Error
}

func queryEDCReportRows(ctx context.Context, params trafficReportParams, window trafficReportWindow, scopeService EDCTrafficScopeService, ownerID uint64) ([]trafficReportEDCRow, error) {
	var rows []trafficReportEDCRow
	q := model.DB.WithContext(ctx).Table("edc_traffic_5m AS t").Select("t.bucket_5m AS create_time, t.entity_id, e.edc_name AS entity_name, COALESCE(t.alias, e.alias, '') AS alias, t.region, t.cp, t.entity_type, t.src_region, t.dst_region, GREATEST(t.service_size, 0) AS service_bytes, GREATEST(t.cache_size, 0) AS cache_bytes").Joins("JOIN edc_entities e ON e.id = t.entity_id AND e.enabled = ? AND e.is_backup = ?", true, false).Where("t.bucket_5m >= ? AND t.bucket_5m < ?", window.Start, window.End.Add(time.Second))
	if params.EntityType != "" {
		q = q.Where("t.entity_type = ?", params.EntityType)
	}
	if params.SrcRegion != "" {
		q = q.Where("t.src_region = ?", params.SrcRegion)
	}
	if params.DstRegion != "" {
		q = q.Where("t.dst_region = ?", params.DstRegion)
	}
	if params.Region != "" {
		q = q.Where("t.region = ?", params.Region)
	}
	if cps := splitReportCSV(params.CP); len(cps) > 0 {
		q = q.Where("t.cp IN ?", cps)
	}
	ids := params.EntityIDs
	if scopeService != nil {
		scope, err := scopeService.ResolveEffectiveScope(ownerID)
		if err != nil {
			return nil, err
		}
		if scope.Source == model.EDCTrafficScopeSourceNone {
			return rows, nil
		}
		if scope.Source != model.EDCTrafficScopeSourceDefaultAdminRole && len(scope.AllowedEntityIDs) == 0 {
			return rows, nil
		}
		if len(ids) == 0 {
			ids = scope.AllowedEntityIDs
		} else {
			allowed := map[uint64]struct{}{}
			for _, id := range scope.AllowedEntityIDs {
				allowed[id] = struct{}{}
			}
			filtered := ids[:0]
			for _, id := range ids {
				if _, ok := allowed[id]; ok {
					filtered = append(filtered, id)
				}
			}
			ids = filtered
		}
	}
	if len(ids) > 0 {
		q = q.Where("t.entity_id IN ?", ids)
	}
	return rows, q.Order("t.bucket_5m ASC, t.entity_id ASC").Scan(&rows).Error
}

func splitReportCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func writeTrafficReportArtifact(runID string, task *model.TrafficReportTask, rows []map[string]interface{}, params trafficReportParams, window trafficReportWindow, summary map[string]interface{}, format string) (*model.TrafficReportArtifact, error) {
	base := sanitizeReportName(task.Name) + "-" + window.Label
	root := filepath.Join(config.GetTrafficReportStorageDir(), runID)
	if format == "xlsx" {
		path := filepath.Join(root, base+".xlsx")
		if err := writeReportXLSX(path, task.DataSourceType, rows, summary); err != nil {
			return nil, err
		}
		st, err := os.Stat(path)
		if err != nil {
			return nil, err
		}
		return &model.TrafficReportArtifact{RunID: runID, FileName: filepath.Base(path), MediaType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", StoragePath: path, FileSize: st.Size()}, nil
	}
	path := filepath.Join(root, base+".csv")
	f, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	keys := reportKeysForRows(task.DataSourceType, rows)
	if len(keys) > 0 {
		if err := w.Write(keys); err != nil {
			return nil, err
		}
		for _, row := range rows {
			vals := make([]string, len(keys))
			for i, k := range keys {
				vals[i] = formatReportValue(row[k])
			}
			if err := w.Write(vals); err != nil {
				return nil, err
			}
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, err
	}
	st, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	return &model.TrafficReportArtifact{RunID: runID, FileName: filepath.Base(path), MediaType: "text/csv; charset=utf-8", StoragePath: path, FileSize: st.Size()}, nil
}

func orderedReportKeys(row map[string]interface{}) []string {
	keys := make([]string, 0, len(row))
	for _, k := range []string{"create_time", "school_id", "school_name", "region", "cp", "entity_id", "entity_name", "alias", "entity_type", "src_region", "dst_region", "recv_bytes", "send_bytes", "service_bytes", "cache_bytes", "total_bytes", "selected_bytes"} {
		if _, ok := row[k]; ok {
			keys = append(keys, k)
		}
	}
	return keys
}

func reportKeysForRows(source string, rows []map[string]interface{}) []string {
	if len(rows) > 0 {
		return orderedReportKeys(rows[0])
	}
	if source == "edc" {
		return []string{"create_time", "entity_id", "entity_name", "alias", "region", "cp", "entity_type", "src_region", "dst_region", "service_bytes", "cache_bytes", "total_bytes", "selected_bytes"}
	}
	return []string{"create_time", "school_id", "school_name", "region", "cp", "recv_bytes", "send_bytes", "total_bytes", "selected_bytes"}
}
func formatReportValue(v interface{}) string {
	if v == nil {
		return ""
	}
	if t, ok := v.(time.Time); ok {
		return t.Format(time.RFC3339)
	}
	return fmt.Sprint(v)
}
func writeReportXLSX(path, source string, rows []map[string]interface{}, summary map[string]interface{}) error {
	f := excelize.NewFile()
	defer f.Close()
	sheet := "流量数据"
	f.SetSheetName("Sheet1", sheet)
	keys := reportKeysForRows(source, rows)
	for i, k := range keys {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		_ = f.SetCellValue(sheet, cell, k)
	}
	for r, row := range rows {
		for c, k := range keys {
			cell, _ := excelize.CoordinatesToCellName(c+1, r+2)
			_ = f.SetCellValue(sheet, cell, formatReportValue(row[k]))
		}
	}
	info := "导出说明"
	f.NewSheet(info)
	r := 1
	for _, item := range []struct {
		k string
		v interface{}
	}{{"计算引擎", trafficReportEngineVersion}, {"窗口开始", summary["window_start"]}, {"窗口结束", summary["window_end"]}, {"原始单位", summary["raw_value_unit"]}, {"采样间隔秒", summary["raw_interval_seconds"]}, {"换算公式", summary["formula"]}, {"区间95 Mbps", summary["range_95_mbps"]}, {"日95平均 Mbps", summary["daily_95_avg_mbps"]}, {"预算区间95", summary["budget_range_value"]}, {"预算日95平均", summary["budget_daily_avg_value"]}} {
		_ = f.SetCellValue(info, fmt.Sprintf("A%d", r), item.k)
		_ = f.SetCellValue(info, fmt.Sprintf("B%d", r), formatReportValue(item.v))
		r++
	}
	return f.SaveAs(path)
}

func sanitizeReportName(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "traffic-report"
	}
	var b strings.Builder
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == ' ' {
			b.WriteRune(r)
		} else {
			b.WriteRune('_')
		}
	}
	return strings.TrimSpace(b.String())
}

func newTrafficReportID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	h := hex.EncodeToString(b)
	return h[0:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:32]
}

func nextTrafficReportTime(now time.Time, scheduleType, expr, timezone string) (time.Time, error) {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return time.Time{}, err
	}
	n := now.In(loc)
	hour, minute := 2, 0
	if strings.TrimSpace(expr) != "" {
		parts := strings.Fields(expr)
		value := parts[len(parts)-1]
		hm := strings.Split(value, ":")
		if len(hm) != 2 {
			return time.Time{}, errors.New("schedule_expr must end with HH:MM")
		}
		hour, _ = strconv.Atoi(hm[0])
		minute, _ = strconv.Atoi(hm[1])
		if hour < 0 || hour > 23 || minute < 0 || minute > 59 {
			return time.Time{}, errors.New("schedule time is invalid")
		}
	}
	var next time.Time
	switch scheduleType {
	case "daily":
		next = time.Date(n.Year(), n.Month(), n.Day(), hour, minute, 0, 0, loc)
		if !next.After(n) {
			next = next.AddDate(0, 0, 1)
		}
	case "weekly":
		weekday := int(n.Weekday())
		target := 1
		if parts := strings.Fields(expr); len(parts) > 0 {
			target, _ = strconv.Atoi(parts[0])
			if target < 1 || target > 7 {
				return time.Time{}, errors.New("weekly schedule_expr must start with weekday 1-7")
			}
		}
		delta := (target - weekday + 7) % 7
		next = time.Date(n.Year(), n.Month(), n.Day(), hour, minute, 0, 0, loc).AddDate(0, 0, delta)
		if !next.After(n) {
			next = next.AddDate(0, 0, 7)
		}
	case "monthly":
		day := 1
		if parts := strings.Fields(expr); len(parts) > 0 {
			day, _ = strconv.Atoi(parts[0])
			if day < 1 || day > 28 {
				return time.Time{}, errors.New("monthly schedule_expr day must be 1-28")
			}
		}
		next = time.Date(n.Year(), n.Month(), day, hour, minute, 0, 0, loc)
		if !next.After(n) {
			next = next.AddDate(0, 1, 0)
		}
	default:
		return time.Time{}, errors.New("schedule_type must be daily, weekly or monthly")
	}
	return next, nil
}
