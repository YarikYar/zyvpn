package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/zyvpn/backend/internal/model"
)

var ErrSubscriptionNotFound = errors.New("subscription not found")

func (r *Repository) GetSubscription(ctx context.Context, id uuid.UUID) (*model.Subscription, error) {
	var sub model.Subscription
	err := r.db.GetContext(ctx, &sub, "SELECT * FROM subscriptions WHERE id = $1", id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSubscriptionNotFound
		}
		return nil, err
	}
	return &sub, nil
}

func (r *Repository) GetActiveSubscription(ctx context.Context, userID int64) (*model.Subscription, error) {
	var sub model.Subscription
	query := `
		SELECT * FROM subscriptions
		WHERE user_id = $1 AND status = 'active'
		ORDER BY created_at DESC
		LIMIT 1`

	err := r.db.GetContext(ctx, &sub, query, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSubscriptionNotFound
		}
		return nil, err
	}
	return &sub, nil
}

func (r *Repository) GetUserSubscriptions(ctx context.Context, userID int64) ([]model.Subscription, error) {
	var subs []model.Subscription
	query := "SELECT * FROM subscriptions WHERE user_id = $1 ORDER BY created_at DESC"
	err := r.db.SelectContext(ctx, &subs, query, userID)
	return subs, err
}

func (r *Repository) CreateSubscription(ctx context.Context, sub *model.Subscription) error {
	query := `
		INSERT INTO subscriptions (
			user_id, plan_id, status, xui_client_id, xui_email, connection_key,
			started_at, expires_at, traffic_limit, traffic_used, max_devices
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, created_at`

	return r.db.QueryRowContext(ctx, query,
		sub.UserID,
		sub.PlanID,
		sub.Status,
		sub.XUIClientID,
		sub.XUIEmail,
		sub.ConnectionKey,
		sub.StartedAt,
		sub.ExpiresAt,
		sub.TrafficLimit,
		sub.TrafficUsed,
		sub.MaxDevices,
	).Scan(&sub.ID, &sub.CreatedAt)
}

func (r *Repository) UpdateSubscription(ctx context.Context, sub *model.Subscription) error {
	query := `
		UPDATE subscriptions SET
			status = $2,
			xui_client_id = $3,
			xui_email = $4,
			connection_key = $5,
			started_at = $6,
			expires_at = $7,
			traffic_limit = $8,
			traffic_used = $9
		WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query,
		sub.ID,
		sub.Status,
		sub.XUIClientID,
		sub.XUIEmail,
		sub.ConnectionKey,
		sub.StartedAt,
		sub.ExpiresAt,
		sub.TrafficLimit,
		sub.TrafficUsed,
	)
	return err
}

func (r *Repository) UpdateSubscriptionStatus(ctx context.Context, id uuid.UUID, status model.SubscriptionStatus) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE subscriptions SET status = $2 WHERE id = $1",
		id, status,
	)
	return err
}

func (r *Repository) UpdateSubscriptionTraffic(ctx context.Context, id uuid.UUID, trafficUsed int64) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE subscriptions SET traffic_used = $2 WHERE id = $1",
		id, trafficUsed,
	)
	return err
}

func (r *Repository) GetExpiredSubscriptions(ctx context.Context) ([]model.Subscription, error) {
	var subs []model.Subscription
	query := `
		SELECT * FROM subscriptions
		WHERE status = 'active' AND expires_at < $1`
	err := r.db.SelectContext(ctx, &subs, query, time.Now())
	return subs, err
}

func (r *Repository) GetExpiringSubscriptions(ctx context.Context, before time.Time) ([]model.Subscription, error) {
	var subs []model.Subscription
	query := `
		SELECT * FROM subscriptions
		WHERE status = 'active'
			AND expires_at > $1
			AND expires_at < $2`
	err := r.db.SelectContext(ctx, &subs, query, time.Now(), before)
	return subs, err
}

func (r *Repository) ExtendSubscription(ctx context.Context, id uuid.UUID, days int) error {
	query := `
		UPDATE subscriptions SET
			expires_at = expires_at + interval '1 day' * $2
		WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id, days)
	return err
}

func (r *Repository) HasUsedTrial(ctx context.Context, userID int64) (bool, error) {
	var count int
	query := `
		SELECT COUNT(*) FROM subscriptions s
		JOIN plans p ON s.plan_id = p.id
		WHERE s.user_id = $1 AND p.price_usd = 0`
	err := r.db.GetContext(ctx, &count, query, userID)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
