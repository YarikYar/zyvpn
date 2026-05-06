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
	HealthCheckInterval     = 30 * time.Second
	TrafficSnapshotInterval = 5 * time.Minute
)

type HealthWorker struct {
	repo      *repository.Repository
	serverSvc *ServerService

	lastSnapshot   map[uuid.UUID]time.Time
	lastSnapshotMu sync.Mutex
}

func NewHealthWorker(repo *repository.Repository, serverSvc *ServerService) *HealthWorker {
	return &HealthWorker{
		repo:         repo,
		serverSvc:    serverSvc,
		lastSnapshot: make(map[uuid.UUID]time.Time),
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

// checkServer probes the XUI panel, updates health, records status-change
// events for uptime calc, and periodically snapshots traffic counters.
func (w *HealthWorker) checkServer(ctx context.Context, srv model.Server) {
	id := srv.ID
	if id == uuid.Nil {
		return
	}

	xuiClient, _, err := w.serverSvc.GetXUIClient(ctx, id)
	if err != nil {
		log.Printf("[Health Worker] %s: get XUI client failed: %v", srv.Name, err)
		w.recordStatus(ctx, id, "offline", nil)
		return
	}

	start := time.Now()
	inbound, err := xuiClient.GetInbound()
	if err != nil {
		log.Printf("[Health Worker] %s: panel probe failed: %v", srv.Name, err)
		w.recordStatus(ctx, id, "offline", nil)
		return
	}
	pingMs := int(time.Since(start).Milliseconds())
	w.recordStatus(ctx, id, "online", &pingMs)

	// Traffic snapshot — rate-limit per server to TrafficSnapshotInterval to
	// keep the table compact.
	w.lastSnapshotMu.Lock()
	prev, ok := w.lastSnapshot[id]
	due := !ok || time.Since(prev) >= TrafficSnapshotInterval
	if due {
		w.lastSnapshot[id] = time.Now()
	}
	w.lastSnapshotMu.Unlock()
	if due {
		snap := model.ServerTrafficSnapshot{
			ServerID:     id,
			UpBytes:      inbound.Up,
			DownBytes:    inbound.Down,
			AllTimeBytes: inbound.AllTime,
		}
		if err := w.repo.RecordTrafficSnapshot(ctx, snap); err != nil {
			log.Printf("[Health Worker] %s: traffic snapshot failed: %v", srv.Name, err)
		}
	}
}

// recordStatus persists the latest probe result to servers and appends a
// health-event row if the status flipped.
func (w *HealthWorker) recordStatus(ctx context.Context, id uuid.UUID, status string, pingMs *int) {
	if err := w.repo.UpdateServerHealth(ctx, id, pingMs, status); err != nil {
		log.Printf("[Health Worker] %s: update health failed: %v", id, err)
	}
	if err := w.repo.RecordHealthEvent(ctx, id, status); err != nil {
		log.Printf("[Health Worker] %s: record event failed: %v", id, err)
	}
}
