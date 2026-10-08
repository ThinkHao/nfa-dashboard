package service

import (
	"archive/zip"
	"context"
	"crypto/rand"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/xuri/excelize/v2"
	"nfa-dashboard/config"
	"nfa-dashboard/internal/model"
	"nfa-dashboard/internal/repository"
	"nfa-dashboard/internal/settlement95"

	"gorm.io/datatypes"
	"gorm.io/gorm"
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

type TrafficReportDownloadMonth struct {
	Month         string                      `json:"month"`
	RunCount      int                         `json:"run_count"`
	ArtifactCount int                         `json:"artifact_count"`
	TotalSize     int64                       `json:"total_size"`
	Files         []TrafficReportDownloadFile `json:"files"`
}

type TrafficReportDownloadFile struct {
	ID        uint64    `json:"id"`
	RunID     string    `json:"run_id"`
	FileName  string    `json:"file_name"`
	MediaType string    `json:"media_type"`
	FileSize  int64     `json:"file_size"`
	RunAt     time.Time `json:"run_at"`
}

type TrafficReportMonthlyArchive struct {
	Path          string
	FileName      string
	ArtifactCount int
}

type trafficReportParams struct {
	SchoolName         string   `json:"school_name"`
	SchoolNames        []string `json:"school_names"`
	School             string   `json:"school"`
	ExcludeSchool      string   `json:"exclude_school"`
	DataSourceInstance string   `json:"data_source_instance"`
	Province           string   `json:"province"`
	ExportRaw          bool     `json:"export_raw"`
	ExportDaily        bool     `json:"export_daily"`
	MonthlyAggregate   bool     `json:"monthly_aggregate"`
	SettlementMode     string   `json:"settlement_mode"`
	AggregateAll       bool     `json:"aggregate_all"`
	CombineV4V6        bool     `json:"combine_v4_v6"`
	MergeKey           string   `json:"merge_key"`
	BatchSize          int      `json:"batch_size"`
	SortBy             string   `json:"sortby"`
	SortOrder          string   `json:"sort_order"`
	EDCName            string   `json:"edc_name"`
	EDCMatchMode       string   `json:"edc_match_mode"`
	Region             string   `json:"region"`
	CP                 string   `json:"cp"`
	EntityIDs          []uint64 `json:"entity_ids"`
	EntityType         string   `json:"entity_type"`
	SrcRegion          string   `json:"src_region"`
	DstRegion          string   `json:"dst_region"`
	Direction          string   `json:"direction"`
	UnitBase           int      `json:"unit_base"`
	BudgetEnabled      bool     `json:"data_budget_enabled"`
	BudgetMul          float64  `json:"data_budget_mul"`
	BudgetDiv          float64  `json:"data_budget_div"`
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

type trafficReportNFASample struct {
	Recv int64
	Send int64
}

type trafficReportNFASeries struct {
	SchoolID   string
	SchoolName string
	Region     string
	CP         string
	Samples    map[time.Time]trafficReportNFASample
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
	ListDownloadMonths(ownerID uint64) ([]TrafficReportDownloadMonth, error)
	CreateMonthlyArchive(ownerID uint64, month string, artifactIDs []uint64) (*TrafficReportMonthlyArchive, error)
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
	input.ExportFormats = []string{"xlsx"}
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

func (s *trafficReportService) ListDownloadMonths(ownerID uint64) ([]TrafficReportDownloadMonth, error) {
	accessOwnerID, err := s.accessOwnerID(ownerID)
	if err != nil {
		return nil, err
	}
	runs, err := s.repo.ListSuccessfulRuns(accessOwnerID)
	if err != nil {
		return nil, err
	}
	runs = latestTrafficReportRuns(runs)
	groups := make(map[string]*TrafficReportDownloadMonth)
	for _, run := range runs {
		month := trafficReportRunMonth(run)
		if month == "" {
			continue
		}
		artifacts := filterTrafficReportXLSXArtifacts(run.Artifacts)
		if len(artifacts) == 0 {
			continue
		}
		group := groups[month]
		if group == nil {
			group = &TrafficReportDownloadMonth{Month: month, Files: make([]TrafficReportDownloadFile, 0)}
			groups[month] = group
		}
		runAt := run.CreatedAt
		if run.FinishedAt != nil {
			runAt = *run.FinishedAt
		}
		group.RunCount++
		group.ArtifactCount += len(artifacts)
		for _, artifact := range artifacts {
			group.TotalSize += artifact.FileSize
			group.Files = append(group.Files, TrafficReportDownloadFile{ID: artifact.ID, RunID: artifact.RunID, FileName: artifact.FileName, MediaType: artifact.MediaType, FileSize: artifact.FileSize, RunAt: runAt})
		}
	}
	months := make([]TrafficReportDownloadMonth, 0, len(groups))
	for _, group := range groups {
		months = append(months, *group)
	}
	sort.Slice(months, func(i, j int) bool { return months[i].Month > months[j].Month })
	return months, nil
}

func (s *trafficReportService) CreateMonthlyArchive(ownerID uint64, month string, artifactIDs []uint64) (*TrafficReportMonthlyArchive, error) {
	accessOwnerID, err := s.accessOwnerID(ownerID)
	if err != nil {
		return nil, err
	}
	if !trafficReportMonthPattern.MatchString(month) {
		return nil, NewBadRequest("month must be formatted as YYYY-MM")
	}
	runs, err := s.repo.ListSuccessfulRuns(accessOwnerID)
	if err != nil {
		return nil, err
	}
	runs = latestTrafficReportRuns(runs)
	filteredRuns := make([]model.TrafficReportRun, 0, len(runs))
	for _, run := range runs {
		if trafficReportRunMonth(run) == month {
			filteredRuns = append(filteredRuns, run)
		}
	}
	runs = filteredRuns
	for runIndex := range runs {
		runs[runIndex].Artifacts = filterTrafficReportXLSXArtifacts(runs[runIndex].Artifacts)
	}
	selectedIDs := make(map[uint64]struct{}, len(artifactIDs))
	for _, artifactID := range artifactIDs {
		if artifactID == 0 {
			return nil, NewBadRequest("artifact_ids must contain positive ids")
		}
		selectedIDs[artifactID] = struct{}{}
	}
	if len(selectedIDs) > 0 {
		matchedIDs := make(map[uint64]struct{}, len(selectedIDs))
		for runIndex := range runs {
			selectedArtifacts := make([]model.TrafficReportArtifact, 0, len(runs[runIndex].Artifacts))
			for _, artifact := range runs[runIndex].Artifacts {
				if _, ok := selectedIDs[artifact.ID]; !ok {
					continue
				}
				selectedArtifacts = append(selectedArtifacts, artifact)
				matchedIDs[artifact.ID] = struct{}{}
			}
			runs[runIndex].Artifacts = selectedArtifacts
		}
		if len(matchedIDs) != len(selectedIDs) {
			return nil, NewBadRequest("所选文件不属于该月份或无权限")
		}
	}
	artifactCount := 0
	for _, run := range runs {
		artifactCount += len(run.Artifacts)
	}
	if artifactCount == 0 {
		return nil, NewBadRequest("该月份没有可下载的成功报表")
	}

	root, err := filepath.Abs(config.GetTrafficReportStorageDir())
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(root, 0o750); err != nil {
		return nil, err
	}
	temp, err := os.CreateTemp(root, ".traffic-reports-"+month+"-*.zip")
	if err != nil {
		return nil, err
	}
	tempPath := temp.Name()
	cleanup := true
	defer func() {
		_ = temp.Close()
		if cleanup {
			_ = os.Remove(tempPath)
		}
	}()

	archive := zip.NewWriter(temp)
	usedNames := map[string]int{}
	for _, run := range runs {
		for _, artifact := range run.Artifacts {
			path, err := safeTrafficReportArtifactPath(root, artifact.StoragePath)
			if err != nil {
				return nil, err
			}
			file, err := os.Open(path)
			if err != nil {
				return nil, err
			}
			name := safeTrafficReportArchiveName(artifact.FileName)
			name = uniqueTrafficReportArchiveName(name, usedNames)
			header := &zip.FileHeader{Name: name, Method: zip.Deflate}
			header.SetMode(0o640)
			writer, err := archive.CreateHeader(header)
			if err == nil {
				_, err = io.Copy(writer, file)
			}
			closeErr := file.Close()
			if err != nil {
				return nil, err
			}
			if closeErr != nil {
				return nil, closeErr
			}
		}
	}
	if err := archive.Close(); err != nil {
		return nil, err
	}
	if err := temp.Close(); err != nil {
		return nil, err
	}
	cleanup = false
	fileName := "traffic-reports-" + month + ".zip"
	if len(selectedIDs) > 0 {
		fileName = "traffic-reports-" + month + "-selected.zip"
	}
	return &TrafficReportMonthlyArchive{Path: tempPath, FileName: fileName, ArtifactCount: artifactCount}, nil
}

var trafficReportMonthPattern = regexp.MustCompile(`^\d{4}-(0[1-9]|1[0-2])$`)

func filterTrafficReportXLSXArtifacts(artifacts []model.TrafficReportArtifact) []model.TrafficReportArtifact {
	filtered := make([]model.TrafficReportArtifact, 0, len(artifacts))
	for _, artifact := range artifacts {
		if strings.EqualFold(filepath.Ext(artifact.FileName), ".xlsx") {
			filtered = append(filtered, artifact)
		}
	}
	return filtered
}

func trafficReportRunMonth(run model.TrafficReportRun) string {
	if window, ok := parseTrafficReportResolvedWindow(run); ok {
		return reportMonth(window.Start)
	}
	t := trafficReportRunAt(run)
	if t.IsZero() {
		return ""
	}
	return reportMonth(t)
}

type trafficReportWindowBounds struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

func parseTrafficReportResolvedWindow(run model.TrafficReportRun) (trafficReportWindowBounds, bool) {
	var window trafficReportWindowBounds
	if len(run.ResolvedWindow) == 0 || json.Unmarshal(run.ResolvedWindow, &window) != nil || window.Start.IsZero() || window.End.IsZero() {
		return trafficReportWindowBounds{}, false
	}
	return window, true
}

func reportMonth(t time.Time) string {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return t.Format("2006-01")
	}
	return t.In(loc).Format("2006-01")
}

func latestTrafficReportRuns(runs []model.TrafficReportRun) []model.TrafficReportRun {
	latestByWindow := make(map[string]int)
	for index, run := range runs {
		if len(filterTrafficReportXLSXArtifacts(run.Artifacts)) == 0 {
			continue
		}
		key := trafficReportRunWindowKey(run)
		if currentIndex, ok := latestByWindow[key]; !ok || trafficReportRunIsNewer(run, runs[currentIndex]) {
			latestByWindow[key] = index
		}
	}

	latest := make([]model.TrafficReportRun, 0, len(latestByWindow))
	for index, run := range runs {
		latestIndex, ok := latestByWindow[trafficReportRunWindowKey(run)]
		if !ok || latestIndex != index {
			continue
		}
		run.Artifacts = filterTrafficReportXLSXArtifacts(run.Artifacts)
		latest = append(latest, run)
	}
	return latest
}

func trafficReportRunWindowKey(run model.TrafficReportRun) string {
	window, ok := parseTrafficReportResolvedWindow(run)
	if !ok {
		// Runs without a resolved window cannot be safely matched to another report.
		return fmt.Sprintf("run:%s", run.ID)
	}
	taskID := strconv.FormatUint(run.TaskID, 10)
	if run.TaskID == 0 {
		// Keep incomplete legacy/test records distinct when they have no task id.
		taskID = "run:" + run.ID
	}
	return fmt.Sprintf("task:%s|window:%s|%s", taskID, window.Start.UTC().Format(time.RFC3339Nano), window.End.UTC().Format(time.RFC3339Nano))
}

func trafficReportRunAt(run model.TrafficReportRun) time.Time {
	if run.FinishedAt != nil {
		return *run.FinishedAt
	}
	return run.CreatedAt
}

func trafficReportRunIsNewer(candidate, current model.TrafficReportRun) bool {
	candidateAt, currentAt := trafficReportRunAt(candidate), trafficReportRunAt(current)
	if !candidateAt.Equal(currentAt) {
		return candidateAt.After(currentAt)
	}
	if !candidate.CreatedAt.Equal(current.CreatedAt) {
		return candidate.CreatedAt.After(current.CreatedAt)
	}
	return candidate.ID > current.ID
}

func safeTrafficReportArtifactPath(root, candidate string) (string, error) {
	path, err := filepath.Abs(candidate)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", errors.New("invalid traffic report artifact path")
	}
	return path, nil
}

func safeTrafficReportArchiveName(value string) string {
	value = filepath.Base(strings.ReplaceAll(value, "\\", "/"))
	if value == "" || value == "." || value == ".." {
		return "artifact"
	}
	var b strings.Builder
	for _, r := range value {
		if r < 0x20 || r == '/' || r == '\\' {
			b.WriteRune('_')
			continue
		}
		b.WriteRune(r)
	}
	if b.Len() == 0 {
		return "artifact"
	}
	return b.String()
}

func uniqueTrafficReportArchiveName(name string, used map[string]int) string {
	if _, exists := used[name]; !exists {
		used[name] = 1
		return name
	}
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)
	for suffix := 2; ; suffix++ {
		candidate := fmt.Sprintf("%s-%d%s", base, suffix, ext)
		if _, exists := used[candidate]; exists {
			continue
		}
		used[candidate] = 1
		return candidate
	}
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
		formats := []string{"xlsx"}
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
	return trafficReportWindow{Start: start, End: end, Label: trafficReportWindowLabel(start, end)}, nil
}

