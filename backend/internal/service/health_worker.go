package service

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/zyvpn/backend/internal/model"
	"github.com/zyvpn/backend/internal/repository"
)

const (
	HealthCheckInterval = 30 * time.Second
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

	w.checkAllServers(ctx)

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
	for i := range servers {
		srv := servers[i]
		if !srv.IsActive {
			continue
		}
		wg.Add(1)
		go func(s model.Server) {
			defer wg.Done()
			w.checkServer(ctx, s)
		}(srv)
	}
	wg.Wait()
}

// checkServer probes a server through its 3x-ui panel rather than TCP-pinging
// the VPN port. The VPN port often refuses bare TCP probes (Reality, IP
// allowlists, anti-scan firewalls), so the panel reachability is a more
// reliable proxy for "this server can issue keys / serve users".
func (w *HealthWorker) checkServer(ctx context.Context, srv model.Server) {
	id := srv.ID
	if id == uuid.Nil {
		return
	}

	xuiClient, _, err := w.serverSvc.GetXUIClient(ctx, id)
	if err != nil {
		log.Printf("[Health Worker] %s: get XUI client failed: %v", srv.Name, err)
		_ = w.repo.UpdateServerHealth(ctx, id, nil, "offline")
		return
	}

	start := time.Now()
	if _, err := xuiClient.GetInboundInfo(); err != nil {
		log.Printf("[Health Worker] %s: panel probe failed: %v", srv.Name, err)
		_ = w.repo.UpdateServerHealth(ctx, id, nil, "offline")
		return
	}
	pingMs := int(time.Since(start).Milliseconds())

	if err := w.repo.UpdateServerHealth(ctx, id, &pingMs, "online"); err != nil {
		log.Printf("[Health Worker] %s: update health failed: %v", srv.Name, err)
	}
}
