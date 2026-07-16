package services

import (
	"database/sql"
	"errors"
	"testing"

	"refer_api/models"
)

// =====================
// Mock UserRepository
// =====================
type mockUserRepo struct {
	user           *models.User
	findByEmailErr error
	findByIDErr    error
	createErr      error
	updateErr      error
	deleteErr      error
	updatePwErr    error
	role           *models.Role
	findRoleErr    error
	findRolesErr   error
	starBalance    int
	starBalErr     error
}

func (m *mockUserRepo) FindByEmail(_ string) (*models.User, error) {
	return m.user, m.findByEmailErr
}
func (m *mockUserRepo) FindByID(_ string) (*models.User, error) {
	return m.user, m.findByIDErr
}
func (m *mockUserRepo) FindAll() ([]models.User, error) {
	if m.user != nil {
		return []models.User{*m.user}, nil
	}
	return []models.User{}, nil
}
func (m *mockUserRepo) Create(_ *models.User) error              { return m.createErr }
func (m *mockUserRepo) Update(_ *models.User) error              { return m.updateErr }
func (m *mockUserRepo) Delete(_ string) error                    { return m.deleteErr }
func (m *mockUserRepo) UpdatePassword(_, _ string) error        { return m.updatePwErr }
func (m *mockUserRepo) FindRoleByName(_ string) (*models.Role, error) {
	return m.role, m.findRoleErr
}
func (m *mockUserRepo) FindRolesByNames(_ []string) ([]models.Role, error) {
	if m.findRolesErr != nil {
		return nil, m.findRolesErr
	}
	if m.role != nil {
		return []models.Role{*m.role}, nil
	}
	return []models.Role{}, nil
}
func (m *mockUserRepo) CreateRole(_ *models.Role) error { return nil }
func (m *mockUserRepo) GetStarBalance(_ string) (int, error) {
	return m.starBalance, m.starBalErr
}

// ===========================
// AuthService Unit Tests
// ===========================

func TestAuthService_Login_UserNotFound(t *testing.T) {
	repo := &mockUserRepo{findByEmailErr: sql.ErrNoRows}
	svc := NewAuthService(repo)
	_, _, err := svc.Login("notexist@example.com", "password")
	if err == nil {
		t.Fatal("Expected error for non-existent user, got nil")
	}
	if err.Error() != "invalid credentials" {
		t.Errorf("Expected 'invalid credentials', got: %s", err.Error())
	}
}

func TestAuthService_Login_WrongPassword(t *testing.T) {
	// We need a real bcrypt hash here. Using a known hash of "correctpassword"
	// Use: bcrypt.GenerateFromPassword([]byte("correctpassword"), 10) offline
	repo := &mockUserRepo{
		user: &models.User{
			ID:       "uuid1",
			Email:    "test@example.com",
			Password: "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy", // hash of "correctpassword"
			Roles:    []models.Role{{Name: "USER"}},
		},
	}
	svc := NewAuthService(repo)
	_, _, err := svc.Login("test@example.com", "wrongpassword")
	if err == nil {
		t.Fatal("Expected error for wrong password, got nil")
	}
	if err.Error() != "invalid credentials" {
		t.Errorf("Expected 'invalid credentials', got: %s", err.Error())
	}
}

func TestAuthService_Register_EmailAlreadyExists(t *testing.T) {
	// When FindByEmail returns no error (user exists), Register should fail
	repo := &mockUserRepo{user: &models.User{Email: "exists@example.com"}}
	svc := NewAuthService(repo)
	_, _, err := svc.Register("John", "exists@example.com", "password123", false)
	if err == nil {
		t.Fatal("Expected error when email already exists, got nil")
	}
	if err.Error() != "email already exists" {
		t.Errorf("Expected 'email already exists', got: %s", err.Error())
	}
}

func TestAuthService_Register_FindByEmailSystemError(t *testing.T) {
	// System error during FindByEmail (not ErrNoRows) should propagate
	repo := &mockUserRepo{findByEmailErr: errors.New("connection error")}
	svc := NewAuthService(repo)
	_, _, err := svc.Register("John", "john@example.com", "password123", false)
	if err == nil {
		t.Fatal("Expected system error to propagate, got nil")
	}
	if err.Error() != "connection error" {
		t.Errorf("Expected 'connection error', got: %s", err.Error())
	}
}

func TestAuthService_ResetPassword_NoCodeRequested(t *testing.T) {
	svc := NewAuthService(&mockUserRepo{})
	err := svc.ResetPassword("nobody@example.com", "123456", "newpass")
	if err == nil {
		t.Fatal("Expected error when no code was requested, got nil")
	}
	if err.Error() != "no recovery code requested for this email" {
		t.Errorf("Expected 'no recovery code requested for this email', got: %s", err.Error())
	}
}

// ===========================
// UserService Unit Tests
// ===========================

func TestUserService_GetProfile_Success(t *testing.T) {
	repo := &mockUserRepo{
		user:        &models.User{ID: "u1", Name: "Test User"},
		starBalance: 100,
	}
	svc := NewUserService(repo)
	user, stars, err := svc.GetProfile("u1")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if user.ID != "u1" {
		t.Errorf("Expected user ID 'u1', got: %s", user.ID)
	}
	if stars != 100 {
		t.Errorf("Expected 100 stars, got: %d", stars)
	}
}

func TestUserService_GetProfile_StarBalanceError_ReturnsZero(t *testing.T) {
	repo := &mockUserRepo{
		user:       &models.User{ID: "u1", Name: "Test"},
		starBalErr: errors.New("star query failed"),
	}
	svc := NewUserService(repo)
	_, stars, err := svc.GetProfile("u1")
	// Should not fail even if star query errors — returns 0 instead
	if err != nil {
		t.Fatalf("Expected no error even with star query failure, got: %v", err)
	}
	if stars != 0 {
		t.Errorf("Expected 0 stars on error, got: %d", stars)
	}
}

func TestUserService_GetProfile_NotFound(t *testing.T) {
	repo := &mockUserRepo{findByIDErr: errors.New("user not found")}
	svc := NewUserService(repo)
	_, _, err := svc.GetProfile("nonexistent")
	if err == nil {
		t.Fatal("Expected error when user not found, got nil")
	}
}

func TestUserService_UpdateProfile_UpdatesName(t *testing.T) {
	repo := &mockUserRepo{user: &models.User{ID: "u1", Name: "Old Name"}}
	svc := NewUserService(repo)
	updatedUser, err := svc.UpdateProfile("u1", map[string]interface{}{"name": "New Name"})
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if updatedUser.Name != "New Name" {
		t.Errorf("Expected name 'New Name', got: %s", updatedUser.Name)
	}
}

func TestUserService_Delete_Success(t *testing.T) {
	repo := &mockUserRepo{}
	svc := NewUserService(repo)
	err := svc.Delete("u1")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
}

func TestUserService_Delete_Error(t *testing.T) {
	repo := &mockUserRepo{deleteErr: errors.New("cannot delete")}
	svc := NewUserService(repo)
	err := svc.Delete("u1")
	if err == nil {
		t.Fatal("Expected error when delete fails, got nil")
	}
}

func TestUserService_FindAll_ReturnsUsers(t *testing.T) {
	repo := &mockUserRepo{user: &models.User{ID: "u1", Name: "Alice"}}
	svc := NewUserService(repo)
	users, err := svc.FindAll()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if len(users) != 1 {
		t.Errorf("Expected 1 user, got: %d", len(users))
	}
	if users[0].Name != "Alice" {
		t.Errorf("Expected user name 'Alice', got: %s", users[0].Name)
	}
}
