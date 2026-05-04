//go:build integration

package service

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/google/uuid"

	"github.com/zyvpn/backend/internal/config"
	"github.com/zyvpn/backend/internal/model"
	"github.com/zyvpn/backend/internal/repository"
)

// integrationDSN returns the test DSN. Set TEST_DATABASE_URL to a postgres
// instance the test can write to (it WILL execute migrations and modify rows).
//
// Example: postgres://zyvpn:zyvpn@localhost:5432/zyvpn_test?sslmode=disable
func integrationDSN(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping subscription integration test")
	}
	return dsn
}

// migrationsPath returns the absolute file:// URL to the migrations directory.
// Tests run from internal/service, so we go up two levels.
func migrationsPath(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	return "file://" + wd + "/../../migrations"
}

var (
	migrationsApplied bool
	migrationsMu      sync.Mutex
)

func ensureMigrations(t *testing.T, dsn string) {
	t.Helper()
	migrationsMu.Lock()
	defer migrationsMu.Unlock()
	if migrationsApplied {
		return
	}
	m, err := migrate.New(migrationsPath(t), dsn)
	if err != nil {
		t.Fatalf("migrate.New: %v", err)
	}
	defer m.Close()
	if err := m.Drop(); err != nil && err != migrate.ErrNoChange {
		t.Fatalf("migrate.Drop: %v", err)
	}
	m2, err := migrate.New(migrationsPath(t), dsn)
	if err != nil {
		t.Fatalf("migrate.New (reopen): %v", err)
	}
	defer m2.Close()
	if err := m2.Up(); err != nil && err != migrate.ErrNoChange {
		t.Fatalf("migrate.Up: %v", err)
	}
	migrationsApplied = true
}

// truncateAll wipes mutable test data between tests so each starts from a
// known state without re-running every migration.
func truncateAll(t *testing.T, repo *repository.Repository) {
	t.Helper()
	tables := []string{
		"payments", "referrals", "subscriptions", "promo_code_uses",
		"promo_codes", "balance_transactions", "user_balances",
		"servers", "users", "plans",
	}
	for _, tbl := range tables {
		// IF EXISTS so we don't fail on tables that may not exist in
		// older migration versions when this list grows.
		_, _ = repo.DB().Exec(fmt.Sprintf("TRUNCATE TABLE %s CASCADE", tbl))
	}
}

func setupDB(t *testing.T) *repository.Repository {
	t.Helper()
	dsn := integrationDSN(t)
	ensureMigrations(t, dsn)

	repo, err := repository.New(dsn)
	if err != nil {
		t.Fatalf("repository.New: %v", err)
	}
	truncateAll(t, repo)
	t.Cleanup(func() { _ = repo.Close() })
	return repo
}

// fakePanel is a minimal httptest 3x-ui stand-in scoped to integration tests.
type fakePanel struct {
	mu          sync.Mutex
	loginCount  int32
	addCount    int32
	updateCount int32
	deleteCount int32
	authed      bool
	trafficObj  string
	inboundObj  string
}

func (p *fakePanel) handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&p.loginCount, 1)
		p.mu.Lock()
		p.authed = true
		p.mu.Unlock()
		http.SetCookie(w, &http.Cookie{Name: "session", Value: "ok", Path: "/"})
		writeJSON(w, 200, `{"success":true,"msg":"ok"}`)
	})

	authed := func(r *http.Request) bool {
		c, _ := r.Cookie("session")
		p.mu.Lock()
		defer p.mu.Unlock()
		return p.authed && c != nil && c.Value == "ok"
	}

	mux.HandleFunc("/panel/api/inbounds/addClient", func(w http.ResponseWriter, r *http.Request) {
		if !authed(r) {
			writeHTML(w)
			return
		}
		atomic.AddInt32(&p.addCount, 1)
		writeJSON(w, 200, `{"success":true,"msg":"ok"}`)
	})

	mux.HandleFunc("/panel/api/inbounds/", func(w http.ResponseWriter, r *http.Request) {
		if !authed(r) {
			writeHTML(w)
			return
		}
		path := r.URL.Path
		switch {
		case strings.Contains(path, "/get/"):
			obj := p.inboundObj
			if obj == "" {
				obj = `null`
			}
			writeJSON(w, 200, fmt.Sprintf(`{"success":true,"obj":%s}`, obj))
		case strings.Contains(path, "/updateClient/"):
			atomic.AddInt32(&p.updateCount, 1)
			writeJSON(w, 200, `{"success":true,"msg":"ok"}`)
		case strings.Contains(path, "/delClient/"):
			atomic.AddInt32(&p.deleteCount, 1)
			writeJSON(w, 200, `{"success":true,"msg":"ok"}`)
		case strings.Contains(path, "/getClientTraffics/"):
			obj := p.trafficObj
			if obj == "" {
				obj = `null`
			}
			writeJSON(w, 200, fmt.Sprintf(`{"success":true,"obj":%s}`, obj))
		default:
			http.NotFound(w, r)
		}
	})

	return mux
}

