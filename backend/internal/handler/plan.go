package handler

import (
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) GetPlans(c *fiber.Ctx) error {
	plans, err := h.planService.GetActivePlans(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get plans",
		})
	}

	return c.JSON(fiber.Map{
		"plans": plans,
	})
}
