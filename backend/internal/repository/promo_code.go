package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/zyvpn/backend/internal/model"
)

// GetPromoCodeByCode retrieves a promo code by its code string
func (r *Repository) GetPromoCodeByCode(ctx context.Context, code string) (*model.PromoCode, error) {
	var promo model.PromoCode
	err := r.db.GetContext(ctx, &promo, `
		SELECT * FROM promo_codes WHERE code = $1`, code)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &promo, err
}

// GetPromoCodeByID retrieves a promo code by ID
func (r *Repository) GetPromoCodeByID(ctx context.Context, id uuid.UUID) (*model.PromoCode, error) {
	var promo model.PromoCode
	err := r.db.GetContext(ctx, &promo, `
		SELECT * FROM promo_codes WHERE id = $1`, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &promo, err
}

// HasUserUsedPromoCode checks if a user has already used a specific promo code
func (r *Repository) HasUserUsedPromoCode(ctx context.Context, userID int64, promoCodeID uuid.UUID) (bool, error) {
	var count int
	err := r.db.GetContext(ctx, &count, `
		SELECT COUNT(*) FROM promo_code_uses
		WHERE user_id = $1 AND promo_code_id = $2`, userID, promoCodeID)
	return count > 0, err
}

// UsePromoCode marks a promo code as used by a user and increments the used count
func (r *Repository) UsePromoCode(ctx context.Context, userID int64, promoCodeID uuid.UUID) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Record the use
	_, err = tx.ExecContext(ctx, `
		INSERT INTO promo_code_uses (promo_code_id, user_id)
		VALUES ($1, $2)`, promoCodeID, userID)
	if err != nil {
		return fmt.Errorf("failed to record promo code use: %w", err)
	}

	// Increment used count
	_, err = tx.ExecContext(ctx, `
		UPDATE promo_codes SET used_count = used_count + 1
		WHERE id = $1`, promoCodeID)
	if err != nil {
		return fmt.Errorf("failed to increment used count: %w", err)
	}

	return tx.Commit()
}

// CreatePromoCode creates a new promo code (for admin use)
func (r *Repository) CreatePromoCode(ctx context.Context, promo *model.PromoCode) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO promo_codes (code, type, value, max_uses, expires_at, is_active, description)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		promo.Code, promo.Type, promo.Value, promo.MaxUses, promo.ExpiresAt, promo.IsActive, promo.Description)
	return err
}

// ListPromoCodes lists all promo codes (for admin use)
func (r *Repository) ListPromoCodes(ctx context.Context, limit, offset int) ([]model.PromoCode, error) {
	var promos []model.PromoCode
	err := r.db.SelectContext(ctx, &promos, `
		SELECT * FROM promo_codes
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`, limit, offset)
	return promos, err
}

// DeactivatePromoCode deactivates a promo code
func (r *Repository) DeactivatePromoCode(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE promo_codes SET is_active = false WHERE id = $1`, id)
	return err
}
