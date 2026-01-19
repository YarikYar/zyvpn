package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/zyvpn/backend/internal/middleware"
	"github.com/zyvpn/backend/internal/service"
)

func (h *Handler) GetMe(c *fiber.Ctx) error {
	telegramUser := middleware.GetTelegramUser(c)
	if telegramUser == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Необходима авторизация",
		})
	}

	// Get or create user
	user, _, err := h.userService.GetOrCreateUser(c.Context(), service.TelegramUser{
		ID:           telegramUser.UserID,
		Username:     &telegramUser.Username,
		FirstName:    &telegramUser.FirstName,
		LastName:     &telegramUser.LastName,
		LanguageCode: &telegramUser.LanguageCode,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Не удалось получить пользователя",
		})
	}

	// Get user with subscription
	userWithSub, err := h.userService.GetUserWithSubscription(c.Context(), user.ID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Не удалось получить данные пользователя",
		})
	}

	return c.JSON(userWithSub)
}
