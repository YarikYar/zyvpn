package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/zyvpn/backend/internal/model"
)

var ErrReferralNotFound = errors.New("referral not found")

func (r *Repository) GetReferral(ctx context.Context, id uuid.UUID) (*model.Referral, error) {
	var referral model.Referral
	err := r.db.GetContext(ctx, &referral, "SELECT * FROM referrals WHERE id = $1", id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrReferralNotFound
		}
		return nil, err
	}
	return &referral, nil
}

func (r *Repository) CreateReferral(ctx context.Context, referral *model.Referral) error {
	query := `
		INSERT INTO referrals (referrer_id, referred_id, bonus_type, bonus_value, bonus_ton, status)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at`

	return r.db.QueryRowContext(ctx, query,
		referral.ReferrerID,
		referral.ReferredID,
		referral.BonusType,
		referral.BonusValue,
		referral.BonusTON,
		referral.Status,
	).Scan(&referral.ID, &referral.CreatedAt)
}

func (r *Repository) GetReferralByReferredID(ctx context.Context, referredID int64) (*model.Referral, error) {
	var referral model.Referral
	query := "SELECT * FROM referrals WHERE referred_id = $1"
	err := r.db.GetContext(ctx, &referral, query, referredID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrReferralNotFound
		}
		return nil, err
	}
	return &referral, nil
}

func (r *Repository) GetPendingReferralsByReferrer(ctx context.Context, referrerID int64) ([]model.Referral, error) {
	var referrals []model.Referral
	query := "SELECT * FROM referrals WHERE referrer_id = $1 AND status = 'pending'"
	err := r.db.SelectContext(ctx, &referrals, query, referrerID)
	return referrals, err
}

func (r *Repository) CreditReferral(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	query := `UPDATE referrals SET status = 'credited', credited_at = $2 WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id, now)
	return err
}

func (r *Repository) GetReferralStats(ctx context.Context, referrerID int64) (*model.ReferralStats, error) {
	stats := &model.ReferralStats{}

	// Total referrals
	err := r.db.GetContext(ctx, &stats.TotalReferrals,
		"SELECT COUNT(*) FROM referrals WHERE referrer_id = $1", referrerID)
	if err != nil {
		return nil, err
	}

	// Pending referrals
	err = r.db.GetContext(ctx, &stats.PendingReferrals,
		"SELECT COUNT(*) FROM referrals WHERE referrer_id = $1 AND status = 'pending'", referrerID)
	if err != nil {
		return nil, err
	}

	// Credited bonus TON
	err = r.db.GetContext(ctx, &stats.CreditedBonusTON,
		"SELECT COALESCE(SUM(bonus_ton), 0) FROM referrals WHERE referrer_id = $1 AND status = 'credited'",
		referrerID)
	if err != nil {
		return nil, err
	}

	return stats, nil
}

func (r *Repository) GetReferredUsers(ctx context.Context, referrerID int64) ([]model.User, error) {
	var users []model.User
	query := `
		SELECT u.* FROM users u
		INNER JOIN referrals r ON r.referred_id = u.id
		WHERE r.referrer_id = $1
		ORDER BY r.created_at DESC`
	err := r.db.SelectContext(ctx, &users, query, referrerID)
	return users, err
}
