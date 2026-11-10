package repository

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"nfa-dashboard/internal/model"
)

type TodoRepository interface {
	GetCustomerRateUserIDs() ([]uint64, error)
	SetCustomerRateUserIDs(ids []uint64) error
	ReconcileCustomerRateTodos() error
	ListMyCustomerRateTodos(userID uint64, status string, ignored *bool) ([]model.TodoListItem, error)
	CountMyPendingCustomerRateTodos(userID uint64) (int64, error)
	SetCustomerRateTodoIgnored(todoID, userID uint64, ignored bool) error
	ListMyCustomerRateTodoEvents(todoID, userID uint64) ([]model.TodoEventListItem, error)
	SyncCustomerRateTodo(region, cp, schoolName string) error
}

type todoRepository struct{}

func NewTodoRepository() TodoRepository { return &todoRepository{} }

func (r *todoRepository) GetCustomerRateUserIDs() ([]uint64, error) {
	return getCustomerRateUserIDs(model.DB)
}

func getCustomerRateUserIDs(db *gorm.DB) ([]uint64, error) {
	var settings model.SystemTodoSettings
	err := db.Order("id ASC").First(&settings).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return []uint64{}, nil
	}
	if err != nil {
		return nil, err
	}
	if len(settings.CustomerRateUserIDs) == 0 || string(settings.CustomerRateUserIDs) == "null" {
		return []uint64{}, nil
	}
	ids := make([]uint64, 0)
	if err := json.Unmarshal(settings.CustomerRateUserIDs, &ids); err != nil {
		return nil, err
	}
	return ids, nil
}

func (r *todoRepository) SetCustomerRateUserIDs(ids []uint64) error {
	ids = normalizeCustomerRateUserIDs(ids)
	data, err := json.Marshal(ids)
	if err != nil {
		return err
	}
	return model.DB.Transaction(func(tx *gorm.DB) error {
		var settings model.SystemTodoSettings
		err := tx.Order("id ASC").First(&settings).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			settings = model.SystemTodoSettings{CustomerRateUserIDs: datatypes.JSON(data)}
			if err := tx.Create(&settings).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		} else if err := tx.Model(&model.SystemTodoSettings{}).Where("id = ?", settings.ID).Update("customer_rate_user_ids", datatypes.JSON(data)).Error; err != nil {
			return err
		}
		return reconcileCustomerRateTodos(tx, ids)
	})
}

func (r *todoRepository) ReconcileCustomerRateTodos() error {
	return model.DB.Transaction(func(tx *gorm.DB) error {
		userIDs, err := getCustomerRateUserIDs(tx)
		if err != nil {
			return err
		}
		return reconcileCustomerRateTodos(tx, userIDs)
	})
}

