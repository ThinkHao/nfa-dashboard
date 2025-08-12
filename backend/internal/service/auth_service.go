package service

import (
	"errors"
	"nfa-dashboard/config"
	"nfa-dashboard/internal/model"
	"nfa-dashboard/internal/repository"
	"nfa-dashboard/internal/security"

	"golang.org/x/crypto/bcrypt"
)

type AuthService interface {
	Login(username, password string) (string, *model.User, []model.Permission, error)
	GetUserByID(id uint64) (*model.User, error)
	GetUserPermissions(userID uint64) ([]model.Permission, error)
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
