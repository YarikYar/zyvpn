package model

import (
	"time"

	"github.com/google/uuid"
)

type SubscriptionStatus string

const (
	SubscriptionStatusActive    SubscriptionStatus = "active"
	SubscriptionStatusExpired   SubscriptionStatus = "expired"
	SubscriptionStatusCancelled SubscriptionStatus = "cancelled"
)

type Subscription struct {
	ID           uuid.UUID          `json:"id" db:"id"`
	UserID       int64              `json:"user_id" db:"user_id"`
	PlanID       uuid.UUID          `json:"plan_id" db:"plan_id"`
	Status       SubscriptionStatus `json:"status" db:"status"`
	SubToken     string             `json:"sub_token" db:"sub_token"`
	StartedAt    *time.Time         `json:"started_at,omitempty" db:"started_at"`
	ExpiresAt    *time.Time         `json:"expires_at,omitempty" db:"expires_at"`
	TrafficLimit int64              `json:"traffic_limit" db:"traffic_limit"`
	TrafficUsed  int64              `json:"traffic_used" db:"traffic_used"`
	MaxDevices   int                `json:"max_devices" db:"max_devices"`
	CreatedAt    time.Time          `json:"created_at" db:"created_at"`
	// Clients — per-server xui-клиенты этой подписки. JOIN c subscription_clients.
	Clients []SubscriptionClient `json:"servers" db:"-"`
}

// SubscriptionClient — xui-клиент этой подписки на конкретном сервере.
// У подписки их N (по числу серверов в её тарифе).
type SubscriptionClient struct {
	ID             uuid.UUID `json:"id" db:"id"`
	SubscriptionID uuid.UUID `json:"subscription_id" db:"subscription_id"`
	ServerID       uuid.UUID `json:"server_id" db:"server_id"`
	XUIClientID    string    `json:"xui_client_id" db:"xui_client_id"`
	XUIEmail       string    `json:"xui_email" db:"xui_email"`
	ConnectionKey  string    `json:"connection_key" db:"connection_key"`
	TrafficUsed    int64     `json:"traffic_used" db:"traffic_used"`
	Enabled        bool      `json:"enabled" db:"enabled"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	// Server — JOIN'нутая инфа о сервере (имя/город/флаг/пинг/статус).
	// Заполняется только в read-методах, иначе nil.
	Server *Server `json:"server,omitempty" db:"-"`
}

type SubscriptionWithPlan struct {
	Subscription
	Plan *Plan `json:"plan,omitempty"`
}

func (s *Subscription) IsActive() bool {
	if s.Status != SubscriptionStatusActive {
		return false
	}
	if s.ExpiresAt != nil && time.Now().After(*s.ExpiresAt) {
		return false
	}
	if s.TrafficLimit > 0 && s.TrafficUsed >= s.TrafficLimit {
		return false
	}
	return true
}

func (s *Subscription) RemainingTrafficGB() float64 {
	if s.TrafficLimit <= 0 {
		return -1 // Unlimited
	}
	remaining := s.TrafficLimit - s.TrafficUsed
	if remaining < 0 {
		remaining = 0
	}
	return float64(remaining) / (1024 * 1024 * 1024)
}

func (s *Subscription) DaysRemaining() int {
	if s.ExpiresAt == nil {
		return -1 // Unlimited
	}
	duration := time.Until(*s.ExpiresAt)
	if duration < 0 {
		return 0
	}
	return int(duration.Hours() / 24)
}
