package controller

import (
	"encoding/csv"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"nfa-dashboard/internal/model"
	"nfa-dashboard/internal/service"

	"github.com/gin-gonic/gin"
)

type customerRateImportTaskRuntime struct {
	Filename     string
	Content      []byte
	ValidateOnly bool
}

const customerRateImportTaskType = "cust_rate_import"

type customerRateImportTaskManager struct {
	mu    sync.Mutex
	items map[int64]customerRateImportTaskRuntime
}

func newCustomerRateImportTaskManager() *customerRateImportTaskManager {
	return &customerRateImportTaskManager{items: make(map[int64]customerRateImportTaskRuntime)}
}

func (m *customerRateImportTaskManager) set(taskID int64, runtime customerRateImportTaskRuntime) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items[taskID] = runtime
}

func (m *customerRateImportTaskManager) get(taskID int64) (customerRateImportTaskRuntime, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	rt, ok := m.items[taskID]
	return rt, ok
}

func (m *customerRateImportTaskManager) delete(taskID int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.items, taskID)
}

type customerRateImportTaskMeta struct {
	ValidateOnly   bool                        `json:"validate_only"`
	Affected       int                         `json:"affected"`
	ErrorCount     int                         `json:"error_count"`
	CreatedCount   int                         `json:"created_count"`
	Errors         []customerRateImportError   `json:"errors,omitempty"`
	MissingUsers   []service.MissingImportUser `json:"missing_users,omitempty"`
	CreatedUsers   []service.CreatedImportUser `json:"created_users,omitempty"`
	Message        string                      `json:"message,omitempty"`
	CanAutoCreate  bool                        `json:"can_auto_create_users"`
	LastUpdateUnix int64                       `json:"last_update_unix"`
}

func (ctl *SettlementRatesController) CreateCustomerImportTask(c *gin.Context) {
	validateOnly := parseBoolImportFlag(c, "validate_only")
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "file is required"})
		return
	}
	f, err := file.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "open file failed"})
		return
	}
	defer f.Close()
	content, err := io.ReadAll(f)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "read import file failed"})
		return
	}

	now := time.Now()
	task := &model.SettlementTask{
		TaskType:       customerRateImportTaskType,
		TaskDate:       now,
		Status:         "pending",
		TaskStage:      "pending",
		ProcessedCount: 0,
		TotalCount:     0,
		CreateTime:     now,
		UpdateTime:     now,
	}
	if err := ctl.settlementRepo.CreateSettlementTask(task); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}

	ctl.importTaskManager.set(task.ID, customerRateImportTaskRuntime{
		Filename:     file.Filename,
		Content:      content,
		ValidateOnly: validateOnly,
	})

	go ctl.runCustomerImportTask(task.ID, false)

	c.JSON(http.StatusOK, gin.H{
		"task_id":       task.ID,
		"status":        task.Status,
		"task_stage":    task.TaskStage,
		"validate_only": validateOnly,
	})
}

func (ctl *SettlementRatesController) ContinueCustomerImportTask(c *gin.Context) {
	taskID, ok := parseImportTaskID(c)
	if !ok {
		return
	}
	task, err := ctl.settlementRepo.GetSettlementTaskByID(taskID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "task not found"})
		return
	}
	if task.TaskType != customerRateImportTaskType {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid task type"})
		return
	}
	if task.Status != "waiting_user_confirm" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "task is not waiting for user confirmation"})
		return
	}
	if _, exists := ctl.importTaskManager.get(taskID); !exists {
		c.JSON(http.StatusBadRequest, gin.H{"message": "导入上下文已失效，请重新上传文件"})
		return
	}

	task.Status = "running"
	task.TaskStage = "resuming"
	task.UpdateTime = time.Now()
	if err := ctl.settlementRepo.UpdateSettlementTask(task); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}

	go ctl.runCustomerImportTask(taskID, true)

	c.JSON(http.StatusOK, gin.H{
		"task_id":    task.ID,
		"status":     task.Status,
		"task_stage": task.TaskStage,
	})
}

