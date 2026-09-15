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
// Password recovery code tests
// ===========================

func TestRecoveryCode_ConsumeWithoutIssue(t *testing.T) {
	_, err := consumeRecoveryCode(&mockUserRepo{}, "nobody@example.com", "123456")
	if err == nil {
		t.Fatal("Expected error when no code was requested, got nil")
	}
	// Uniform message: must not reveal whether a code was requested.
	if !errors.Is(err, ErrRecoveryCodeInvalid) {
		t.Errorf("Expected ErrRecoveryCodeInvalid, got: %s", err.Error())
	}
}

func TestRecoveryCode_WrongCodeBurnsAfterMaxAttempts(t *testing.T) {
	repo := &mockUserRepo{user: &models.User{ID: "u1", Email: "a@example.com"}}
	if err := issueRecoveryCode(repo, "a@example.com"); err != nil {
		t.Fatalf("issueRecoveryCode failed: %v", err)
	}
	for i := 0; i < resetCodeMaxAttempts; i++ {
		if _, err := consumeRecoveryCode(repo, "a@example.com", "000000"); !errors.Is(err, ErrRecoveryCodeInvalid) {
			t.Fatalf("attempt %d: expected ErrRecoveryCodeInvalid, got: %v", i, err)
		}
	}
	// Code is burned: even the right code (unknown here) must fail uniformly.
	if _, err := consumeRecoveryCode(repo, "a@example.com", "000000"); !errors.Is(err, ErrRecoveryCodeInvalid) {
		t.Errorf("Expected burned code to fail uniformly, got: %v", err)
	}
}

func TestRecoveryCode_IssueUnknownUserUniformSuccess(t *testing.T) {
	repo := &mockUserRepo{findByEmailErr: sql.ErrNoRows}
	if err := issueRecoveryCode(repo, "ghost@example.com"); err != nil {
		t.Errorf("Expected uniform success for unknown user, got: %v", err)
	}
}

func TestRecoveryCode_ConsumeSuccessSingleUse(t *testing.T) {
	repo := &mockUserRepo{user: &models.User{ID: "u2", Email: "b@example.com"}}
	if err := issueRecoveryCode(repo, "b@example.com"); err != nil {
		t.Fatalf("issueRecoveryCode failed: %v", err)
	}
	val, ok := resetCodes.Load("b@example.com")
	if !ok {
		t.Fatal("Expected a stored code, found none")
	}
	code := val.(resetCodeInfo).code
	user, err := consumeRecoveryCode(repo, "b@example.com", code)
	if err != nil {
		t.Fatalf("Expected code to validate, got: %v", err)
	}
	if user.ID != "u2" {
		t.Errorf("Expected user u2, got: %s", user.ID)
	}
	// Single-use: the same code must fail now.
	if _, err := consumeRecoveryCode(repo, "b@example.com", code); !errors.Is(err, ErrRecoveryCodeInvalid) {
		t.Errorf("Expected consumed code to fail, got: %v", err)
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