func normalizeCustomerRateUserIDs(ids []uint64) []uint64 {
	seen := make(map[uint64]struct{}, len(ids))
	result := make([]uint64, 0, len(ids))
	for _, id := range ids {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}

func reconcileCustomerRateTodos(tx *gorm.DB, userIDs []uint64) error {
	userIDs = normalizeCustomerRateUserIDs(userIDs)
	var rates []model.RateCustomer
	if err := tx.Where("fee_mode = ? AND (customer_fee IS NULL OR network_line_fee IS NULL OR general_fee IS NULL)", "auto").Order("id ASC").Find(&rates).Error; err != nil {
		return err
	}
	for _, rate := range rates {
		if err := ensurePendingCustomerRateTodo(tx, rate); err != nil {
			return err
		}
	}

	var todoIDs []uint64
	if err := tx.Model(&model.Todo{}).
		Where("todo_type = ? AND status = ?", model.TodoTypeCustomerRatePrices, model.TodoStatusPending).
		Order("id ASC").Pluck("id", &todoIDs).Error; err != nil {
		return err
	}
	for _, todoID := range todoIDs {
		if err := reconcileTodoAssignees(tx, todoID, userIDs); err != nil {
			return err
		}
	}
	return nil
}

func ensurePendingCustomerRateTodo(tx *gorm.DB, rate model.RateCustomer) error {
	var todo model.Todo
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("todo_type = ? AND target_id = ?", model.TodoTypeCustomerRatePrices, rate.ID).
		First(&todo).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		todo = model.Todo{TodoType: model.TodoTypeCustomerRatePrices, TargetID: rate.ID, Status: model.TodoStatusPending}
		result := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "todo_type"}, {Name: "target_id"}}, DoNothing: true}).Create(&todo)
		if result.Error != nil {
			return result.Error
		}
		created := result.RowsAffected > 0
		if err := tx.Where("todo_type = ? AND target_id = ?", model.TodoTypeCustomerRatePrices, rate.ID).First(&todo).Error; err != nil {
			return err
		}
		if created {
			if err := insertTodoEvent(tx, todo.ID, nil, "created", nil, &todo.Status, "历史自动导入院校费率存在未填写单价字段"); err != nil {
				return err
			}
		}
	} else if err != nil {
		return err
	}
	if todo.Status == model.TodoStatusCompleted {
		if err := tx.Model(&todo).Updates(map[string]any{"status": model.TodoStatusPending, "completed_at": nil}).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.TodoAssignee{}).Where("todo_id = ?", todo.ID).Updates(map[string]any{"ignored": false, "ignored_at": nil}).Error; err != nil {
			return err
		}
		return insertTodoEvent(tx, todo.ID, nil, "reopened", &todo.Status, ptrString(model.TodoStatusPending), "必填单价字段再次为空")
	}
	return nil
}

func reconcileTodoAssignees(tx *gorm.DB, todoID uint64, userIDs []uint64) error {
	var current []model.TodoAssignee
	if err := tx.Where("todo_id = ?", todoID).Find(&current).Error; err != nil {
		return err
	}
	currentIDs := make(map[uint64]struct{}, len(current))
	for _, assignment := range current {
		currentIDs[assignment.UserID] = struct{}{}
	}
	desiredIDs := make(map[uint64]struct{}, len(userIDs))
	for _, userID := range userIDs {
		if userID != 0 {
			desiredIDs[userID] = struct{}{}
		}
	}

	removed := make([]uint64, 0)
	for userID := range currentIDs {
		if _, keep := desiredIDs[userID]; !keep {
			removed = append(removed, userID)
		}
	}
	sort.Slice(removed, func(i, j int) bool { return removed[i] < removed[j] })
	if len(removed) > 0 {
		if err := tx.Where("todo_id = ? AND user_id IN ?", todoID, removed).Delete(&model.TodoAssignee{}).Error; err != nil {
			return err
		}
		for _, userID := range removed {
			if err := insertTodoEvent(tx, todoID, nil, "unassigned", nil, nil, fmt.Sprintf("系统提醒设置移除接收人 user_id=%d", userID)); err != nil {
				return err
			}
		}
	}

	added := make([]uint64, 0)
	for userID := range desiredIDs {
		if _, exists := currentIDs[userID]; !exists {
			added = append(added, userID)
		}
	}
	sort.Slice(added, func(i, j int) bool { return added[i] < added[j] })
	for _, userID := range added {
		created, err := insertTodoAssignee(tx, todoID, userID)
		if err != nil {
			return err
		}
		if created {
			if err := insertTodoEvent(tx, todoID, nil, "assigned", nil, nil, fmt.Sprintf("系统提醒设置新增接收人 user_id=%d", userID)); err != nil {
				return err
			}
		}
	}
	return nil
}

type todoRateRow struct {
	ID             uint64
	TodoID         uint64
	Status         string
	Region         string
	CP             string
	SchoolName     string
	CustomerFee    *float64
	NetworkLineFee *float64
	GeneralFee     *float64
	Ignored        bool
	CreatedAt      time.Time
}