func writeJSON(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = io.WriteString(w, body)
}

func writeHTML(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(200)
	_, _ = io.WriteString(w, `<!DOCTYPE html><html>login</html>`)
}

// fixture builds the standard integration setup: clean DB, fake XUI panel,
// active server pointing to it, default user, and a single 30-day plan.
type fixture struct {
	repo      *repository.Repository
	panel     *fakePanel
	panelSrv  *httptest.Server
	server    *model.Server
	user      *model.User
	plan      *model.Plan
	subSvc    *SubscriptionService
	serverSvc *ServerService
}

func setupFixture(t *testing.T) *fixture {
	t.Helper()
	repo := setupDB(t)

	p := &fakePanel{
		// Inbound metadata used by GenerateConnectionKey via server fields,
		// but the service reads server.PublicKey/ShortID from DB. The inbound
		// payload is consumed by GetInboundInfo (admin "test connection") and
		// DeleteClientByEmail.
		inboundObj: `{"id":1,"port":443,"settings":"{\"clients\":[]}","streamSettings":"{\"network\":\"tcp\",\"security\":\"reality\",\"realitySettings\":{\"serverNames\":[\"example.com\"],\"shortIds\":[\"sid\"],\"settings\":{\"publicKey\":\"PK\",\"serverName\":\"example.com\"}}}"}`,
	}
	srv := httptest.NewServer(p.handler())
	t.Cleanup(srv.Close)

	ctx := context.Background()

	// Server pointing at the fake panel
	server := &model.Server{
		Name:          "test-de",
		Country:       "DE",
		FlagEmoji:     "DE",
		XUIBaseURL:    srv.URL,
		XUIUsername:   "admin",
		XUIPassword:   "admin",
		XUIInboundID:  1,
		ServerAddress: "test.example.com",
		ServerPort:    443,
		PublicKey:     "PK",
		ShortID:       "sid",
		ServerName:    "example.com",
		IsActive:      true,
		SortOrder:     1,
	}
	if err := repo.CreateServer(ctx, server); err != nil {
		t.Fatalf("CreateServer: %v", err)
	}
	// Mark online so GetBestServer picks it.
	pingMs := 5
	if err := repo.UpdateServerHealth(ctx, server.ID, &pingMs, "online"); err != nil {
		t.Fatalf("UpdateServerHealth: %v", err)
	}

	// User. referral_code is VARCHAR(20), so keep it short.
	user := &model.User{
		ID:           42,
		Username:     stringPtr("integration"),
		FirstName:    stringPtr("Test"),
		ReferralCode: shortRef(),
	}
	if err := repo.CreateUser(ctx, user); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	// Plan: 30 days, 100 GB
	plan, err := repo.CreatePlan(ctx, "Test30", "30d/100GB", 30, 100, 3, 1.5, 150, 4.99, 100)
	if err != nil {
		t.Fatalf("CreatePlan: %v", err)
	}

	cfg := &config.Config{}
	serverSvc := NewServerService(repo)
	subSvc := NewSubscriptionService(repo, serverSvc, cfg)
	subSvc.SetServerService(serverSvc)

	return &fixture{
		repo:      repo,
		panel:     p,
		panelSrv:  srv,
		server:    server,
		user:      user,
		plan:      plan,
		subSvc:    subSvc,
		serverSvc: serverSvc,
	}
}

