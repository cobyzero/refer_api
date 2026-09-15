package services

import (
	"errors"
	"testing"

	"refer_api/models"
)

// =====================
// Mock StarRepository
// =====================
type mockStarRepo struct {
	balance        int
	balanceErr     error
	history        []models.StarTransaction
	rewards        []models.Reward
	rewardByID     *models.Reward
	rewardByIDErr  error
	createTxErr    error
	createRRErr    error
	redeemedRewards []models.RedeemedReward
	createRewardErr error
	updateRewardErr error
	deleteRewardErr error
}

func (m *mockStarRepo) GetStarBalance(_ string) (int, error) { return m.balance, m.balanceErr }
func (m *mockStarRepo) FindHistory(_ string) ([]models.StarTransaction, error) {
	return m.history, nil
}
func (m *mockStarRepo) FindHistoryRecent(_ string, limit int) ([]models.StarTransaction, error) {
	if limit > 0 && len(m.history) > limit {
		return m.history[:limit], nil
	}
	return m.history, nil
}
func (m *mockStarRepo) FindRewards() ([]models.Reward, error)   { return m.rewards, nil }
func (m *mockStarRepo) FindRewardByID(_ string) (*models.Reward, error) {
	return m.rewardByID, m.rewardByIDErr
}
func (m *mockStarRepo) CreateTransaction(_ *models.StarTransaction) error { return m.createTxErr }
func (m *mockStarRepo) CreateRedeemedReward(_ *models.RedeemedReward) error { return m.createRRErr }
func (m *mockStarRepo) FindRedeemedRewards(_ string) ([]models.RedeemedReward, error) {
	return m.redeemedRewards, nil
}
func (m *mockStarRepo) FindRedeemedRewardByID(_ string) (*models.RedeemedReward, error) {
	return nil, nil
}
func (m *mockStarRepo) UpdateRedeemedReward(_ *models.RedeemedReward) error { return nil }
func (m *mockStarRepo) GetTotalStarsAwarded() (int, error)                  { return 0, nil }
func (m *mockStarRepo) GetTotalStarsRedeemed() (int, error)                 { return 0, nil }
func (m *mockStarRepo) GetAllTransactions() ([]models.StarTransaction, error) {
	return m.history, nil
}
func (m *mockStarRepo) CreateReward(_ *models.Reward) error  { return m.createRewardErr }
func (m *mockStarRepo) UpdateReward(_ *models.Reward) error  { return m.updateRewardErr }
func (m *mockStarRepo) DeleteReward(_ string) error          { return m.deleteRewardErr }

// ===========================
// StarService Unit Tests
// ===========================

func TestStarService_GetPerks_ReturnsTwoPerks(t *testing.T) {
	svc := NewStarService(&mockStarRepo{})
	perks := svc.GetPerks()
	if len(perks) != 2 {
		t.Fatalf("Expected 2 perks, got: %d", len(perks))
	}
	if perks[0]["id"] != "birthday" {
		t.Errorf("Expected first perk to be 'birthday', got: %s", perks[0]["id"])
	}
	if perks[1]["id"] != "double-stars" {
		t.Errorf("Expected second perk to be 'double-stars', got: %s", perks[1]["id"])
	}
}

func TestStarService_GetBalance_Success(t *testing.T) {
	svc := NewStarService(&mockStarRepo{balance: 150})
	bal, err := svc.GetBalance("user1")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if bal != 150 {
		t.Errorf("Expected balance 150, got: %d", bal)
	}
}

func TestStarService_GetBalance_Error(t *testing.T) {
	svc := NewStarService(&mockStarRepo{balanceErr: errors.New("db error")})
	_, err := svc.GetBalance("user1")
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
}

func TestStarService_GetHistory_ReturnsEmptySliceNotNil(t *testing.T) {
	// Service should return empty slice, not nil, for nil history
	svc := NewStarService(&mockStarRepo{history: nil})
	history, err := svc.GetHistory("user1")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if history == nil {
		t.Error("Expected empty slice, got nil")
	}
}

func TestStarService_GetRewards_ReturnsEmptySliceNotNil(t *testing.T) {
	svc := NewStarService(&mockStarRepo{rewards: nil})
	rewards, err := svc.GetRewards()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if rewards == nil {
		t.Error("Expected empty slice, got nil")
	}
}

