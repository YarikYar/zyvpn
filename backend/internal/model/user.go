package model

import (
	"time"
)

type User struct {
	ID           int64      `json:"id" db:"id"`
	Username     *string    `json:"username,omitempty" db:"username"`
	FirstName    *string    `json:"first_name,omitempty" db:"first_name"`
	LastName     *string    `json:"last_name,omitempty" db:"last_name"`
	LanguageCode *string    `json:"language_code,omitempty" db:"language_code"`
	ReferralCode string     `json:"referral_code" db:"referral_code"`
	ReferredBy   *int64     `json:"referred_by,omitempty" db:"referred_by"`
	Balance      float64    `json:"balance" db:"balance"` // Balance in TON
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at" db:"updated_at"`
}

type UserWithSubscription struct {
	User
	Subscription *Subscription `json:"subscription,omitempty"`
}
