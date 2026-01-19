package model

import (
	"time"

	"github.com/google/uuid"
)

type BonusType string

const (
	BonusTypeDays     BonusType = "days"
	BonusTypeDiscount BonusType = "discount"
	BonusTypeBalance  BonusType = "balance"
)

type ReferralStatus string

const (
	ReferralStatusPending  ReferralStatus = "pending"
	ReferralStatusCredited ReferralStatus = "credited"
)

type Referral struct {
	ID         uuid.UUID      `json:"id" db:"id"`
	ReferrerID int64          `json:"referrer_id" db:"referrer_id"`
	ReferredID int64          `json:"referred_id" db:"referred_id"`
	BonusType  BonusType      `json:"bonus_type" db:"bonus_type"`
	BonusValue int            `json:"bonus_value" db:"bonus_value"`
	BonusTON   float64        `json:"bonus_ton" db:"bonus_ton"` // Bonus amount in TON
	Status     ReferralStatus `json:"status" db:"status"`
	CreatedAt  time.Time      `json:"created_at" db:"created_at"`
	CreditedAt *time.Time     `json:"credited_at,omitempty" db:"credited_at"`
}

type ReferralStats struct {
	TotalReferrals    int     `json:"total_referrals"`
	PendingReferrals  int     `json:"pending_referrals"`
	CreditedBonusTON  float64 `json:"credited_bonus_ton"`
}

// Default bonus configuration
const (
	DefaultReferralBonusType = BonusTypeBalance
	DefaultReferralBonusTON  = 0.1 // 0.1 TON bonus for referrer when referred user pays
)