func (r *todoRepository) ListMyCustomerRateTodos(userID uint64, status string, ignored *bool) ([]model.TodoListItem, error) {
	rows := make([]todoRateRow, 0)
	q := model.DB.Table("nfa_todos t").
		Select("t.id AS todo_id, t.status, t.created_at, rc.id, rc.region, rc.cp, rc.school_name, rc.customer_fee, rc.network_line_fee, rc.general_fee, a.ignored").
		Joins("JOIN nfa_todo_assignees a ON a.todo_id = t.id AND a.user_id = ?", userID).
		Joins("JOIN rate_customer rc ON rc.id = t.target_id").
		Where("t.todo_type = ? AND t.status = ?", model.TodoTypeCustomerRatePrices, status)
	if status == model.TodoStatusPending && ignored != nil {
		q = q.Where("a.ignored = ?", *ignored)
	}
	if err := q.Order("t.created_at ASC, t.id ASC").Scan(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]model.TodoListItem, 0, len(rows))
	for _, row := range rows {
		rate := model.RateCustomer{CustomerFee: row.CustomerFee, NetworkLineFee: row.NetworkLineFee, GeneralFee: row.GeneralFee}
		items = append(items, model.TodoListItem{
			ID: row.TodoID, Status: row.Status, Region: row.Region, CP: row.CP,
			SchoolName: row.SchoolName, MissingFields: rate.MissingCustomerPriceFields(),
			Ignored: row.Ignored, CreatedAt: row.CreatedAt,
		})
	}
	return items, nil
}

func (r *todoRepository) CountMyPendingCustomerRateTodos(userID uint64) (int64, error) {
	var count int64
	err := model.DB.Table("nfa_todos t").
		Joins("JOIN nfa_todo_assignees a ON a.todo_id = t.id AND a.user_id = ? AND a.ignored = 0", userID).
		Where("t.todo_type = ? AND t.status = ?", model.TodoTypeCustomerRatePrices, model.TodoStatusPending).
		Count(&count).Error
	return count, err
}

func (r *todoRepository) SetCustomerRateTodoIgnored(todoID, userID uint64, ignored bool) error {
	return model.DB.Transaction(func(tx *gorm.DB) error {
		var assignment model.TodoAssignee
		if err := tx.Joins("JOIN nfa_todos ON nfa_todos.id = nfa_todo_assignees.todo_id").
			Where("nfa_todo_assignees.todo_id = ? AND nfa_todo_assignees.user_id = ? AND nfa_todos.todo_type = ? AND nfa_todos.status = ?", todoID, userID, model.TodoTypeCustomerRatePrices, model.TodoStatusPending).
			First(&assignment).Error; err != nil {
			return err
		}
		if assignment.Ignored == ignored {
			return nil
		}
		now := time.Now()
		updates := map[string]any{"ignored": ignored}
		if ignored {
			updates["ignored_at"] = now
		} else {
			updates["ignored_at"] = nil
		}
		if err := tx.Model(&model.TodoAssignee{}).Where("id = ?", assignment.ID).Updates(updates).Error; err != nil {
			return err
		}
		actorID := userID
		eventType := "unignored"
		detail := "用户恢复该待办提醒"
		if ignored {
			eventType = "ignored"
			detail = "用户忽略该待办提醒"
		}
		return insertTodoEvent(tx, todoID, &actorID, eventType, nil, nil, detail)
	})
}

func (r *todoRepository) ListMyCustomerRateTodoEvents(todoID, userID uint64) ([]model.TodoEventListItem, error) {
	var assignmentCount int64
	if err := model.DB.Table("nfa_todo_assignees a").
		Joins("JOIN nfa_todos t ON t.id = a.todo_id").
		Where("a.todo_id = ? AND a.user_id = ? AND t.todo_type = ?", todoID, userID, model.TodoTypeCustomerRatePrices).
		Count(&assignmentCount).Error; err != nil {
		return nil, err
	}
	if assignmentCount == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	items := make([]model.TodoEventListItem, 0)
	err := model.DB.Table("nfa_todo_events e").
		Select("e.id, e.actor_user_id, COALESCE(NULLIF(u.alias, ''), NULLIF(u.username, ''), '系统') AS actor_name, e.event_type, e.old_status, e.new_status, e.detail, e.created_at").
		Joins("LEFT JOIN users u ON u.id = e.actor_user_id").
		Where("e.todo_id = ?", todoID).
		Order("e.id ASC").
		Scan(&items).Error
	return items, err
}