func trafficReportWindowLabel(start, end time.Time) string {
	if start.Day() == 1 && start.Hour() == 0 && start.Minute() == 0 && start.Second() == 0 &&
		end.Year() == start.Year() && end.Month() == start.Month() &&
		end.Day() == start.AddDate(0, 1, 0).Add(-time.Second).Day() &&
		end.Hour() == 23 && end.Minute() == 59 && end.Second() == 59 {
		return start.Format("200601")
	}
	return start.Format("20060102") + "-" + end.Format("20060102")
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
		points := map[time.Time]float64{}
		for _, row := range rows {
			value := reportDirectionValue(row.Recv, row.Send, params.Direction)
			points[row.CreateTime] += value
		}
		out := buildNFAReportRows(rows, window, params)
		return out, buildReportSummary(task.DataSourceType, params, points, window, len(out)), nil
	}
	rows, err := queryEDCReportRows(ctx, params, window, s.edcScope, task.OwnerUserID)
	if err != nil {
		return nil, nil, err
	}
	params.Direction = "both"
	points := map[time.Time]float64{}
	for _, row := range rows {
		points[row.CreateTime.In(window.Start.Location())] += float64(row.ServiceSize)
	}
	rankIndex := config.AppConfig.TrafficReport.EDC.DailyRankIndex
	if rankIndex <= 0 {
		rankIndex = 14
	}
	out := buildEDCReportRows(rows, window, params, rankIndex)
	return out, buildReportSummary(task.DataSourceType, params, points, window, len(rows)), nil
}