func (ctl *SettlementRatesController) GetCustomerImportTask(c *gin.Context) {
	taskID, ok := parseImportTaskID(c)
	if !ok {
		return
	}
	task, err := ctl.settlementRepo.GetSettlementTaskByID(taskID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "task not found"})
		return
	}
	if task.TaskType != customerRateImportTaskType {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid task type"})
		return
	}
	meta := parseCustomerRateImportTaskMeta(task.TaskMeta)

	c.JSON(http.StatusOK, gin.H{
		"id":              task.ID,
		"task_type":       task.TaskType,
		"task_date":       task.TaskDate,
		"status":          task.Status,
		"task_stage":      task.TaskStage,
		"start_time":      task.StartTime,
		"end_time":        task.EndTime,
		"processed_count": task.ProcessedCount,
		"total_count":     task.TotalCount,
		"error_message":   task.ErrorMessage,
		"create_time":     task.CreateTime,
		"update_time":     task.UpdateTime,
		"result": gin.H{
			"validate_only":         meta.ValidateOnly,
			"affected":              meta.Affected,
			"error_count":           meta.ErrorCount,
			"created_count":         meta.CreatedCount,
			"errors_preview":        previewImportErrors(meta.Errors, 20),
			"missing_users_preview": previewMissingUsers(meta.MissingUsers, 20),
			"created_users_preview": previewCreatedUsers(meta.CreatedUsers, 20),
			"can_auto_create_users": meta.CanAutoCreate,
			"can_continue":          task.Status == "waiting_user_confirm",
			"errors_csv_url":        "/api/v1/settlement/rates/customer/import/tasks/" + strconv.FormatInt(task.ID, 10) + "/errors.csv",
			"created_users_csv_url": "/api/v1/settlement/rates/customer/import/tasks/" + strconv.FormatInt(task.ID, 10) + "/created-users.csv",
		},
	})
}

func (ctl *SettlementRatesController) DownloadCustomerImportTaskErrorsCSV(c *gin.Context) {
	task, ok := ctl.getCustomerImportTaskByID(c)
	if !ok {
		return
	}
	meta := parseCustomerRateImportTaskMeta(task.TaskMeta)
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename=customer_rate_import_errors.csv")
	_, _ = c.Writer.Write([]byte{0xEF, 0xBB, 0xBF})
	w := csv.NewWriter(c.Writer)
	_ = w.Write([]string{"line", "message"})
	for _, item := range meta.Errors {
		_ = w.Write([]string{strconv.Itoa(item.Line), item.Message})
	}
	w.Flush()
}

func (ctl *SettlementRatesController) DownloadCustomerImportTaskCreatedUsersCSV(c *gin.Context) {
	task, ok := ctl.getCustomerImportTaskByID(c)
	if !ok {
		return
	}
	meta := parseCustomerRateImportTaskMeta(task.TaskMeta)
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename=customer_rate_import_created_users.csv")
	_, _ = c.Writer.Write([]byte{0xEF, 0xBB, 0xBF})
	w := csv.NewWriter(c.Writer)
	_ = w.Write([]string{"alias", "username", "password"})
	for _, item := range meta.CreatedUsers {
		_ = w.Write([]string{item.Alias, item.Username, item.Password})
	}
	w.Flush()
}

