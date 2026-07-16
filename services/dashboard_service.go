package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"refer_api/models"
	"refer_api/repository"

	"github.com/uptrace/bun"
)

// DashboardService defines business logic for home summary views.
type DashboardService interface {
	GetSummary(userID string) (*models.UserSummaryResponse, error)
	GetAdminSummary() (*models.AdminSummaryResponse, error)
}

type dashboardService struct {
	db       *bun.DB
	userRepo repository.UserRepository
	starRepo repository.StarRepository
}

// NewDashboardService constructs a new DashboardService.
func NewDashboardService(db *bun.DB, userRepo repository.UserRepository, starRepo repository.StarRepository) DashboardService {
	return &dashboardService{
		db:       db,
		userRepo: userRepo,
		starRepo: starRepo,
	}
}

func (s *dashboardService) GetSummary(userID string) (*models.UserSummaryResponse, error) {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return nil, err
	}

	stars, err := s.starRepo.GetStarBalance(userID)
	if err != nil {
		stars = 0
	}

	rewards, err := s.starRepo.FindRewards()
	if err != nil {
		rewards = []models.Reward{}
	}

	history, err := s.starRepo.FindHistory(userID)
	if err != nil {
		history = []models.StarTransaction{}
	}

	recentActivity := []models.ActivitySummaryDto{}
	limit := 5
	if len(history) < 5 {
		limit = len(history)
	}
	for i := 0; i < limit; i++ {
		h := history[i]
		recentActivity = append(recentActivity, models.ActivitySummaryDto{
			ID:    h.ID,
			Type:  h.Type,
			Title: h.Title,
			Stars: h.Stars,
			Date:  h.Date.Format(time.RFC3339),
		})
	}

	nextRewardProgress := float64(stars) / 100.0
	if nextRewardProgress > 1.0 {
		nextRewardProgress = 1.0
	}

	return &models.UserSummaryResponse{
		User: models.UserSummaryUserDto{
			Name:     user.Name,
			PhotoURL: user.PhotoURL,
			Stars:    stars,
		},
		Stars:              stars,
		Rewards:            rewards,
		RecentActivity:     recentActivity,
		NextRewardProgress: nextRewardProgress,
	}, nil
}

func (s *dashboardService) GetAdminSummary() (*models.AdminSummaryResponse, error) {
	users, err := s.userRepo.FindAll()
	if err != nil {
		return nil, err
	}

	var orders []models.Order
	if err := s.db.NewSelect().Model(&orders).
		Relation("User").
		OrderExpr("?TableAlias.createdAt DESC").
		Scan(context.Background()); err != nil {
		return nil, err
	}

	totalStarsAwarded, err := s.starRepo.GetTotalStarsAwarded()
	if err != nil {
		totalStarsAwarded = 0
	}

	totalStarsRedeemed, err := s.starRepo.GetTotalStarsRedeemed()
	if err != nil {
		totalStarsRedeemed = 0
	}

	totalOrders := len(orders)
	totalUsers := len(users)

	var starsAwardedStr string
	if totalStarsAwarded > 1000 {
		starsAwardedStr = fmt.Sprintf("%.1fK", float64(totalStarsAwarded)/1000.0)
	} else {
		starsAwardedStr = fmt.Sprintf("%d", totalStarsAwarded)
	}

	stats := []models.AdminStatDto{
		{Label: "Usuarios Totales", Value: fmt.Sprintf("%d", totalUsers), Change: "+5%", Icon: "group", Color: "primary"},
		{Label: "Órdenes Totales", Value: fmt.Sprintf("%d", totalOrders), Change: "+12%", Icon: "shopping_cart", Color: "secondary"},
		{Label: "Estrellas Repartidas", Value: starsAwardedStr, Change: "+8%", Icon: "stars", Color: "accent"},
		{Label: "Puntos Canjeados", Value: fmt.Sprintf("%d", totalStarsRedeemed), Change: "+15%", Icon: "redeem", Color: "success"},
	}

	recentActivity := []models.AdminActivitySummary{}
	limit := 5
	if len(orders) < 5 {
		limit = len(orders)
	}
	for i := 0; i < limit; i++ {
		o := orders[i]
		userName := "Usuario"
		avatar := "U"
		if o.User != nil {
			userName = o.User.Name
			if len(userName) >= 2 {
				avatar = strings.ToUpper(userName[0:2])
			} else if len(userName) == 1 {
				avatar = strings.ToUpper(userName[0:1])
			}
		}

		recentActivity = append(recentActivity, models.AdminActivitySummary{
			User:   userName,
			Action: fmt.Sprintf("Realizó el pedido %s", o.OrderNumber),
			Time:   formatTime(o.CreatedAt),
			Avatar: avatar,
		})
	}

	return &models.AdminSummaryResponse{
		Stats:          stats,
		RecentActivity: recentActivity,
	}, nil
}

func formatTime(t time.Time) string {
	diff := time.Since(t)
	diffMin := int(diff.Minutes())

	if diffMin < 60 {
		return fmt.Sprintf("hace %d min", diffMin)
	}
	diffHrs := int(diff.Hours())
	if diffHrs < 24 {
		plural := ""
		if diffHrs > 1 {
			plural = "s"
		}
		return fmt.Sprintf("hace %d hora%s", diffHrs, plural)
	}
	return t.Format("2006-01-02")
}
