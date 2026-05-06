package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/zyvpn/backend/internal/middleware"
)

// GetPlans returns plans visible to the calling user. Public plans show up
// for everyone; plans with visible_to_referrer_id are filtered to users
// referred by that referrer.
//
//	@Summary	Plans visible to caller
//	@Tags		subscription
//	@Produce	json
//	@Success	200	{array}		model.Plan
//	@Failure	500	{object}	map[string]string
//	@Router		/api/plans [get]
//	@Security	TelegramInitData
func (h *Handler) GetPlans(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	plans, err := h.planService.GetActivePlansForUser(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get plans",
		})
	}

	return c.JSON(fiber.Map{
		"plans": plans,
	})
}
