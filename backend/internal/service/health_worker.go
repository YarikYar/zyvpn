package service

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/zyvpn/backend/internal/repository"
)

const (
	HealthCheckInterval = 10 * time.Second
	PingTimeout         = 5 * time.Second
)

type HealthWorker struct {
	repo      *repository.Repository
	serverSvc *ServerService
}

func NewHealthWorker(repo *repository.Repository, serverSvc *ServerService) *HealthWorker {
	return &HealthWorker{
		repo:      repo,
		serverSvc: serverSvc,
	}
}

func (w *HealthWorker) Start(ctx context.Context) {
	log.Printf("[Health Worker] Started, checking every %v", HealthCheckInterval)

	// Initial check
	w.checkAllServers(ctx)

	// Sync server loads on startup
	if err := w.repo.SyncAllServerLoads(ctx); err != nil {
		log.Printf("[Health Worker] Failed to sync server loads: %v", err)
	}

	ticker := time.NewTicker(HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("[Health Worker] Stopped")
			return
		case <-ticker.C:
			w.checkAllServers(ctx)
		}
	}
}

func (w *HealthWorker) checkAllServers(ctx context.Context) {
	servers, err := w.repo.GetAllServers(ctx)
	if err != nil {
		log.Printf("[Health Worker] Failed to get servers: %v", err)
		return
	}

	if len(servers) == 0 {
		return
	}

	var wg sync.WaitGroup
	for _, server := range servers {
		if !server.IsActive {
			continue
		}

		wg.Add(1)
		go func(serverID, address string, port int) {
			defer wg.Done()
			w.checkServer(ctx, serverID, address, port)
		}(server.ID.String(), server.ServerAddress, server.ServerPort)
	}
	wg.Wait()
}

func (w *HealthWorker) checkServer(ctx context.Context, serverID, address string, port int) {
	id, err := uuid.Parse(serverID)
	if err != nil {
		log.Printf("[Health Worker] Invalid server ID %s: %v", serverID, err)
		return
	}

	// Measure TCP connection time as ping
	start := time.Now()
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", address, port), PingTimeout)
	pingMs := int(time.Since(start).Milliseconds())

	if err != nil {
		// Server is offline
		if updateErr := w.repo.UpdateServerHealth(ctx, id, nil, "offline"); updateErr != nil {
			log.Printf("[Health Worker] Failed to update server %s health: %v", serverID, updateErr)
		}
		return
	}
	conn.Close()

	// Server is online
	if updateErr := w.repo.UpdateServerHealth(ctx, id, &pingMs, "online"); updateErr != nil {
		log.Printf("[Health Worker] Failed to update server %s health: %v", serverID, updateErr)
	}
}
