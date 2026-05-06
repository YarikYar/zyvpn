package middleware

import (
	"crypto/subtle"

	"github.com/gofiber/fiber/v2"
)

// InternalSecret guards endpoints meant for cron jobs / self-hosted runners
// (e.g. /internal/cron/*). Protects against random internet hits since the
// reverse proxy exposes /internal/* publicly.
//
// Behavior:
//   - secret == ""  → middleware refuses every request (fail-closed). This
//     prevents accidentally shipping with no protection.
//   - secret set    → caller must send an exact match in the
//     X-Internal-Secret header.
func InternalSecret(secret string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if secret == "" {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"error": "internal endpoint disabled",
			})
		}
		got := c.Get("X-Internal-Secret")
		if subtle.ConstantTimeCompare([]byte(got), []byte(secret)) != 1 {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "unauthorized",
			})
		}
		return c.Next()
	}
}
