//go:build integration

package xui

import (
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"
)

// liveClient builds a real XUI client from environment variables. The test is
// skipped when the variables aren't set, so unit-test runs aren't affected.
//
// Required env:
//
//	XUI_TEST_BASE_URL       http(s)://host:port[/prefix]
//	XUI_TEST_USERNAME
//	XUI_TEST_PASSWORD
//	XUI_TEST_INBOUND_ID     (numeric)
func liveClient(t *testing.T) *Client {
	t.Helper()
	base := os.Getenv("XUI_TEST_BASE_URL")
	user := os.Getenv("XUI_TEST_USERNAME")
	pass := os.Getenv("XUI_TEST_PASSWORD")
	inboundStr := os.Getenv("XUI_TEST_INBOUND_ID")

	if base == "" || user == "" || pass == "" || inboundStr == "" {
		t.Skip("XUI_TEST_* env vars not set; skipping live XUI integration test")
	}

	inbound, err := strconv.Atoi(inboundStr)
	if err != nil {
		t.Fatalf("XUI_TEST_INBOUND_ID must be numeric: %v", err)
	}

	c, err := NewClient(base, user, pass, inbound)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c
}

// TestLive_FullClientLifecycle exercises the whole CRUD flow against a real
// 3x-ui panel. Use against a non-prod inbound — it creates and deletes test
// clients.
func TestLive_FullClientLifecycle(t *testing.T) {
	c := liveClient(t)

	email := fmt.Sprintf("zyvpn_it_%d", time.Now().UnixNano())

	// 1) Create
	created, err := c.AddClient(email, 1, 1, 3)
	if err != nil {
		t.Fatalf("AddClient: %v", err)
	}
	t.Logf("created client uuid=%s email=%s", created.ID, created.Email)

	// Best-effort cleanup at the end.
	defer func() {
		if err := c.DeleteClient(created.ID); err != nil {
			t.Logf("cleanup DeleteClient: %v", err)
		}
	}()

	// 2) Read traffic for the new client (should not be nil)
	tr, err := c.GetClientTraffic(email)
	if err != nil {
		t.Fatalf("GetClientTraffic: %v", err)
	}
	if tr == nil {
		t.Fatalf("GetClientTraffic returned nil for fresh client")
	}
	if tr.Up != 0 || tr.Down != 0 {
		t.Logf("note: fresh client already has traffic Up=%d Down=%d", tr.Up, tr.Down)
	}

	// 3) Update traffic limit (3 GB) and extend expiry by 1 day
	newExpiry := time.Now().Add(2 * 24 * time.Hour).UnixMilli()
	if err := c.UpdateClientTraffic(created.ID, email, 3*bytesInGB, newExpiry, 3); err != nil {
		t.Fatalf("UpdateClientTraffic: %v", err)
	}

	// 4) Get inbound info — common pre-flight call from admin "test connection"
	info, err := c.GetInboundInfo()
	if err != nil {
		t.Fatalf("GetInboundInfo: %v", err)
	}
	if info.Port == 0 {
		t.Errorf("GetInboundInfo: zero port")
	}
}

// TestLive_GetClientTraffic_Missing checks behaviour for a non-existent email.
// 3x-ui returns success=true with obj=null in this case; the client must
// translate that into (nil, nil).
func TestLive_GetClientTraffic_Missing(t *testing.T) {
	c := liveClient(t)

	tr, err := c.GetClientTraffic(fmt.Sprintf("zyvpn_it_doesnotexist_%d", time.Now().UnixNano()))
	if err != nil {
		t.Fatalf("GetClientTraffic on missing email: %v", err)
	}
	if tr != nil {
		t.Errorf("expected nil for missing client, got %+v", tr)
	}
}

// TestLive_GetInboundInfo verifies that we can parse the Reality settings
// returned by the live panel.
func TestLive_GetInboundInfo(t *testing.T) {
	c := liveClient(t)

	info, err := c.GetInboundInfo()
	if err != nil {
		t.Fatalf("GetInboundInfo: %v", err)
	}
	if info.Port == 0 || info.PublicKey == "" || info.ServerName == "" {
		t.Errorf("incomplete inbound info: %+v", info)
	}
}
