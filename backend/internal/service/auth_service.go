package service

import (
	"errors"
	"nfa-dashboard/config"
	"nfa-dashboard/internal/model"
	"nfa-dashboard/internal/repository"
	"nfa-dashboard/internal/security"
	"regexp"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

type AuthService interface {
	Login(username, password string) (string, *model.User, []model.Permission, error)
	GetUserByID(id uint64) (*model.User, error)
	GetUserPermissions(userID uint64) ([]model.Permission, error)
	ChangePassword(userID uint64, oldPassword, newPassword string) error
}

type authService struct {
	userRepo repository.UserRepository
}

func NewAuthService(userRepo repository.UserRepository) AuthService {
	return &authService{userRepo: userRepo}
}

func (s *authService) Login(username, password string) (string, *model.User, []model.Permission, error) {
	u, err := s.userRepo.GetByUsername(username)
	if err != nil {
		return "", nil, nil, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return "", nil, nil, errors.New("invalid username or password")
	}
	perms, err := s.userRepo.GetUserPermissions(u.ID)
	if err != nil {
		return "", nil, nil, err
	}
	// build JWT
	token, err := security.GenerateToken(u.ID, u.Username, config.GetAccessTokenTTLMinutes())
	if err != nil {
		return "", nil, nil, err
	}
	return token, u, perms, nil
}

func (s *authService) GetUserByID(id uint64) (*model.User, error) {
	return s.userRepo.GetByID(id)
}

func (s *authService) GetUserPermissions(userID uint64) ([]model.Permission, error) {
	return s.userRepo.GetUserPermissions(userID)
}

func (s *authService) ChangePassword(userID uint64, oldPassword, newPassword string) error {
	if userID == 0 {
		return NewBadRequest("invalid user id")
	}
	if strings.TrimSpace(oldPassword) == "" || strings.TrimSpace(newPassword) == "" {
		return NewBadRequest("old password and new password are required")
	}
	if oldPassword == newPassword {
		return NewBadRequest("new password must be different from current password")
	}
	if err := validatePasswordStrength(newPassword); err != nil {
		return err
	}

	u, err := s.userRepo.GetByID(userID)
	if err != nil {
		return errors.New("user not found or disabled")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(oldPassword)); err != nil {
		return errors.New("current password is incorrect")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return s.userRepo.UpdatePasswordHash(userID, string(hash))
}

func validatePasswordStrength(password string) error {
	if len(password) < 8 {
		return NewBadRequest("new password must be at least 8 characters")
	}
	if len(password) > 64 {
		return NewBadRequest("new password must not exceed 64 characters")
	}
	hasLower, _ := regexp.MatchString(`[a-z]`, password)
	hasUpper, _ := regexp.MatchString(`[A-Z]`, password)
	hasDigit, _ := regexp.MatchString(`\d`, password)
	hasSpecial, _ := regexp.MatchString(`[^A-Za-z0-9]`, password)
	if !hasLower || !hasUpper || !hasDigit || !hasSpecial {
		return NewBadRequest("new password must include upper/lowercase letters, numbers and symbols")
	}
	return nil
}
