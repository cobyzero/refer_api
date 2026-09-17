package services

import (
	"errors"
	"time"

	"refer_api/models"
	"refer_api/repository"

	"github.com/google/uuid"
)

// UserService defines the business logic interface for managing users.
type UserService interface {
	GetProfile(userID string) (*models.User, int, error)
	UpdateProfile(userID string, updateData map[string]interface{}) (*models.User, error)
	UpdatePhoto(userID string, photoURL string) error
	FindAll() ([]models.User, error)
	FindByID(id string) (*models.User, error)
	Create(userData map[string]interface{}, roleNames []string) (*models.User, error)
	Update(id string, userData map[string]interface{}, roleNames []string) (*models.User, error)
	Delete(id string) error
}

type userService struct {
	userRepo repository.UserRepository
	auth     AuthProvisioner
}

// AuthProvisioner mirrors operator-managed accounts in Kratos so they
// can sign in immediately. Implemented by *auth.Client.
type AuthProvisioner interface {
	EnsureUser(email, password, name string) (string, error)
	AdminGetUserByEmail(email string) (string, error)
	AdminUpdatePassword(authUserID, newPassword string) error
	AdminDeleteUser(authUserID string) error
}

// NewUserService constructs a new UserService instance. The optional
// AuthProvisioner (Kratos client) is required for Create/Update with a
// password and for Delete cleanup; without it those operations fail loudly.
func NewUserService(userRepo repository.UserRepository, auth ...AuthProvisioner) UserService {
	s := &userService{userRepo: userRepo}
	if len(auth) > 0 {
		s.auth = auth[0]
	}
	return s
}

func (s *userService) GetProfile(userID string) (*models.User, int, error) {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return nil, 0, err
	}
	stars, err := s.userRepo.GetStarBalance(userID)
	if err != nil {
		stars = 0 // Return 0 if stars query fails, don't crash profile query
	}
	return user, stars, nil
}

func (s *userService) UpdateProfile(userID string, updateData map[string]interface{}) (*models.User, error) {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return nil, err
	}

	if name, ok := updateData["name"].(string); ok && name != "" {
		user.Name = name
	}
	if keepUpdated, ok := updateData["keepUpdated"].(bool); ok {
		user.KeepUpdated = keepUpdated
	}
	if photoUrl, ok := updateData["photoUrl"].(string); ok {
		user.PhotoURL = &photoUrl
	}
	if phoneNumber, ok := updateData["phoneNumber"].(string); ok {
		user.PhoneNumber = &phoneNumber
	}
	if birthDate, ok := updateData["birthDate"].(string); ok {
		user.BirthDate = &birthDate
	}
	user.UpdatedAt = time.Now()

	if err := s.userRepo.Update(user); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *userService) UpdatePhoto(userID string, photoURL string) error {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return err
	}
	user.PhotoURL = &photoURL
	user.UpdatedAt = time.Now()
	return s.userRepo.Update(user)
}

func (s *userService) FindAll() ([]models.User, error) {
	return s.userRepo.FindAll()
}

func (s *userService) FindByID(id string) (*models.User, error) {
	return s.userRepo.FindByID(id)
}

func (s *userService) Create(userData map[string]interface{}, roleNames []string) (*models.User, error) {
	email, _ := userData["email"].(string)
	name, _ := userData["name"].(string)
	password, _ := userData["password"].(string)
	keepUpdated, _ := userData["keepUpdated"].(bool)
	photoUrl, _ := userData["photoUrl"].(string)
	phoneNumber, _ := userData["phoneNumber"].(string)
	birthDate, _ := userData["birthDate"].(string)

	if email == "" {
		return nil, errors.New("email is required")
	}

	if s.auth == nil {
		return nil, errors.New("auth provisioner is required to create users")
	}
	if password == "" {
		password = "password123"
	}
	// Kratos owns the credentials: provision (verified) first and
	// mirror its id, so the account can sign in immediately. Passwords are
	// never stored locally.
	authID, err := s.auth.EnsureUser(email, password, name)
	if err != nil {
		return nil, err
	}

	var roles []models.Role
	if len(roleNames) > 0 {
		var err error
		roles, err = s.userRepo.FindRolesByNames(roleNames)
		if err != nil {
			return nil, err
		}
	} else {
		userRole, err := s.userRepo.FindRoleByName("USER")
		if err != nil {
			userRole = &models.Role{
				ID:   uuid.New().String(),
				Name: "USER",
			}
			s.userRepo.CreateRole(userRole)
		}
		roles = []models.Role{*userRole}
	}

	user := &models.User{
		ID:          authID,
		Name:        name,
		Email:       email,
		Password:    "",
		KeepUpdated: keepUpdated,
		Roles:       roles,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if photoUrl != "" {
		user.PhotoURL = &photoUrl
	}
	if phoneNumber != "" {
		user.PhoneNumber = &phoneNumber
	}
	if birthDate != "" {
		user.BirthDate = &birthDate
	}

	if err := s.userRepo.Create(user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *userService) Update(id string, userData map[string]interface{}, roleNames []string) (*models.User, error) {
	user, err := s.userRepo.FindByID(id)
	if err != nil {
		return nil, err
	}
	oldEmail := user.Email

	if name, ok := userData["name"].(string); ok && name != "" {
		user.Name = name
	}
	if email, ok := userData["email"].(string); ok && email != "" {
		user.Email = email
	}
	if password, ok := userData["password"].(string); ok && password != "" {
		if s.auth == nil {
			return nil, errors.New("auth provisioner is required to set passwords")
		}
		// Resolve the Kratos account by the previous email; if it has no
		// auth row yet, provision one (same pattern as login auto-create).
		// The public.users.id never changes, so FKs are untouched.
		authID, resolveErr := s.auth.AdminGetUserByEmail(oldEmail)
		if resolveErr != nil || authID == "" {
			var ensureErr error
			authID, ensureErr = s.auth.EnsureUser(user.Email, password, user.Name)
			if ensureErr != nil {
				return nil, ensureErr
			}
		}
		if err := s.auth.AdminUpdatePassword(authID, password); err != nil {
			return nil, err
		}
		user.Password = ""
	}
	if keepUpdated, ok := userData["keepUpdated"].(bool); ok {
		user.KeepUpdated = keepUpdated
	}
	if photoUrl, ok := userData["photoUrl"].(string); ok {
		user.PhotoURL = &photoUrl
	}
	if phoneNumber, ok := userData["phoneNumber"].(string); ok {
		user.PhoneNumber = &phoneNumber
	}
	if birthDate, ok := userData["birthDate"].(string); ok {
		user.BirthDate = &birthDate
	}

	if len(roleNames) > 0 {
		roles, err := s.userRepo.FindRolesByNames(roleNames)
		if err != nil {
			return nil, err
		}
		user.Roles = roles
	}

	user.UpdatedAt = time.Now()

	if err := s.userRepo.Update(user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *userService) Delete(id string) error {
	user, err := s.userRepo.FindByID(id)
	if err != nil {
		return err
	}
	// Best-effort Kratos cleanup so the deleted account cannot sign in.
	// The local row is always removed, even if Auth cleanup fails.
	if s.auth != nil && user != nil {
		if authID, resolveErr := s.auth.AdminGetUserByEmail(user.Email); resolveErr == nil && authID != "" {
			_ = s.auth.AdminDeleteUser(authID)
		}
	}
	return s.userRepo.Delete(id)
}