func (ctl *SettlementRatesController) runCustomerImportTask(taskID int64, autoCreateMissingUsers bool) {
	runtime, ok := ctl.importTaskManager.get(taskID)
	if !ok {
		_ = ctl.failCustomerImportTask(taskID, "导入上下文已失效，请重新上传文件")
		return
	}

	task, err := ctl.settlementRepo.GetSettlementTaskByID(taskID)
	if err != nil {
		return
	}
	now := time.Now()
	if task.StartTime == nil {
		task.StartTime = &now
	}
	task.Status = "running"
	task.TaskStage = "parsing"
	task.UpdateTime = now
	_ = ctl.settlementRepo.UpdateSettlementTask(task)

	header, rows, err := parseCustomerRateImportFile(runtime.Filename, runtime.Content)
	if err != nil {
		msg := "read import file failed"
		if service.IsBadRequest(err) {
			msg = err.Error()
		}
		_ = ctl.failCustomerImportTask(taskID, msg)
		return
	}

	preparedRows, prepareErrors := prepareCustomerRateImportRows(header, rows)
	missingUsers, missingLookupErrors, err := collectMissingCustomerRateImportUsers(ctl.svc, preparedRows)
	if err != nil {
		_ = ctl.failCustomerImportTask(taskID, err.Error())
		return
	}
	allErrors := append([]customerRateImportError{}, prepareErrors...)
	allErrors = append(allErrors, missingLookupErrors...)

	task, err = ctl.settlementRepo.GetSettlementTaskByID(taskID)
	if err != nil {
		return
	}
	task.TotalCount = len(preparedRows)
	task.ProcessedCount = 0
	task.TaskStage = "processing"
	task.TaskMeta = marshalCustomerRateImportTaskMeta(customerRateImportTaskMeta{
		ValidateOnly:   runtime.ValidateOnly,
		ErrorCount:     len(allErrors),
		Errors:         allErrors,
		MissingUsers:   missingUsers,
		CanAutoCreate:  len(missingUsers) > 0,
		LastUpdateUnix: time.Now().Unix(),
	})
	task.UpdateTime = time.Now()
	_ = ctl.settlementRepo.UpdateSettlementTask(task)

	if len(missingUsers) > 0 && !autoCreateMissingUsers {
		task.Status = "waiting_user_confirm"
		task.TaskStage = "waiting_user_confirm"
		task.TaskMeta = marshalCustomerRateImportTaskMeta(customerRateImportTaskMeta{
			ValidateOnly:   runtime.ValidateOnly,
			ErrorCount:     len(allErrors),
			Errors:         allErrors,
			MissingUsers:   missingUsers,
			CanAutoCreate:  true,
			Message:        "检测到未匹配用户，等待确认自动创建后继续",
			LastUpdateUnix: time.Now().Unix(),
		})
		task.UpdateTime = time.Now()
		_ = ctl.settlementRepo.UpdateSettlementTask(task)
		return
	}

	createdUsers := make([]service.CreatedImportUser, 0)
	if autoCreateMissingUsers && len(missingUsers) > 0 {
		createdUsers, err = ctl.svc.CreateCustomerRateImportUsers(missingUsers)
		if err != nil {
			_ = ctl.failCustomerImportTask(taskID, err.Error())
			return
		}
	}

	affected := 0
	executionErrors := make([]customerRateImportError, 0)
	for idx, row := range preparedRows {
		rate := clonePreparedImportRate(row.Rate)
		if err := ctl.svc.ResolveCustomerRateOwnerIDsByDisplayName(rate, row.OwnerNames); err != nil {
			executionErrors = append(executionErrors, customerRateImportError{Line: row.Line, Message: err.Error()})
		} else if err := ctl.svc.ValidateCustomerRate(rate); err != nil {
			executionErrors = append(executionErrors, customerRateImportError{Line: row.Line, Message: err.Error()})
		} else if !runtime.ValidateOnly {
			if err := ctl.svc.UpsertCustomerRate(rate); err != nil {
				executionErrors = append(executionErrors, customerRateImportError{Line: row.Line, Message: err.Error()})
			} else {
				affected++
			}
		} else {
			affected++
		}

		processed := idx + 1
		if processed == len(preparedRows) || processed%20 == 0 {
			task, getErr := ctl.settlementRepo.GetSettlementTaskByID(taskID)
			if getErr == nil {
				task.ProcessedCount = processed
				task.TotalCount = len(preparedRows)
				task.TaskStage = "processing"
				task.TaskMeta = marshalCustomerRateImportTaskMeta(customerRateImportTaskMeta{
					ValidateOnly:   runtime.ValidateOnly,
					Affected:       affected,
					ErrorCount:     len(allErrors) + len(executionErrors),
					CreatedCount:   len(createdUsers),
					CanAutoCreate:  len(missingUsers) > 0,
					LastUpdateUnix: time.Now().Unix(),
				})
				task.UpdateTime = time.Now()
				_ = ctl.settlementRepo.UpdateSettlementTask(task)
			}
		}
	}

	allErrors = append(allErrors, executionErrors...)
	task, err = ctl.settlementRepo.GetSettlementTaskByID(taskID)
	if err != nil {
		return
	}
	doneAt := time.Now()
	task.Status = "success"
	task.TaskStage = "completed"
	task.EndTime = &doneAt
	task.ProcessedCount = len(preparedRows)
	task.TotalCount = len(preparedRows)
	task.ErrorMessage = ""
	task.TaskMeta = marshalCustomerRateImportTaskMeta(customerRateImportTaskMeta{
		ValidateOnly:   runtime.ValidateOnly,
		Affected:       affected,
		ErrorCount:     len(allErrors),
		CreatedCount:   len(createdUsers),
		Errors:         allErrors,
		CreatedUsers:   createdUsers,
		MissingUsers:   []service.MissingImportUser{},
		CanAutoCreate:  false,
		LastUpdateUnix: time.Now().Unix(),
	})
	task.UpdateTime = doneAt
	_ = ctl.settlementRepo.UpdateSettlementTask(task)
	ctl.importTaskManager.delete(taskID)
}

