package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/zyvpn/backend/internal/model"
	"github.com/zyvpn/backend/internal/repository"
)

type BalanceService struct {
	repo *repository.Repository
}

func NewBalanceService(repo *repository.Repository) *BalanceService {
	return &BalanceService{repo: repo}
}

// GetBalance returns user's current balance in TON
func (s *BalanceService) GetBalance(ctx context.Context, userID int64) (float64, error) {
	return s.repo.GetUserBalance(ctx, userID)
}

// CreditReferralBonus adds referral bonus to user balance
func (s *BalanceService) CreditReferralBonus(ctx context.Context, userID int64, amount float64, referralID uuid.UUID) (float64, error) {
	description := fmt.Sprintf("Реферальный бонус: +%.4f TON", amount)
	return s.repo.UpdateBalance(ctx, userID, amount, model.TransactionTypeReferralBonus, description, &referralID)
}

// CreditGiveaway adds giveaway prize to user balance
func (s *BalanceService) CreditGiveaway(ctx context.Context, userID int64, amount float64, description string) (float64, error) {
	return s.repo.UpdateBalance(ctx, userID, amount, model.TransactionTypeGiveaway, description, nil)
}

// DebitForSubscription deducts amount for subscription payment
func (s *BalanceService) DebitForSubscription(ctx context.Context, userID int64, amount float64, paymentID uuid.UUID) (float64, error) {
	description := fmt.Sprintf("Оплата подписки: -%.4f TON", amount)
	return s.repo.UpdateBalance(ctx, userID, -amount, model.TransactionTypeSubscriptionPayment, description, &paymentID)
}

// CreditRefund adds refund to user balance
func (s *BalanceService) CreditRefund(ctx context.Context, userID int64, amount float64, paymentID uuid.UUID) (float64, error) {
	description := fmt.Sprintf("Возврат средств: +%.4f TON", amount)
	return s.repo.UpdateBalance(ctx, userID, amount, model.TransactionTypeRefund, description, &paymentID)
}

// CreditManual adds manual adjustment (admin operation)
func (s *BalanceService) CreditManual(ctx context.Context, userID int64, amount float64, description string) (float64, error) {
	return s.repo.UpdateBalance(ctx, userID, amount, model.TransactionTypeManual, description, nil)
}

// CreditTopUp adds balance from top-up with optional bonus
func (s *BalanceService) CreditTopUp(ctx context.Context, userID int64, amount float64, paymentID uuid.UUID) (float64, error) {
	// Get bonus percentage from settings
	bonusPercent, err := s.repo.GetSettingFloat(ctx, "topup_bonus_percent")
	if err != nil {
		bonusPercent = 0 // Default to no bonus if setting not found
	}

	// Calculate total with bonus
	bonusAmount := amount * bonusPercent / 100
	totalAmount := amount + bonusAmount

	var description string
	if bonusAmount > 0 {
		description = fmt.Sprintf("Пополнение баланса: +%.4f TON (+%.1f%% бонус = %.4f)", amount, bonusPercent, totalAmount)
	} else {
		description = fmt.Sprintf("Пополнение баланса: +%.4f TON", amount)
	}

	return s.repo.UpdateBalance(ctx, userID, totalAmount, model.TransactionTypeTopUp, description, &paymentID)
}

// CreditPromoCode adds balance from promo code
func (s *BalanceService) CreditPromoCode(ctx context.Context, userID int64, amount float64, promoCodeID uuid.UUID, code string) (float64, error) {
	description := fmt.Sprintf("Промокод %s: +%.4f TON", code, amount)
	return s.repo.UpdateBalance(ctx, userID, amount, model.TransactionTypePromoCode, description, &promoCodeID)
}

// GetTransactions returns balance transaction history
func (s *BalanceService) GetTransactions(ctx context.Context, userID int64, limit, offset int) ([]model.BalanceTransaction, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return s.repo.GetBalanceTransactions(ctx, userID, limit, offset)
}

// CanAfford checks if user has enough balance
func (s *BalanceService) CanAfford(ctx context.Context, userID int64, amount float64) (bool, error) {
	balance, err := s.repo.GetUserBalance(ctx, userID)
	if err != nil {
		return false, err
	}
	return balance >= amount, nil
}
