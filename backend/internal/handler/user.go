package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/zyvpn/backend/internal/middleware"
	"github.com/zyvpn/backend/internal/service"
)

// GetMe returns the authenticated user with their active subscription
// embedded.
//
//	@Summary		Current authenticated user
//	@Tags			user
//	@Produce		json
//	@Success		200	{object}	model.UserWithSubscription
//	@Failure		401	{object}	map[string]string
//	@Router			/api/user/me [get]
//	@Security		TelegramInitData
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

	isAdmin, err := h.adminSvc.IsAdmin(c.Context(), user.ID)
	if err == nil {
		userWithSub.IsAdmin = isAdmin
	}

	return c.JSON(userWithSub)
}