func (ctl *SettlementRatesController) failCustomerImportTask(taskID int64, message string) error {
	task, err := ctl.settlementRepo.GetSettlementTaskByID(taskID)
	if err != nil {
		return err
	}
	now := time.Now()
	task.Status = "failed"
	task.TaskStage = "failed"
	task.EndTime = &now
	task.ErrorMessage = message
	task.TaskMeta = marshalCustomerRateImportTaskMeta(customerRateImportTaskMeta{
		Message:        message,
		LastUpdateUnix: time.Now().Unix(),
	})
	task.UpdateTime = now
	updateErr := ctl.settlementRepo.UpdateSettlementTask(task)
	ctl.importTaskManager.delete(taskID)
	return updateErr
}

func (ctl *SettlementRatesController) getCustomerImportTaskByID(c *gin.Context) (*model.SettlementTask, bool) {
	taskID, ok := parseImportTaskID(c)
	if !ok {
		return nil, false
	}
	task, err := ctl.settlementRepo.GetSettlementTaskByID(taskID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "task not found"})
		return nil, false
	}
	if task.TaskType != customerRateImportTaskType {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid task type"})
		return nil, false
	}
	return task, true
}

func parseImportTaskID(c *gin.Context) (int64, bool) {
	idStr := strings.TrimSpace(c.Param("id"))
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid task id"})
		return 0, false
	}
	return id, true
}

func marshalCustomerRateImportTaskMeta(meta customerRateImportTaskMeta) string {
	meta.LastUpdateUnix = time.Now().Unix()
	raw, err := json.Marshal(meta)
	if err != nil {
		return "{}"
	}
	return string(raw)
}

func parseCustomerRateImportTaskMeta(raw string) customerRateImportTaskMeta {
	if strings.TrimSpace(raw) == "" {
		return customerRateImportTaskMeta{}
	}
	var out customerRateImportTaskMeta
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return customerRateImportTaskMeta{}
	}
	return out
}

func previewImportErrors(items []customerRateImportError, limit int) []customerRateImportError {
	if len(items) <= limit {
		return items
	}
	return items[:limit]
}

func previewMissingUsers(items []service.MissingImportUser, limit int) []service.MissingImportUser {
	if len(items) <= limit {
		return items
	}
	return items[:limit]
}

func previewCreatedUsers(items []service.CreatedImportUser, limit int) []service.CreatedImportUser {
	if len(items) <= limit {
		return items
	}
	return items[:limit]
}
