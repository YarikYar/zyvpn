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
	XUIClientID  string             `json:"xui_client_id" db:"xui_client_id"`
	XUIEmail     string             `json:"xui_email" db:"xui_email"`
	ConnectionKey string            `json:"connection_key,omitempty" db:"connection_key"`
	StartedAt    *time.Time         `json:"started_at,omitempty" db:"started_at"`
	ExpiresAt    *time.Time         `json:"expires_at,omitempty" db:"expires_at"`
	TrafficLimit int64              `json:"traffic_limit" db:"traffic_limit"`
	TrafficUsed  int64              `json:"traffic_used" db:"traffic_used"`
	CreatedAt    time.Time          `json:"created_at" db:"created_at"`
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
