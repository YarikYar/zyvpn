package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/zyvpn/backend/internal/service"
)

const (
	AdminKey   = "is_admin"
	AdminIDKey = "admin_id"
)

// AdminAuth middleware checks if the authenticated user is an admin
func AdminAuth(adminSvc *service.AdminService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID := GetUserID(c)
		if userID == 0 {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "unauthorized",
			})
		}

		isAdmin, err := adminSvc.IsAdmin(c.Context(), userID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to check admin status",
			})
		}

		if !isAdmin {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "access denied",
			})
		}

		c.Locals(AdminKey, true)
		c.Locals(AdminIDKey, userID)

		return c.Next()
	}
}

// BanCheck middleware checks if the user or their IP is banned
func BanCheck(adminSvc *service.AdminService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID := GetUserID(c)

		// Check user ban
		if userID != 0 {
			banned, err := adminSvc.IsUserBanned(c.Context(), userID)
			if err == nil && banned {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
					"error": "account banned",
				})
			}
		}

		// Check IP ban
		ip := c.IP()
		if ip != "" {
			banned, err := adminSvc.IsIPBanned(c.Context(), ip)
			if err == nil && banned {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
					"error": "ip banned",
				})
			}
		}

		return c.Next()
	}
}

// GetAdminID returns the admin user ID from context
func GetAdminID(c *fiber.Ctx) int64 {
	adminID, ok := c.Locals(AdminIDKey).(int64)
	if !ok {
		return 0
	}
	return adminID
}

// IsAdmin checks if the current user is an admin
func IsAdmin(c *fiber.Ctx) bool {
	isAdmin, ok := c.Locals(AdminKey).(bool)
	return ok && isAdmin
}