func buildEDCReportRows(rows []trafficReportEDCRow, window trafficReportWindow, params trafficReportParams, rankIndex int) []map[string]interface{} {
	if params.ExportRaw {
		out := make([]map[string]interface{}, 0, len(rows))
		instance := strings.TrimSpace(params.DataSourceInstance)
		if instance == "" {
			instance = "ali"
		}
		for _, row := range rows {
			selected := reportDirectionValue(row.ServiceSize, row.CacheSize, "both")
			out = append(out, map[string]interface{}{
				"create_time": row.CreateTime, "entity_id": row.EntityID, "edc_name": row.EntityName,
				"alias": row.Alias, "region": row.Region, "cp": row.CP, "entity_type": row.EntityType,
				"src_region": row.SrcRegion, "dst_region": row.DstRegion, "service_size": row.ServiceSize,
				"cache_size": row.CacheSize, "total_bytes": row.ServiceSize + row.CacheSize, "selected_bytes": selected,
				"data_source_type": "edc", "data_source_instance": instance, "requested_edc_name": params.EDCName,
				"unit_base": params.UnitBase, "saler_group": "", "saler": "",
				"service_size_mbps_1000": float64(row.ServiceSize) * 8 / 300 / 1000 / 1000,
				"service_size_mbps_1024": float64(row.ServiceSize) * 8 / 300 / 1024 / 1024,
			})
		}
		return sortTrafficReportRows(out, params.SortBy, params.SortOrder)
	}
	daily := buildEDCDailyReportRows(rows, window, params, rankIndex)
	var out []map[string]interface{}
	switch {
	case params.MonthlyAggregate:
		out = buildEDCMonthlyReportRows(rows, daily, window, params, rankIndex)
	case params.ExportDaily:
		out = daily
	default:
		out = buildEDCSummaryReportRows(rows, daily, params, rankIndex, window)
	}
	return sortTrafficReportRows(out, params.SortBy, params.SortOrder)
}

