package handler

import (
	"github.com/gofiber/fiber/v2"
)

// TelegramWebhook is deprecated - we use long polling instead
func (h *Handler) TelegramWebhook(c *fiber.Ctx) error {
	// Long polling is used instead of webhooks
	return c.SendStatus(fiber.StatusOK)
}

type TONWebhookPayload struct {
	TransactionHash string `json:"transaction_hash"`
	Comment         string `json:"comment"` // Contains payment_id
	Amount          string `json:"amount"`
	FromAddress     string `json:"from_address"`
}

func (h *Handler) TONWebhook(c *fiber.Ctx) error {
	// This would be called by a TON blockchain monitoring service
	// For production, you'd use something like TON Center API or your own indexer
	// The comment in the transaction contains the payment_id

	// For now, this is a placeholder
	// In production, you'd verify the transaction and complete the payment

	return c.SendStatus(fiber.StatusOK)
}

func (h *Handler) StarsWebhook(c *fiber.Ctx) error {
	// This is handled by the Telegram bot via long polling
	// Successful payments come as regular Telegram updates

	return c.SendStatus(fiber.StatusOK)
}
