package service

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"strings"

	"github.com/zyvpn/backend/internal/model"
	"github.com/zyvpn/backend/internal/repository"
)

type UserService struct {
	repo *repository.Repository
}

func NewUserService(repo *repository.Repository) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) GetOrCreateUser(ctx context.Context, telegramUser TelegramUser) (*model.User, bool, error) {
	existingUser, err := s.repo.GetUser(ctx, telegramUser.ID)
	if err == nil {
		// Update user info if changed
		existingUser.Username = telegramUser.Username
		existingUser.FirstName = telegramUser.FirstName
		existingUser.LastName = telegramUser.LastName
		existingUser.LanguageCode = telegramUser.LanguageCode
		if err := s.repo.UpdateUser(ctx, existingUser); err != nil {
			return nil, false, err
		}
		return existingUser, false, nil
	}

	if err != repository.ErrUserNotFound {
		return nil, false, err
	}

	// Create new user
	referralCode, err := generateReferralCode()
	if err != nil {
		return nil, false, err
	}

	user := &model.User{
		ID:           telegramUser.ID,
		Username:     telegramUser.Username,
		FirstName:    telegramUser.FirstName,
		LastName:     telegramUser.LastName,
		LanguageCode: telegramUser.LanguageCode,
		ReferralCode: referralCode,
		ReferredBy:   telegramUser.ReferredBy,
	}

	if err := s.repo.CreateUser(ctx, user); err != nil {
		return nil, false, err
	}

	return user, true, nil
}

func (s *UserService) GetUser(ctx context.Context, id int64) (*model.User, error) {
	return s.repo.GetUser(ctx, id)
}

func (s *UserService) GetUserWithSubscription(ctx context.Context, id int64) (*model.UserWithSubscription, error) {
	return s.repo.GetUserWithSubscription(ctx, id)
}

func (s *UserService) GetUserByReferralCode(ctx context.Context, code string) (*model.User, error) {
	return s.repo.GetUserByReferralCode(ctx, code)
}

type TelegramUser struct {
	ID           int64
	Username     *string
	FirstName    *string
	LastName     *string
	LanguageCode *string
	ReferredBy   *int64
}

func generateReferralCode() (string, error) {
	bytes := make([]byte, 5)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	code := base32.StdEncoding.EncodeToString(bytes)
	code = strings.TrimRight(code, "=")
	return strings.ToLower(code[:8]), nil
}
