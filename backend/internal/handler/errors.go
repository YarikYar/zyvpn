package handler

import (
	"log"

	"github.com/gofiber/fiber/v2"
)

// respondInternalError logs the underlying error with full context (method,
// path, original error) and returns a generic 500 to the client. Use this
// for any 5xx response — never hand `err.Error()` back, since service-layer
// errors often contain SQL fragments, file paths or upstream details.
func respondInternalError(c *fiber.Ctx, err error) error {
	log.Printf("[%s %s] internal: %v", c.Method(), c.Path(), err)
	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
		"error": "Внутренняя ошибка сервера",
	})
}
