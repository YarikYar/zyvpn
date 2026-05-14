package model

import (
	"time"

	"github.com/google/uuid"
)

type PaymentProvider string

const (
	PaymentProviderTON     PaymentProvider = "ton"
	PaymentProviderStars   PaymentProvider = "stars"
	PaymentProviderBalance PaymentProvider = "balance"
)

type PaymentStatus string

const (
	PaymentStatusPending    PaymentStatus = "pending"
	PaymentStatusAwaitingTx PaymentStatus = "awaiting_tx" // Waiting for blockchain confirmation
	PaymentStatusCompleted  PaymentStatus = "completed"
	PaymentStatusFailed     PaymentStatus = "failed"
	PaymentStatusRefunded   PaymentStatus = "refunded"
)

type PaymentType string

const (
	PaymentTypeSubscription PaymentType = "subscription"
	PaymentTypeTopUp        PaymentType = "top_up"
)

type Payment struct {
	ID             uuid.UUID       `json:"id" db:"id"`
	UserID         int64           `json:"user_id" db:"user_id"`
	SubscriptionID *uuid.UUID      `json:"subscription_id,omitempty" db:"subscription_id"`
	PlanID         *uuid.UUID      `json:"plan_id,omitempty" db:"plan_id"`
	ServerID       *uuid.UUID      `json:"server_id,omitempty" db:"server_id"`
	PaymentType    PaymentType     `json:"payment_type" db:"payment_type"`
	Provider       PaymentProvider `json:"provider" db:"provider"`
	Amount         float64         `json:"amount" db:"amount"`
	Currency       string          `json:"currency" db:"currency"`
	Status         PaymentStatus   `json:"status" db:"status"`
	ExternalID     *string         `json:"external_id,omitempty" db:"external_id"`
	Metadata       *string         `json:"metadata,omitempty" db:"metadata"`
	CreatedAt      time.Time       `json:"created_at" db:"created_at"`
	CompletedAt    *time.Time      `json:"completed_at,omitempty" db:"completed_at"`
}

type CreatePaymentRequest struct {
	PlanID   uuid.UUID       `json:"plan_id" validate:"required"`
	ServerID *uuid.UUID      `json:"server_id,omitempty"`
	Provider PaymentProvider `json:"provider" validate:"required,oneof=ton stars"`
}

type TONPaymentInfo struct {
	PaymentID     uuid.UUID `json:"payment_id"`
	WalletAddress string    `json:"wallet_address"`
	Amount        string    `json:"amount"`
	Comment       string    `json:"comment"`
	DeepLink      string    `json:"deep_link"`
}

type StarsPaymentInfo struct {
	PaymentID  uuid.UUID `json:"payment_id"`
	InvoiceURL string    `json:"invoice_url"`
}
