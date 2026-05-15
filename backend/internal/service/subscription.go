package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	mathrand "math/rand/v2"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/zyvpn/backend/internal/config"
	"github.com/zyvpn/backend/internal/model"
	"github.com/zyvpn/backend/internal/repository"
)

var (
	ErrSubscriptionActive    = errors.New("У пользователя уже есть активная подписка")
	ErrSubscriptionNotActive = errors.New("Подписка неактивна")
	ErrTrialAlreadyUsed      = errors.New("Пробный период уже использован")
	ErrNoServersAvailable    = errors.New("Нет доступных серверов")
	ErrPlanHasNoServers      = errors.New("В тарифе не указаны серверы")
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

func (s *SubscriptionService) GetActiveSubscription(ctx context.Context, userID int64) (*model.Subscription, error) {
	sub, err := s.repo.GetActiveSubscription(ctx, userID)
	if err != nil {
		return nil, err
	}
	clients, err := s.repo.GetSubscriptionClients(ctx, sub.ID)
	if err != nil {
		return nil, err
	}
	s.hydrateLiveKeys(clients)
	sub.Clients = clients
	return sub, nil
}

func (s *SubscriptionService) GetSubscription(ctx context.Context, id uuid.UUID) (*model.Subscription, error) {
	sub, err := s.repo.GetSubscription(ctx, id)
	if err != nil {
		return nil, err
	}
	clients, err := s.repo.GetSubscriptionClients(ctx, sub.ID)
	if err != nil {
		return nil, err
	}
	s.hydrateLiveKeys(clients)
	sub.Clients = clients
	return sub, nil
}

// hydrateLiveKeys перегенерит connection_key каждого клиента из текущих
// настроек серверов. Сохранённый в БД ключ — снэпшот момента покупки и
// не используется для отдачи юзеру. Так любая правка сервера в админке
// (Reality keys, port, sni, address) автоматически прилетает в /sub и в
// status response — без ре-провизии клиентов в xui.
func (s *SubscriptionService) hydrateLiveKeys(clients []model.SubscriptionClient) {
	if s.serverSvc == nil {
		return
	}
	for i := range clients {
		if clients[i].Server == nil {
			continue
		}
		clients[i].ConnectionKey = s.serverSvc.GenerateConnectionKey(
			clients[i].Server, clients[i].XUIClientID, clients[i].XUIEmail,
		)
	}
}

// CreateSubscription активирует тариф: создаёт subscription row, генерит
// sub_token, провижинит xui-клиента на каждом server'е из plan.Servers,
// сохраняет subscription_clients. Если у юзера уже есть активная подписка —
// extend'им её вместо создания новой.
func (s *SubscriptionService) CreateSubscription(ctx context.Context, userID int64, plan *model.Plan) (*model.Subscription, error) {
	if s.serverSvc == nil {
		return nil, ErrNoServersAvailable
	}

	// Hydrate plan.Servers if caller didn't.
	if len(plan.Servers) == 0 {
		if err := s.repo.HydratePlanWithServers(ctx, plan); err != nil {
			return nil, fmt.Errorf("hydrate plan servers: %w", err)
		}
	}
	if len(plan.Servers) == 0 {
		return nil, ErrPlanHasNoServers
	}

	// Existing active sub — extend instead of new.
	existing, err := s.repo.GetActiveSubscription(ctx, userID)
	if err == nil && existing.IsActive() {
		log.Printf("Extending existing subscription %s for user %d by %d days, +%d GB",
			existing.ID, userID, plan.DurationDays, plan.TrafficGB)
		if err := s.ExtendSubscriptionWithTraffic(ctx, existing.ID, plan.DurationDays, plan.TrafficBytes()); err != nil {
			return nil, fmt.Errorf("failed to extend subscription: %w", err)
		}
		return s.GetSubscription(ctx, existing.ID)
	}

	maxDevices := plan.MaxDevices
	if maxDevices <= 0 {
		maxDevices = 3
	}

	now := time.Now()
	expiresAt := now.Add(time.Duration(plan.DurationDays) * 24 * time.Hour)

	subToken, err := generateSubToken()
	if err != nil {
		return nil, fmt.Errorf("generate sub token: %w", err)
	}

	sub := &model.Subscription{
		UserID:       userID,
		PlanID:       plan.ID,
		Status:       model.SubscriptionStatusActive,
		SubToken:     subToken,
		StartedAt:    &now,
		ExpiresAt:    &expiresAt,
		TrafficLimit: plan.TrafficBytes(),
		TrafficUsed:  0,
		MaxDevices:   maxDevices,
	}
	if err := s.repo.CreateSubscription(ctx, sub); err != nil {
		return nil, err
	}

	// Provision xui клиента на каждом сервере плана.
	// trafficLimitGB=0 — лимит enforce'ится централизованно (см. EnforceTrafficLimit).
	created := make([]model.SubscriptionClient, 0, len(plan.Servers))
	for _, srv := range plan.Servers {
		if !srv.IsActive {
			log.Printf("CreateSubscription: skip inactive server %s for sub %s", srv.ID, sub.ID)
			continue
		}
		xuiClientAPI, server, err := s.serverSvc.GetXUIClient(ctx, srv.ID)
		if err != nil {
			log.Printf("CreateSubscription: get xui client for server %s failed: %v — rolling back", srv.ID, err)
			s.rollbackProvisioning(ctx, sub.ID, created)
			return nil, fmt.Errorf("failed to get server %s: %w", srv.ID, err)
		}
		email := fmt.Sprintf("u%d_s%s_%d", userID, shortID(srv.ID), time.Now().Unix())

		xuiClient, err := xuiClientAPI.AddClient(email, 0 /*unlimited, enforce centrally*/, plan.DurationDays, maxDevices)
		if err != nil {
			log.Printf("CreateSubscription: AddClient on %s failed: %v — rolling back", server.Name, err)
			s.rollbackProvisioning(ctx, sub.ID, created)
			return nil, fmt.Errorf("failed to create VPN client on %s: %w", server.Name, err)
		}
		connectionKey := s.serverSvc.GenerateConnectionKey(server, xuiClient.ID, email)

		client := &model.SubscriptionClient{
			SubscriptionID: sub.ID,
			ServerID:       server.ID,
			XUIClientID:    xuiClient.ID,
			XUIEmail:       email,
			ConnectionKey:  connectionKey,
			Enabled:        true,
		}
		if err := s.repo.CreateSubscriptionClient(ctx, client); err != nil {
			_ = xuiClientAPI.DeleteClient(xuiClient.ID)
			s.rollbackProvisioning(ctx, sub.ID, created)
			return nil, fmt.Errorf("failed to save subscription_client: %w", err)
		}
		created = append(created, *client)

		if err := s.serverSvc.IncrementLoad(ctx, server.ID); err != nil {
			log.Printf("WARNING: failed to increment load for server %s: %v", server.ID, err)
		}
	}

	if len(created) == 0 {
		// все серверы плана не активны / упали
		_ = s.repo.UpdateSubscriptionStatus(ctx, sub.ID, model.SubscriptionStatusCancelled)
		return nil, ErrNoServersAvailable
	}

	sub.Clients = created
	return sub, nil
}

// rollbackProvisioning удаляет xui-клиентов, созданных в рамках частично
// успешной активации, и помечает подписку cancelled.
func (s *SubscriptionService) rollbackProvisioning(ctx context.Context, subID uuid.UUID, created []model.SubscriptionClient) {
	for _, c := range created {
		xuiClientAPI, _, err := s.serverSvc.GetXUIClient(ctx, c.ServerID)
		if err == nil {
			_ = xuiClientAPI.DeleteClient(c.XUIClientID)
		}
		_ = s.repo.DeleteSubscriptionClient(ctx, c.ID)
		_ = s.serverSvc.DecrementLoad(ctx, c.ServerID)
	}
	_ = s.repo.UpdateSubscriptionStatus(ctx, subID, model.SubscriptionStatusCancelled)
}

func (s *SubscriptionService) ExtendSubscription(ctx context.Context, subID uuid.UUID, days int) error {
	return s.ExtendSubscriptionWithTraffic(ctx, subID, days, 0)
}

func (s *SubscriptionService) ExtendSubscriptionWithTraffic(ctx context.Context, subID uuid.UUID, days int, additionalTrafficBytes int64) error {
	sub, err := s.repo.GetSubscription(ctx, subID)
	if err != nil {
		return err
	}
	if sub.Status != model.SubscriptionStatusActive {
		return ErrSubscriptionNotActive
	}

	clients, err := s.repo.GetSubscriptionClients(ctx, subID)
	if err != nil {
		return err
	}

	now := time.Now()
	base := now
	if sub.ExpiresAt != nil && sub.ExpiresAt.After(now) {
		base = *sub.ExpiresAt
	}
	newExpiry := base.Add(time.Duration(days) * 24 * time.Hour)

	maxDevices := sub.MaxDevices
	if maxDevices <= 0 {
		maxDevices = 3
	}

	// Обновляем expiry на каждом xui-клиенте подписки. Трафик лимит на стороне
	// xui остаётся 0 (unlimited) — суммарный лимит мы enforce'им сами.
	for _, c := range clients {
		xuiClientAPI, _, err := s.serverSvc.GetXUIClient(ctx, c.ServerID)
		if err != nil {
			log.Printf("ExtendSubscription: get xui client for server %s failed: %v — пропускаю", c.ServerID, err)
			continue
		}
		if err := xuiClientAPI.UpdateClientTraffic(c.XUIClientID, c.XUIEmail, 0 /*unlimited*/, newExpiry.UnixMilli(), maxDevices); err != nil {
			log.Printf("ExtendSubscription: update xui client %s failed: %v", c.XUIClientID, err)
		}
		// Если был disabled из-за лимита — снова включаем при расширении лимита.
		if !c.Enabled && additionalTrafficBytes > 0 {
			_ = s.repo.SetSubscriptionClientEnabled(ctx, c.ID, true)
		}
	}

	return s.repo.SetSubscriptionExpiryAndTraffic(ctx, subID, newExpiry, additionalTrafficBytes)
}

func (s *SubscriptionService) CancelSubscription(ctx context.Context, subID uuid.UUID) error {
	clients, err := s.repo.GetSubscriptionClients(ctx, subID)
	if err != nil {
		return err
	}
	for _, c := range clients {
		xuiClientAPI, _, err := s.serverSvc.GetXUIClient(ctx, c.ServerID)
		if err == nil {
			if err := xuiClientAPI.DeleteClient(c.XUIClientID); err != nil {
				log.Printf("CancelSubscription: DeleteClient %s on server %s: %v",
					c.XUIClientID, c.ServerID, err)
			}
		}
		_ = s.serverSvc.DecrementLoad(ctx, c.ServerID)
		_ = s.repo.DeleteSubscriptionClient(ctx, c.ID)
	}
	return s.repo.UpdateSubscriptionStatus(ctx, subID, model.SubscriptionStatusCancelled)
}

func (s *SubscriptionService) ExpireSubscription(ctx context.Context, subID uuid.UUID) error {
	clients, err := s.repo.GetSubscriptionClients(ctx, subID)
	if err != nil {
		return err
	}
	for _, c := range clients {
		xuiClientAPI, _, err := s.serverSvc.GetXUIClient(ctx, c.ServerID)
		if err == nil {
			if err := xuiClientAPI.DeleteClient(c.XUIClientID); err != nil {
				log.Printf("ExpireSubscription: DeleteClient %s on server %s: %v",
					c.XUIClientID, c.ServerID, err)
			}
		}
		_ = s.serverSvc.DecrementLoad(ctx, c.ServerID)
		_ = s.repo.DeleteSubscriptionClient(ctx, c.ID)
	}
	return s.repo.UpdateSubscriptionStatus(ctx, subID, model.SubscriptionStatusExpired)
}

// SyncTraffic — устаревший fast-path (по-клиентный pull). Сохранён для
// бот-флоу и handler /api/subscription, чтобы юзер не ждал тика HealthWorker'а.
// Под капотом ходит по всем клиентам подписки и суммирует.
func (s *SubscriptionService) SyncTraffic(ctx context.Context, subID uuid.UUID) error {
	clients, err := s.repo.GetSubscriptionClients(ctx, subID)
	if err != nil {
		return err
	}
	var total int64
	for _, c := range clients {
		xuiClientAPI, _, err := s.serverSvc.GetXUIClient(ctx, c.ServerID)
		if err != nil {
			log.Printf("SyncTraffic: get xui for server %s failed: %v", c.ServerID, err)
			continue
		}
		t, err := xuiClientAPI.GetClientTraffic(c.XUIEmail)
		if err != nil || t == nil {
			continue
		}
		used := t.Up + t.Down
		_ = s.repo.UpdateSubscriptionClientTraffic(ctx, c.ID, used)
		total += used
	}
	if err := s.repo.UpdateSubscriptionTraffic(ctx, subID, total); err != nil {
		return err
	}
	return s.EnforceTrafficLimit(ctx, subID)
}

// EnforceTrafficLimit — если суммарный traffic_used ≥ traffic_limit (и лимит
// не безлимит), отключаем всех xui-клиентов этой подписки. Идемпотентно:
// если уже disabled — не дёргаем xui повторно.
func (s *SubscriptionService) EnforceTrafficLimit(ctx context.Context, subID uuid.UUID) error {
	sub, err := s.repo.GetSubscription(ctx, subID)
	if err != nil {
		return err
	}
	if sub.TrafficLimit <= 0 {
		return nil // unlimited
	}
	if sub.TrafficUsed < sub.TrafficLimit {
		return nil
	}
	clients, err := s.repo.GetSubscriptionClients(ctx, subID)
	if err != nil {
		return err
	}
	var expiryMs int64
	if sub.ExpiresAt != nil {
		expiryMs = sub.ExpiresAt.UnixMilli()
	}
	maxDevices := sub.MaxDevices
	if maxDevices <= 0 {
		maxDevices = 3
	}
	for _, c := range clients {
		if !c.Enabled {
			continue
		}
		xuiClientAPI, _, err := s.serverSvc.GetXUIClient(ctx, c.ServerID)
		if err != nil {
			log.Printf("EnforceTrafficLimit: get xui for server %s failed: %v", c.ServerID, err)
			continue
		}
		if err := xuiClientAPI.SetClientEnabled(c.XUIClientID, c.XUIEmail, false, 0 /*unlimited на стороне xui*/, expiryMs, maxDevices); err != nil {
			log.Printf("EnforceTrafficLimit: disable xui client %s failed: %v", c.XUIClientID, err)
			continue
		}
		_ = s.repo.SetSubscriptionClientEnabled(ctx, c.ID, false)
	}
	return nil
}

func (s *SubscriptionService) ProcessExpiredSubscriptions(ctx context.Context) error {
	expired, err := s.repo.GetExpiredSubscriptions(ctx)
	if err != nil {
		return err
	}
	for _, sub := range expired {
		if err := s.ExpireSubscription(ctx, sub.ID); err != nil {
			log.Printf("Failed to expire subscription %s: %v", sub.ID, err)
		}
	}
	return nil
}

func (s *SubscriptionService) GetExpiringSubscriptions(ctx context.Context, withinHours int) ([]model.Subscription, error) {
	before := time.Now().Add(time.Duration(withinHours) * time.Hour)
	return s.repo.GetExpiringSubscriptions(ctx, before)
}

// HasUsedTrial checks if user has already used trial
func (s *SubscriptionService) HasUsedTrial(ctx context.Context, userID int64) (bool, error) {
	return s.repo.HasUsedTrial(ctx, userID)
}

func (s *SubscriptionService) ActivateTrial(ctx context.Context, userID int64) (*model.Subscription, error) {
	existing, err := s.repo.GetActiveSubscription(ctx, userID)
	if err == nil && existing.IsActive() {
		return nil, ErrSubscriptionActive
	}
	hasUsedTrial, err := s.repo.HasUsedTrial(ctx, userID)
	if err != nil {
		return nil, err
	}
	if hasUsedTrial {
		return nil, ErrTrialAlreadyUsed
	}
	plan, err := s.repo.GetTrialPlan(ctx)
	if err != nil {
		return nil, fmt.Errorf("trial plan not found: %w", err)
	}
	if err := s.repo.HydratePlanWithServers(ctx, plan); err != nil {
		return nil, err
	}
	return s.CreateSubscription(ctx, userID, plan)
}

// ActivateTrialWithDays creates a subscription with custom days (for promo codes)
func (s *SubscriptionService) ActivateTrialWithDays(ctx context.Context, userID int64, days int) (*model.Subscription, error) {
	existing, err := s.repo.GetActiveSubscription(ctx, userID)
	if err == nil && existing.IsActive() {
		return nil, ErrSubscriptionActive
	}
	plan, err := s.repo.GetTrialPlan(ctx)
	if err != nil {
		return nil, fmt.Errorf("trial plan not found: %w", err)
	}
	if err := s.repo.HydratePlanWithServers(ctx, plan); err != nil {
		return nil, err
	}
	customPlan := *plan
	customPlan.DurationDays = days
	return s.CreateSubscription(ctx, userID, &customPlan)
}

// BuildSubscriptionContent рендерит ответ для /sub/<token>: base64(joined \n
// share-links) + Subscription-Userinfo заголовок. Если подписка не активна
// или не найдена — возвращает ErrSubscriptionNotActive.
type SubscriptionContent struct {
	Body            []byte
	UserInfoHeader  string // Subscription-Userinfo
	ProfileTitle    string
	UpdateInterval  int // часы для Profile-Update-Interval
	ContentTypeHint string
}

func (s *SubscriptionService) BuildSubscriptionContent(ctx context.Context, token string) (*SubscriptionContent, error) {
	sub, err := s.repo.GetSubscriptionByToken(ctx, token)
	if err != nil {
		return nil, err
	}
	if !sub.IsActive() {
		return nil, ErrSubscriptionNotActive
	}
	clients, err := s.repo.GetSubscriptionClients(ctx, sub.ID)
	if err != nil {
		return nil, err
	}
	s.hydrateLiveKeys(clients)
	plan, err := s.repo.GetPlan(ctx, sub.PlanID)
	if err != nil {
		return nil, err
	}

	// Live load-balancing: фильтруем offline-серверы, сортируем по
	// (ping_ms ASC, current_load ASC). Внутри одной «тяжести» — рандом,
	// чтобы N юзеров с одинаковым best-сервером не уходили все на один.
	ranked := make([]model.SubscriptionClient, 0, len(clients))
	for _, c := range clients {
		if !c.Enabled {
			continue
		}
		if c.Server == nil || !c.Server.IsActive {
			continue
		}
		if c.Server.Status != "" && c.Server.Status != "online" {
			continue
		}
		ranked = append(ranked, c)
	}
	// pre-shuffle to give равным-по-метрикам серверам случайный порядок
	mathrand.Shuffle(len(ranked), func(i, j int) {
		ranked[i], ranked[j] = ranked[j], ranked[i]
	})
	sort.SliceStable(ranked, func(i, j int) bool {
		pi, pj := pingScore(ranked[i].Server.PingMs), pingScore(ranked[j].Server.PingMs)
		if pi != pj {
			return pi < pj
		}
		return ranked[i].Server.CurrentLoad < ranked[j].Server.CurrentLoad
	})

	links := make([]string, 0, len(ranked))
	for _, c := range ranked {
		links = append(links, c.ConnectionKey)
	}
	plaintext := strings.Join(links, "\n")
	body := []byte(base64.StdEncoding.EncodeToString([]byte(plaintext)))

	// Subscription-Userinfo header. expire — UNIX-секунды, 0 если бессрочно.
	var expire int64
	if sub.ExpiresAt != nil {
		expire = sub.ExpiresAt.Unix()
	}
	header := fmt.Sprintf("upload=0; download=%d; total=%d; expire=%d",
		sub.TrafficUsed, sub.TrafficLimit, expire)

	return &SubscriptionContent{
		Body:            body,
		UserInfoHeader:  header,
		ProfileTitle:    "CoreVPN — " + plan.Name,
		UpdateInterval:  1,
		ContentTypeHint: "text/plain; charset=utf-8",
	}, nil
}

// GetSubscriptionURLForUser возвращает subscription URL активной подписки
// юзера. Используется для нотификаций (бот, payment success) — теперь вместо
// одного connection_key даём ссылку, которую юзер вставляет в VPN-клиент.
func (s *SubscriptionService) GetSubscriptionURLForUser(ctx context.Context, userID int64) (string, error) {
	sub, err := s.repo.GetActiveSubscription(ctx, userID)
	if err != nil {
		return "", err
	}
	if !sub.IsActive() {
		return "", ErrSubscriptionNotActive
	}
	return s.BuildSubscriptionURL(sub), nil
}

// BuildSubscriptionURL — публичная ссылка которую отдаём фронту в /api/subscription.
func (s *SubscriptionService) BuildSubscriptionURL(sub *model.Subscription) string {
	base := strings.TrimRight(s.cfg.Server.PublicAPIBaseURL, "/")
	if base == "" {
		return "/sub/" + url.PathEscape(sub.SubToken)
	}
	return base + "/sub/" + url.PathEscape(sub.SubToken)
}

// ReconcileSubscriptionClients добивает subscription_clients подписки до
// набора серверов из её плана. Идемпотентно — для серверов где клиент уже
// есть ничего не делает. Для отсутствующих — создаёт xui-клиента и пишет
// строку в subscription_clients. Серверы которых больше нет в plan_servers
// сейчас не трогает (пусть юзер ими пока пользуется — decommission отдельно).
func (s *SubscriptionService) ReconcileSubscriptionClients(ctx context.Context, subID uuid.UUID) (int, error) {
	if s.serverSvc == nil {
		return 0, ErrNoServersAvailable
	}
	sub, err := s.repo.GetSubscription(ctx, subID)
	if err != nil {
		return 0, err
	}
	plan, err := s.repo.GetPlan(ctx, sub.PlanID)
	if err != nil {
		return 0, err
	}
	if err := s.repo.HydratePlanWithServers(ctx, plan); err != nil {
		return 0, err
	}
	if len(plan.Servers) == 0 {
		return 0, ErrPlanHasNoServers
	}
	existing, err := s.repo.GetSubscriptionClients(ctx, subID)
	if err != nil {
		return 0, err
	}
	have := make(map[uuid.UUID]struct{}, len(existing))
	for _, c := range existing {
		have[c.ServerID] = struct{}{}
	}

	maxDevices := sub.MaxDevices
	if maxDevices <= 0 {
		maxDevices = 3
	}
	expiryDays := 0
	if sub.ExpiresAt != nil {
		expiryDays = int(time.Until(*sub.ExpiresAt).Hours()/24) + 1
		if expiryDays < 1 {
			expiryDays = 1
		}
	}

	added := 0
	for _, srv := range plan.Servers {
		if !srv.IsActive {
			continue
		}
		if _, ok := have[srv.ID]; ok {
			continue
		}
		xuiClientAPI, server, err := s.serverSvc.GetXUIClient(ctx, srv.ID)
		if err != nil {
			log.Printf("Reconcile sub=%s: get xui for %s failed: %v", subID, srv.ID, err)
			continue
		}
		email := fmt.Sprintf("u%d_s%s_%d", sub.UserID, shortID(srv.ID), time.Now().Unix())
		xuiClient, err := xuiClientAPI.AddClient(email, 0, expiryDays, maxDevices)
		if err != nil {
			log.Printf("Reconcile sub=%s server=%s: AddClient failed: %v", subID, srv.Name, err)
			continue
		}
		connectionKey := s.serverSvc.GenerateConnectionKey(server, xuiClient.ID, email)
		c := &model.SubscriptionClient{
			SubscriptionID: subID,
			ServerID:       server.ID,
			XUIClientID:    xuiClient.ID,
			XUIEmail:       email,
			ConnectionKey:  connectionKey,
			Enabled:        true,
		}
		if err := s.repo.CreateSubscriptionClient(ctx, c); err != nil {
			_ = xuiClientAPI.DeleteClient(xuiClient.ID)
			log.Printf("Reconcile sub=%s server=%s: insert failed: %v", subID, srv.Name, err)
			continue
		}
		_ = s.serverSvc.IncrementLoad(ctx, server.ID)
		added++
	}
	return added, nil
}

// RotateSubToken — администратор/юзер просит выпустить новый токен (если
// старый утёк). Старый перестаёт работать сразу.
func (s *SubscriptionService) RotateSubToken(ctx context.Context, subID uuid.UUID) (string, error) {
	tok, err := generateSubToken()
	if err != nil {
		return "", err
	}
	if err := s.repo.RotateSubscriptionToken(ctx, subID, tok); err != nil {
		return "", err
	}
	return tok, nil
}

func generateSubToken() (string, error) {
	b := make([]byte, 24) // 24 bytes → 48 hex chars
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func shortID(id uuid.UUID) string {
	s := id.String()
	return strings.ReplaceAll(s[:8], "-", "")
}

// pingScore нормализует ping_ms для сортировки. nil → большой штраф
// (но не «бесконечность», чтобы непрозвонившийся сервер не убегал
// в самый конец — он ещё может быть жив, просто HealthWorker не успел
// сделать первый probe).
func pingScore(ms *int) int {
	if ms == nil {
		return 999
	}
	return *ms
}
