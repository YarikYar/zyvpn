package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
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
	calculatedHash := hex.EncodeToString(h.Sum(nil))

	if calculatedHash != hash {
		return nil, fiber.NewError(fiber.StatusUnauthorized, "invalid hash")
	}

	// Parse user data
	userData := &TelegramInitData{
		QueryID:  values.Get("query_id"),
		AuthDate: authDate,
		Hash:     hash,
	}

	userJSON := values.Get("user")
	if userJSON != "" {
		// Parse user JSON manually
		userJSON, _ = url.QueryUnescape(userJSON)
		userData.UserID = parseJSONInt(userJSON, "id")
		userData.Username = parseJSONString(userJSON, "username")
		userData.FirstName = parseJSONString(userJSON, "first_name")
		userData.LastName = parseJSONString(userJSON, "last_name")
		userData.LanguageCode = parseJSONString(userJSON, "language_code")
	}

	return userData, nil
}

func parseJSONString(json, key string) string {
	search := `"` + key + `":"`
	start := strings.Index(json, search)
	if start == -1 {
		return ""
	}
	start += len(search)
	end := strings.Index(json[start:], `"`)
	if end == -1 {
		return ""
	}
	return json[start : start+end]
}

func parseJSONInt(json, key string) int64 {
	search := `"` + key + `":`
	start := strings.Index(json, search)
	if start == -1 {
		return 0
	}
	start += len(search)

	// Find end of number
	end := start
	for end < len(json) && (json[end] >= '0' && json[end] <= '9') {
		end++
	}

	val, _ := strconv.ParseInt(json[start:end], 10, 64)
	return val
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
