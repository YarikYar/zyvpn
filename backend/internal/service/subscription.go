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
	ErrNoServersAvailable    = errors.New("Нет доступных серверов")
)

type SubscriptionService struct {
	repo      *repository.Repository
	serverSvc *ServerService
	cfg       *config.Config
}

func NewSubscriptionService(repo *repository.Repository, serverSvc *ServerService, cfg *config.Config) *SubscriptionService {
	return &SubscriptionService{
		repo:      repo,
		serverSvc: serverSvc,
		cfg:       cfg,
	}
}

// SetServerService sets the server service (to avoid circular dependency)
func (s *SubscriptionService) SetServerService(serverSvc *ServerService) {
	s.serverSvc = serverSvc
}

// getXUIClientForSubscription returns the appropriate XUI client for a subscription
func (s *SubscriptionService) getXUIClientForSubscription(ctx context.Context, sub *model.Subscription) (*xui.Client, *model.Server, error) {
	if s.serverSvc == nil {
		return nil, nil, fmt.Errorf("server service not available")
	}

	// If subscription has a server_id, use it
	if sub.ServerID != nil {
		return s.serverSvc.GetXUIClient(ctx, *sub.ServerID)
	}

	// Fall back to default server for old subscriptions without server_id
	return s.serverSvc.GetXUIClientForDefault(ctx)
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
	return s.CreateSubscriptionWithServer(ctx, userID, plan, nil)
}

