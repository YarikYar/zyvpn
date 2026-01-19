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
	PlanID   string `json:"plan_id"`
	Provider string `json:"provider"`
}

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

	payment, err := h.paymentSvc.CreatePayment(c.Context(), userID, planID, provider)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to create payment: " + err.Error(),
		})
	}

	// Return payment info based on provider
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

	// Stars payment - would create Telegram invoice
	return c.JSON(fiber.Map{
		"payment": payment,
	})
}

func (h *Handler) GetSubscriptionKey(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Необходима авторизация",
		})
	}

	key, err := h.subscriptionSvc.GetConnectionKey(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Нет активной подписки",
		})
	}

	return c.JSON(fiber.Map{
		"key": key,
	})
}

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

	// Sync traffic
	_ = h.subscriptionSvc.SyncTraffic(c.Context(), sub.ID)
	sub, _ = h.subscriptionSvc.GetSubscription(c.Context(), sub.ID)

	return c.JSON(fiber.Map{
		"active":         sub.IsActive(),
		"subscription":   sub,
		"days_remaining": sub.DaysRemaining(),
		"traffic_gb": fiber.Map{
			"used":      float64(sub.TrafficUsed) / (1024 * 1024 * 1024),
			"limit":     float64(sub.TrafficLimit) / (1024 * 1024 * 1024),
			"remaining": sub.RemainingTrafficGB(),
		},
	})
}

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
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success":      true,
		"subscription": sub,
		"key":          sub.ConnectionKey,
	})
}
