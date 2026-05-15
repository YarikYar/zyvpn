package handler

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/zyvpn/backend/internal/middleware"
	"github.com/zyvpn/backend/internal/model"
	"github.com/zyvpn/backend/internal/service"
)

type BuySubscriptionRequest struct {
	PlanID   string `json:"plan_id" example:"a3b1f8a2-..."`
	Provider string `json:"provider" enums:"ton,stars,cash"`
}

// BuySubscription initiates plan purchase. Provider determines response
// shape: ton returns TON deep link; stars returns payment with id; cash
// creates a pending payment and notifies admins.
//
//	@Summary	Initiate plan purchase
//	@Tags		subscription
//	@Accept		json
//	@Produce	json
//	@Param		body	body		BuySubscriptionRequest	true	"plan and provider"
//	@Success	200		{object}	map[string]interface{}
//	@Failure	400		{object}	map[string]string
//	@Failure	500		{object}	map[string]string
//	@Router		/api/subscription/buy [post]
//	@Security	TelegramInitData
func (h *Handler) BuySubscription(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Необходима авторизация",
		})
	}

	var req BuySubscriptionRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Неверный формат запроса",
		})
	}

	planID, err := uuid.Parse(req.PlanID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Неверный ID тарифа",
		})
	}

	var provider model.PaymentProvider
	switch req.Provider {
	case "ton":
		provider = model.PaymentProviderTON
	case "stars":
		provider = model.PaymentProviderStars
	default:
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Неверный способ оплаты, выберите 'ton' или 'stars'",
		})
	}

	// Серверы определяются тарифом — server_id больше не выбирается юзером.
	payment, err := h.paymentSvc.CreatePaymentWithServer(c.Context(), userID, planID, nil, provider)
	if err != nil {
		return respondInternalError(c, err)
	}

	if provider == model.PaymentProviderTON {
		tonInfo, err := h.paymentSvc.GetTONPaymentInfo(c.Context(), payment.ID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Не удалось получить информацию о платеже",
			})
		}
		return c.JSON(fiber.Map{
			"payment":  payment,
			"ton_info": tonInfo,
		})
	}

	return c.JSON(fiber.Map{
		"payment": payment,
	})
}

// GetSubscriptionURL returns the subscription URL for the user's active
// subscription. UI кладёт это в QR + копи-кнопку.
//
//	@Summary	Subscription URL (для VPN-клиента)
//	@Tags		subscription
//	@Produce	json
//	@Success	200	{object}	map[string]string
//	@Failure	404	{object}	map[string]string
//	@Router		/api/subscription/url [get]
//	@Security	TelegramInitData
func (h *Handler) GetSubscriptionURL(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Необходима авторизация",
		})
	}

	subURL, err := h.subscriptionSvc.GetSubscriptionURLForUser(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Нет активной подписки",
		})
	}

	return c.JSON(fiber.Map{
		"subscription_url": subURL,
	})
}

// GetSubscriptionStatus returns active subscription status including
// remaining days, traffic and the list of accessible servers.
//
//	@Summary	Subscription status
//	@Tags		subscription
//	@Produce	json
//	@Success	200	{object}	map[string]interface{}
//	@Router		/api/subscription/status [get]
//	@Security	TelegramInitData
func (h *Handler) GetSubscriptionStatus(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Необходима авторизация",
		})
	}

	sub, err := h.subscriptionSvc.GetActiveSubscription(c.Context(), userID)
	if err != nil {
		return c.JSON(fiber.Map{
			"active": false,
		})
	}

	// Sync traffic перед ответом — чтобы фронт видел свежее used.
	// Это тяжёлый вызов (N походов в xui), HealthWorker должен закрыть
	// эту задачу фоном; здесь оставлено как «горячий» путь когда юзер
	// открывает экран.
	_ = h.subscriptionSvc.SyncTraffic(c.Context(), sub.ID)
	sub, _ = h.subscriptionSvc.GetSubscription(c.Context(), sub.ID)

	// Имя тарифа — best-effort.
	planName := ""
	if plan, perr := h.planService.GetPlan(c.Context(), sub.PlanID); perr == nil && plan != nil {
		planName = plan.Name
	}

	// Flatten servers: фронт ждёт ServerEntry с полями вверху, не вложенный
	// SubscriptionClient.server.
	servers := make([]fiber.Map, 0, len(sub.Clients))
	for _, c := range sub.Clients {
		entry := fiber.Map{
			"id":             c.ServerID,
			"connection_key": c.ConnectionKey,
		}
		if c.Server != nil {
			entry["name"] = c.Server.Name
			entry["country"] = c.Server.Country
			if c.Server.City != nil {
				entry["city"] = *c.Server.City
			}
			entry["flag"] = c.Server.FlagEmoji
			entry["status"] = c.Server.Status
			if c.Server.PingMs != nil {
				entry["ping_ms"] = *c.Server.PingMs
			}
		}
		servers = append(servers, entry)
	}

	trafficLimitGB := float64(sub.TrafficLimit) / (1024 * 1024 * 1024)
	var limit any = trafficLimitGB
	if sub.TrafficLimit <= 0 {
		limit = nil // безлимит
	}

	return c.JSON(fiber.Map{
		"active": sub.IsActive(),
		"subscription": fiber.Map{
			"plan_id":          sub.PlanID,
			"plan_name":        planName,
			"started_at":       sub.StartedAt,
			"ends_at":          sub.ExpiresAt,
			"auto_renew":       false,
			"subscription_url": h.subscriptionSvc.BuildSubscriptionURL(sub),
			"servers":          servers,
		},
		"days_remaining": sub.DaysRemaining(),
		"traffic_gb": fiber.Map{
			"used":  float64(sub.TrafficUsed) / (1024 * 1024 * 1024),
			"limit": limit,
		},
	})
}

// ActivateTrial enables the free trial subscription for the user.
//
//	@Summary	Activate free trial
//	@Tags		subscription
//	@Produce	json
//	@Success	200	{object}	map[string]interface{}
//	@Failure	409	{object}	map[string]string	"trial already used or active sub exists"
//	@Failure	500	{object}	map[string]string
//	@Router		/api/subscription/trial [post]
//	@Security	TelegramInitData
func (h *Handler) ActivateTrial(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Необходима авторизация",
		})
	}

	sub, err := h.subscriptionSvc.ActivateTrial(c.Context(), userID)
	if err != nil {
		if errors.Is(err, service.ErrTrialAlreadyUsed) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": "Пробный период уже использован",
			})
		}
		if errors.Is(err, service.ErrSubscriptionActive) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": "У вас уже есть активная подписка",
			})
		}
		return respondInternalError(c, err)
	}

	return c.JSON(fiber.Map{
		"success":          true,
		"subscription":     sub,
		"subscription_url": h.subscriptionSvc.BuildSubscriptionURL(sub),
	})
}
