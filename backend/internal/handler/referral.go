package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/zyvpn/backend/internal/middleware"
)

type ApplyReferralRequest struct {
	Code string `json:"code"`
}

func (h *Handler) GetReferralStats(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Необходима авторизация",
		})
	}

	stats, err := h.referralSvc.GetReferralStats(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Не удалось получить статистику",
		})
	}

	return c.JSON(stats)
}

func (h *Handler) GetReferralLink(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Необходима авторизация",
		})
	}

	// Get bot username from config or hardcode
	botUsername := "zyvpn_bot" // This should come from config

	link, err := h.referralSvc.GetReferralLink(c.Context(), userID, botUsername)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Не удалось получить ссылку",
		})
	}

	// Get user's referral code
	user, err := h.userService.GetUser(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Не удалось получить пользователя",
		})
	}

	return c.JSON(fiber.Map{
		"link": link,
		"code": user.ReferralCode,
	})
}

func (h *Handler) ApplyReferralCode(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Необходима авторизация",
		})
	}

	var req ApplyReferralRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Неверный формат запроса",
		})
	}

	if req.Code == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Введите код",
		})
	}

	if err := h.referralSvc.ApplyReferralCode(c.Context(), userID, req.Code); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Реферальный код применён! Бонусные дни начислятся после первой оплаты.",
	})
}

func (h *Handler) GetReferredUsers(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Необходима авторизация",
		})
	}

	users, err := h.referralSvc.GetReferredUsers(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Не удалось получить список пользователей",
		})
	}

	return c.JSON(fiber.Map{
		"users": users,
	})
}
