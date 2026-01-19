package service

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/zyvpn/backend/internal/model"
	"github.com/zyvpn/backend/internal/repository"
	"github.com/zyvpn/backend/internal/xui"
)

type ServerService struct {
	repo    *repository.Repository
	clients map[uuid.UUID]*xui.Client
	mu      sync.RWMutex
}

func NewServerService(repo *repository.Repository) *ServerService {
	return &ServerService{
		repo:    repo,
		clients: make(map[uuid.UUID]*xui.Client),
	}
}

// GetActiveServers returns all active servers for users
func (s *ServerService) GetActiveServers(ctx context.Context) ([]model.ServerPublic, error) {
	servers, err := s.repo.GetActiveServers(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]model.ServerPublic, len(servers))
	for i, srv := range servers {
		result[i] = srv.ToPublic()
	}
	return result, nil
}

// GetAllServers returns all servers for admin
func (s *ServerService) GetAllServers(ctx context.Context) ([]model.ServerAdmin, error) {
	servers, err := s.repo.GetAllServers(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]model.ServerAdmin, len(servers))
	for i, srv := range servers {
		result[i] = srv.ToAdmin()
	}
	return result, nil
}

// GetServer returns a server by ID
func (s *ServerService) GetServer(ctx context.Context, id uuid.UUID) (*model.Server, error) {
	return s.repo.GetServer(ctx, id)
}

// GetDefaultServer returns the default server
func (s *ServerService) GetDefaultServer(ctx context.Context) (*model.Server, error) {
	return s.repo.GetDefaultServer(ctx)
}

// CreateServer creates a new server
func (s *ServerService) CreateServer(ctx context.Context, server *model.Server) error {
	return s.repo.CreateServer(ctx, server)
}

// UpdateServer updates a server
func (s *ServerService) UpdateServer(ctx context.Context, server *model.Server) error {
	// Invalidate cached client
	s.mu.Lock()
	delete(s.clients, server.ID)
	s.mu.Unlock()

	return s.repo.UpdateServer(ctx, server)
}

// DeleteServer deletes a server
func (s *ServerService) DeleteServer(ctx context.Context, id uuid.UUID) error {
	// Invalidate cached client
	s.mu.Lock()
	delete(s.clients, id)
	s.mu.Unlock()

	return s.repo.DeleteServer(ctx, id)
}

// GetXUIClient returns an XUI client for a specific server (with caching)
func (s *ServerService) GetXUIClient(ctx context.Context, serverID uuid.UUID) (*xui.Client, *model.Server, error) {
	// Check cache first
	s.mu.RLock()
	client, exists := s.clients[serverID]
	s.mu.RUnlock()

	server, err := s.repo.GetServer(ctx, serverID)
	if err != nil {
		return nil, nil, err
	}

	if exists && client != nil {
		return client, server, nil
	}

	// Create new client
	client, err = xui.NewClient(server.XUIBaseURL, server.XUIUsername, server.XUIPassword, server.XUIInboundID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create XUI client: %w", err)
	}

	// Cache it
	s.mu.Lock()
	s.clients[serverID] = client
	s.mu.Unlock()

	return client, server, nil
}

// GetXUIClientForDefault returns XUI client for default server
func (s *ServerService) GetXUIClientForDefault(ctx context.Context) (*xui.Client, *model.Server, error) {
	server, err := s.repo.GetDefaultServer(ctx)
	if err != nil {
		return nil, nil, err
	}
	return s.GetXUIClient(ctx, server.ID)
}

// GenerateConnectionKey generates a VLESS connection key for a subscription
func (s *ServerService) GenerateConnectionKey(server *model.Server, clientID, email string) string {
	return fmt.Sprintf("vless://%s@%s:%d?type=tcp&security=reality&pbk=%s&fp=chrome&sni=%s&sid=%s&spx=%%2F&flow=xtls-rprx-vision#%s",
		clientID,
		server.ServerAddress,
		server.ServerPort,
		server.PublicKey,
		server.ServerName,
		server.ShortID,
		email,
	)
}

// GetBestServer returns the best available server based on load balancing
func (s *ServerService) GetBestServer(ctx context.Context) (*model.Server, error) {
	return s.repo.GetBestServer(ctx)
}

// GetOnlineServers returns all online active servers
func (s *ServerService) GetOnlineServers(ctx context.Context) ([]model.ServerPublic, error) {
	servers, err := s.repo.GetOnlineServers(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]model.ServerPublic, len(servers))
	for i, srv := range servers {
		result[i] = srv.ToPublic()
	}
	return result, nil
}

// IncrementLoad increments server load when subscription is created
func (s *ServerService) IncrementLoad(ctx context.Context, serverID uuid.UUID) error {
	return s.repo.IncrementServerLoad(ctx, serverID)
}

// DecrementLoad decrements server load when subscription is cancelled/expired
func (s *ServerService) DecrementLoad(ctx context.Context, serverID uuid.UUID) error {
	return s.repo.DecrementServerLoad(ctx, serverID)
}

// UpdateServerHealth updates server health status
func (s *ServerService) UpdateServerHealth(ctx context.Context, serverID uuid.UUID, pingMs *int, status string) error {
	return s.repo.UpdateServerHealth(ctx, serverID, pingMs, status)
}
