package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/zyvpn/backend/internal/model"
	"github.com/zyvpn/backend/internal/repository"
)

var (
	ErrReferralAlreadyExists = errors.New("Реферал уже существует")
	ErrSelfReferral          = errors.New("Нельзя пригласить самого себя")
)

type ReferralService struct {
	repo *repository.Repository
}

func NewReferralService(repo *repository.Repository) *ReferralService {
	return &ReferralService{repo: repo}
}

func (s *ReferralService) CreateReferral(ctx context.Context, referrerID, referredID int64) error {
	if referrerID == referredID {
		return ErrSelfReferral
	}

	// Check if referral already exists
	_, err := s.repo.GetReferralByReferredID(ctx, referredID)
	if err == nil {
		return ErrReferralAlreadyExists
	}
	if err != repository.ErrReferralNotFound {
		return err
	}

	referral := &model.Referral{
		ReferrerID: referrerID,
		ReferredID: referredID,
		BonusType:  model.DefaultReferralBonusType,
		BonusValue: 0,
		BonusTON:   model.DefaultReferralBonusTON,
		Status:     model.ReferralStatusPending,
	}

	return s.repo.CreateReferral(ctx, referral)
}

// GetPendingReferral returns pending referral for a user (if exists)
func (s *ReferralService) GetPendingReferral(ctx context.Context, referredID int64) (*model.Referral, error) {
	referral, err := s.repo.GetReferralByReferredID(ctx, referredID)
	if err != nil {
		return nil, err
	}

	if referral.Status == model.ReferralStatusCredited {
		return nil, nil // Already credited
	}

	return referral, nil
}

// MarkReferralCredited marks referral as credited
func (s *ReferralService) MarkReferralCredited(ctx context.Context, referralID uuid.UUID) error {
	return s.repo.CreditReferral(ctx, referralID)
}

func (s *ReferralService) GetReferralStats(ctx context.Context, userID int64) (*model.ReferralStats, error) {
	return s.repo.GetReferralStats(ctx, userID)
}

func (s *ReferralService) GetReferralLink(ctx context.Context, userID int64, botUsername string) (string, error) {
	user, err := s.repo.GetUser(ctx, userID)
	if err != nil {
		return "", err
	}

	return "https://t.me/" + botUsername + "?start=ref_" + user.ReferralCode, nil
}

func (s *ReferralService) GetReferredUsers(ctx context.Context, referrerID int64) ([]model.User, error) {
	return s.repo.GetReferredUsers(ctx, referrerID)
}

func (s *ReferralService) ApplyReferralCode(ctx context.Context, userID int64, code string) error {
	// Find referrer by code
	referrer, err := s.repo.GetUserByReferralCode(ctx, code)
	if err != nil {
		return err
	}

	// Create referral relationship
	return s.CreateReferral(ctx, referrer.ID, userID)
}
