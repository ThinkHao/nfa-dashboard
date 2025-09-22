package repository

import (
    "nfa-dashboard/internal/model"

    "gorm.io/gorm"
    "gorm.io/gorm/clause"
)

// UserSchoolRepository 提供 user_schools 的数据访问
type UserSchoolRepository interface {
	// SetSchoolOwner 以“替换”的方式为 school 设置单一可见用户；
	// 若 userID 为空或 0，则清空该 school 的所有绑定
	SetSchoolOwner(schoolID string, userID *uint64) error
}

type userSchoolRepository struct{}

func NewUserSchoolRepository() UserSchoolRepository { return &userSchoolRepository{} }

func (r *userSchoolRepository) SetSchoolOwner(schoolID string, userID *uint64) error {
	return model.DB.Transaction(func(tx *gorm.DB) error {
		// 先清空该 school 的所有绑定
		if err := tx.Where("school_id = ?", schoolID).Delete(&model.UserSchool{}).Error; err != nil {
			return err
		}
		// 如需设置新 owner，则插入一条新记录（幂等：唯一键(user_id,school_id)保证）
		if userID != nil && *userID > 0 {
			us := model.UserSchool{UserID: *userID, SchoolID: schoolID}
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&us).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
