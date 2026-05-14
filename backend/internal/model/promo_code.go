package model

import (
	"time"

	"github.com/google/uuid"
)

type PromoCodeType string

const (
	PromoCodeTypeBalance PromoCodeType = "balance" // Credits TON to balance
	PromoCodeTypeDays    PromoCodeType = "days"    // Adds days to subscription
)

type PromoCode struct {
	ID          uuid.UUID     `json:"id" db:"id"`
	Code        string        `json:"code" db:"code"`
	Type        PromoCodeType `json:"type" db:"type"`
	Value       float64       `json:"value" db:"value"` // TON amount or days count
	MaxUses     *int          `json:"max_uses,omitempty" db:"max_uses"`
	UsedCount   int           `json:"used_count" db:"used_count"`
	ExpiresAt   *time.Time    `json:"expires_at,omitempty" db:"expires_at"`
	IsActive    bool          `json:"is_active" db:"is_active"`
	Description *string       `json:"description,omitempty" db:"description"`
	CreatedAt   time.Time     `json:"created_at" db:"created_at"`
}

type PromoCodeUse struct {
	ID          uuid.UUID `json:"id" db:"id"`
	PromoCodeID uuid.UUID `json:"promo_code_id" db:"promo_code_id"`
	UserID      int64     `json:"user_id" db:"user_id"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// IsValid checks if the promo code can be used
func (p *PromoCode) IsValid() bool {
	if !p.IsActive {
		return false
	}

	// Check expiration
	if p.ExpiresAt != nil && time.Now().After(*p.ExpiresAt) {
		return false
	}

	// Check usage limit
	if p.MaxUses != nil && p.UsedCount >= *p.MaxUses {
		return false
	}

	return true
}
