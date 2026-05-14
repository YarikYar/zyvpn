package model

import (
	"time"

	"github.com/google/uuid"
)

type PromoCodeType string

const (
	PromoCodeTypeBalance PromoCodeType = "balance" // Credits TON to balance
	PromoCodeTypeDays    PromoCodeType = "days"    // Adds days to subscription
	// PromoCodeTypeCashPlan unlocks a specific plan at a discounted cash
	// price. Redeeming the code creates a pending cash payment for the
	// bound plan and amount; admin approval delivers the subscription.
	PromoCodeTypeCashPlan PromoCodeType = "cash_plan"
)

type PromoCode struct {
	ID            uuid.UUID     `json:"id" db:"id"`
	Code          string        `json:"code" db:"code"`
	Type          PromoCodeType `json:"type" db:"type"`
	Value         float64       `json:"value" db:"value"` // TON amount or days count
	MaxUses       *int          `json:"max_uses,omitempty" db:"max_uses"`
	UsedCount     int           `json:"used_count" db:"used_count"`
	ExpiresAt     *time.Time    `json:"expires_at,omitempty" db:"expires_at"`
	IsActive      bool          `json:"is_active" db:"is_active"`
	Description   *string       `json:"description,omitempty" db:"description"`
	// CashPlan-specific fields. Both NULL for non-cash_plan promos.
	PlanID        *uuid.UUID    `json:"plan_id,omitempty" db:"plan_id"`
	CashAmountRUB *float64      `json:"cash_amount_rub,omitempty" db:"cash_amount_rub"`
	CreatedAt     time.Time     `json:"created_at" db:"created_at"`
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
