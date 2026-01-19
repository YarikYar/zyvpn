package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/zyvpn/backend/internal/config"
	"github.com/zyvpn/backend/internal/model"
	"github.com/zyvpn/backend/internal/repository"
	"github.com/zyvpn/backend/internal/xui"
)

var (
	ErrSubscriptionActive    = errors.New("У пользователя уже есть активная подписка")
	ErrSubscriptionNotActive = errors.New("Подписка неактивна")
	ErrTrialAlreadyUsed      = errors.New("Пробный период уже использован")
)

type SubscriptionService struct {
	repo        *repository.Repository
	xuiClient   *xui.Client
	cfg         *config.Config
	inboundInfo *xui.InboundInfo
}

func NewSubscriptionService(repo *repository.Repository, xuiClient *xui.Client, cfg *config.Config) *SubscriptionService {
	svc := &SubscriptionService{
		repo:      repo,
		xuiClient: xuiClient,
		cfg:       cfg,
	}

	// Try to load inbound info from 3x-ui
	info, err := xuiClient.GetInboundInfo()
	if err != nil {
		log.Printf("WARNING: Failed to get inbound info from 3x-ui: %v", err)
		log.Printf("Using config values for VPN server settings")
	} else {
		svc.inboundInfo = info
		log.Printf("Loaded inbound info: Port=%d, PublicKey=%s..., ShortID=%s, ServerName=%s",
			info.Port,
			info.PublicKey[:min(10, len(info.PublicKey))],
			info.ShortID,
			info.ServerName,
		)
	}

	return svc
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (s *SubscriptionService) GetActiveSubscription(ctx context.Context, userID int64) (*model.Subscription, error) {
	return s.repo.GetActiveSubscription(ctx, userID)
}

func (s *SubscriptionService) GetSubscription(ctx context.Context, id uuid.UUID) (*model.Subscription, error) {
	return s.repo.GetSubscription(ctx, id)
}

func (s *SubscriptionService) CreateSubscription(ctx context.Context, userID int64, plan *model.Plan) (*model.Subscription, error) {
	// Check for existing active subscription - extend it instead of creating new
	existing, err := s.repo.GetActiveSubscription(ctx, userID)
	if err == nil && existing.IsActive() {
		// Extend existing subscription
		log.Printf("Extending existing subscription %s for user %d by %d days", existing.ID, userID, plan.DurationDays)
		if err := s.ExtendSubscription(ctx, existing.ID, plan.DurationDays); err != nil {
			return nil, fmt.Errorf("failed to extend subscription: %w", err)
		}
		// Return updated subscription
		return s.repo.GetSubscription(ctx, existing.ID)
	}

	// Generate unique email for 3x-ui client
	email := fmt.Sprintf("user_%d_%d", userID, time.Now().Unix())

	maxDevices := plan.MaxDevices
	if maxDevices <= 0 {
		maxDevices = 3
	}

	log.Printf("Creating VPN client for user %d, email: %s, traffic: %d GB, days: %d, devices: %d", userID, email, plan.TrafficGB, plan.DurationDays, maxDevices)

	// Create client in 3x-ui
	xuiClient, err := s.xuiClient.AddClient(email, int64(plan.TrafficGB), plan.DurationDays, maxDevices)
	if err != nil {
		log.Printf("ERROR: Failed to create VPN client for user %d: %v", userID, err)
		return nil, fmt.Errorf("failed to create VPN client: %w", err)
	}

	log.Printf("VPN client created successfully: ID=%s, Email=%s", xuiClient.ID, xuiClient.Email)

	now := time.Now()
	expiresAt := now.Add(time.Duration(plan.DurationDays) * 24 * time.Hour)

	// Generate connection key (this would need actual server details)
	connectionKey := s.generateConnectionKey(xuiClient.ID, email)

	sub := &model.Subscription{
		UserID:        userID,
		PlanID:        plan.ID,
		Status:        model.SubscriptionStatusActive,
		XUIClientID:   xuiClient.ID,
		XUIEmail:      email,
		ConnectionKey: connectionKey,
		StartedAt:     &now,
		ExpiresAt:     &expiresAt,
		TrafficLimit:  plan.TrafficBytes(),
		TrafficUsed:   0,
		MaxDevices:    maxDevices,
	}

	if err := s.repo.CreateSubscription(ctx, sub); err != nil {
		// Try to cleanup 3x-ui client
		_ = s.xuiClient.DeleteClient(xuiClient.ID)
		return nil, err
	}

	return sub, nil
}

func (s *SubscriptionService) ExtendSubscription(ctx context.Context, subID uuid.UUID, days int) error {
	sub, err := s.repo.GetSubscription(ctx, subID)
	if err != nil {
		return err
	}

	if sub.Status != model.SubscriptionStatusActive {
		return ErrSubscriptionNotActive
	}

	// Update in 3x-ui FIRST (before database, so we can fail early)
	newExpiry := sub.ExpiresAt.Add(time.Duration(days) * 24 * time.Hour)
	maxDevices := sub.MaxDevices
	if maxDevices <= 0 {
		maxDevices = 3
	}
	if err := s.xuiClient.UpdateClientTraffic(sub.XUIClientID, sub.XUIEmail, sub.TrafficLimit/(1024*1024*1024), newExpiry.UnixMilli(), maxDevices); err != nil {
		return fmt.Errorf("failed to update VPN client: %w", err)
	}

	// Only extend in database after 3x-ui succeeded
	if err := s.repo.ExtendSubscription(ctx, subID, days); err != nil {
		return err
	}

	return nil
}

func (s *SubscriptionService) CancelSubscription(ctx context.Context, subID uuid.UUID) error {
	sub, err := s.repo.GetSubscription(ctx, subID)
	if err != nil {
		return err
	}

	// Delete from 3x-ui
	if sub.XUIClientID != "" {
		if err := s.xuiClient.DeleteClient(sub.XUIClientID); err != nil {
			return fmt.Errorf("failed to delete VPN client: %w", err)
		}
	}

	return s.repo.UpdateSubscriptionStatus(ctx, subID, model.SubscriptionStatusCancelled)
}

func (s *SubscriptionService) ExpireSubscription(ctx context.Context, subID uuid.UUID) error {
	sub, err := s.repo.GetSubscription(ctx, subID)
	if err != nil {
		return err
	}

	// Delete from 3x-ui
	if sub.XUIClientID != "" {
		if err := s.xuiClient.DeleteClient(sub.XUIClientID); err != nil {
			// Log error but continue with expiration
			fmt.Printf("Failed to delete VPN client %s: %v\n", sub.XUIClientID, err)
		}
	}

	return s.repo.UpdateSubscriptionStatus(ctx, subID, model.SubscriptionStatusExpired)
}

func (s *SubscriptionService) SyncTraffic(ctx context.Context, subID uuid.UUID) error {
	sub, err := s.repo.GetSubscription(ctx, subID)
	if err != nil {
		return err
	}

	traffic, err := s.xuiClient.GetClientTraffic(sub.XUIEmail)
	if err != nil {
		return fmt.Errorf("failed to get traffic: %w", err)
	}

	totalUsed := traffic.Up + traffic.Down
	return s.repo.UpdateSubscriptionTraffic(ctx, subID, totalUsed)
}

func (s *SubscriptionService) GetConnectionKey(ctx context.Context, userID int64) (string, error) {
	sub, err := s.repo.GetActiveSubscription(ctx, userID)
	if err != nil {
		return "", err
	}

	if !sub.IsActive() {
		return "", ErrSubscriptionNotActive
	}

	return sub.ConnectionKey, nil
}

func (s *SubscriptionService) ProcessExpiredSubscriptions(ctx context.Context) error {
	expired, err := s.repo.GetExpiredSubscriptions(ctx)
	if err != nil {
		return err
	}

	for _, sub := range expired {
		if err := s.ExpireSubscription(ctx, sub.ID); err != nil {
			fmt.Printf("Failed to expire subscription %s: %v\n", sub.ID, err)
		}
	}

	return nil
}

func (s *SubscriptionService) GetExpiringSubscriptions(ctx context.Context, withinHours int) ([]model.Subscription, error) {
	before := time.Now().Add(time.Duration(withinHours) * time.Hour)
	return s.repo.GetExpiringSubscriptions(ctx, before)
}

func (s *SubscriptionService) generateConnectionKey(clientID, email string) string {
	var serverAddress string
	var serverPort int
	var publicKey, shortID, serverName string

	// Use inbound info if available, otherwise fall back to config
	if s.inboundInfo != nil {
		serverPort = s.inboundInfo.Port
		publicKey = s.inboundInfo.PublicKey
		shortID = s.inboundInfo.ShortID
		serverName = s.inboundInfo.ServerName
	} else {
		serverPort = s.cfg.XUI.ServerPort
		publicKey = s.cfg.XUI.PublicKey
		shortID = s.cfg.XUI.ShortID
		serverName = s.cfg.XUI.ServerName
	}

	// Server address must come from config (API doesn't know external IP)
	serverAddress = s.cfg.XUI.ServerAddress
	if serverAddress == "" {
		serverAddress = "YOUR_SERVER_IP"
	}

	return fmt.Sprintf("vless://%s@%s:%d?type=tcp&security=reality&pbk=%s&fp=chrome&sni=%s&sid=%s&spx=%%2F&flow=xtls-rprx-vision#%s",
		clientID,
		serverAddress,
		serverPort,
		publicKey,
		serverName,
		shortID,
		email,
	)
}

func (s *SubscriptionService) ActivateTrial(ctx context.Context, userID int64) (*model.Subscription, error) {
	// Check if user already has active subscription
	existing, err := s.repo.GetActiveSubscription(ctx, userID)
	if err == nil && existing.IsActive() {
		return nil, ErrSubscriptionActive
	}

	// Check if user already used trial
	hasUsedTrial, err := s.repo.HasUsedTrial(ctx, userID)
	if err != nil {
		return nil, err
	}
	if hasUsedTrial {
		return nil, ErrTrialAlreadyUsed
	}

	// Get trial plan
	plan, err := s.repo.GetTrialPlan(ctx)
	if err != nil {
		return nil, fmt.Errorf("trial plan not found: %w", err)
	}

	// Create subscription
	return s.CreateSubscription(ctx, userID, plan)
}
