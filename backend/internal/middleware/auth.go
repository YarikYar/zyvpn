package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/zyvpn/backend/internal/config"
)

const (
	TelegramUserKey = "telegram_user"
	UserIDKey       = "user_id"
)

type TelegramInitData struct {
	QueryID      string `json:"query_id"`
	UserID       int64  `json:"user_id"`
	Username     string `json:"username"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
	LanguageCode string `json:"language_code"`
	AuthDate     int64  `json:"auth_date"`
	Hash         string `json:"hash"`
}

func TelegramAuth(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		initData := c.Get("X-Telegram-Init-Data")
		if initData == "" {
			initData = c.Get("Authorization")
			if strings.HasPrefix(initData, "tma ") {
				initData = strings.TrimPrefix(initData, "tma ")
			}
		}

		// Dev bypass — for redesign testing without a real Telegram WebApp.
		// Accepts X-User-ID header as the identity, OR parses initData
		// without HMAC validation. NEVER enable in production.
		if cfg.Server.InsecureTGAuthBypass {
			if uidStr := c.Get("X-User-ID"); uidStr != "" {
				if uid, err := strconv.ParseInt(uidStr, 10, 64); err == nil && uid > 0 {
					c.Locals(TelegramUserKey, &TelegramInitData{UserID: uid})
					c.Locals(UserIDKey, uid)
					return c.Next()
				}
			}
			if initData != "" {
				userData, err := parseInitDataNoVerify(initData)
				if err == nil && userData.UserID > 0 {
					c.Locals(TelegramUserKey, userData)
					c.Locals(UserIDKey, userData.UserID)
					return c.Next()
				}
			}
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "bypass mode: provide X-User-ID header or initData with user.id",
			})
		}

		if initData == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "missing telegram init data",
			})
		}

		userData, err := ValidateTelegramInitData(initData, cfg.Telegram.BotToken)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "invalid telegram init data: " + err.Error(),
			})
		}

		c.Locals(TelegramUserKey, userData)
		c.Locals(UserIDKey, userData.UserID)

		return c.Next()
	}
}

// parseInitDataNoVerify extracts user info from a Telegram initData query
// string without checking the HMAC. Used only when InsecureTGAuthBypass is
// on. Returns an error if the user JSON is missing or malformed.
func parseInitDataNoVerify(initData string) (*TelegramInitData, error) {
	values, err := url.ParseQuery(initData)
	if err != nil {
		return nil, err
	}
	authDate, _ := strconv.ParseInt(values.Get("auth_date"), 10, 64)
	out := &TelegramInitData{
		QueryID:  values.Get("query_id"),
		AuthDate: authDate,
		Hash:     values.Get("hash"),
	}
	if rawUser := values.Get("user"); rawUser != "" {
		var u struct {
			ID           int64  `json:"id"`
			Username     string `json:"username"`
			FirstName    string `json:"first_name"`
			LastName     string `json:"last_name"`
			LanguageCode string `json:"language_code"`
		}
		if err := json.Unmarshal([]byte(rawUser), &u); err != nil {
			return nil, err
		}
		out.UserID = u.ID
		out.Username = u.Username
		out.FirstName = u.FirstName
		out.LastName = u.LastName
		out.LanguageCode = u.LanguageCode
	}
	return out, nil
}

func ValidateTelegramInitData(initData, botToken string) (*TelegramInitData, error) {
	values, err := url.ParseQuery(initData)
	if err != nil {
		return nil, err
	}

	hash := values.Get("hash")
	if hash == "" {
		return nil, fiber.NewError(fiber.StatusUnauthorized, "missing hash")
	}

	// Check auth_date
	authDateStr := values.Get("auth_date")
	authDate, err := strconv.ParseInt(authDateStr, 10, 64)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusUnauthorized, "invalid auth_date")
	}

	// Check if auth_date is not too old (1 hour)
	if time.Now().Unix()-authDate > 3600 {
		return nil, fiber.NewError(fiber.StatusUnauthorized, "auth_date expired")
	}

	// Build data check string
	values.Del("hash")
	var keys []string
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var dataCheckParts []string
	for _, key := range keys {
		dataCheckParts = append(dataCheckParts, key+"="+values.Get(key))
	}
	dataCheckString := strings.Join(dataCheckParts, "\n")

	// Calculate secret key
	secretKey := hmac.New(sha256.New, []byte("WebAppData"))
	secretKey.Write([]byte(botToken))

	// Calculate hash
	h := hmac.New(sha256.New, secretKey.Sum(nil))
	h.Write([]byte(dataCheckString))
	calculatedHash := h.Sum(nil)

	providedHash, err := hex.DecodeString(hash)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusUnauthorized, "invalid hash encoding")
	}

	// Constant-time comparison to avoid leaking timing information.
	if !hmac.Equal(calculatedHash, providedHash) {
		return nil, fiber.NewError(fiber.StatusUnauthorized, "invalid hash")
	}

	userData := &TelegramInitData{
		QueryID:  values.Get("query_id"),
		AuthDate: authDate,
		Hash:     hash,
	}

	if rawUser := values.Get("user"); rawUser != "" {
		var u struct {
			ID           int64  `json:"id"`
			Username     string `json:"username"`
			FirstName    string `json:"first_name"`
			LastName     string `json:"last_name"`
			LanguageCode string `json:"language_code"`
		}
		// Telegram passes user as a JSON string (already URL-decoded by ParseQuery).
		if err := json.Unmarshal([]byte(rawUser), &u); err != nil {
			return nil, fiber.NewError(fiber.StatusUnauthorized, "invalid user payload")
		}
		userData.UserID = u.ID
		userData.Username = u.Username
		userData.FirstName = u.FirstName
		userData.LastName = u.LastName
		userData.LanguageCode = u.LanguageCode
	}

	return userData, nil
}

func GetUserID(c *fiber.Ctx) int64 {
	userID, ok := c.Locals(UserIDKey).(int64)
	if !ok {
		return 0
	}
	return userID
}

func GetTelegramUser(c *fiber.Ctx) *TelegramInitData {
	userData, ok := c.Locals(TelegramUserKey).(*TelegramInitData)
	if !ok {
		return nil
	}
	return userData
}
