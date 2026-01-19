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

	if err := h.paymentSvc.VerifyTONPayment(c.Context(), paymentID, req.TxHash); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Get subscription key for response
	key, _ := h.subscriptionSvc.GetConnectionKey(c.Context(), userID)

	// Send notification via bot
	if h.bot != nil {
		sub, _ := h.subscriptionSvc.GetActiveSubscription(c.Context(), userID)
		if sub != nil {
			_ = h.bot.SendSubscriptionActivated(userID, sub.ExpiresAt.Format("02.01.2006"))
		}
	}

	return c.JSON(fiber.Map{
		"success": true,
		"key":     key,
	})
}

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
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to refund: " + err.Error(),
		})
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

// GetPaymentStatus returns current payment status for polling
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

	// Add subscription key if payment completed
	if payment.Status == "completed" && payment.PaymentType == "subscription" {
		key, _ := h.subscriptionSvc.GetConnectionKey(c.Context(), userID)
		response["key"] = key
	}

	// Add new balance if top-up completed
	if payment.Status == "completed" && payment.PaymentType == "top_up" {
		balance, _ := h.balanceSvc.GetBalance(c.Context(), userID)
		response["new_balance"] = balance
	}

	return c.JSON(response)
}

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
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to create invoice: " + err.Error(),
		})
	}
	log.Printf("Stars invoice created: %s", invoiceLink)

	return c.JSON(fiber.Map{
		"payment_id":   payment.ID,
		"amount":       int(payment.Amount),
		"currency":     "XTR",
		"invoice_link": invoiceLink,
	})
}