func buildEDCDailyReportRows(rows []trafficReportEDCRow, window trafficReportWindow, params trafficReportParams, rankIndex int) []map[string]interface{} {
	loc := window.Start.Location()
	pointsByDay := make(map[string]map[time.Time]float64)
	for _, row := range rows {
		at := row.CreateTime.In(loc)
		day := at.Format("2006-01-02")
		points := pointsByDay[day]
		if points == nil {
			points = make(map[time.Time]float64)
			pointsByDay[day] = points
		}
		points[at] += float64(row.ServiceSize)
	}
	instance := strings.TrimSpace(params.DataSourceInstance)
	if instance == "" {
		instance = "ali"
	}
	rowsOut := make([]map[string]interface{}, 0, legacyTotalDays(window))
	name := strings.TrimSpace(params.EDCName)
	for day := time.Date(window.Start.In(loc).Year(), window.Start.In(loc).Month(), window.Start.In(loc).Day(), 0, 0, 0, 0, loc); !day.After(window.End.In(loc)); day = day.AddDate(0, 0, 1) {
		date := day.Format("2006-01-02")
		dailyPoints := pointsByDay[date]
		values := make([]float64, 0, len(dailyPoints))
		for _, raw := range dailyPoints {
			values = append(values, raw)
		}
		raw95 := legacyNth95(values, rankIndex)
		rowsOut = append(rowsOut, map[string]interface{}{
			"date":                       date,
			"edc_name":                   name,
			"data_source_instance":       instance,
			"daily_95th_percentile_raw":  raw95,
			"daily_95th_percentile_mbps": raw95 * 8 / 300 / float64(params.UnitBase) / float64(params.UnitBase),
			"data_points_daily":          len(values),
			"saler_group":                "",
			"saler":                      "",
		})
	}
	return rowsOut
}

func buildNFAReportRows(rows []trafficReportNFARow, window trafficReportWindow, params trafficReportParams) []map[string]interface{} {
	series := buildNFAReportSeries(rows, params)
	var out []map[string]interface{}
	switch {
	case params.ExportRaw:
		out = buildNFARawReportRows(series, params)
	case params.MonthlyAggregate:
		out = buildNFAMonthlyReportRows(series, window, params)
	case params.ExportDaily:
		out = buildNFADailyReportRows(series, window, params)
	default:
		out = buildNFASummaryReportRows(series, window, params)
	}
	return sortTrafficReportRows(out, params.SortBy, params.SortOrder)
}

