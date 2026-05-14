package service

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/zyvpn/backend/internal/model"
	"github.com/zyvpn/backend/internal/repository"
	"github.com/zyvpn/backend/internal/xui"
)

const (
	HealthCheckInterval     = 30 * time.Second
	TrafficSnapshotInterval = 5 * time.Minute
)

type HealthWorker struct {
	repo            *repository.Repository
	serverSvc       *ServerService
	subscriptionSvc *SubscriptionService

	lastSnapshot   map[uuid.UUID]time.Time
	lastSnapshotMu sync.Mutex
}

func NewHealthWorker(repo *repository.Repository, serverSvc *ServerService, subscriptionSvc *SubscriptionService) *HealthWorker {
	return &HealthWorker{
		repo:            repo,
		serverSvc:       serverSvc,
		subscriptionSvc: subscriptionSvc,
		lastSnapshot:    make(map[uuid.UUID]time.Time),
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
// events for uptime calc, snapshots inbound traffic, and — крутая часть —
// bulk-sync'ит per-client traffic в `subscription_clients`. После каждого
// тика триггерим EnforceTrafficLimit для подписок чьи клиенты обновились,
// чтобы вовремя отрубать перерасход.
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

	// Bulk-sync трафика по клиентам этого инбаунда. inbound.ClientStats
	// уже есть в ответе getInbound — ноль дополнительных запросов в xui.
	w.syncClientTraffic(ctx, srv, inbound.ClientStats)

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

// syncClientTraffic обновляет subscription_clients.traffic_used по данным
// xui clientStats. Затем по уникальным subscription_id триггерит
// EnforceTrafficLimit — если суммарный used ≥ limit, клиенты этой подписки
// будут disabled во всех серверах.
//
// Безопасно если подписки на этом сервере нет — UPDATE по (server_id, email)
// просто не задевает строк.
func (w *HealthWorker) syncClientTraffic(ctx context.Context, srv model.Server, stats []xui.Traffic) {
	if len(stats) == 0 {
		return
	}

	// 1) batch-апдейт per-client used + параллельно пересчитываем агрегат
	//    по subscription_id (читая sub_id из subscription_clients по email).
	updatedSubs := make(map[uuid.UUID]struct{})
	for _, st := range stats {
		used := st.Up + st.Down
		client, err := w.repo.GetSubscriptionClientByServerEmail(ctx, srv.ID, st.Email)
		if err != nil {
			// клиент xui не зарегистрирован у нас — это нормально для
			// импортированных вручную или тестовых клиентов.
			continue
		}
		if client.TrafficUsed != used {
			if err := w.repo.UpdateSubscriptionClientTraffic(ctx, client.ID, used); err != nil {
				log.Printf("[Health Worker] %s: update client traffic for %s: %v",
					srv.Name, st.Email, err)
				continue
			}
		}
		updatedSubs[client.SubscriptionID] = struct{}{}
	}

	// 2) Для каждой задетой подписки — пересчитать сумму и enforcement.
	for subID := range updatedSubs {
		if err := w.repo.RecomputeSubscriptionTrafficUsed(ctx, subID); err != nil {
			log.Printf("[Health Worker] recompute sub %s traffic: %v", subID, err)
			continue
		}
		if w.subscriptionSvc != nil {
			if err := w.subscriptionSvc.EnforceTrafficLimit(ctx, subID); err != nil {
				log.Printf("[Health Worker] enforce limit sub %s: %v", subID, err)
			}
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
