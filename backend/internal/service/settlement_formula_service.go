package service

import (
	"encoding/json"
	"fmt"
	"nfa-dashboard/internal/model"
	"nfa-dashboard/internal/repository"
)

type SettlementFormulaService interface {
	List(limit, offset int) ([]model.SettlementFormula, int64, error)
	GetByID(id uint64) (*model.SettlementFormula, error)
	GetFirstEnabled() (*model.SettlementFormula, error)
	Create(name, desc string, tokens json.RawMessage, enabled bool, updatedBy string) (*model.SettlementFormula, error)
	Update(id uint64, name, desc string, tokens json.RawMessage, enabled bool, updatedBy string) error
	Delete(id uint64) error
}

type settlementFormulaService struct {
	repo repository.SettlementFormulaRepository
}

func NewSettlementFormulaService(repo repository.SettlementFormulaRepository) SettlementFormulaService {
	return &settlementFormulaService{repo: repo}
}

func (s *settlementFormulaService) List(limit, offset int) ([]model.SettlementFormula, int64, error) {
	if limit <= 0 { limit = 20 }
	if offset < 0 { offset = 0 }
	return s.repo.List(limit, offset)
}

func (s *settlementFormulaService) GetByID(id uint64) (*model.SettlementFormula, error) {
	return s.repo.GetByID(id)
}

func (s *settlementFormulaService) GetFirstEnabled() (*model.SettlementFormula, error) {
	return s.repo.GetFirstEnabled()
}

func (s *settlementFormulaService) Create(name, desc string, tokens json.RawMessage, enabled bool, updatedBy string) (*model.SettlementFormula, error) {
	if len(tokens) == 0 || !json.Valid(tokens) {
		return nil, fmt.Errorf("tokens 必须是有效的 JSON 数组")
	}
	item := &model.SettlementFormula{
		Name:        name,
		Description: desc,
		Tokens:      string(tokens),
		Enabled:     enabled,
		UpdatedBy:   updatedBy,
	}
	if err := s.repo.Create(item); err != nil { return nil, err }
	return item, nil
}

func (s *settlementFormulaService) Update(id uint64, name, desc string, tokens json.RawMessage, enabled bool, updatedBy string) error {
	if len(tokens) > 0 && !json.Valid(tokens) {
		return fmt.Errorf("tokens 不是有效的 JSON")
	}
	// 若 tokens 为空表示不更新 tokens
	item, err := s.repo.GetByID(id)
	if err != nil { return err }
	item.Name = name
	item.Description = desc
	if len(tokens) > 0 { item.Tokens = string(tokens) }
	item.Enabled = enabled
	item.UpdatedBy = updatedBy
	return s.repo.Update(item)
}

func (s *settlementFormulaService) Delete(id uint64) error {
	return s.repo.Delete(id)
}
