package config

import (
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	Telegram TelegramConfig
	TON      TONConfig
}

type ServerConfig struct {
	Port         string
	Environment  string
	JWTSecret    string
	AllowOrigins string
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

type TelegramConfig struct {
	BotToken  string
	WebAppURL string
}

type TONConfig struct {
	Testnet       bool
	WalletAddress string
}

func (d DatabaseConfig) DSN() string {
	return "postgres://" + d.User + ":" + d.Password + "@" + d.Host + ":" + d.Port + "/" + d.Name + "?sslmode=" + d.SSLMode
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	redisDB, _ := strconv.Atoi(getEnv("REDIS_DB", "0"))
	tonTestnet, _ := strconv.ParseBool(getEnv("TON_TESTNET", "true"))

	cfg := &Config{
		Server: ServerConfig{
			Port:         getEnv("SERVER_PORT", "8080"),
			Environment:  getEnv("ENVIRONMENT", "development"),
			JWTSecret:    getEnv("JWT_SECRET", "your-secret-key-change-in-production"),
			AllowOrigins: getEnv("ALLOW_ORIGINS", "*"),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "zyvpn"),
			Password: getEnv("DB_PASSWORD", "zyvpn"),
			Name:     getEnv("DB_NAME", "zyvpn"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       redisDB,
		},
		Telegram: TelegramConfig{
			BotToken:  getEnv("TELEGRAM_BOT_TOKEN", ""),
			WebAppURL: getEnv("TELEGRAM_WEBAPP_URL", ""),
		},
		TON: TONConfig{
			Testnet:       tonTestnet,
			WalletAddress: getEnv("TON_WALLET_ADDRESS", ""),
		},
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Subscription durations
const (
	SubscriptionCheckInterval = 1 * time.Hour
	NotifyBeforeExpiry3Days   = 3 * 24 * time.Hour
	NotifyBeforeExpiry1Day    = 24 * time.Hour
)
