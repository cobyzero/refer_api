package services

import (
	"database/sql"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"refer_api/models"
	"refer_api/repository"
	"refer_api/utils"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// AuthService defines the business logic interface for authentication.
type AuthService interface {
	Register(name, email, password string, keepUpdated bool) (*models.User, string, error)
	Login(email, password string) (*models.User, string, error)
	ChangePassword(userID, currentPassword, newPassword string) error
	ForgotPassword(email string) error
	ResetPassword(email, code, newPassword string) error
}

type authService struct {
	userRepo repository.UserRepository
}

// NewAuthService creates a new AuthService instance with its dependencies.
func NewAuthService(userRepo repository.UserRepository) AuthService {
	return &authService{userRepo: userRepo}
}

func (s *authService) Register(name, email, password string, keepUpdated bool) (*models.User, string, error) {
	// Check if already exists
	_, err := s.userRepo.FindByEmail(email)
	if err == nil {
		return nil, "", errors.New("email already exists")
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, "", err
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	if err != nil {
		return nil, "", err
	}

	// Get USER role
	userRole, err := s.userRepo.FindRoleByName("USER")
	if err != nil {
		// Fallback role creation
		userRole = &models.Role{
			ID:   uuid.New().String(),
			Name: "USER",
		}
		if createErr := s.userRepo.CreateRole(userRole); createErr != nil {
			return nil, "", createErr
		}
	}

	// Create user
	user := &models.User{
		ID:          uuid.New().String(),
		Name:        name,
		Email:       email,
		Password:    string(hashedPassword),
		KeepUpdated: keepUpdated,
		Roles:       []models.Role{*userRole},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.userRepo.Create(user); err != nil {
		return nil, "", err
	}

	token, err := utils.GenerateToken(user.ID, user.Email, []string{userRole.Name})
	if err != nil {
		return nil, "", err
	}

	return user, token, nil
}

func (s *authService) Login(email, password string) (*models.User, string, error) {
	user, err := s.userRepo.FindByEmail(email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, "", errors.New("invalid credentials")
		}
		return nil, "", err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, "", errors.New("invalid credentials")
	}

	var rolesList []string
	for _, r := range user.Roles {
		rolesList = append(rolesList, r.Name)
	}

	token, err := utils.GenerateToken(user.ID, user.Email, rolesList)
	if err != nil {
		return nil, "", err
	}

	return user, token, nil
}

func (s *authService) ChangePassword(userID, currentPassword, newPassword string) error {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("user not found")
		}
		return err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(currentPassword)); err != nil {
		return errors.New("current password incorrect")
	}

	newHashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), 10)
	if err != nil {
		return err
	}

	return s.userRepo.UpdatePassword(userID, string(newHashedPassword))
}

type resetCodeInfo struct {
	code      string
	expiresAt time.Time
}

var resetCodes sync.Map // maps email (string) to resetCodeInfo

func (s *authService) ForgotPassword(email string) error {
	user, err := s.userRepo.FindByEmail(email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("user not found")
		}
		return err
	}

	// Generate 6-digit random code
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	code := fmt.Sprintf("%06d", r.Intn(1000000))

	// Store code in memory with 15 minutes expiration
	resetCodes.Store(email, resetCodeInfo{
		code:      code,
		expiresAt: time.Now().Add(15 * time.Minute),
	})

	// Print code to server console for development testing
	fmt.Printf("\n[RECOVERY CODE] Generated recovery code for %s: %s (expires in 15 mins)\n\n", user.Email, code)
	return nil
}

func (s *authService) ResetPassword(email, code, newPassword string) error {
	val, ok := resetCodes.Load(email)
	if !ok {
		return errors.New("no recovery code requested for this email")
	}

	info := val.(resetCodeInfo)
	if time.Now().After(info.expiresAt) {
		resetCodes.Delete(email)
		return errors.New("recovery code has expired")
	}

	if info.code != code {
		return errors.New("invalid recovery code")
	}

	user, err := s.userRepo.FindByEmail(email)
	if err != nil {
		return err
	}

	newHashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), 10)
	if err != nil {
		return err
	}

	err = s.userRepo.UpdatePassword(user.ID, string(newHashedPassword))
	if err != nil {
		return err
	}

	// Clean up code from memory
	resetCodes.Delete(email)
	return nil
}
