package handler

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/zyvpn/backend/internal/middleware"
)

type VerifyTONPaymentRequest struct {
	PaymentID string `json:"payment_id"`
	TxHash    string `json:"tx_hash"`
}

// InitTONPayment returns the TON deep link for an existing payment.
//
//	@Summary	TON deep link
//	@Tags		payments
//	@Produce	json
//	@Param		payment_id	query		string	true	"payment uuid"
//	@Success	200			{object}	model.TONPaymentInfo
//	@Failure	404			{object}	map[string]string
//	@Router		/api/payment/ton/init [get]
//	@Security	TelegramInitData
func (h *Handler) InitTONPayment(c *fiber.Ctx) error {
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

	tonInfo, err := h.paymentSvc.GetTONPaymentInfo(c.Context(), paymentID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Платёж не найден",
		})
	}

	return c.JSON(tonInfo)
}

// VerifyTONPayment submits a TON tx hash for verification.
//
//	@Summary	Verify TON payment
//	@Tags		payments
//	@Accept		json
//	@Produce	json
//	@Param		body	body		VerifyTONPaymentRequest	true	"payment id + BOC"
//	@Success	200		{object}	map[string]interface{}
//	@Failure	400		{object}	map[string]string
//	@Failure	403		{object}	map[string]string
//	@Failure	404		{object}	map[string]string
//	@Router		/api/payment/ton/check [post]
//	@Security	TelegramInitData
func (h *Handler) VerifyTONPayment(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Необходима авторизация",
		})
	}

	var req VerifyTONPaymentRequest
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

	// Ownership check before kicking off TON verification — иначе любой
	// залогиненный юзер мог бы спамить проверки чужих платежей.
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

	if err := h.paymentSvc.VerifyTONPayment(c.Context(), paymentID, req.TxHash); err != nil {
		log.Printf("[VerifyTONPayment] %s: %v", paymentID, err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Не удалось проверить платёж",
		})
	}

	// Subscription URL вместо одиночного connection_key
	subURL, _ := h.subscriptionSvc.GetSubscriptionURLForUser(c.Context(), userID)

	// Send notification via bot
	if h.bot != nil {
		sub, _ := h.subscriptionSvc.GetActiveSubscription(c.Context(), userID)
		if sub != nil {
			_ = h.bot.SendSubscriptionActivated(userID, sub.ExpiresAt.Format("02.01.2006"))
		}
	}

	return c.JSON(fiber.Map{
		"success":          true,
		"subscription_url": subURL,
	})
}

// RefundStarsPayment refunds a completed Telegram Stars payment.
//
//	@Summary	Refund Stars payment
//	@Tags		payments
//	@Produce	json
//	@Param		payment_id	query		string	true	"payment uuid"
//	@Success	200			{object}	map[string]interface{}
//	@Failure	403			{object}	map[string]string
//	@Failure	404			{object}	map[string]string
//	@Failure	503			{object}	map[string]string
//	@Router		/api/payment/stars/refund [post]
//	@Security	TelegramInitData
func (h *Handler) RefundStarsPayment(c *fiber.Ctx) error {
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

	// Get telegram charge ID
	paymentUserID, chargeID, err := h.paymentSvc.GetTelegramChargeID(c.Context(), paymentID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Verify user owns this payment
	if paymentUserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Доступ запрещён",
		})
	}

	if h.bot == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "Сервис возврата недоступен",
		})
	}

	// Refund via Telegram API
	if err := h.bot.RefundStarsPayment(userID, chargeID); err != nil {
		log.Printf("Failed to refund Stars payment: %v", err)
		return respondInternalError(c, err)
	}

	// Update payment status
	if err := h.paymentSvc.RefundPayment(c.Context(), paymentID); err != nil {
		log.Printf("Failed to update payment status: %v", err)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Возврат успешно обработан",
	})
}

// GetPaymentStatus returns current payment status for polling.
//
//	@Summary	Poll payment status
//	@Tags		payments
//	@Produce	json
//	@Param		payment_id	query		string	true	"payment uuid"
//	@Success	200			{object}	map[string]interface{}
//	@Failure	403			{object}	map[string]string
//	@Failure	404			{object}	map[string]string
//	@Router		/api/payment/status [get]
//	@Security	TelegramInitData
func (h *Handler) GetPaymentStatus(c *fiber.Ctx) error {
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

	payment, err := h.paymentSvc.GetPaymentStatus(c.Context(), paymentID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Платёж не найден",
		})
	}

	// Verify user owns this payment
	if payment.UserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Доступ запрещён",
		})
	}

	response := fiber.Map{
		"payment_id": payment.ID,
		"status":     payment.Status,
		"amount":     payment.Amount,
		"currency":   payment.Currency,
	}

	// Add subscription URL if payment completed
	if payment.Status == "completed" && payment.PaymentType == "subscription" {
		subURL, _ := h.subscriptionSvc.GetSubscriptionURLForUser(c.Context(), userID)
		response["subscription_url"] = subURL
	}

	// Add new balance if top-up completed
	if payment.Status == "completed" && payment.PaymentType == "top_up" {
		balance, _ := h.balanceSvc.GetBalance(c.Context(), userID)
		response["new_balance"] = balance
	}

	return c.JSON(response)
}

// InitStarsPayment returns a Telegram Stars invoice link.
//
//	@Summary	Stars invoice link
//	@Tags		payments
//	@Produce	json
//	@Param		payment_id	query		string	true	"payment uuid"
//	@Success	200			{object}	map[string]interface{}
//	@Failure	404			{object}	map[string]string
//	@Router		/api/payment/stars/init [get]
//	@Security	TelegramInitData
func (h *Handler) InitStarsPayment(c *fiber.Ctx) error {
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

	if payment.PlanID == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Платёж не привязан к тарифу",
		})
	}

	// Get plan for title/description
	plan, err := h.planService.GetPlan(c.Context(), *payment.PlanID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Тариф не найден",
		})
	}

	if h.bot == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "Сервис оплаты недоступен",
		})
	}

	// Create invoice via bot
	log.Printf("Creating Stars invoice for user %d, plan %s, amount %d", userID, plan.Name, int(payment.Amount))
	invoiceLink, err := h.bot.CreateStarsInvoice(
		userID,
		plan.Name,
		plan.Description,
		int(payment.Amount),
		payment.ID.String(),
	)
	if err != nil {
		log.Printf("Failed to create Stars invoice: %v", err)
		return respondInternalError(c, err)
	}
	log.Printf("Stars invoice created: %s", invoiceLink)

	return c.JSON(fiber.Map{
		"payment_id":   payment.ID,
		"amount":       int(payment.Amount),
		"currency":     "XTR",
		"invoice_link": invoiceLink,
	})
}
