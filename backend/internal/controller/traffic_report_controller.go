package controller

import (
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gin-gonic/gin"
	"nfa-dashboard/config"
	"nfa-dashboard/internal/service"
)

type TrafficReportController struct{ svc service.TrafficReportService }

func NewTrafficReportController(svc service.TrafficReportService) *TrafficReportController {
	return &TrafficReportController{svc: svc}
}

func (c *TrafficReportController) ListTasks(ctx *gin.Context) {
	uid, ok := currentUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
		return
	}
	page := parseIntDefault(ctx.Query("page"), 1)
	size := parseIntDefault(ctx.Query("page_size"), 50)
	items, total, err := c.svc.ListTasks(uid, page, size)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "list report tasks failed", "error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"items": items, "total": total, "page": page, "page_size": size})
}

func (c *TrafficReportController) CreateTask(ctx *gin.Context) {
	uid, ok := currentUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
		return
	}
	var input service.TrafficReportTaskInput
	if err := ctx.ShouldBindJSON(&input); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "invalid report task", "error": err.Error()})
		return
	}
	task, err := c.svc.CreateTask(uid, input)
	if err != nil {
		writeTrafficReportServiceError(ctx, err)
		return
	}
	resp := gin.H{"task": task}
	if task.Kind == "one_off" {
		runID, e := c.svc.StartRun(uid, task.ID)
		if e != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"message": "task created but run could not start", "error": e.Error(), "task": task})
			return
		}
		resp["run_id"] = runID
	}
	ctx.JSON(http.StatusAccepted, resp)
}

func (c *TrafficReportController) UpdateTask(ctx *gin.Context) {
	uid, ok := currentUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
		return
	}
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
	if err != nil || id == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "invalid task id"})
		return
	}
	var input service.TrafficReportTaskUpdate
	if err := ctx.ShouldBindJSON(&input); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "invalid report task update", "error": err.Error()})
		return
	}
	task, err := c.svc.UpdateTask(uid, id, input)
	if err != nil {
		writeTrafficReportServiceError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"task": task})
}

func (c *TrafficReportController) StartRun(ctx *gin.Context) {
	uid, ok := currentUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
		return
	}
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
	if err != nil || id == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "invalid task id"})
		return
	}
	runID, err := c.svc.StartRun(uid, id)
	if err != nil {
		writeTrafficReportServiceError(ctx, err)
		return
	}
	ctx.JSON(http.StatusAccepted, gin.H{"run_id": runID})
}

func (c *TrafficReportController) ListRuns(ctx *gin.Context) {
	uid, ok := currentUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
		return
	}
	page := parseIntDefault(ctx.Query("page"), 1)
	size := parseIntDefault(ctx.Query("page_size"), 50)
	var taskID uint64
	if raw := ctx.Query("task_id"); raw != "" {
		taskID, _ = strconv.ParseUint(raw, 10, 64)
	}
	items, total, err := c.svc.ListRuns(uid, taskID, page, size)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "list report runs failed", "error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"items": items, "total": total, "page": page, "page_size": size})
}

func (c *TrafficReportController) GetRun(ctx *gin.Context) {
	uid, ok := currentUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
		return
	}
	run, err := c.svc.GetRun(uid, ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"message": "report run not found"})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"run": run})
}

func (c *TrafficReportController) DownloadArtifact(ctx *gin.Context) {
	uid, ok := currentUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
		return
	}
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
	if err != nil || id == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "invalid artifact id"})
		return
	}
	artifact, err := c.svc.DownloadArtifact(uid, id)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"message": "artifact not found"})
		return
	}
	root, _ := filepath.Abs(config.GetTrafficReportStorageDir())
	path, _ := filepath.Abs(artifact.StoragePath)
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == ".." || len(rel) >= 3 && rel[:3] == ".."+string(os.PathSeparator) {
		ctx.JSON(http.StatusForbidden, gin.H{"message": "invalid artifact path"})
		return
	}
	if _, err := os.Stat(path); err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"message": "artifact file is unavailable"})
		return
	}
	ctx.Header("Content-Type", artifact.MediaType)
	ctx.FileAttachment(path, artifact.FileName)
}

func writeTrafficReportServiceError(ctx *gin.Context, err error) {
	if service.IsBadRequest(err) {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}
	ctx.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
}