func (r *todoRepository) SyncCustomerRateTodo(region, cp, schoolName string) error {
	return model.DB.Transaction(func(tx *gorm.DB) error {
		var rate model.RateCustomer
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("region = ? AND cp = ? AND school_name = ?", region, cp, schoolName).First(&rate).Error; err != nil {
			return err
		}
		var recipients []uint64
		var settings model.SystemTodoSettings
		if err := tx.Order("id ASC").First(&settings).Error; err == nil && len(settings.CustomerRateUserIDs) > 0 && string(settings.CustomerRateUserIDs) != "null" {
			if err := json.Unmarshal(settings.CustomerRateUserIDs, &recipients); err != nil {
				return err
			}
		} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		var todo model.Todo
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("todo_type = ? AND target_id = ?", model.TodoTypeCustomerRatePrices, rate.ID).First(&todo).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if rate.HasCompleteCustomerPrices() {
				return nil
			}
			todo = model.Todo{TodoType: model.TodoTypeCustomerRatePrices, TargetID: rate.ID, Status: model.TodoStatusPending}
			created := false
			createResult := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "todo_type"}, {Name: "target_id"}}, DoNothing: true}).Create(&todo)
			if createResult.Error != nil {
				return createResult.Error
			}
			created = createResult.RowsAffected > 0
			if err := tx.Where("todo_type = ? AND target_id = ?", model.TodoTypeCustomerRatePrices, rate.ID).First(&todo).Error; err != nil {
				return err
			}
			if created {
				if err := insertTodoEvent(tx, todo.ID, nil, "created", nil, &todo.Status, "院校客户费率存在未填写单价字段"); err != nil {
					return err
				}
			}
			return reconcileTodoAssignees(tx, todo.ID, recipients)
		}
		if err != nil {
			return err
		}

		if rate.HasCompleteCustomerPrices() && todo.Status == model.TodoStatusPending {
			now := time.Now()
			if err := tx.Model(&todo).Updates(map[string]any{"status": model.TodoStatusCompleted, "completed_at": &now}).Error; err != nil {
				return err
			}
			return insertTodoEvent(tx, todo.ID, nil, "auto_completed", &todo.Status, ptrString(model.TodoStatusCompleted), "客户费、线路费、节点通用费均已填写")
		}
		if !rate.HasCompleteCustomerPrices() && todo.Status == model.TodoStatusCompleted {
			if err := tx.Model(&todo).Updates(map[string]any{"status": model.TodoStatusPending, "completed_at": nil}).Error; err != nil {
				return err
			}
			if err := tx.Model(&model.TodoAssignee{}).Where("todo_id = ?", todo.ID).Updates(map[string]any{"ignored": false, "ignored_at": nil}).Error; err != nil {
				return err
			}
			if err := insertTodoEvent(tx, todo.ID, nil, "reopened", &todo.Status, ptrString(model.TodoStatusPending), "必填单价字段再次为空"); err != nil {
				return err
			}
		}
		if todo.Status == model.TodoStatusPending || !rate.HasCompleteCustomerPrices() {
			return reconcileTodoAssignees(tx, todo.ID, recipients)
		}
		return nil
	})
}

func insertTodoAssignee(tx *gorm.DB, todoID, userID uint64) (bool, error) {
	assignment := model.TodoAssignee{TodoID: todoID, UserID: userID}
	result := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "todo_id"}, {Name: "user_id"}}, DoNothing: true}).Create(&assignment)
	return result.RowsAffected > 0, result.Error
}

func insertTodoEvent(tx *gorm.DB, todoID uint64, actorID *uint64, eventType string, oldStatus, newStatus *string, detail string) error {
	var detailPtr *string
	if detail != "" {
		detailPtr = &detail
	}
	return tx.Create(&model.TodoEvent{TodoID: todoID, ActorUserID: actorID, EventType: eventType, OldStatus: oldStatus, NewStatus: newStatus, Detail: detailPtr}).Error
}

func ptrString(value string) *string { return &value }
