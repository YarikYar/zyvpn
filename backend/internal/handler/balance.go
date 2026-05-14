package handler

import (
	"fmt"
	"log"
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

// GetBalance returns current TON balance.
//
//	@Summary	Current balance
//	@Tags		balance
//	@Produce	json
//	@Success	200	{object}	map[string]interface{}
//	@Router		/api/balance [get]
//	@Security	TelegramInitData
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

// GetBalanceTransactions returns paginated balance history.
//
//	@Summary	Balance transactions
//	@Tags		balance
//	@Produce	json
//	@Param		limit	query		int	false	"page size"	default(20)
//	@Param		offset	query		int	false	"offset"	default(0)
//	@Success	200		{object}	map[string]interface{}
//	@Router		/api/balance/transactions [get]
//	@Security	TelegramInitData
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

// PayFromBalance buys a plan from existing TON balance.
//
//	@Summary	Pay from balance
//	@Tags		balance
//	@Accept		json
//	@Produce	json
//	@Param		body	body		PayFromBalanceRequest	true	"plan to buy"
//	@Success	200		{object}	map[string]interface{}
//	@Failure	400		{object}	map[string]string
//	@Failure	402		{object}	map[string]string	"insufficient balance"
//	@Router		/api/balance/pay [post]
//	@Security	TelegramInitData
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
		return respondInternalError(c, err)
	}

	// Complete payment and create/extend subscription
	if err := h.paymentSvc.CompletePayment(c.Context(), payment.ID); err != nil {
		// Refund on failure
		h.balanceSvc.CreditRefund(c.Context(), userID, plan.PriceTON, payment.ID)
		return respondInternalError(c, err)
	}

	// Get subscription URL (вместо одного connection_key теперь даём ссылку)
	subURL, _ := h.subscriptionSvc.GetSubscriptionURLForUser(c.Context(), userID)

	// Notify via bot
	if h.bot != nil {
		sub, _ := h.subscriptionSvc.GetActiveSubscription(c.Context(), userID)
		if sub != nil {
			_ = h.bot.SendSubscriptionActivated(userID, sub.ExpiresAt.Format("02.01.2006"))
		}
	}

	return c.JSON(fiber.Map{
		"success":          true,
		"new_balance":      newBalance,
		"subscription_url": subURL,
	})
}

// InitTopUp creates a pending top-up payment for the chosen provider.
//
//	@Summary	Init balance top-up
//	@Tags		balance
//	@Accept		json
//	@Produce	json
//	@Param		body	body		TopUpRequest	true	"amount + provider"
//	@Success	200		{object}	map[string]interface{}
//	@Failure	400		{object}	map[string]string
//	@Router		/api/balance/topup [post]
//	@Security	TelegramInitData
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
		return respondInternalError(c, err)
	}

	return c.JSON(fiber.Map{
		"payment_id": payment.ID,
		"amount":     payment.Amount,
		"currency":   payment.Currency,
		"provider":   payment.Provider,
	})
}

// GetTopUpTONInfo returns TON deep link for a top-up payment.
//
//	@Summary	TON deep link for top-up
//	@Tags		balance
//	@Produce	json
//	@Param		payment_id	query		string	true	"payment uuid"
//	@Success	200			{object}	model.TONPaymentInfo
//	@Failure	404			{object}	map[string]string
//	@Router		/api/balance/topup/ton [get]
//	@Security	TelegramInitData
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

// InitTopUpStars creates a Stars invoice for a top-up payment.
//
//	@Summary	Stars invoice for top-up
//	@Tags		balance
//	@Produce	json
//	@Param		payment_id	query		string	true	"payment uuid"
//	@Success	200			{object}	map[string]interface{}
//	@Failure	404			{object}	map[string]string
//	@Router		/api/balance/topup/stars [get]
//	@Security	TelegramInitData
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
		return respondInternalError(c, err)
	}

	return c.JSON(fiber.Map{
		"payment_id":   payment.ID,
		"amount":       int(payment.Amount),
		"currency":     "XTR",
		"invoice_link": invoiceLink,
	})
}

// VerifyTopUp verifies a TON top-up transaction and credits balance.
//
//	@Summary	Verify TON top-up
//	@Tags		balance
//	@Accept		json
//	@Produce	json
//	@Param		body	body		VerifyTopUpRequest	true	"payment id + BOC"
//	@Success	200		{object}	map[string]interface{}
//	@Failure	400		{object}	map[string]string
//	@Failure	403		{object}	map[string]string
//	@Failure	404		{object}	map[string]string
//	@Router		/api/balance/topup/verify [post]
//	@Security	TelegramInitData
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

	// Ownership check.
	payment, err := h.paymentSvc.GetPayment(c.Context(), paymentID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Платёж не найден",
		})
	}
	if payment.UserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Доступ запрещён",
		})
	}

	if err := h.paymentSvc.CompleteTopUpPayment(c.Context(), paymentID, req.TxHash); err != nil {
		log.Printf("[VerifyTopUp] %s: %v", paymentID, err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Не удалось проверить платёж",
		})
	}

	// Get updated balance
	balance, _ := h.balanceSvc.GetBalance(c.Context(), userID)

	return c.JSON(fiber.Map{
		"success":     true,
		"new_balance": balance,
	})
}