func buildNFAReportSeries(rows []trafficReportNFARow, params trafficReportParams) []trafficReportNFASeries {
	seriesByKey := make(map[string]*trafficReportNFASeries)
	for _, row := range rows {
		key := ""
		schoolName := strings.TrimSpace(row.SchoolName)
		if params.AggregateAll {
			key = "all"
		} else if params.CombineV4V6 {
			switch params.MergeKey {
			case "school_id":
				key = row.SchoolID
			case "school_name", "ipgroup_name":
				key = schoolName
			case "school_name_plus_cp":
				key = schoolName + "\x00" + row.CP
			default:
				schoolName = trimNFAFamilySuffix(schoolName)
				key = schoolName
			}
		} else {
			key = strings.Join([]string{row.SchoolID, schoolName, row.Region, row.CP}, "\x00")
		}
		group := seriesByKey[key]
		if group == nil {
			name := schoolName
			id, region, cp := row.SchoolID, row.Region, row.CP
			if params.AggregateAll {
				name, id, region, cp = "全部院校汇总", "", "", ""
			} else if params.CombineV4V6 {
				switch params.MergeKey {
				case "school_id":
					name = schoolName
				case "school_name_plus_cp":
					name = strings.Trim(schoolName+"_"+row.CP, "_")
				default:
					name = schoolName
				}
				if params.MergeKey == "ipgroup_name_base" || params.MergeKey == "" {
					name = trimNFAFamilySuffix(name)
				}
			}
			group = &trafficReportNFASeries{SchoolID: id, SchoolName: name, Region: region, CP: cp, Samples: make(map[time.Time]trafficReportNFASample)}
			seriesByKey[key] = group
		}
		sample := group.Samples[row.CreateTime]
		sample.Recv += row.Recv
		sample.Send += row.Send
		group.Samples[row.CreateTime] = sample
	}
	out := make([]trafficReportNFASeries, 0, len(seriesByKey))
	for _, group := range seriesByKey {
		out = append(out, *group)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].SchoolName != out[j].SchoolName {
			return out[i].SchoolName < out[j].SchoolName
		}
		if out[i].Region != out[j].Region {
			return out[i].Region < out[j].Region
		}
		return out[i].CP < out[j].CP
	})
	return out
}

func trimNFAFamilySuffix(name string) string {
	lower := strings.ToLower(name)
	if strings.HasSuffix(lower, "_v4") || strings.HasSuffix(lower, "_v6") {
		return name[:len(name)-3]
	}
	return name
}

func buildNFARawReportRows(series []trafficReportNFASeries, params trafficReportParams) []map[string]interface{} {
	out := make([]map[string]interface{}, 0)
	instance := strings.TrimSpace(params.DataSourceInstance)
	if instance == "" {
		instance = "default"
	}
	for _, group := range series {
		times := make([]time.Time, 0, len(group.Samples))
		for at := range group.Samples {
			times = append(times, at)
		}
		sort.Slice(times, func(i, j int) bool { return times[i].Before(times[j]) })
		for _, at := range times {
			sample := group.Samples[at]
			selected := reportDirectionValue(sample.Recv, sample.Send, params.Direction)
			out = append(out, map[string]interface{}{
				"create_time": at, "school_id": group.SchoolID, "school_name": group.SchoolName,
				"region": group.Region, "cp": group.CP, "recv_bytes": sample.Recv, "send_bytes": sample.Send,
				"total_bytes": sample.Recv + sample.Send, "selected_bytes": selected,
				"data_source_type": "nfa", "data_source_instance": instance, "unit_base": params.UnitBase, "direction": params.Direction,
				"recv_mbps_1000": float64(sample.Recv) * 8 / 60 / 1000 / 1000,
				"recv_mbps_1024": float64(sample.Recv) * 8 / 60 / 1024 / 1024,
				"send_mbps_1000": float64(sample.Send) * 8 / 60 / 1000 / 1000,
				"send_mbps_1024": float64(sample.Send) * 8 / 60 / 1024 / 1024,
			})
		}
	}
	return out
}

func buildNFADailyReportRows(series []trafficReportNFASeries, window trafficReportWindow, params trafficReportParams) []map[string]interface{} {
	out := make([]map[string]interface{}, 0)
	for _, group := range series {
		byDay := make(map[string][]float64)
		for at, sample := range group.Samples {
			value := reportDirectionValue(sample.Recv, sample.Send, params.Direction)
			byDay[at.In(window.Start.Location()).Format("2006-01-02")] = append(byDay[at.In(window.Start.Location()).Format("2006-01-02")], value*8/60/float64(params.UnitBase)/float64(params.UnitBase))
		}
		days := make([]string, 0, len(byDay))
		for day := range byDay {
			days = append(days, day)
		}
		sort.Strings(days)
		for _, day := range days {
			mbps := legacyNFA95(byDay[day])
			out = append(out, map[string]interface{}{
				"school_id": group.SchoolID, "school_name": group.SchoolName, "region": group.Region, "cp": group.CP,
				"date": day, "daily_95th_percentile_raw": mbps * 60 * float64(params.UnitBase) * float64(params.UnitBase) / 8,
				"daily_95th_percentile_mbps": mbps, "direction": params.Direction, "data_points_daily": len(byDay[day]),
			})
		}
	}
	return out
}

func buildNFAMonthlyReportRows(series []trafficReportNFASeries, window trafficReportWindow, params trafficReportParams) []map[string]interface{} {
	out := make([]map[string]interface{}, 0)
	for _, group := range series {
		byMonth := make(map[string][]float64)
		for at, sample := range group.Samples {
			value := reportDirectionValue(sample.Recv, sample.Send, params.Direction)
			byMonth[at.In(window.Start.Location()).Format("2006-01")] = append(byMonth[at.In(window.Start.Location()).Format("2006-01")], value*8/60/float64(params.UnitBase)/float64(params.UnitBase))
		}
		months := make([]string, 0, len(byMonth))
		for month := range byMonth {
			months = append(months, month)
		}
		sort.Strings(months)
		for _, month := range months {
			mbps := legacyNFA95(byMonth[month])
			out = append(out, map[string]interface{}{
				"school_id": group.SchoolID, "school_name": group.SchoolName, "region": group.Region, "cp": group.CP,
				"month": month, "95th_percentile_raw": mbps * 60 * float64(params.UnitBase) * float64(params.UnitBase) / 8,
				"95th_percentile_mbps": mbps, "settlement_mode": "range_95", "data_points": len(byMonth[month]), "direction": params.Direction,
			})
		}
	}
	return out
}

