package model

import (
	"time"

	"github.com/google/uuid"
)

type TransactionType string

const (
	TransactionTypeReferralBonus       TransactionType = "referral_bonus"
	TransactionTypeGiveaway            TransactionType = "giveaway"
	TransactionTypeSubscriptionPayment TransactionType = "subscription_payment"
	TransactionTypeRefund              TransactionType = "refund"
	TransactionTypeManual              TransactionType = "manual"
	TransactionTypeTopUp               TransactionType = "top_up"
	TransactionTypePromoCode           TransactionType = "promo_code"
)

type BalanceTransaction struct {
	ID            uuid.UUID       `json:"id" db:"id"`
	UserID        int64           `json:"user_id" db:"user_id"`
	Amount        float64         `json:"amount" db:"amount"` // positive = credit, negative = debit
	Type          TransactionType `json:"type" db:"type"`
	Description   *string         `json:"description,omitempty" db:"description"`
	ReferenceID   *uuid.UUID      `json:"reference_id,omitempty" db:"reference_id"`
	BalanceBefore float64         `json:"balance_before" db:"balance_before"`
	BalanceAfter  float64         `json:"balance_after" db:"balance_after"`
	CreatedAt     time.Time       `json:"created_at" db:"created_at"`
}
