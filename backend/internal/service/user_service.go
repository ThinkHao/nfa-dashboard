package service

import (
	"nfa-dashboard/internal/model"
	"nfa-dashboard/internal/repository"
	"strings"
	"golang.org/x/crypto/bcrypt"
)

type UserService interface {
	List(username string, status *int8, page, pageSize int) ([]model.User, int64, error)
	SetRoles(userID uint64, roleIDs []uint64) error
	UpdateStatus(userID uint64, status int8) error
	Create(username, password string, email, phone *string, status *int8, roleIDs []uint64) (*model.User, error)
	GetUserRoles(userID uint64) ([]model.Role, error)
}

func (s *userService) GetUserRoles(userID uint64) ([]model.Role, error) {
    return s.userRepo.GetUserRoles(userID)
}

type userService struct{ 
	userRepo repository.UserRepository
	roleRepo repository.RoleRepository
}

func NewUserService(userRepo repository.UserRepository, roleRepo repository.RoleRepository) UserService { 
	return &userService{userRepo: userRepo, roleRepo: roleRepo}
}

func (s *userService) List(username string, status *int8, page, pageSize int) ([]model.User, int64, error) {
	return s.userRepo.List(username, status, page, pageSize)
}
func (s *userService) SetRoles(userID uint64, roleIDs []uint64) error { 
	// validate user exists
	exists, err := s.userRepo.Exists(userID)
	if err != nil { return err }
	if !exists { return NewBadRequestf("user %d not found", userID) }

	// dedup roleIDs
	uniq := make([]uint64, 0, len(roleIDs))
	seen := make(map[uint64]struct{}, len(roleIDs))
	for _, id := range roleIDs {
		if id == 0 { continue }
		if _, ok := seen[id]; ok { continue }
		seen[id] = struct{}{}
		uniq = append(uniq, id)
	}
	// verify roles exist
	if len(uniq) > 0 {
		roles, err := s.roleRepo.FindByIDs(uniq)
		if err != nil { return err }
		if len(roles) != len(uniq) {
			present := make(map[uint64]struct{}, len(roles))
			for _, r := range roles { present[r.ID] = struct{}{} }
			missing := make([]uint64, 0)
			for _, id := range uniq { if _, ok := present[id]; !ok { missing = append(missing, id) } }
			return NewBadRequestf("roles not found: %v", missing)
		}
	}
	return s.userRepo.SetRoles(userID, uniq)
}
func (s *userService) UpdateStatus(userID uint64, status int8) error { return s.userRepo.UpdateStatus(userID, status) }

func (s *userService) Create(username, password string, email, phone *string, status *int8, roleIDs []uint64) (*model.User, error) {
	// basic validation
	if username == "" { return nil, NewBadRequest("username is required") }
	if len(password) < 6 { return nil, NewBadRequest("password must be at least 6 chars") }

	// hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil { return nil, err }

	st := int8(1)
	if status != nil { st = *status }

	u := &model.User{ Username: username, PasswordHash: string(hash), Email: email, Phone: phone, Status: st }
	created, err := s.userRepo.Create(u)
	if err != nil {
		// handle duplicate username (unique key)
		if strings.Contains(strings.ToLower(err.Error()), "duplicate") && strings.Contains(err.Error(), "uk_users_username") {
			return nil, NewBadRequest("username already exists")
		}
		return nil, err
	}

	// handle roles if provided
	if len(roleIDs) > 0 {
		// dedup & validate
		uniq := make([]uint64, 0, len(roleIDs))
		seen := make(map[uint64]struct{}, len(roleIDs))
		for _, id := range roleIDs { if id != 0 { if _, ok := seen[id]; !ok { seen[id] = struct{}{}; uniq = append(uniq, id) } } }
		if len(uniq) > 0 {
			roles, err := s.roleRepo.FindByIDs(uniq)
			if err != nil { return nil, err }
			if len(roles) != len(uniq) {
				present := make(map[uint64]struct{}, len(roles))
				for _, r := range roles { present[r.ID] = struct{}{} }
				missing := make([]uint64, 0)
				for _, id := range uniq { if _, ok := present[id]; !ok { missing = append(missing, id) } }
				return nil, NewBadRequestf("roles not found: %v", missing)
			}
			if err := s.userRepo.SetRoles(created.ID, uniq); err != nil { return nil, err }
		}
	}
	return created, nil
}