func buildNFASummaryReportRows(series []trafficReportNFASeries, window trafficReportWindow, params trafficReportParams) []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(series))
	for _, group := range series {
		values := make([]float64, 0, len(group.Samples))
		byDay := make(map[string][]float64)
		for at, sample := range group.Samples {
			value := reportDirectionValue(sample.Recv, sample.Send, params.Direction) * 8 / 60 / float64(params.UnitBase) / float64(params.UnitBase)
			values = append(values, value)
			day := at.In(window.Start.Location()).Format("2006-01-02")
			byDay[day] = append(byDay[day], value)
		}
		mbps, points := legacyNFA95(values), len(values)
		if params.SettlementMode == "daily_95_avg" {
			mbps, points = 0, len(byDay)
			for _, daily := range byDay {
				mbps += legacyNFA95(daily)
			}
			if totalDays := legacyTotalDays(window); totalDays > 0 {
				mbps /= float64(totalDays)
			}
		}
		out = append(out, map[string]interface{}{
			"school_id": group.SchoolID, "school_name": group.SchoolName, "region": group.Region, "cp": group.CP,
			"95th_percentile_raw":  mbps * 60 * float64(params.UnitBase) * float64(params.UnitBase) / 8,
			"95th_percentile_mbps": mbps, "settlement_mode": params.SettlementMode, "data_points": points, "direction": params.Direction,
		})
	}
	return out
}

func buildEDCMonthlyReportRows(rows []trafficReportEDCRow, daily []map[string]interface{}, window trafficReportWindow, params trafficReportParams, rankIndex int) []map[string]interface{} {
	pointsByMonth := make(map[string]map[time.Time]float64)
	monthsSet := make(map[string]struct{})
	loc := window.Start.Location()
	for _, row := range rows {
		at := row.CreateTime.In(loc)
		month := at.Format("2006-01")
		monthsSet[month] = struct{}{}
		points := pointsByMonth[month]
		if points == nil {
			points = make(map[time.Time]float64)
			pointsByMonth[month] = points
		}
		points[at] += float64(row.ServiceSize)
	}
	months := make([]string, 0, len(monthsSet))
	for month := range monthsSet {
		months = append(months, month)
	}
	sort.Strings(months)
	instance := strings.TrimSpace(params.DataSourceInstance)
	if instance == "" {
		instance = "ali"
	}
	byDailyMonth := make(map[string][]float64)
	for _, row := range daily {
		value, _ := strconv.ParseFloat(fmt.Sprint(row["daily_95th_percentile_raw"]), 64)
		month := fmt.Sprint(row["date"])
		if len(month) >= 7 {
			byDailyMonth[month[:7]] = append(byDailyMonth[month[:7]], value)
		}
	}
	name := strings.TrimSpace(params.EDCName)
	out := make([]map[string]interface{}, 0, len(months))
	for _, month := range months {
		points := pointsByMonth[month]
		if points == nil {
			continue
		}
		raw, dataPoints := 0.0, 0
		if params.SettlementMode == "daily_95_avg" {
			values := byDailyMonth[month]
			for _, value := range values {
				raw += value
			}
			dataPoints = len(values)
			if dataPoints > 0 {
				raw /= float64(dataPoints)
			}
		} else {
			values := make([]float64, 0, len(points))
			for _, value := range points {
				values = append(values, value)
			}
			raw = legacyNth95(values, rankIndex)
			dataPoints = len(values)
		}
		out = append(out, map[string]interface{}{
			"month": month, "edc_name": name, "data_source_instance": instance,
			"95th_percentile_raw": raw, "95th_percentile_mbps": raw * 8 / 300 / float64(params.UnitBase) / float64(params.UnitBase),
			"settlement_mode": params.SettlementMode, "data_points": dataPoints,
		})
	}
	return out
}

func buildEDCSummaryReportRows(rows []trafficReportEDCRow, daily []map[string]interface{}, params trafficReportParams, rankIndex int, window trafficReportWindow) []map[string]interface{} {
	points := make(map[time.Time]float64)
	loc := window.Start.Location()
	for _, row := range rows {
		at := row.CreateTime.In(loc)
		points[at] += float64(row.ServiceSize)
	}
	if len(points) == 0 {
		return nil
	}
	dailyValues := make([]float64, 0, len(daily))
	for _, row := range daily {
		value, _ := strconv.ParseFloat(fmt.Sprint(row["daily_95th_percentile_raw"]), 64)
		dailyValues = append(dailyValues, value)
	}
	instance := strings.TrimSpace(params.DataSourceInstance)
	if instance == "" {
		instance = "ali"
	}
	values := make([]float64, 0, len(points))
	for _, value := range points {
		values = append(values, value)
	}
	raw, dataPoints := legacyNth95(values, rankIndex), len(values)
	if params.SettlementMode == "daily_95_avg" {
		raw, dataPoints = 0, len(dailyValues)
		for _, value := range dailyValues {
			raw += value
		}
		if totalDays := legacyTotalDays(window); totalDays > 0 {
			raw /= float64(totalDays)
		}
	}
	return []map[string]interface{}{{
		"edc_name": strings.TrimSpace(params.EDCName), "data_source_instance": instance, "95th_percentile_raw": raw,
		"95th_percentile_mbps": raw * 8 / 300 / float64(params.UnitBase) / float64(params.UnitBase),
		"settlement_mode":      params.SettlementMode, "data_points": dataPoints,
	}}
}