func (s *SubscriptionService) CreateSubscriptionWithServer(ctx context.Context, userID int64, plan *model.Plan, serverID *uuid.UUID) (*model.Subscription, error) {
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

	if s.serverSvc == nil {
		return nil, ErrNoServersAvailable
	}

	// Get XUI client for the selected server (or best available)
	var xuiClientAPI *xui.Client
	var server *model.Server

	if serverID != nil {
		xuiClientAPI, server, err = s.serverSvc.GetXUIClient(ctx, *serverID)
		if err != nil {
			return nil, fmt.Errorf("failed to get server: %w", err)
		}
	} else {
		// Auto-select best server based on load balancing
		server, err = s.serverSvc.GetBestServer(ctx)
		if err != nil {
			return nil, ErrNoServersAvailable
		}
		xuiClientAPI, server, err = s.serverSvc.GetXUIClient(ctx, server.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get server client: %w", err)
		}
	}

	// Generate unique email for 3x-ui client
	email := fmt.Sprintf("user_%d_%d", userID, time.Now().Unix())

	maxDevices := plan.MaxDevices
	if maxDevices <= 0 {
		maxDevices = 3
	}

	log.Printf("Creating VPN client for user %d, email: %s, traffic: %d GB, days: %d, devices: %d", userID, email, plan.TrafficGB, plan.DurationDays, maxDevices)

	// Create client in 3x-ui
	xuiClient, err := xuiClientAPI.AddClient(email, int64(plan.TrafficGB), plan.DurationDays, maxDevices)
	if err != nil {
		log.Printf("ERROR: Failed to create VPN client for user %d: %v", userID, err)
		return nil, fmt.Errorf("failed to create VPN client: %w", err)
	}

	log.Printf("VPN client created successfully: ID=%s, Email=%s", xuiClient.ID, xuiClient.Email)

	now := time.Now()
	expiresAt := now.Add(time.Duration(plan.DurationDays) * 24 * time.Hour)

	// Generate connection key
	connectionKey := s.serverSvc.GenerateConnectionKey(server, xuiClient.ID, email)

	sub := &model.Subscription{
		UserID:        userID,
		PlanID:        plan.ID,
		ServerID:      &server.ID,
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
		_ = xuiClientAPI.DeleteClient(xuiClient.ID)
		return nil, err
	}

	// Increment server load
	if err := s.serverSvc.IncrementLoad(ctx, server.ID); err != nil {
		log.Printf("WARNING: Failed to increment server load: %v", err)
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

	// Get appropriate XUI client for this subscription
	xuiClientAPI, _, err := s.getXUIClientForSubscription(ctx, sub)
	if err != nil {
		return fmt.Errorf("failed to get XUI client: %w", err)
	}

	// Update in 3x-ui FIRST (before database, so we can fail early)
	newExpiry := sub.ExpiresAt.Add(time.Duration(days) * 24 * time.Hour)
	maxDevices := sub.MaxDevices
	if maxDevices <= 0 {
		maxDevices = 3
	}
	if err := xuiClientAPI.UpdateClientTraffic(sub.XUIClientID, sub.XUIEmail, sub.TrafficLimit/(1024*1024*1024), newExpiry.UnixMilli(), maxDevices); err != nil {
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
		xuiClientAPI, _, err := s.getXUIClientForSubscription(ctx, sub)
		if err != nil {
			log.Printf("WARNING: Failed to get XUI client for subscription %s: %v", subID, err)
		} else {
			if err := xuiClientAPI.DeleteClient(sub.XUIClientID); err != nil {
				return fmt.Errorf("failed to delete VPN client: %w", err)
			}
		}
	}

	// Decrement server load
	if sub.ServerID != nil && s.serverSvc != nil {
		if err := s.serverSvc.DecrementLoad(ctx, *sub.ServerID); err != nil {
			log.Printf("WARNING: Failed to decrement server load: %v", err)
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
		xuiClientAPI, _, err := s.getXUIClientForSubscription(ctx, sub)
		if err != nil {
			log.Printf("WARNING: Failed to get XUI client for subscription %s: %v", subID, err)
		} else {
			if err := xuiClientAPI.DeleteClient(sub.XUIClientID); err != nil {
				// Log error but continue with expiration
				log.Printf("Failed to delete VPN client %s: %v", sub.XUIClientID, err)
			}
		}
	}

	// Decrement server load
	if sub.ServerID != nil && s.serverSvc != nil {
		if err := s.serverSvc.DecrementLoad(ctx, *sub.ServerID); err != nil {
			log.Printf("WARNING: Failed to decrement server load: %v", err)
		}
	}

	return s.repo.UpdateSubscriptionStatus(ctx, subID, model.SubscriptionStatusExpired)
}

func (s *SubscriptionService) SyncTraffic(ctx context.Context, subID uuid.UUID) error {
	sub, err := s.repo.GetSubscription(ctx, subID)
	if err != nil {
		return err
	}

	xuiClientAPI, _, err := s.getXUIClientForSubscription(ctx, sub)
	if err != nil {
		return fmt.Errorf("failed to get XUI client: %w", err)
	}

	traffic, err := xuiClientAPI.GetClientTraffic(sub.XUIEmail)
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

// SwitchServer switches an active subscription to a different server
func (s *SubscriptionService) SwitchServer(ctx context.Context, userID int64, newServerID uuid.UUID) (*model.Subscription, error) {
	// Get current active subscription
	sub, err := s.repo.GetActiveSubscription(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("no active subscription found")
	}

	if !sub.IsActive() {
		return nil, ErrSubscriptionNotActive
	}

	// Check if already on this server
	if sub.ServerID != nil && *sub.ServerID == newServerID {
		return sub, nil // Already on this server
	}

	// Get the new server
	newXUIClient, newServer, err := s.serverSvc.GetXUIClient(ctx, newServerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get new server: %w", err)
	}

	if !newServer.IsOnline() {
		return nil, fmt.Errorf("selected server is not available")
	}

	// Delete client from old server
	if sub.XUIClientID != "" {
		oldXUIClient, _, err := s.getXUIClientForSubscription(ctx, sub)
		if err != nil {
			log.Printf("WARNING: Failed to get old XUI client: %v", err)
		} else {
			if err := oldXUIClient.DeleteClient(sub.XUIClientID); err != nil {
				log.Printf("WARNING: Failed to delete client from old server: %v", err)
			}
		}
	}

	// Decrement old server load
	if sub.ServerID != nil {
		if err := s.serverSvc.DecrementLoad(ctx, *sub.ServerID); err != nil {
			log.Printf("WARNING: Failed to decrement old server load: %v", err)
		}
	}

	// Calculate remaining time and traffic
	remainingDays := int(time.Until(*sub.ExpiresAt).Hours() / 24)
	if remainingDays < 1 {
		remainingDays = 1
	}
	remainingTrafficGB := int((sub.TrafficLimit - sub.TrafficUsed) / (1024 * 1024 * 1024))
	if remainingTrafficGB < 1 {
		remainingTrafficGB = 1
	}

	// Generate new email for XUI client
	email := fmt.Sprintf("user_%d_%d", userID, time.Now().Unix())

	maxDevices := sub.MaxDevices
	if maxDevices <= 0 {
		maxDevices = 3
	}

	log.Printf("Switching user %d to server %s, email: %s, traffic: %d GB, days: %d", userID, newServer.Name, email, remainingTrafficGB, remainingDays)

	// Create client on new server
	newClient, err := newXUIClient.AddClient(email, int64(remainingTrafficGB), remainingDays, maxDevices)
	if err != nil {
		log.Printf("ERROR: Failed to create VPN client on new server: %v", err)
		return nil, fmt.Errorf("failed to create VPN client on new server: %w", err)
	}

	log.Printf("VPN client created on new server: ID=%s, Email=%s", newClient.ID, newClient.Email)

	// Generate new connection key
	connectionKey := s.serverSvc.GenerateConnectionKey(newServer, newClient.ID, email)

	// Update subscription in database
	sub.ServerID = &newServer.ID
	sub.XUIClientID = newClient.ID
	sub.XUIEmail = email
	sub.ConnectionKey = connectionKey

	if err := s.repo.UpdateSubscriptionServer(ctx, sub.ID, newServer.ID, newClient.ID, email, connectionKey); err != nil {
		// Try to cleanup new client
		_ = newXUIClient.DeleteClient(newClient.ID)
		return nil, fmt.Errorf("failed to update subscription: %w", err)
	}

	// Increment new server load
	if err := s.serverSvc.IncrementLoad(ctx, newServer.ID); err != nil {
		log.Printf("WARNING: Failed to increment new server load: %v", err)
	}

	return sub, nil
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
