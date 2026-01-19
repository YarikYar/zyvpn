package repository

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/zyvpn/backend/internal/model"
)

// IsAdmin checks if a user is an admin
func (r *Repository) IsAdmin(ctx context.Context, userID int64) (bool, error) {
	var count int
	err := r.db.GetContext(ctx, &count, `SELECT COUNT(*) FROM admins WHERE user_id = $1`, userID)
	return count > 0, err
}

// GetAdmin retrieves admin info by user ID
func (r *Repository) GetAdmin(ctx context.Context, userID int64) (*model.Admin, error) {
	var admin model.Admin
	err := r.db.GetContext(ctx, &admin, `SELECT * FROM admins WHERE user_id = $1`, userID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &admin, err
}

// CreateAdmin creates a new admin
func (r *Repository) CreateAdmin(ctx context.Context, admin *model.Admin) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO admins (user_id, role, created_by)
		VALUES ($1, $2, $3)`,
		admin.UserID, admin.Role, admin.CreatedBy)
	return err
}

// ListAdmins lists all admins
func (r *Repository) ListAdmins(ctx context.Context) ([]model.Admin, error) {
	var admins []model.Admin
	err := r.db.SelectContext(ctx, &admins, `SELECT * FROM admins ORDER BY created_at DESC`)
	return admins, err
}

// RemoveAdmin removes an admin
func (r *Repository) RemoveAdmin(ctx context.Context, userID int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM admins WHERE user_id = $1`, userID)
	return err
}

// BanUser bans a user by user_id
func (r *Repository) BanUser(ctx context.Context, ban *model.BannedUser) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO banned_users (user_id, ip_address, reason, banned_by, expires_at, is_active)
		VALUES ($1, $2, $3, $4, $5, true)`,
		ban.UserID, ban.IPAddress, ban.Reason, ban.BannedBy, ban.ExpiresAt)
	return err
}

// UnbanUser unbans a user
func (r *Repository) UnbanUser(ctx context.Context, userID int64) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE banned_users SET is_active = false
		WHERE user_id = $1 AND is_active = true`, userID)
	return err
}

// UnbanIP unbans an IP address
func (r *Repository) UnbanIP(ctx context.Context, ip string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE banned_users SET is_active = false
		WHERE ip_address = $1 AND is_active = true`, ip)
	return err
}

// IsUserBanned checks if a user is banned
func (r *Repository) IsUserBanned(ctx context.Context, userID int64) (bool, error) {
	var ban model.BannedUser
	err := r.db.GetContext(ctx, &ban, `
		SELECT * FROM banned_users
		WHERE user_id = $1 AND is_active = true
		ORDER BY banned_at DESC LIMIT 1`, userID)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	// Check if ban is expired
	if ban.IsExpired() {
		// Deactivate expired ban
		_, _ = r.db.ExecContext(ctx, `UPDATE banned_users SET is_active = false WHERE id = $1`, ban.ID)
		return false, nil
	}
	return true, nil
}

// IsIPBanned checks if an IP address is banned
func (r *Repository) IsIPBanned(ctx context.Context, ip string) (bool, error) {
	var ban model.BannedUser
	err := r.db.GetContext(ctx, &ban, `
		SELECT * FROM banned_users
		WHERE ip_address = $1 AND is_active = true
		ORDER BY banned_at DESC LIMIT 1`, ip)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	// Check if ban is expired
	if ban.IsExpired() {
		// Deactivate expired ban
		_, _ = r.db.ExecContext(ctx, `UPDATE banned_users SET is_active = false WHERE id = $1`, ban.ID)
		return false, nil
	}
	return true, nil
}

// GetUserBan gets active ban for a user
func (r *Repository) GetUserBan(ctx context.Context, userID int64) (*model.BannedUser, error) {
	var ban model.BannedUser
	err := r.db.GetContext(ctx, &ban, `
		SELECT * FROM banned_users
		WHERE user_id = $1 AND is_active = true
		ORDER BY banned_at DESC LIMIT 1`, userID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &ban, err
}

// ListBannedUsers lists all active bans
func (r *Repository) ListBannedUsers(ctx context.Context, limit, offset int) ([]model.BannedUser, error) {
	var bans []model.BannedUser
	err := r.db.SelectContext(ctx, &bans, `
		SELECT * FROM banned_users
		WHERE is_active = true
		ORDER BY banned_at DESC
		LIMIT $1 OFFSET $2`, limit, offset)
	return bans, err
}

// CreateAdminLog creates an admin action log entry
func (r *Repository) CreateAdminLog(ctx context.Context, log *model.AdminLog) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO admin_logs (admin_id, action, target_user_id, details)
		VALUES ($1, $2, $3, $4)`,
		log.AdminID, log.Action, log.TargetUserID, log.Details)
	return err
}