func sortTrafficReportRows(rows []map[string]interface{}, sortBy, sortOrder string) []map[string]interface{} {
	if strings.TrimSpace(sortBy) == "" || len(rows) < 2 {
		return rows
	}
	ascending := sortOrder == "asc"
	sort.SliceStable(rows, func(i, j int) bool {
		left, leftOK := rows[i][sortBy]
		right, rightOK := rows[j][sortBy]
		if !leftOK || !rightOK {
			return false
		}
		leftNumber, leftErr := strconv.ParseFloat(fmt.Sprint(left), 64)
		rightNumber, rightErr := strconv.ParseFloat(fmt.Sprint(right), 64)
		if leftErr == nil && rightErr == nil {
			if ascending {
				return leftNumber < rightNumber
			}
			return leftNumber > rightNumber
		}
		if ascending {
			return fmt.Sprint(left) < fmt.Sprint(right)
		}
		return fmt.Sprint(left) > fmt.Sprint(right)
	})
	return rows
}

func normalizeTrafficReportParams(params trafficReportParams) trafficReportParams {
	if params.Region == "" {
		params.Region = params.Province
	}
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
	if params.SettlementMode != "daily_95_avg" {
		params.SettlementMode = "range_95"
	}
	if params.BatchSize < 10 {
		params.BatchSize = 200
	}
	if params.SortOrder != "asc" {
		params.SortOrder = "desc"
	}
	if params.EDCMatchMode != "exact" {
		params.EDCMatchMode = "prefix"
	}
	if !params.CombineV4V6 {
		params.MergeKey = ""
	}
	if params.ExportRaw {
		params.ExportDaily = false
		params.MonthlyAggregate = false
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
	schools := splitReportCSV(params.School)
	if len(params.SchoolNames) > 0 {
		q = q.Where("school_name IN ?", params.SchoolNames)
	}
	if excluded := splitReportCSV(params.ExcludeSchool); len(excluded) > 0 {
		q = q.Where("school_name NOT IN ?", excluded)
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
			q = q.Where("("+strings.Join(parts, " OR ")+")", args...)
		}
	}
	if len(schools) == 0 {
		return rows, q.Order("create_time ASC, school_name ASC, region ASC, cp ASC").Scan(&rows).Error
	}
	batchSize := params.BatchSize
	if batchSize < 10 {
		batchSize = 200
	}
	rows = make([]trafficReportNFARow, 0)
	for start := 0; start < len(schools); start += batchSize {
		end := start + batchSize
		if end > len(schools) {
			end = len(schools)
		}
		var batch []trafficReportNFARow
		query := q.Session(&gorm.Session{}).Where("school_name IN ?", schools[start:end])
		if err := query.Order("create_time ASC, school_name ASC, region ASC, cp ASC").Scan(&batch).Error; err != nil {
			return nil, err
		}
		rows = append(rows, batch...)
	}
	return rows, nil
}

