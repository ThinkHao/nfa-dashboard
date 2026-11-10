package controller

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"nfa-dashboard/internal/middleware"
	"nfa-dashboard/internal/model"
	"nfa-dashboard/internal/service"
)

type TodoController struct {
	svc service.TodoService
}

func NewTodoController(svc service.TodoService) *TodoController { return &TodoController{svc: svc} }

func (ctl *TodoController) GetCustomerRateSettings(c *gin.Context) {
	settings, err := ctl.svc.GetSettings()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, settings)
}

func (ctl *TodoController) UpdateCustomerRateSettings(c *gin.Context) {
	var req model.TodoSettingsResponse
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request"})
		return
	}
	settings, err := ctl.svc.UpdateSettings(req.CustomerRateUserIDs)
	if err != nil {
		if errors.Is(err, service.ErrInvalidTodoRecipients) {
			c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	c.Set("auditMessage", "更新客户费率待办接收人，共 "+strconv.Itoa(len(settings.CustomerRateUserIDs))+" 人")
	c.JSON(http.StatusOK, settings)
}

func (ctl *TodoController) ListMine(c *gin.Context) {
	user, ok := c.Get(middleware.ContextUserKey)
	if !ok || user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
		return
	}
	currentUser, ok := user.(*model.User)
	if !ok || currentUser.ID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
		return
	}
	status := strings.TrimSpace(c.Query("status"))
	if status == "" {
		status = model.TodoStatusPending
	}
	if status != model.TodoStatusPending && status != model.TodoStatusCompleted {
		c.JSON(http.StatusBadRequest, gin.H{"message": "status must be pending or completed"})
		return
	}
	var ignored *bool
	if value := strings.TrimSpace(c.Query("ignored")); value != "" {
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "ignored must be true or false"})
			return
		}
		ignored = &parsed
	}
	items, err := ctl.svc.ListMyTodos(currentUser.ID, status, ignored)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (ctl *TodoController) CountMine(c *gin.Context) {
	currentUser, ok := todoCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
		return
	}
	count, err := ctl.svc.CountMyPendingTodos(currentUser.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"count": count})
}

func (ctl *TodoController) SetIgnored(c *gin.Context) {
	currentUser, ok := todoCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
		return
	}
	todoID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || todoID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid todo id"})
		return
	}
	var req struct {
		Ignored *bool `json:"ignored" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Ignored == nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "ignored is required"})
		return
	}
	if err := ctl.svc.SetIgnored(todoID, currentUser.ID, *req.Ignored); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"message": "todo not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	action := "恢复"
	if *req.Ignored {
		action = "忽略"
	}
	c.Set("auditMessage", action+"客户费率待办 #"+strconv.FormatUint(todoID, 10))
	c.Status(http.StatusNoContent)
}

func (ctl *TodoController) ListEvents(c *gin.Context) {
	currentUser, ok := todoCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
		return
	}
	todoID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || todoID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid todo id"})
		return
	}
	items, err := ctl.svc.ListMyTodoEvents(todoID, currentUser.ID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"message": "todo not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func todoCurrentUser(c *gin.Context) (*model.User, bool) {
	value, ok := c.Get(middleware.ContextUserKey)
	if !ok || value == nil {
		return nil, false
	}
	user, ok := value.(*model.User)
	return user, ok && user.ID > 0
}
