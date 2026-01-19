package handler

import (
	"fmt"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/zyvpn/backend/internal/middleware"
	"github.com/zyvpn/backend/internal/model"
)

type TopUpRequest struct {
	Amount   float64             `json:"amount"`   // Amount in TON
	Provider model.PaymentProvider `json:"provider"` // ton or stars
}

// GetBalance returns user's current balance
func (h *Handler) GetBalance(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Необходима авторизация",
		})
	}

	balance, err := h.balanceSvc.GetBalance(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Не удалось получить баланс",
		})
	}

	return c.JSON(fiber.Map{
		"balance":  balance,
		"currency": "TON",
	})
}

// GetBalanceTransactions returns balance history
func (h *Handler) GetBalanceTransactions(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Необходима авторизация",
		})
	}

	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))

	transactions, err := h.balanceSvc.GetTransactions(c.Context(), userID, limit, offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Не удалось получить историю операций",
		})
	}

	return c.JSON(fiber.Map{
		"transactions": transactions,
	})
}

type PayFromBalanceRequest struct {
	PlanID string `json:"plan_id"`
}

// PayFromBalance pays for subscription using balance
func (h *Handler) PayFromBalance(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Необходима авторизация",
		})
	}

	var req PayFromBalanceRequest
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

	// Get plan
	plan, err := h.planService.GetPlan(c.Context(), planID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Тариф не найден",
		})
	}

	// Check if user can afford
	canAfford, err := h.balanceSvc.CanAfford(c.Context(), userID, plan.PriceTON)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Не удалось проверить баланс",
		})
	}

	if !canAfford {
		balance, _ := h.balanceSvc.GetBalance(c.Context(), userID)
		return c.Status(fiber.StatusPaymentRequired).JSON(fiber.Map{
			"error":    "Недостаточно средств",
			"balance":  balance,
			"required": plan.PriceTON,
		})
	}

	// Create payment record
	payment, err := h.paymentSvc.CreatePayment(c.Context(), userID, plan.ID, model.PaymentProviderBalance)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Не удалось создать платёж",
		})
	}

	// Debit balance
	newBalance, err := h.balanceSvc.DebitForSubscription(c.Context(), userID, plan.PriceTON, payment.ID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to debit balance: " + err.Error(),
		})
	}

	// Complete payment and create/extend subscription
	if err := h.paymentSvc.CompletePayment(c.Context(), payment.ID); err != nil {
		// Refund on failure
		h.balanceSvc.CreditRefund(c.Context(), userID, plan.PriceTON, payment.ID)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to complete payment: " + err.Error(),
		})
	}

	// Get subscription key
	key, _ := h.subscriptionSvc.GetConnectionKey(c.Context(), userID)

	// Notify via bot
	if h.bot != nil {
		sub, _ := h.subscriptionSvc.GetActiveSubscription(c.Context(), userID)
		if sub != nil {
			_ = h.bot.SendSubscriptionActivated(userID, sub.ExpiresAt.Format("02.01.2006"))
		}
	}

	return c.JSON(fiber.Map{
		"success":     true,
		"new_balance": newBalance,
		"key":         key,
	})
}

// InitTopUp creates a payment for balance top-up
func (h *Handler) InitTopUp(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Необходима авторизация",
		})
	}

	var req TopUpRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Неверный формат запроса",
		})
	}

	if req.Amount <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Сумма должна быть положительной",
		})
	}

	if req.Provider != model.PaymentProviderTON && req.Provider != model.PaymentProviderStars {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Способ оплаты должен быть 'ton' или 'stars'",
		})
	}

	payment, err := h.paymentSvc.CreateTopUpPayment(c.Context(), userID, req.Amount, req.Provider)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to create payment: " + err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"payment_id": payment.ID,
		"amount":     payment.Amount,
		"currency":   payment.Currency,
		"provider":   payment.Provider,
	})
}

// GetTopUpTONInfo returns TON payment info for balance top-up
func (h *Handler) GetTopUpTONInfo(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Необходима авторизация",
		})
	}

	paymentIDStr := c.Query("payment_id")
	if paymentIDStr == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Отсутствует ID платежа",
		})
	}

	paymentID, err := uuid.Parse(paymentIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Неверный ID платежа",
		})
	}

	tonInfo, err := h.paymentSvc.GetTONTopUpInfo(c.Context(), paymentID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(tonInfo)
}

type VerifyTopUpRequest struct {
	PaymentID string `json:"payment_id"`
	TxHash    string `json:"tx_hash"`
}

// InitTopUpStars creates a Stars invoice for balance top-up
func (h *Handler) InitTopUpStars(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Необходима авторизация",
		})
	}

	paymentIDStr := c.Query("payment_id")
	if paymentIDStr == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Отсутствует ID платежа",
		})
	}

	paymentID, err := uuid.Parse(paymentIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Неверный ID платежа",
		})
	}

	payment, err := h.paymentSvc.GetPayment(c.Context(), paymentID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Платёж не найден",
		})
	}

	if payment.PaymentType != model.PaymentTypeTopUp {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Платёж не является пополнением",
		})
	}

	if h.bot == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "Сервис оплаты недоступен",
		})
	}

	// Calculate TON amount for description
	tonAmount := payment.Amount
	if payment.Currency == "XTR" {
		tonAmount = payment.Amount / 100 // Convert Stars to TON
	}

	// Create invoice via bot
	invoiceLink, err := h.bot.CreateStarsInvoice(
		userID,
		"Пополнение баланса",
		fmt.Sprintf("Пополнение баланса на %.4f TON", tonAmount),
		int(payment.Amount),
		payment.ID.String(),
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to create invoice: " + err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"payment_id":   payment.ID,
		"amount":       int(payment.Amount),
		"currency":     "XTR",
		"invoice_link": invoiceLink,
	})
}

// VerifyTopUp verifies TON transaction and credits balance
func (h *Handler) VerifyTopUp(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Необходима авторизация",
		})
	}

	var req VerifyTopUpRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Неверный формат запроса",
		})
	}

	paymentID, err := uuid.Parse(req.PaymentID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Неверный ID платежа",
		})
	}

	if err := h.paymentSvc.CompleteTopUpPayment(c.Context(), paymentID, req.TxHash); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Get updated balance
	balance, _ := h.balanceSvc.GetBalance(c.Context(), userID)

	return c.JSON(fiber.Map{
		"success":     true,
		"new_balance": balance,
	})
}
