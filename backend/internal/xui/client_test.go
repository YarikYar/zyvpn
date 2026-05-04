package xui

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

// fakePanel is a tiny stub of the 3x-ui HTTP API. It supports configurable
// responses per endpoint and counts how many times each endpoint was hit so
// tests can assert recovery flows.
type fakePanel struct {
	mu          sync.Mutex
	loginCount  int32
	addCount    int32
	updateCount int32
	deleteCount int32
	getInbound  int32
	getTraffic  int32

	// behavior knobs
	authValidAfter int32 // login N times before sessions are accepted; -1 = always valid
	updateBehavior func(call int32) (int, string)
	addBehavior    func(call int32) (int, string)
	deleteBehavior func(call int32) (int, string)
	trafficObj     string // raw json for getTraffic obj
	inboundObj     string // raw json for getInbound obj

	// session state
	authedSession bool
}

func (p *fakePanel) handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&p.loginCount, 1)
		p.mu.Lock()
		p.authedSession = true
		p.mu.Unlock()
		http.SetCookie(w, &http.Cookie{Name: "session", Value: "ok", Path: "/"})
		writeJSON(w, 200, `{"success":true,"msg":"ok"}`)
	})

	mux.HandleFunc("/panel/api/inbounds/addClient", func(w http.ResponseWriter, r *http.Request) {
		if !p.checkAuth(w, r) {
			return
		}
		n := atomic.AddInt32(&p.addCount, 1)
		if p.addBehavior != nil {
			status, body := p.addBehavior(n)
			writeJSON(w, status, body)
			return
		}
		writeJSON(w, 200, `{"success":true,"msg":"ok"}`)
	})

	mux.HandleFunc("/panel/api/inbounds/", func(w http.ResponseWriter, r *http.Request) {
		if !p.checkAuth(w, r) {
			return
		}
		// Routes:
		//   /panel/api/inbounds/get/{id}
		//   /panel/api/inbounds/{id}/updateClient/{uuid}
		//   /panel/api/inbounds/{id}/delClient/{uuid}
		//   /panel/api/inbounds/{id}/resetClientTraffic/{email}
		//   /panel/api/inbounds/getClientTraffics/{email}
		path := r.URL.Path
		switch {
		case strings.Contains(path, "/get/"):
			atomic.AddInt32(&p.getInbound, 1)
			obj := p.inboundObj
			if obj == "" {
				obj = `null`
			}
			writeJSON(w, 200, fmt.Sprintf(`{"success":true,"obj":%s}`, obj))
		case strings.Contains(path, "/updateClient/"):
			n := atomic.AddInt32(&p.updateCount, 1)
			if p.updateBehavior != nil {
				status, body := p.updateBehavior(n)
				writeJSON(w, status, body)
				return
			}
			writeJSON(w, 200, `{"success":true,"msg":"ok"}`)
		case strings.Contains(path, "/delClient/"):
			n := atomic.AddInt32(&p.deleteCount, 1)
			if p.deleteBehavior != nil {
				status, body := p.deleteBehavior(n)
				writeJSON(w, status, body)
				return
			}
			writeJSON(w, 200, `{"success":true,"msg":"ok"}`)
		case strings.Contains(path, "/resetClientTraffic/"):
			writeJSON(w, 200, `{"success":true,"msg":"ok"}`)
		case strings.Contains(path, "/getClientTraffics/"):
			atomic.AddInt32(&p.getTraffic, 1)
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

func (p *fakePanel) checkAuth(w http.ResponseWriter, r *http.Request) bool {
	cookie, _ := r.Cookie("session")
	p.mu.Lock()
	authed := p.authedSession && cookie != nil && cookie.Value == "ok"
	p.mu.Unlock()
	if !authed {
		// Simulate 3x-ui's behaviour: when session expired, panel sometimes
		// returns the login HTML page with status 200.
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(200)
		_, _ = io.WriteString(w, `<!DOCTYPE html><html><body>Login</body></html>`)
		return false
	}
	return true
}

// killSession invalidates the current session (simulates panel restart or
// timeout).
func (p *fakePanel) killSession() {
	p.mu.Lock()
	p.authedSession = false
	p.mu.Unlock()
}

func writeJSON(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = io.WriteString(w, body)
}

func newTestClient(t *testing.T, p *fakePanel) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(p.handler())
	c, err := NewClient(srv.URL, "admin", "admin", 1)
	if err != nil {
		srv.Close()
		t.Fatalf("NewClient: %v", err)
	}
	return c, srv
}

func TestClient_AddClient_Success(t *testing.T) {
	p := &fakePanel{}
	c, srv := newTestClient(t, p)
	defer srv.Close()

	got, err := c.AddClient("user_42_123", 5, 30, 3)
	if err != nil {
		t.Fatalf("AddClient: %v", err)
	}
	if got.Email != "user_42_123" {
		t.Errorf("email mismatch: %s", got.Email)
	}
	if got.TotalGB != 5*bytesInGB {
		t.Errorf("TotalGB = %d, want %d", got.TotalGB, 5*bytesInGB)
	}
	if got.ExpiryTime == 0 {
		t.Errorf("ExpiryTime should not be 0 for 30-day plan")
	}
	if atomic.LoadInt32(&p.loginCount) != 1 {
		t.Errorf("login count = %d, want 1", p.loginCount)
	}
	if atomic.LoadInt32(&p.addCount) != 1 {
		t.Errorf("add count = %d, want 1", p.addCount)
	}
}

func TestClient_AddClient_UnlimitedAndNoExpiry(t *testing.T) {
	p := &fakePanel{}
	c, srv := newTestClient(t, p)
	defer srv.Close()

	got, err := c.AddClient("user_unlim", 0, 0, 5)
	if err != nil {
		t.Fatalf("AddClient: %v", err)
	}
	if got.TotalGB != 0 {
		t.Errorf("TotalGB = %d, want 0 for unlimited", got.TotalGB)
	}
	if got.ExpiryTime != 0 {
		t.Errorf("ExpiryTime = %d, want 0 for no-expiry", got.ExpiryTime)
	}
	if got.LimitIP != 5 {
		t.Errorf("LimitIP = %d, want 5", got.LimitIP)
	}
}

func TestClient_AutoRelogin_OnSessionLost(t *testing.T) {
	p := &fakePanel{}
	c, srv := newTestClient(t, p)
	defer srv.Close()

	// Initial login + first AddClient
	if _, err := c.AddClient("user_1", 1, 1, 3); err != nil {
		t.Fatalf("first AddClient: %v", err)
	}
	if got := atomic.LoadInt32(&p.loginCount); got != 1 {
		t.Errorf("login count = %d, want 1", got)
	}

	// Simulate panel session loss (cookie no longer recognized)
	p.killSession()

	// Second call should transparently re-login and succeed
	if _, err := c.AddClient("user_2", 1, 1, 3); err != nil {
		t.Fatalf("second AddClient after session loss: %v", err)
	}
	if got := atomic.LoadInt32(&p.loginCount); got != 2 {
		t.Errorf("login count after relogin = %d, want 2", got)
	}
	if got := atomic.LoadInt32(&p.addCount); got != 2 {
		// addCount only increments after checkAuth succeeds: 1 ok + 1 retry
		t.Errorf("add count = %d, want 2 (1 ok + 1 retry after relogin)", got)
	}
}

func TestClient_UpdateClientTraffic_RecreateOnNotFound(t *testing.T) {
	p := &fakePanel{}
	p.updateBehavior = func(call int32) (int, string) {
		// Always return not-found for update calls
		return 200, `{"success":false,"msg":"client not found"}`
	}
	// Make sure GetInbound returns empty client list (so DeleteClientByEmail
	// finds nothing and returns nil without errors).
	p.inboundObj = `{"id":1,"settings":"{\"clients\":[]}","streamSettings":"{}"}`

	c, srv := newTestClient(t, p)
	defer srv.Close()

	err := c.UpdateClientTraffic("uuid-x", "user_recreate", 5*bytesInGB, 0, 3)
	if err != nil {
		t.Fatalf("UpdateClientTraffic: %v", err)
	}
	if atomic.LoadInt32(&p.updateCount) != 1 {
		t.Errorf("expected 1 update attempt, got %d", p.updateCount)
	}
	if atomic.LoadInt32(&p.addCount) != 1 {
		t.Errorf("expected 1 recreate via AddClient, got %d", p.addCount)
	}
}

func TestClient_UpdateClientTraffic_DuplicateEmailFallback(t *testing.T) {
	p := &fakePanel{}
	p.updateBehavior = func(call int32) (int, string) {
		// Update returns "Email Already Exists" — current XUI phrasing.
		return 200, `{"success":false,"msg":"Email Already Exists"}`
	}
	p.addBehavior = func(call int32) (int, string) {
		// First add (recreate) also collides; second succeeds.
		// But for duplicate-on-update path we expect addClientWithUUIDBytes
		// called once (with new email).
		return 200, `{"success":true}`
	}
	p.inboundObj = `{"id":1,"settings":"{\"clients\":[]}","streamSettings":"{}"}`

	c, srv := newTestClient(t, p)
	defer srv.Close()

	err := c.UpdateClientTraffic("uuid-x", "user_dup", 2*bytesInGB, 0, 3)
	if err != nil {
		t.Fatalf("UpdateClientTraffic: %v", err)
	}
	if atomic.LoadInt32(&p.addCount) != 1 {
		t.Errorf("expected 1 add (with new email), got %d", p.addCount)
	}
}

func TestClient_GetClientTraffic_NullObj(t *testing.T) {
	p := &fakePanel{trafficObj: `null`}
	c, srv := newTestClient(t, p)
	defer srv.Close()

	tr, err := c.GetClientTraffic("nonexistent")
	if err != nil {
		t.Fatalf("GetClientTraffic: %v", err)
	}
	if tr != nil {
		t.Errorf("expected nil traffic for null obj, got %+v", tr)
	}
}

func TestClient_GetClientTraffic_RealValue(t *testing.T) {
	p := &fakePanel{trafficObj: `{"email":"u","enable":true,"up":100,"down":200,"total":1000,"expiryTime":0}`}
	c, srv := newTestClient(t, p)
	defer srv.Close()

	tr, err := c.GetClientTraffic("u")
	if err != nil {
		t.Fatalf("GetClientTraffic: %v", err)
	}
	if tr == nil || tr.Up != 100 || tr.Down != 200 {
		t.Errorf("unexpected traffic: %+v", tr)
	}
}

func TestClient_DeleteClient(t *testing.T) {
	p := &fakePanel{}
	c, srv := newTestClient(t, p)
	defer srv.Close()

	if err := c.DeleteClient("some-uuid"); err != nil {
		t.Fatalf("DeleteClient: %v", err)
	}
	if atomic.LoadInt32(&p.deleteCount) != 1 {
		t.Errorf("delete count = %d, want 1", p.deleteCount)
	}
}

func TestClient_DeleteClientByEmail_NotFound_NoError(t *testing.T) {
	// inbound returns empty client list — no client with given email
	p := &fakePanel{
		inboundObj: `{"id":1,"settings":"{\"clients\":[]}","streamSettings":"{}"}`,
	}
	c, srv := newTestClient(t, p)
	defer srv.Close()

	if err := c.DeleteClientByEmail("ghost"); err != nil {
		t.Fatalf("expected nil for missing client, got %v", err)
	}
	if atomic.LoadInt32(&p.deleteCount) != 0 {
		t.Errorf("expected no delete call, got %d", p.deleteCount)
	}
}

func TestClient_DeleteClientByEmail_FoundAndDeleted(t *testing.T) {
	settings := `{"clients":[{"id":"abc-uuid","email":"target","enable":true}]}`
	settingsEsc, _ := json.Marshal(settings)
	p := &fakePanel{
		inboundObj: fmt.Sprintf(`{"id":1,"settings":%s,"streamSettings":"{}"}`, settingsEsc),
	}
	c, srv := newTestClient(t, p)
	defer srv.Close()

	if err := c.DeleteClientByEmail("target"); err != nil {
		t.Fatalf("DeleteClientByEmail: %v", err)
	}
	if atomic.LoadInt32(&p.deleteCount) != 1 {
		t.Errorf("expected 1 delete call, got %d", p.deleteCount)
	}
}

func TestClient_GetInboundInfo(t *testing.T) {
	stream := `{"network":"tcp","security":"reality","realitySettings":{"serverNames":["www.example.com"],"shortIds":["abcd"],"settings":{"publicKey":"PK","fingerprint":"chrome","serverName":"www.example.com"}}}`
	streamEsc, _ := json.Marshal(stream)
	p := &fakePanel{
		inboundObj: fmt.Sprintf(`{"id":1,"port":443,"settings":"{}","streamSettings":%s}`, streamEsc),
	}
	c, srv := newTestClient(t, p)
	defer srv.Close()

	info, err := c.GetInboundInfo()
	if err != nil {
		t.Fatalf("GetInboundInfo: %v", err)
	}
	if info.Port != 443 || info.PublicKey != "PK" || info.ServerName != "www.example.com" || info.ShortID != "abcd" {
		t.Errorf("unexpected info: %+v", info)
	}
}

func TestClient_LoginFailure_NotMarkedAuthed(t *testing.T) {
	mux := http.NewServeMux()
	var loginHits int32
	mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&loginHits, 1)
		writeJSON(w, 401, `{"success":false,"msg":"bad creds"}`)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c, err := NewClient(srv.URL, "x", "x", 1)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	if _, err := c.AddClient("u", 1, 1, 1); err == nil {
		t.Fatalf("expected AddClient to fail when login fails")
	}
	if c.loggedIn {
		t.Errorf("loggedIn should remain false after failed login")
	}
}