func stringPtr(s string) *string { return &s }

// shortRef returns a fresh short referral code (under 20 chars).
func shortRef() string {
	return fmt.Sprintf("R%d", time.Now().UnixNano()%1_000_000_000)
}

// TestInt_CreateSubscription verifies the happy path: subscription row is
// created and a panel client was added.
func TestInt_CreateSubscription(t *testing.T) {
	f := setupFixture(t)
	ctx := context.Background()

	sub, err := f.subSvc.CreateSubscriptionWithServer(ctx, f.user.ID, f.plan, &f.server.ID)
	if err != nil {
		t.Fatalf("CreateSubscriptionWithServer: %v", err)
	}
	if sub.Status != model.SubscriptionStatusActive {
		t.Errorf("status = %s, want active", sub.Status)
	}
	if sub.TrafficLimit != int64(f.plan.TrafficGB)*1024*1024*1024 {
		t.Errorf("traffic_limit mismatch: got %d", sub.TrafficLimit)
	}
	if atomic.LoadInt32(&f.panel.addCount) != 1 {
		t.Errorf("expected 1 AddClient call, got %d", f.panel.addCount)
	}
}

// TestInt_BuyOnTopExtendsExisting reproduces the "buying a plan when one is
// already active" flow — must extend the existing subscription, not create a
// second one.
func TestInt_BuyOnTopExtendsExisting(t *testing.T) {
	f := setupFixture(t)
	ctx := context.Background()

	first, err := f.subSvc.CreateSubscriptionWithServer(ctx, f.user.ID, f.plan, &f.server.ID)
	if err != nil {
		t.Fatalf("first buy: %v", err)
	}
	originalExpiry := *first.ExpiresAt
	originalLimit := first.TrafficLimit

	second, err := f.subSvc.CreateSubscriptionWithServer(ctx, f.user.ID, f.plan, &f.server.ID)
	if err != nil {
		t.Fatalf("second buy: %v", err)
	}

	if second.ID != first.ID {
		t.Errorf("expected same subscription id, got new one")
	}
	if !second.ExpiresAt.After(originalExpiry) {
		t.Errorf("expiry didn't move forward: %v -> %v", originalExpiry, *second.ExpiresAt)
	}
	if second.TrafficLimit <= originalLimit {
		t.Errorf("traffic_limit didn't grow: %d -> %d", originalLimit, second.TrafficLimit)
	}
	if atomic.LoadInt32(&f.panel.updateCount) < 1 {
		t.Errorf("expected at least 1 UpdateClient call, got %d", f.panel.updateCount)
	}
}

// TestInt_ExtendFromPastExpiry reproduces issue #6 — a subscription whose
// expires_at is already in the past should still end up with a future
// expiry after extending.
func TestInt_ExtendFromPastExpiry(t *testing.T) {
	f := setupFixture(t)
	ctx := context.Background()

	sub, err := f.subSvc.CreateSubscriptionWithServer(ctx, f.user.ID, f.plan, &f.server.ID)
	if err != nil {
		t.Fatalf("CreateSubscription: %v", err)
	}

	// Force expires_at into the past.
	pastExpiry := time.Now().Add(-2 * 24 * time.Hour)
	if _, err := f.repo.DB().ExecContext(ctx,
		"UPDATE subscriptions SET expires_at = $2 WHERE id = $1", sub.ID, pastExpiry); err != nil {
		t.Fatalf("force past expiry: %v", err)
	}

	// Extend by 7 days. New expiry should be roughly now + 7 days, NOT
	// pastExpiry + 7 days (which would still be in the past).
	if err := f.subSvc.ExtendSubscriptionWithTraffic(ctx, sub.ID, 7, 0); err != nil {
		t.Fatalf("ExtendSubscriptionWithTraffic: %v", err)
	}

	updated, err := f.repo.GetSubscription(ctx, sub.ID)
	if err != nil {
		t.Fatalf("re-read: %v", err)
	}
	if updated.ExpiresAt == nil || !updated.ExpiresAt.After(time.Now()) {
		t.Errorf("expected future expiry, got %v", updated.ExpiresAt)
	}
	// Should be within +6/+8 days from now (sanity).
	delta := time.Until(*updated.ExpiresAt)
	if delta < 6*24*time.Hour || delta > 8*24*time.Hour {
		t.Errorf("delta from now = %v, expected ~7 days", delta)
	}
}

