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
	PlanID   string  `json:"plan_id"`
	ServerID *string `json:"server_id,omitempty"`
	Provider string  `json:"provider"`
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

	// Parse optional server ID
	var serverID *uuid.UUID
	if req.ServerID != nil && *req.ServerID != "" {
		sid, err := uuid.Parse(*req.ServerID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Неверный ID сервера",
			})
		}
		serverID = &sid
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

	payment, err := h.paymentSvc.CreatePaymentWithServer(c.Context(), userID, planID, serverID, provider)
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

type SwitchServerRequest struct {
	ServerID string `json:"server_id"`
}

// GetSwitchServerInfo returns info about region switching (price and free switches)
func (h *Handler) GetSwitchServerInfo(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Необходима авторизация",
		})
	}

	// Get user for free switches count
	user, err := h.userService.GetUser(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Не удалось получить данные пользователя",
		})
	}

	// Get region switch price
	price, err := h.adminSvc.GetRegionSwitchPrice(c.Context())
	if err != nil {
		price = 0.1 // Default fallback
	}

	return c.JSON(fiber.Map{
		"price":         price,
		"free_switches": user.FreeRegionSwitches,
	})
}

// SwitchServer switches the active subscription to a different server
func (h *Handler) SwitchServer(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Необходима авторизация",
		})
	}

	var req SwitchServerRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Неверный формат запроса",
		})
	}

	serverID, err := uuid.Parse(req.ServerID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Неверный ID сервера",
		})
	}

	// Check if user has free switches
	usedFree, err := h.userService.UseFreeRegionSwitch(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Ошибка при проверке бесплатных переключений",
		})
	}

	// If no free switch available, charge from balance
	if !usedFree {
		// Get price
		price, err := h.adminSvc.GetRegionSwitchPrice(c.Context())
		if err != nil {
			price = 0.1
		}

		// Try to charge from balance
		_, err = h.balanceSvc.ChargeRegionSwitch(c.Context(), userID, price)
		if err != nil {
			if errors.Is(err, service.ErrInsufficientBalance) {
				return c.Status(fiber.StatusPaymentRequired).JSON(fiber.Map{
					"error":          "Недостаточно средств на балансе",
					"price":          price,
					"need_more":      true,
				})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
	}

	sub, err := h.subscriptionSvc.SwitchServer(c.Context(), userID, serverID)
	if err != nil {
		if errors.Is(err, service.ErrSubscriptionNotActive) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": "Нет активной подписки",
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
		"used_free":    usedFree,
	})
}