func queryEDCReportRows(ctx context.Context, params trafficReportParams, window trafficReportWindow, scopeService EDCTrafficScopeService, ownerID uint64) ([]trafficReportEDCRow, error) {
	var rows []trafficReportEDCRow
	q := model.DB.WithContext(ctx).Table("edc_traffic_5m AS t").Select("t.bucket_5m AS create_time, t.entity_id, e.edc_name AS entity_name, COALESCE(t.alias, e.alias, '') AS alias, t.region, t.cp, t.entity_type, t.src_region, t.dst_region, GREATEST(t.service_size, 0) AS service_bytes, GREATEST(t.cache_size, 0) AS cache_bytes").Joins("JOIN edc_entities e ON e.id = t.entity_id AND e.enabled = ? AND e.is_backup = ?", true, false).Where("t.bucket_5m >= ? AND t.bucket_5m < ?", window.Start, window.End.Add(time.Second))
	if names := splitReportCSV(params.EDCName); len(names) > 0 {
		parts := make([]string, 0, len(names)+1)
		args := make([]interface{}, 0, len(names)+1)
		exactNames := make([]string, 0, len(names))
		for _, name := range names {
			if params.EDCMatchMode == "exact" && !strings.ContainsAny(name, "*?") {
				exactNames = append(exactNames, name)
				continue
			}
			pattern := strings.NewReplacer("*", "%", "?", "_").Replace(name)
			if params.EDCMatchMode == "prefix" && !strings.ContainsAny(name, "*?") {
				pattern += "%"
			}
			parts = append(parts, "e.edc_name LIKE ?")
			args = append(args, pattern)
		}
		if len(exactNames) > 0 {
			parts = append(parts, "e.edc_name IN ?")
			args = append(args, exactNames)
		}
		if len(parts) == 1 {
			q = q.Where(parts[0], args...)
		} else if len(parts) > 1 {
			q = q.Where("("+strings.Join(parts, " OR ")+")", args...)
		}
	}
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
		if err := writeReportXLSX(path, task.DataSourceType, rows, summary, params); err != nil {
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
	keys := reportKeysForRows(task.DataSourceType, rows, params)
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

func orderedReportKeys(source string, row map[string]interface{}) []string {
	if _, ok := row["date"]; ok {
		if source == "edc" {
			return []string{"date", "edc_name", "data_source_instance", "daily_95th_percentile_raw", "daily_95th_percentile_mbps", "data_points_daily", "saler_group", "saler"}
		}
		return []string{"school_id", "school_name", "region", "cp", "date", "daily_95th_percentile_raw", "daily_95th_percentile_mbps", "direction", "data_points_daily"}
	}
	if _, ok := row["month"]; ok {
		if source == "edc" {
			return []string{"month", "edc_name", "data_source_instance", "95th_percentile_raw", "95th_percentile_mbps", "settlement_mode", "data_points"}
		}
		return []string{"school_id", "school_name", "region", "cp", "month", "95th_percentile_raw", "95th_percentile_mbps", "settlement_mode", "data_points", "direction"}
	}
	if source == "edc" {
		if _, ok := row["95th_percentile_mbps"]; ok {
			return []string{"edc_name", "data_source_instance", "95th_percentile_raw", "95th_percentile_mbps", "settlement_mode", "data_points"}
		}
		return []string{"create_time", "edc_name", "service_size", "data_source_type", "data_source_instance", "requested_edc_name", "unit_base", "saler_group", "saler", "service_size_mbps_1000", "service_size_mbps_1024", "entity_id", "alias", "region", "cp", "entity_type", "src_region", "dst_region", "cache_size", "total_bytes", "selected_bytes"}
	}
	if _, ok := row["95th_percentile_mbps"]; ok {
		return []string{"school_id", "school_name", "region", "cp", "95th_percentile_raw", "95th_percentile_mbps", "settlement_mode", "data_points", "direction"}
	}
	return []string{"create_time", "school_id", "school_name", "region", "cp", "recv_bytes", "send_bytes", "total_bytes", "selected_bytes", "recv_mbps_1000", "recv_mbps_1024", "send_mbps_1000", "send_mbps_1024", "data_source_type", "data_source_instance", "unit_base", "direction"}
}

func reportKeysForRows(source string, rows []map[string]interface{}, params ...trafficReportParams) []string {
	if len(rows) > 0 {
		return orderedReportKeys(source, rows[0])
	}
	var mode trafficReportParams
	if len(params) > 0 {
		mode = params[0]
	}
	if source == "edc" {
		if mode.ExportRaw {
			return []string{"create_time", "edc_name", "service_size", "data_source_type", "data_source_instance", "requested_edc_name", "unit_base", "saler_group", "saler", "service_size_mbps_1000", "service_size_mbps_1024", "entity_id", "alias", "region", "cp", "entity_type", "src_region", "dst_region", "cache_size", "total_bytes", "selected_bytes"}
		}
		if mode.MonthlyAggregate {
			return []string{"month", "edc_name", "data_source_instance", "95th_percentile_raw", "95th_percentile_mbps", "settlement_mode", "data_points"}
		}
		if mode.ExportDaily {
			return []string{"date", "edc_name", "data_source_instance", "daily_95th_percentile_raw", "daily_95th_percentile_mbps", "data_points_daily", "saler_group", "saler"}
		}
		return []string{"edc_name", "data_source_instance", "95th_percentile_raw", "95th_percentile_mbps", "settlement_mode", "data_points"}
	}
	if mode.ExportRaw {
		return []string{"create_time", "school_id", "school_name", "region", "cp", "recv_bytes", "send_bytes", "total_bytes", "selected_bytes", "recv_mbps_1000", "recv_mbps_1024", "send_mbps_1000", "send_mbps_1024", "data_source_type", "data_source_instance", "unit_base", "direction"}
	}
	if mode.MonthlyAggregate {
		return []string{"school_id", "school_name", "region", "cp", "month", "95th_percentile_raw", "95th_percentile_mbps", "settlement_mode", "data_points", "direction"}
	}
	if mode.ExportDaily {
		return []string{"school_id", "school_name", "region", "cp", "date", "daily_95th_percentile_raw", "daily_95th_percentile_mbps", "direction", "data_points_daily"}
	}
	return []string{"school_id", "school_name", "region", "cp", "95th_percentile_raw", "95th_percentile_mbps", "settlement_mode", "data_points", "direction"}
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
func writeReportXLSX(path, source string, rows []map[string]interface{}, summary map[string]interface{}, params trafficReportParams) error {
	f := excelize.NewFile()
	defer f.Close()
	sheet := "流量数据"
	f.SetSheetName("Sheet1", sheet)
	keys := reportKeysForRows(source, rows, params)
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
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' || r == ' ' {
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
