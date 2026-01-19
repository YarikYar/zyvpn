package handler

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/zyvpn/backend/internal/middleware"
	"github.com/zyvpn/backend/internal/service"
)

type ApplyPromoCodeRequest struct {
	Code string `json:"code"`
}

// ApplyPromoCode applies a promo code for the current user
func (h *Handler) ApplyPromoCode(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Необходима авторизация",
		})
	}

	var req ApplyPromoCodeRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Неверный формат запроса",
		})
	}

	if req.Code == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Введите промокод",
		})
	}

	result, err := h.promoCodeSvc.ApplyPromoCode(c.Context(), req.Code, userID)
	if err != nil {
		status := fiber.StatusBadRequest
		if errors.Is(err, service.ErrPromoCodeNotFound) {
			status = fiber.StatusNotFound
		}
		return c.Status(status).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success":     true,
		"type":        result.Type,
		"value":       result.Value,
		"new_balance": result.NewBalance,
		"message":     result.Message,
	})
}

// ValidatePromoCode checks if a promo code is valid (without applying it)
func (h *Handler) ValidatePromoCode(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Необходима авторизация",
		})
	}

	code := c.Query("code")
	if code == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Введите промокод",
		})
	}

	promo, err := h.promoCodeSvc.ValidatePromoCode(c.Context(), code, userID)
	if err != nil {
		status := fiber.StatusBadRequest
		if errors.Is(err, service.ErrPromoCodeNotFound) {
			status = fiber.StatusNotFound
		}
		return c.Status(status).JSON(fiber.Map{
			"error": err.Error(),
			"valid": false,
		})
	}

	return c.JSON(fiber.Map{
		"valid":       true,
		"type":        promo.Type,
		"value":       promo.Value,
		"description": promo.Description,
	})
}