func TestStarService_RedeemReward_RewardNotFound(t *testing.T) {
	repo := &mockStarRepo{balance: 200, rewardByIDErr: errors.New("not found")}
	svc := NewStarService(repo)
	_, err := svc.RedeemReward("user1", "reward-999")
	if err == nil {
		t.Fatal("Expected error when reward not found, got nil")
	}
	if err.Error() != "reward not found" {
		t.Errorf("Expected 'reward not found', got: %s", err.Error())
	}
}

func TestStarService_RedeemReward_InactiveReward(t *testing.T) {
	repo := &mockStarRepo{
		balance:       200,
		rewardByID:    &models.Reward{ID: "r1", Title: "Test", StarsRequired: 100, IsActive: false},
		rewardByIDErr: nil,
	}
	svc := NewStarService(repo)
	_, err := svc.RedeemReward("user1", "r1")
	if err == nil {
		t.Fatal("Expected error for inactive reward, got nil")
	}
	if err.Error() != "reward not found" {
		t.Errorf("Expected 'reward not found' for inactive, got: %s", err.Error())
	}
}

func TestStarService_RedeemReward_NotEnoughStars(t *testing.T) {
	repo := &mockStarRepo{
		balance:    50,
		rewardByID: &models.Reward{ID: "r1", Title: "Test", StarsRequired: 100, IsActive: true},
	}
	svc := NewStarService(repo)
	_, err := svc.RedeemReward("user1", "r1")
	if err == nil {
		t.Fatal("Expected error for insufficient stars, got nil")
	}
	if err.Error() != "not enough stars" {
		t.Errorf("Expected 'not enough stars', got: %s", err.Error())
	}
}

func TestStarService_RedeemReward_Success(t *testing.T) {
	repo := &mockStarRepo{
		balance:    200,
		rewardByID: &models.Reward{ID: "r1", Title: "Free Coffee", StarsRequired: 100, IsActive: true},
	}
	svc := NewStarService(repo)
	result, err := svc.RedeemReward("user1", "r1")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if result["success"] != true {
		t.Error("Expected success to be true")
	}
	if result["remainingStars"].(int) != 100 {
		t.Errorf("Expected remainingStars to be 100, got: %v", result["remainingStars"])
	}
}

func TestStarService_CreateReward_Success(t *testing.T) {
	svc := NewStarService(&mockStarRepo{})
	reward, err := svc.CreateReward(map[string]interface{}{
		"title":         "Test Reward",
		"description":   "A test reward",
		"starsRequired": float64(50),
		"isActive":      true,
	})
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if reward.Title != "Test Reward" {
		t.Errorf("Expected title 'Test Reward', got: %s", reward.Title)
	}
	if reward.StarsRequired != 50 {
		t.Errorf("Expected starsRequired 50, got: %d", reward.StarsRequired)
	}
}

func TestStarService_CreateReward_DefaultsIsActiveTrue(t *testing.T) {
	svc := NewStarService(&mockStarRepo{})
	// Not passing isActive — should default to true
	reward, err := svc.CreateReward(map[string]interface{}{
		"title":         "Default Active",
		"description":   "Test",
		"starsRequired": float64(10),
	})
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if !reward.IsActive {
		t.Error("Expected IsActive to default to true")
	}
}

func TestStarService_DeleteReward_Error(t *testing.T) {
	svc := NewStarService(&mockStarRepo{deleteRewardErr: errors.New("delete failed")})
	err := svc.DeleteReward("r-999")
	if err == nil {
		t.Fatal("Expected error when delete fails, got nil")
	}
}

func TestStarService_UpdateReward_NotFound(t *testing.T) {
	svc := NewStarService(&mockStarRepo{rewardByIDErr: errors.New("not found")})
	_, err := svc.UpdateReward("bad-id", map[string]interface{}{"title": "New Title"})
	if err == nil {
		t.Fatal("Expected error when reward not found, got nil")
	}
}

func TestStarService_UpdateReward_Success(t *testing.T) {
	repo := &mockStarRepo{
		rewardByID: &models.Reward{ID: "r1", Title: "Old Title", StarsRequired: 50, IsActive: true},
	}
	svc := NewStarService(repo)
	reward, err := svc.UpdateReward("r1", map[string]interface{}{
		"title":         "New Title",
		"starsRequired": float64(75),
	})
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if reward.Title != "New Title" {
		t.Errorf("Expected title 'New Title', got: %s", reward.Title)
	}
	if reward.StarsRequired != 75 {
		t.Errorf("Expected starsRequired 75, got: %d", reward.StarsRequired)
	}
}
