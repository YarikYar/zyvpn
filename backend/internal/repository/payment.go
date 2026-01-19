package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/zyvpn/backend/internal/model"
)

var ErrPaymentNotFound = errors.New("payment not found")

func (r *Repository) GetPayment(ctx context.Context, id uuid.UUID) (*model.Payment, error) {
	var payment model.Payment
	err := r.db.GetContext(ctx, &payment, "SELECT * FROM payments WHERE id = $1", id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPaymentNotFound
		}
		return nil, err
	}
	return &payment, nil
}

func (r *Repository) GetPendingPayment(ctx context.Context, userID int64, planID uuid.UUID) (*model.Payment, error) {
	var payment model.Payment
	query := `
		SELECT * FROM payments
		WHERE user_id = $1 AND plan_id = $2 AND status = 'pending'
		ORDER BY created_at DESC
		LIMIT 1`
	err := r.db.GetContext(ctx, &payment, query, userID, planID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPaymentNotFound
		}
		return nil, err
	}
	return &payment, nil
}

func (r *Repository) GetUserPayments(ctx context.Context, userID int64) ([]model.Payment, error) {
	var payments []model.Payment
	query := "SELECT * FROM payments WHERE user_id = $1 ORDER BY created_at DESC"
	err := r.db.SelectContext(ctx, &payments, query, userID)
	return payments, err
}

func (r *Repository) CreatePayment(ctx context.Context, payment *model.Payment) error {
	query := `
		INSERT INTO payments (user_id, plan_id, server_id, payment_type, provider, amount, currency, status, external_id, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at`

	return r.db.QueryRowContext(ctx, query,
		payment.UserID,
		payment.PlanID,
		payment.ServerID,
		payment.PaymentType,
		payment.Provider,
		payment.Amount,
		payment.Currency,
		payment.Status,
		payment.ExternalID,
		payment.Metadata,
	).Scan(&payment.ID, &payment.CreatedAt)
}

func (r *Repository) UpdatePaymentStatus(ctx context.Context, id uuid.UUID, status model.PaymentStatus) error {
	var completedAt *time.Time
	if status == model.PaymentStatusCompleted {
		now := time.Now()
		completedAt = &now
	}

	query := `UPDATE payments SET status = $2, completed_at = $3 WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id, status, completedAt)
	return err
}

func (r *Repository) UpdatePaymentSubscription(ctx context.Context, paymentID uuid.UUID, subscriptionID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE payments SET subscription_id = $2 WHERE id = $1",
		paymentID, subscriptionID,
	)
	return err
}

func (r *Repository) UpdatePaymentExternalID(ctx context.Context, id uuid.UUID, externalID string) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE payments SET external_id = $2 WHERE id = $1",
		id, externalID,
	)
	return err
}

func (r *Repository) GetPaymentByExternalID(ctx context.Context, externalID string) (*model.Payment, error) {
	var payment model.Payment
	query := "SELECT * FROM payments WHERE external_id = $1"
	err := r.db.GetContext(ctx, &payment, query, externalID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPaymentNotFound
		}
		return nil, err
	}
	return &payment, nil
}

func (r *Repository) GetPendingPayments(ctx context.Context, olderThan time.Duration) ([]model.Payment, error) {
	var payments []model.Payment
	query := `
		SELECT * FROM payments
		WHERE status = 'pending' AND created_at < $1`
	err := r.db.SelectContext(ctx, &payments, query, time.Now().Add(-olderThan))
	return payments, err
}

func (r *Repository) HasCompletedPayment(ctx context.Context, userID int64) (bool, error) {
	var count int
	query := "SELECT COUNT(*) FROM payments WHERE user_id = $1 AND status = 'completed'"
	err := r.db.GetContext(ctx, &count, query, userID)
	return count > 0, err
}

// GetAwaitingTxPayments returns TON payments waiting for blockchain confirmation
func (r *Repository) GetAwaitingTxPayments(ctx context.Context) ([]model.Payment, error) {
	var payments []model.Payment
	query := `
		SELECT * FROM payments
		WHERE status = 'awaiting_tx' AND provider = 'ton'
		ORDER BY created_at ASC`
	err := r.db.SelectContext(ctx, &payments, query)
	return payments, err
}

// CreateTopUpPayment creates a balance top-up payment (no plan_id)
func (r *Repository) CreateTopUpPayment(ctx context.Context, payment *model.Payment) error {
	query := `
		INSERT INTO payments (user_id, payment_type, provider, amount, currency, status, external_id, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at`

	return r.db.QueryRowContext(ctx, query,
		payment.UserID,
		payment.PaymentType,
		payment.Provider,
		payment.Amount,
		payment.Currency,
		payment.Status,
		payment.ExternalID,
		payment.Metadata,
	).Scan(&payment.ID, &payment.CreatedAt)
}