// TestInt_SyncTrafficWithMissingClientNoCrash verifies the GetClientTraffic
// nil-Obj fix: when 3x-ui no longer knows about the client, SyncTraffic must
// not panic and must leave the DB row alone.
func TestInt_SyncTrafficWithMissingClientNoCrash(t *testing.T) {
	f := setupFixture(t)
	ctx := context.Background()

	sub, err := f.subSvc.CreateSubscriptionWithServer(ctx, f.user.ID, f.plan, &f.server.ID)
	if err != nil {
		t.Fatalf("CreateSubscription: %v", err)
	}

	// Panel returns null obj for traffic by default — simulates client gone.
	if err := f.subSvc.SyncTraffic(ctx, sub.ID); err != nil {
		t.Fatalf("SyncTraffic: %v", err)
	}

	updated, err := f.repo.GetSubscription(ctx, sub.ID)
	if err != nil {
		t.Fatalf("re-read: %v", err)
	}
	if updated.TrafficUsed != 0 {
		t.Errorf("expected traffic_used unchanged at 0, got %d", updated.TrafficUsed)
	}
}

// TestInt_CancelSubscription verifies that cancellation deletes from XUI and
// flips the DB status.
func TestInt_CancelSubscription(t *testing.T) {
	f := setupFixture(t)
	ctx := context.Background()

	sub, err := f.subSvc.CreateSubscriptionWithServer(ctx, f.user.ID, f.plan, &f.server.ID)
	if err != nil {
		t.Fatalf("CreateSubscription: %v", err)
	}

	if err := f.subSvc.CancelSubscription(ctx, sub.ID); err != nil {
		t.Fatalf("CancelSubscription: %v", err)
	}
	if atomic.LoadInt32(&f.panel.deleteCount) != 1 {
		t.Errorf("expected 1 DeleteClient call, got %d", f.panel.deleteCount)
	}

	updated, err := f.repo.GetSubscription(ctx, sub.ID)
	if err != nil {
		t.Fatalf("re-read: %v", err)
	}
	if updated.Status != model.SubscriptionStatusCancelled {
		t.Errorf("status = %s, want cancelled", updated.Status)
	}
}

// TestInt_PanelSessionLossRecovers exercises the auth-retry path end to end:
// drop the panel session between calls and ensure the next subscription
// operation transparently re-logs in.
func TestInt_PanelSessionLossRecovers(t *testing.T) {
	f := setupFixture(t)
	ctx := context.Background()

	if _, err := f.subSvc.CreateSubscriptionWithServer(ctx, f.user.ID, f.plan, &f.server.ID); err != nil {
		t.Fatalf("first create: %v", err)
	}
	loginsBefore := atomic.LoadInt32(&f.panel.loginCount)

	// Wipe panel session — like a panel restart.
	f.panel.mu.Lock()
	f.panel.authed = false
	f.panel.mu.Unlock()

	// Simulate "system" extending the same subscription. This goes through
	// UpdateClientTraffic, which must auto re-login.
	sub, err := f.subSvc.GetActiveSubscription(ctx, f.user.ID)
	if err != nil {
		t.Fatalf("GetActiveSubscription: %v", err)
	}
	if err := f.subSvc.ExtendSubscriptionWithTraffic(ctx, sub.ID, 1, 0); err != nil {
		t.Fatalf("ExtendSubscriptionWithTraffic after session loss: %v", err)
	}
	loginsAfter := atomic.LoadInt32(&f.panel.loginCount)
	if loginsAfter <= loginsBefore {
		t.Errorf("expected at least one extra login, before=%d after=%d", loginsBefore, loginsAfter)
	}
}

// Ensure the SQL driver is linked even though we only use it via repository.New.
var _ = sql.Drivers
var _ = uuid.New
