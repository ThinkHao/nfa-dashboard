package repository

import (
    "nfa-dashboard/internal/model"

    "gorm.io/gorm"
)

type SettlementFormulaRepository interface {
    List(limit, offset int) ([]model.SettlementFormula, int64, error)
    GetByID(id uint64) (*model.SettlementFormula, error)
    GetByName(name string) (*model.SettlementFormula, error)
    GetFirstEnabled() (*model.SettlementFormula, error)
    Create(item *model.SettlementFormula) error
    Update(item *model.SettlementFormula) error
    Delete(id uint64) error
}

type settlementFormulaRepository struct{}

func NewSettlementFormulaRepository() SettlementFormulaRepository {
    return &settlementFormulaRepository{}
}

func (r *settlementFormulaRepository) List(limit, offset int) ([]model.SettlementFormula, int64, error) {
    var items []model.SettlementFormula
    var count int64
    q := model.DB.Model(&model.SettlementFormula{})
    if err := q.Count(&count).Error; err != nil {
        return nil, 0, err
    }
    if err := q.Order("update_time DESC, id DESC").Limit(limit).Offset(offset).Find(&items).Error; err != nil {
        return nil, 0, err
    }
    return items, count, nil
}

func (r *settlementFormulaRepository) GetByID(id uint64) (*model.SettlementFormula, error) {
    var item model.SettlementFormula
    if err := model.DB.First(&item, id).Error; err != nil {
        return nil, err
    }
    return &item, nil
}

func (r *settlementFormulaRepository) GetByName(name string) (*model.SettlementFormula, error) {
    var item model.SettlementFormula
    if err := model.DB.Where("name = ?", name).First(&item).Error; err != nil {
        return nil, err
    }
    return &item, nil
}

func (r *settlementFormulaRepository) GetFirstEnabled() (*model.SettlementFormula, error) {
    var item model.SettlementFormula
    if err := model.DB.Where("enabled = ?", true).Order("update_time DESC, id DESC").First(&item).Error; err != nil {
        return nil, err
    }
    return &item, nil
}

func (r *settlementFormulaRepository) Create(item *model.SettlementFormula) error {
    return model.DB.Create(item).Error
}

func (r *settlementFormulaRepository) Update(item *model.SettlementFormula) error {
    return model.DB.Model(&model.SettlementFormula{}).Where("id = ?", item.ID).Updates(map[string]interface{}{
        "name":        item.Name,
        "description": item.Description,
        "tokens":      item.Tokens,
        "enabled":     item.Enabled,
        "updated_by":  item.UpdatedBy,
    }).Error
}

func (r *settlementFormulaRepository) Delete(id uint64) error {
    return model.DB.Transaction(func(tx *gorm.DB) error {
        if err := tx.Delete(&model.SettlementFormula{}, id).Error; err != nil {
            return err
        }
        return nil
    })
}
