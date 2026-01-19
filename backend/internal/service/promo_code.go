package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/zyvpn/backend/internal/model"
	"github.com/zyvpn/backend/internal/repository"
)

var (
	ErrPromoCodeNotFound          = errors.New("Промокод не найден")
	ErrPromoCodeExpired           = errors.New("Срок действия промокода истёк")
	ErrPromoCodeUsageLimitReached = errors.New("Лимит использований промокода исчерпан")
	ErrPromoCodeAlreadyUsed       = errors.New("Вы уже использовали этот промокод")
	ErrPromoCodeInactive          = errors.New("Промокод неактивен")
	ErrNoActiveSubscription       = errors.New("Нет активной подписки для продления")
)

type PromoCodeService struct {
	repo           *repository.Repository
	balanceSvc     *BalanceService
	subscriptionSvc *SubscriptionService
}

func NewPromoCodeService(repo *repository.Repository) *PromoCodeService {
	return &PromoCodeService{repo: repo}
}

// SetBalanceService sets the balance service (to avoid circular deps)
func (s *PromoCodeService) SetBalanceService(balanceSvc *BalanceService) {
	s.balanceSvc = balanceSvc
}

// SetSubscriptionService sets the subscription service (to avoid circular deps)
func (s *PromoCodeService) SetSubscriptionService(subscriptionSvc *SubscriptionService) {
	s.subscriptionSvc = subscriptionSvc
}

// ValidatePromoCode checks if a promo code is valid for a user
func (s *PromoCodeService) ValidatePromoCode(ctx context.Context, code string, userID int64) (*model.PromoCode, error) {
	promo, err := s.repo.GetPromoCodeByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	if promo == nil {
		return nil, ErrPromoCodeNotFound
	}

	if !promo.IsActive {
		return nil, ErrPromoCodeInactive
	}

	if !promo.IsValid() {
		if promo.ExpiresAt != nil {
			return nil, ErrPromoCodeExpired
		}
		return nil, ErrPromoCodeUsageLimitReached
	}

	// Check if user already used this code
	used, err := s.repo.HasUserUsedPromoCode(ctx, userID, promo.ID)
	if err != nil {
		return nil, err
	}
	if used {
		return nil, ErrPromoCodeAlreadyUsed
	}

	return promo, nil
}

// ApplyResult holds the result of applying a promo code
type ApplyResult struct {
	Type       model.PromoCodeType `json:"type"`
	Value      float64             `json:"value"`
	NewBalance *float64            `json:"new_balance,omitempty"`
	Message    string              `json:"message"`
}

// ApplyPromoCode applies a promo code to a user
func (s *PromoCodeService) ApplyPromoCode(ctx context.Context, code string, userID int64) (*ApplyResult, error) {
	promo, err := s.ValidatePromoCode(ctx, code, userID)
	if err != nil {
		return nil, err
	}

	result := &ApplyResult{
		Type:  promo.Type,
		Value: promo.Value,
	}

	switch promo.Type {
	case model.PromoCodeTypeBalance:
		if s.balanceSvc == nil {
			return nil, errors.New("balance service not configured")
		}
		newBalance, err := s.balanceSvc.CreditPromoCode(ctx, userID, promo.Value, promo.ID, promo.Code)
		if err != nil {
			return nil, fmt.Errorf("failed to credit balance: %w", err)
		}
		result.NewBalance = &newBalance
		result.Message = fmt.Sprintf("На ваш баланс зачислено %.4f TON", promo.Value)

	case model.PromoCodeTypeDays:
		if s.subscriptionSvc == nil {
			return nil, errors.New("subscription service not configured")
		}
		// Get active subscription
		sub, err := s.subscriptionSvc.GetActiveSubscription(ctx, userID)
		if err != nil {
			return nil, err
		}
		if sub == nil {
			return nil, ErrNoActiveSubscription
		}
		// Extend subscription
		if err := s.subscriptionSvc.ExtendSubscription(ctx, sub.ID, int(promo.Value)); err != nil {
			return nil, fmt.Errorf("failed to extend subscription: %w", err)
		}
		result.Message = fmt.Sprintf("Ваша подписка продлена на %d дней", int(promo.Value))

	case model.PromoCodeTypeRegionSwitch:
		// Add free region switches to user
		count := int(promo.Value)
		if count < 1 {
			count = 1
		}
		if err := s.repo.AddFreeRegionSwitches(ctx, userID, count); err != nil {
			return nil, fmt.Errorf("failed to add free region switches: %w", err)
		}
		if count == 1 {
			result.Message = "Вам начислена 1 бесплатная смена региона"
		} else {
			result.Message = fmt.Sprintf("Вам начислено %d бесплатных смен региона", count)
		}
	}

	// Mark promo code as used
	if err := s.repo.UsePromoCode(ctx, userID, promo.ID); err != nil {
		return nil, fmt.Errorf("failed to record promo code use: %w", err)
	}

	return result, nil
}

// CreatePromoCode creates a new promo code (admin function)
func (s *PromoCodeService) CreatePromoCode(ctx context.Context, promo *model.PromoCode) error {
	return s.repo.CreatePromoCode(ctx, promo)
}

// ListPromoCodes lists all promo codes (admin function)
func (s *PromoCodeService) ListPromoCodes(ctx context.Context, limit, offset int) ([]model.PromoCode, error) {
	if limit <= 0 {
		limit = 50
	}
	return s.repo.ListPromoCodes(ctx, limit, offset)
}

// DeactivatePromoCode deactivates a promo code (admin function)
func (s *PromoCodeService) DeactivatePromoCode(ctx context.Context, code string) error {
	promo, err := s.repo.GetPromoCodeByCode(ctx, code)
	if err != nil {
		return err
	}
	if promo == nil {
		return ErrPromoCodeNotFound
	}
	return s.repo.DeactivatePromoCode(ctx, promo.ID)
}
