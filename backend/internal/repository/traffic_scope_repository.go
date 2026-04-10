package repository

import (
	"strings"

	"nfa-dashboard/internal/model"
	"gorm.io/gorm"
)

type TrafficScopeRuleRepository interface {
	ListByUser(userID uint64) ([]model.TrafficScopeRuleGroup, error)
	ReplaceByUser(userID uint64, groups []model.TrafficScopeRuleGroup) error
}

type TrafficScopeSchoolRepository interface {
	MatchSchools(dimension, value string) ([]model.School, error)
	GetSchoolsByKeys(keys []model.TrafficScopeSchoolKey) ([]model.School, error)
	ExpandSchoolIDsToKeys(ids []string) ([]model.TrafficScopeSchoolKey, error)
	ListAllSchools() ([]model.School, error)
}

type trafficScopeRuleRepository struct{}

func NewTrafficScopeRuleRepository() TrafficScopeRuleRepository { return &trafficScopeRuleRepository{} }

func (r *trafficScopeRuleRepository) ListByUser(userID uint64) ([]model.TrafficScopeRuleGroup, error) {
	out := make([]model.TrafficScopeRuleGroup, 0)
	if userID == 0 {
		return out, nil
	}
	err := model.DB.
		Where("user_id = ?", userID).
		Order("id ASC").
		Preload("Conditions", func(db *gorm.DB) *gorm.DB {
			return db.Order("id ASC")
		}).
		Find(&out).Error
	return out, err
}

func (r *trafficScopeRuleRepository) ReplaceByUser(userID uint64, groups []model.TrafficScopeRuleGroup) error {
	return model.DB.Transaction(func(tx *gorm.DB) error {
		existingIDs := make([]uint64, 0)
		if err := tx.Model(&model.TrafficScopeRuleGroup{}).Where("user_id = ?", userID).Pluck("id", &existingIDs).Error; err != nil {
			return err
		}
		if len(existingIDs) > 0 {
			if err := tx.Where("group_id IN ?", existingIDs).Delete(&model.TrafficScopeCondition{}).Error; err != nil {
				return err
			}
		}
		if err := tx.Where("user_id = ?", userID).Delete(&model.TrafficScopeRuleGroup{}).Error; err != nil {
			return err
		}
		for _, group := range groups {
			group.ID = 0
			group.UserID = userID
			group.LegacyRuleID = nil
			conditions := make([]model.TrafficScopeCondition, 0, len(group.Conditions))
			for _, condition := range group.Conditions {
				condition.ID = 0
				condition.GroupID = 0
				conditions = append(conditions, condition)
			}
			group.Conditions = nil
			if err := tx.Create(&group).Error; err != nil {
				return err
			}
			for i := range conditions {
				conditions[i].GroupID = group.ID
			}
			if len(conditions) > 0 {
				if err := tx.Create(&conditions).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
}

type trafficScopeSchoolRepository struct{}

func NewTrafficScopeSchoolRepository() TrafficScopeSchoolRepository { return &trafficScopeSchoolRepository{} }

func (r *trafficScopeSchoolRepository) MatchSchools(dimension, value string) ([]model.School, error) {
	out := make([]model.School, 0)
	v := strings.TrimSpace(value)
	if v == "" {
		return out, nil
	}
	query := model.DB.Model(&model.School{})
	switch dimension {
	case model.TrafficScopeDimensionRegion:
		query = query.Where("region = ?", v)
	case model.TrafficScopeDimensionCP:
		query = query.Where("cp = ?", v)
	case model.TrafficScopeDimensionSchool:
		query = query.Where("school_id = ? OR school_name = ?", v, v)
	default:
		return out, nil
	}
	if err := query.Order("school_id ASC").Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

func (r *trafficScopeSchoolRepository) GetSchoolsByKeys(keys []model.TrafficScopeSchoolKey) ([]model.School, error) {
	out := make([]model.School, 0)
	if len(keys) == 0 {
		return out, nil
	}
	query := model.DB.Model(&model.School{})
	query = applySchoolKeysFilter(query, keys)
	err := query.Order("school_id ASC, region ASC, cp ASC").Find(&out).Error
	return out, err
}

func (r *trafficScopeSchoolRepository) ListAllSchools() ([]model.School, error) {
	out := make([]model.School, 0)
	err := model.DB.Order("school_id ASC").Find(&out).Error
	return out, err
}

func (r *trafficScopeSchoolRepository) ExpandSchoolIDsToKeys(ids []string) ([]model.TrafficScopeSchoolKey, error) {
	if len(ids) == 0 {
		return []model.TrafficScopeSchoolKey{}, nil
	}
	schools := make([]model.School, 0)
	if err := model.DB.Where("school_id IN ?", ids).Order("school_id ASC, region ASC, cp ASC").Find(&schools).Error; err != nil {
		return nil, err
	}
	keys := make([]model.TrafficScopeSchoolKey, 0, len(schools))
	for _, school := range schools {
		keys = append(keys, model.TrafficScopeSchoolKey{
			SchoolID: school.SchoolID,
			Region:   school.Region,
			CP:       school.CP,
		})
	}
	return keys, nil
}

func applySchoolKeysFilter(query *gorm.DB, keys []model.TrafficScopeSchoolKey) *gorm.DB {
	if len(keys) == 0 {
		return query
	}
	clauses := make([]string, 0, len(keys))
	args := make([]interface{}, 0, len(keys)*3)
	for _, key := range keys {
		clauses = append(clauses, "(school_id = ? AND region = ? AND cp = ?)")
		args = append(args, key.SchoolID, key.Region, key.CP)
	}
	return query.Where("("+strings.Join(clauses, " OR ")+")", args...)
}
