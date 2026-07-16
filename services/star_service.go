package services

import (
	"errors"
	"time"

	"refer_api/models"
	"refer_api/repository"

	"github.com/google/uuid"
)

// StarService defines the business logic interface for managing star balances, history, rewards, and perks.
type StarService interface {
	GetBalance(userID string) (int, error)
	GetHistory(userID string) ([]models.StarTransaction, error)
	GetRewards() ([]models.Reward, error)
	RedeemReward(userID, rewardID string) (map[string]interface{}, error)
	GetRedeemedRewards(userID string) ([]models.RedeemedReward, error)
	GetPerks() []map[string]string
	GetAllTransactions() ([]models.StarTransaction, error)
	CreateReward(data map[string]interface{}) (*models.Reward, error)
	UpdateReward(id string, data map[string]interface{}) (*models.Reward, error)
	DeleteReward(id string) error
}

type starService struct {
	starRepo repository.StarRepository
}

// NewStarService constructs a new StarService.
func NewStarService(starRepo repository.StarRepository) StarService {
	return &starService{starRepo: starRepo}
}

func (s *starService) GetBalance(userID string) (int, error) {
	return s.starRepo.GetStarBalance(userID)
}

func (s *starService) GetHistory(userID string) ([]models.StarTransaction, error) {
	txs, err := s.starRepo.FindHistory(userID)
	if err != nil {
		return nil, err
	}
	if txs == nil {
		txs = []models.StarTransaction{}
	}
	return txs, nil
}

func (s *starService) GetRewards() ([]models.Reward, error) {
	rewards, err := s.starRepo.FindRewards()
	if err != nil {
		return nil, err
	}
	if rewards == nil {
		rewards = []models.Reward{}
	}
	return rewards, nil
}

func (s *starService) RedeemReward(userID, rewardID string) (map[string]interface{}, error) {
	currentStars, err := s.GetBalance(userID)
	if err != nil {
		return nil, err
	}

	reward, err := s.starRepo.FindRewardByID(rewardID)
	if err != nil {
		return nil, errors.New("reward not found")
	}

	if !reward.IsActive {
		return nil, errors.New("reward not found")
	}

	if currentStars < reward.StarsRequired {
		return nil, errors.New("not enough stars")
	}

	// 1. Create history transaction record
	transaction := &models.StarTransaction{
		ID:     uuid.New().String(),
		UserID: userID,
		Type:   "redeem",
		Title:  reward.Title,
		Stars:  -reward.StarsRequired,
		Date:   time.Now(),
	}
	if err := s.starRepo.CreateTransaction(transaction); err != nil {
		return nil, err
	}

	// 2. Create the redeemed reward claim record
	redeemedReward := &models.RedeemedReward{
		ID:          uuid.New().String(),
		UserID:      userID,
		RewardID:    reward.ID,
		RewardTitle: reward.Title,
		IsClaimed:   false,
		CreatedAt:   time.Now(),
	}
	if err := s.starRepo.CreateRedeemedReward(redeemedReward); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"success":        true,
		"message":        "Redeemed " + reward.Title,
		"remainingStars": currentStars - reward.StarsRequired,
		"redeemedReward": redeemedReward,
	}, nil
}

func (s *starService) GetRedeemedRewards(userID string) ([]models.RedeemedReward, error) {
	rrs, err := s.starRepo.FindRedeemedRewards(userID)
	if err != nil {
		return nil, err
	}
	if rrs == nil {
		rrs = []models.RedeemedReward{}
	}
	return rrs, nil
}

func (s *starService) GetPerks() []map[string]string {
	return []map[string]string{
		{
			"id":          "birthday",
			"title":       "Birthday Treat",
			"description": "Get a free drink on your birthday!",
			"status":      "active",
		},
		{
			"id":          "double-stars",
			"title":       "Double Star Day",
			"description": "Earn 2x stars every Tuesday.",
			"status":      "active",
		},
	}
}

func (s *starService) GetAllTransactions() ([]models.StarTransaction, error) {
	return s.starRepo.GetAllTransactions()
}

func (s *starService) CreateReward(data map[string]interface{}) (*models.Reward, error) {
	title, _ := data["title"].(string)
	description, _ := data["description"].(string)
	starsRequiredFloat, _ := data["starsRequired"].(float64)
	starsRequired := int(starsRequiredFloat)
	imageUrl, _ := data["imageUrl"].(string)
	icon, _ := data["icon"].(string)
	productId, _ := data["productId"].(string)
	isActive, ok := data["isActive"].(bool)
	if !ok {
		isActive = true
	}

	reward := &models.Reward{
		ID:            uuid.New().String(),
		Title:         title,
		Description:   description,
		StarsRequired: starsRequired,
		IsActive:      isActive,
	}

	if imageUrl != "" {
		reward.ImageURL = &imageUrl
	}
	if icon != "" {
		reward.Icon = &icon
	}
	if productId != "" {
		reward.ProductID = &productId
	}

	if err := s.starRepo.CreateReward(reward); err != nil {
		return nil, err
	}
	return reward, nil
}

func (s *starService) UpdateReward(id string, data map[string]interface{}) (*models.Reward, error) {
	reward, err := s.starRepo.FindRewardByID(id)
	if err != nil {
		return nil, err
	}

	if title, ok := data["title"].(string); ok {
		reward.Title = title
	}
	if description, ok := data["description"].(string); ok {
		reward.Description = description
	}
	if starsRequired, ok := data["starsRequired"].(float64); ok {
		reward.StarsRequired = int(starsRequired)
	}
	if imageUrl, ok := data["imageUrl"].(string); ok {
		reward.ImageURL = &imageUrl
	}
	if icon, ok := data["icon"].(string); ok {
		reward.Icon = &icon
	}
	if productId, ok := data["productId"].(string); ok {
		reward.ProductID = &productId
	}
	if isActive, ok := data["isActive"].(bool); ok {
		reward.IsActive = isActive
	}

	if err := s.starRepo.UpdateReward(reward); err != nil {
		return nil, err
	}
	return reward, nil
}

func (s *starService) DeleteReward(id string) error {
	return s.starRepo.DeleteReward(id)
}
