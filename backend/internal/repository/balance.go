package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/zyvpn/backend/internal/model"
)

// GetUserBalance returns the current balance of a user
func (r *Repository) GetUserBalance(ctx context.Context, userID int64) (float64, error) {
	var balance float64
	err := r.db.GetContext(ctx, &balance, "SELECT balance FROM users WHERE id = $1", userID)
	return balance, err
}

// UpdateBalance updates user balance atomically and creates a transaction record
// Returns the new balance and error
func (r *Repository) UpdateBalance(ctx context.Context, userID int64, amount float64, txType model.TransactionType, description string, referenceID *uuid.UUID) (float64, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	// Get current balance with lock
	var balanceBefore float64
	err = tx.GetContext(ctx, &balanceBefore, "SELECT balance FROM users WHERE id = $1 FOR UPDATE", userID)
	if err != nil {
		return 0, fmt.Errorf("failed to get balance: %w", err)
	}

	balanceAfter := balanceBefore + amount

	// Check for negative balance (only for debits)
	if amount < 0 && balanceAfter < 0 {
		return balanceBefore, fmt.Errorf("insufficient balance: have %.9f, need %.9f", balanceBefore, -amount)
	}

	// Update balance
	_, err = tx.ExecContext(ctx, "UPDATE users SET balance = $1, updated_at = NOW() WHERE id = $2", balanceAfter, userID)
	if err != nil {
		return 0, fmt.Errorf("failed to update balance: %w", err)
	}

	// Create transaction record
	var desc *string
	if description != "" {
		desc = &description
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO balance_transactions (user_id, amount, type, description, reference_id, balance_before, balance_after)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		userID, amount, txType, desc, referenceID, balanceBefore, balanceAfter)
	if err != nil {
		return 0, fmt.Errorf("failed to create transaction record: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	return balanceAfter, nil
}

// GetBalanceTransactions returns balance transaction history for a user
func (r *Repository) GetBalanceTransactions(ctx context.Context, userID int64, limit, offset int) ([]model.BalanceTransaction, error) {
	var transactions []model.BalanceTransaction
	err := r.db.SelectContext(ctx, &transactions, `
		SELECT * FROM balance_transactions
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`,
		userID, limit, offset)
	return transactions, err
}
