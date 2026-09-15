package handlers

import (
	"refer_api/middleware"
	"refer_api/services"

	"github.com/gofiber/fiber/v2"
)

// StarHandler coordinates HTTP requests for stars and perks endpoints.
type StarHandler struct {
	starService services.StarService
}

// NewStarHandler constructs a new StarHandler.
func NewStarHandler(starService services.StarService) *StarHandler {
	return &StarHandler{starService: starService}
}

// GetBalance returns the user's star count.
func (h *StarHandler) GetBalance(c *fiber.Ctx) error {
	userID := c.Locals("userId").(string)
	balance, err := h.starService.GetBalance(userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": err.Error()})
	}
	return c.JSON(fiber.Map{"stars": balance})
}

// GetHistory returns user star earnings and redemptions history.
func (h *StarHandler) GetHistory(c *fiber.Ctx) error {
	userID := c.Locals("userId").(string)
	history, err := h.starService.GetHistory(userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": err.Error()})
	}
	return c.JSON(history)
}

// GetRewards returns all active rewards.
func (h *StarHandler) GetRewards(c *fiber.Ctx) error {
	rewards, err := h.starService.GetRewards()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": err.Error()})
	}
	return c.JSON(rewards)
}

// RedeemReward claims a reward for the user.
func (h *StarHandler) RedeemReward(c *fiber.Ctx) error {
	userID := c.Locals("userId").(string)

	var body struct {
		RewardID string `json:"rewardId"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid request body"})
	}

	result, err := h.starService.RedeemReward(userID, body.RewardID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": err.Error()})
	}
	middleware.AttachAudit(c, "reward.redeem", "reward", body.RewardID, nil)

	return c.Status(fiber.StatusCreated).JSON(result)
}

// GetRedeemedRewards returns history of claimed rewards.
func (h *StarHandler) GetRedeemedRewards(c *fiber.Ctx) error {
	userID := c.Locals("userId").(string)
	rrs, err := h.starService.GetRedeemedRewards(userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": err.Error()})
	}
	return c.JSON(rrs)
}

// GetPerks returns active store perks.
func (h *StarHandler) GetPerks(c *fiber.Ctx) error {
	perks := h.starService.GetPerks()
	return c.JSON(perks)
}

// GetAllTransactions returns all transactions (Admin only).
func (h *StarHandler) GetAllTransactions(c *fiber.Ctx) error {
	txs, err := h.starService.GetAllTransactions()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": err.Error()})
	}
	return c.JSON(txs)
}

// CreateReward inserts a reward manually (Admin only).
func (h *StarHandler) CreateReward(c *fiber.Ctx) error {
	var body map[string]interface{}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid request body"})
	}

	reward, err := h.starService.CreateReward(body)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": err.Error()})
	}
	middleware.AttachAudit(c, "reward.create", "reward", reward.ID, map[string]interface{}{"title": body["title"]})
	return c.Status(fiber.StatusCreated).JSON(reward)
}

// UpdateReward modifies reward attributes (Admin only).
func (h *StarHandler) UpdateReward(c *fiber.Ctx) error {
	id := c.Params("id")
	var body map[string]interface{}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid request body"})
	}

	reward, err := h.starService.UpdateReward(id, body)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": err.Error()})
	}
	middleware.AttachAudit(c, "reward.update", "reward", id, nil)
	return c.JSON(reward)
}

// DeleteReward removes a reward by ID (Admin only).
func (h *StarHandler) DeleteReward(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := h.starService.DeleteReward(id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": err.Error()})
	}
	middleware.AttachAudit(c, "reward.delete", "reward", id, nil)
	return c.JSON(fiber.Map{"success": true})
}
