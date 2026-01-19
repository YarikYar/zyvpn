package model

import (
	"time"

	"github.com/google/uuid"
)

type AdminRole string

const (
	AdminRoleAdmin      AdminRole = "admin"
	AdminRoleSuperAdmin AdminRole = "superadmin"
)

type Admin struct {
	ID        uuid.UUID `json:"id" db:"id"`
	UserID    int64     `json:"user_id" db:"user_id"`
	Role      AdminRole `json:"role" db:"role"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	CreatedBy *int64    `json:"created_by,omitempty" db:"created_by"`
}

type BannedUser struct {
	ID        uuid.UUID  `json:"id" db:"id"`
	UserID    *int64     `json:"user_id,omitempty" db:"user_id"`
	IPAddress *string    `json:"ip_address,omitempty" db:"ip_address"`
	Reason    *string    `json:"reason,omitempty" db:"reason"`
	BannedAt  time.Time  `json:"banned_at" db:"banned_at"`
	BannedBy  *int64     `json:"banned_by,omitempty" db:"banned_by"`
	ExpiresAt *time.Time `json:"expires_at,omitempty" db:"expires_at"`
	IsActive  bool       `json:"is_active" db:"is_active"`
}

// IsExpired checks if ban has expired
func (b *BannedUser) IsExpired() bool {
	if b.ExpiresAt == nil {
		return false // Permanent ban
	}
	return time.Now().After(*b.ExpiresAt)
}

type AdminLog struct {
	ID           uuid.UUID `json:"id" db:"id"`
	AdminID      int64     `json:"admin_id" db:"admin_id"`
	Action       string    `json:"action" db:"action"`
	TargetUserID *int64    `json:"target_user_id,omitempty" db:"target_user_id"`
	Details      []byte    `json:"details,omitempty" db:"details"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

// Admin action constants
const (
	AdminActionSetBalance       = "set_balance"
	AdminActionAddBalance       = "add_balance"
	AdminActionExtendSub        = "extend_subscription"
	AdminActionCancelSub        = "cancel_subscription"
	AdminActionBanUser          = "ban_user"
	AdminActionBanIP            = "ban_ip"
	AdminActionUnbanUser        = "unban_user"
	AdminActionCreatePromoCode  = "create_promo_code"
	AdminActionDeactivatePromo  = "deactivate_promo_code"
)
