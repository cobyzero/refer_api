package services

import (
	"errors"
	"time"

	"refer_api/models"
	"refer_api/repository"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
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
}

// NewUserService constructs a new UserService instance.
func NewUserService(userRepo repository.UserRepository) UserService {
	return &userService{userRepo: userRepo}
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

	var hashedPassword string
	if password == "" {
		password = "password123"
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	if err != nil {
		return nil, err
	}
	hashedPassword = string(hashed)

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
		ID:          uuid.New().String(),
		Name:        name,
		Email:       email,
		Password:    hashedPassword,
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

	if name, ok := userData["name"].(string); ok && name != "" {
		user.Name = name
	}
	if email, ok := userData["email"].(string); ok && email != "" {
		user.Email = email
	}
	if password, ok := userData["password"].(string); ok && password != "" {
		hashed, err := bcrypt.GenerateFromPassword([]byte(password), 10)
		if err != nil {
			return nil, err
		}
		user.Password = string(hashed)
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
	return s.userRepo.Delete(id)
}
