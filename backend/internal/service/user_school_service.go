package service

import (
	"errors"
	"fmt"
	"strings"
	"nfa-dashboard/config"
	"nfa-dashboard/internal/model"
	"nfa-dashboard/internal/repository"
)

// UserSchoolService 提供用户-院校绑定的业务能力
// 约束：每个院校仅绑定一个可见用户；传入 nil/0 等价于解绑（清空绑定）
type UserSchoolService interface {
	SetOwner(schoolID string, userID *uint64) error
}

type userSchoolService struct {
	userRepo       repository.UserRepository
	schoolRepo     repository.SchoolRepository
	userSchoolRepo repository.UserSchoolRepository
}

func NewUserSchoolService(userRepo repository.UserRepository, schoolRepo repository.SchoolRepository, userSchoolRepo repository.UserSchoolRepository) UserSchoolService {
	return &userSchoolService{userRepo: userRepo, schoolRepo: schoolRepo, userSchoolRepo: userSchoolRepo}
}

func (s *userSchoolService) SetOwner(schoolID string, userID *uint64) error {
	if schoolID == "" {
		return errors.New("school_id is required")
	}
	// 校验 school 是否存在
	var school model.School
	if err := model.DB.Where("school_id = ?", schoolID).First(&school).Error; err != nil {
		return fmt.Errorf("school not found: %w", err)
	}
	// 如需绑定用户，则校验用户是否存在且启用
	if userID != nil && *userID > 0 {
		if ok, err := s.userRepo.Exists(*userID); err != nil {
			return err
		} else if !ok {
			return fmt.Errorf("user not found: %d", *userID)
		}
		// 角色白名单校验（可配置）。为空则不限制。绑定院校owner使用销售角色白名单。
        allowed := config.GetAllowedSalesRoles()
		if len(allowed) > 0 {
			// 构建小写集合
			allowSet := make(map[string]struct{}, len(allowed))
			for _, n := range allowed {
				if n = strings.TrimSpace(strings.ToLower(n)); n != "" {
					allowSet[n] = struct{}{}
				}
			}
			roles, err := s.userRepo.GetUserRoles(*userID)
			if err != nil {
				return err
			}
			ok := false
			for _, r := range roles {
				if _, hit := allowSet[strings.ToLower(r.Name)]; hit {
					ok = true
					break
				}
			}
			if !ok {
				return fmt.Errorf("user %d is not in allowed roles for binding", *userID)
			}
		}
	}
	return s.userSchoolRepo.SetSchoolOwner(schoolID, userID)
}
