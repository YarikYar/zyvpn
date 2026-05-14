package handler

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"

	"github.com/zyvpn/backend/internal/repository"
	"github.com/zyvpn/backend/internal/service"
)

// GetSubscriptionPublic — публичный endpoint для VPN-клиентов
// (v2rayNG, Hiddify, Streisand, Shadowrocket и т.п.). Без авторизации —
// токен в URL и есть авторизация.
//
// Возвращает base64-encoded plaintext списка VLESS-ссылок + стандартные
// заголовки, которые читают клиенты:
//   - Subscription-Userinfo: upload/download/total/expire
//   - Profile-Update-Interval: часов до автоматического обновления
//   - Profile-Title: отображаемое имя подписки в клиенте
//
//	@Summary	Subscription content (для VPN-клиентов)
//	@Tags		public
//	@Produce	plain
//	@Param		token	path	string	true	"subscription token"
//	@Success	200	{string}	string	"base64-encoded share links"
//	@Failure	404	{object}	map[string]string
//	@Router		/sub/{token} [get]
func (h *Handler) GetSubscriptionPublic(c *fiber.Ctx) error {
	token := c.Params("token")
	if token == "" {
		return c.Status(fiber.StatusNotFound).SendString("not found")
	}

	content, err := h.subscriptionSvc.BuildSubscriptionContent(c.Context(), token)
	if err != nil {
		if errors.Is(err, repository.ErrSubscriptionNotFound) ||
			errors.Is(err, service.ErrSubscriptionNotActive) {
			return c.Status(fiber.StatusNotFound).SendString("not found")
		}
		return c.Status(fiber.StatusInternalServerError).SendString("error")
	}

	c.Set("Content-Type", content.ContentTypeHint)
	c.Set("Subscription-Userinfo", content.UserInfoHeader)
	c.Set("Profile-Update-Interval", strconv.Itoa(content.UpdateInterval))
	c.Set("Profile-Title", content.ProfileTitle)
	// inline-имя файла для клиентов которые умеют его читать как
	// отображаемое имя подписки.
	c.Set("Content-Disposition", `inline; filename="corevpn"`)
	return c.Send(content.Body)
}