// LogAdminAction is a helper to create admin log with JSON details
func (r *Repository) LogAdminAction(ctx context.Context, adminID int64, action string, targetUserID *int64, details interface{}) error {
	var detailsJSON []byte
	if details != nil {
		var err error
		detailsJSON, err = json.Marshal(details)
		if err != nil {
			return err
		}
	}
	return r.CreateAdminLog(ctx, &model.AdminLog{
		AdminID:      adminID,
		Action:       action,
		TargetUserID: targetUserID,
		Details:      detailsJSON,
	})
}

// GetAdminLogs retrieves admin action logs
func (r *Repository) GetAdminLogs(ctx context.Context, limit, offset int) ([]model.AdminLog, error) {
	var logs []model.AdminLog
	err := r.db.SelectContext(ctx, &logs, `
		SELECT * FROM admin_logs
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`, limit, offset)
	return logs, err
}

// GetAdminLogsByTarget retrieves admin logs for a specific target user
func (r *Repository) GetAdminLogsByTarget(ctx context.Context, targetUserID int64, limit int) ([]model.AdminLog, error) {
	var logs []model.AdminLog
	err := r.db.SelectContext(ctx, &logs, `
		SELECT * FROM admin_logs
		WHERE target_user_id = $1
		ORDER BY created_at DESC
		LIMIT $2`, targetUserID, limit)
	return logs, err
}

// SetUserBalance sets user balance to a specific value
func (r *Repository) SetUserBalance(ctx context.Context, userID int64, balance float64) error {
	_, err := r.db.ExecContext(ctx, `UPDATE users SET balance = $1, updated_at = NOW() WHERE id = $2`, balance, userID)
	return err
}

// ListUsers lists users with pagination and optional search
func (r *Repository) ListUsers(ctx context.Context, limit, offset int, search string) ([]model.User, int, error) {
	var users []model.User
	var total int

	if search != "" {
		searchPattern := "%" + search + "%"
		// Count total
		err := r.db.GetContext(ctx, &total, `
			SELECT COUNT(*) FROM users
			WHERE username ILIKE $1 OR first_name ILIKE $1 OR CAST(id AS TEXT) LIKE $1`, searchPattern)
		if err != nil {
			return nil, 0, err
		}
		// Get users
		err = r.db.SelectContext(ctx, &users, `
			SELECT * FROM users
			WHERE username ILIKE $1 OR first_name ILIKE $1 OR CAST(id AS TEXT) LIKE $1
			ORDER BY created_at DESC
			LIMIT $2 OFFSET $3`, searchPattern, limit, offset)
		return users, total, err
	}

	// Count total
	err := r.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM users`)
	if err != nil {
		return nil, 0, err
	}
	// Get users
	err = r.db.SelectContext(ctx, &users, `
		SELECT * FROM users
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`, limit, offset)
	return users, total, err
}

// GetSubscriptionByUserID gets subscription by user ID
func (r *Repository) GetSubscriptionByUserID(ctx context.Context, userID int64) (*model.Subscription, error) {
	var sub model.Subscription
	err := r.db.GetContext(ctx, &sub, `
		SELECT * FROM subscriptions
		WHERE user_id = $1 AND status = 'active'
		ORDER BY created_at DESC LIMIT 1`, userID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &sub, err
}

// GetPromoCodeStats returns promo code statistics
func (r *Repository) GetPromoCodeStats(ctx context.Context, promoCodeID uuid.UUID) (int, error) {
	var count int
	err := r.db.GetContext(ctx, &count, `
		SELECT COUNT(*) FROM promo_code_uses WHERE promo_code_id = $1`, promoCodeID)
	return count, err
}
