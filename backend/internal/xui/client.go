package xui

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

const bytesInGB = int64(1024 * 1024 * 1024)

type Client struct {
	baseURL   string
	username  string
	password  string
	inboundID int
	client    *http.Client
	mu        sync.Mutex
	loggedIn  bool
}

type ClientConfig struct {
	ID         string `json:"id"`
	Email      string `json:"email"`
	Enable     bool   `json:"enable"`
	Flow       string `json:"flow"`
	LimitIP    int    `json:"limitIp"`
	TotalGB    int64  `json:"totalGB"`
	ExpiryTime int64  `json:"expiryTime"` // milliseconds timestamp
}

type Traffic struct {
	Email      string `json:"email"`
	Enable     bool   `json:"enable"`
	Up         int64  `json:"up"`
	Down       int64  `json:"down"`
	Total      int64  `json:"total"`
	ExpiryTime int64  `json:"expiryTime"`
}

type InboundSettings struct {
	Clients []ClientConfig `json:"clients"`
}

type Inbound struct {
	ID             int    `json:"id"`
	Remark         string `json:"remark"`
	Enable         bool   `json:"enable"`
	Protocol       string `json:"protocol"`
	Settings       string `json:"settings"`
	StreamSettings string `json:"streamSettings"`
	Port           int    `json:"port"`
	Tag            string `json:"tag"`
}

type Response struct {
	Success bool        `json:"success"`
	Msg     string      `json:"msg"`
	Obj     interface{} `json:"obj"`
}

func NewClient(baseURL, username, password string, inboundID int) (*Client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}

	// 3x-ui panels are commonly served behind self-signed certs on a random
	// port. Skipping verification is acceptable here because the credentials
	// already authenticate the panel; we don't rely on TLS for trust.
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	return &Client{
		baseURL:   strings.TrimRight(baseURL, "/"),
		username:  username,
		password:  password,
		inboundID: inboundID,
		client: &http.Client{
			Timeout:   30 * time.Second,
			Jar:       jar,
			Transport: transport,
		},
	}, nil
}

// bytesToTotalGB converts a byte count to the value 3x-ui expects in TotalGB
// (which is actually bytes despite the name). Returns 0 only if input is 0
// (means unlimited in 3x-ui). Sub-GB plans are rounded up to 1 GB to avoid
// accidentally creating an unlimited client.
func bytesToTotalGB(bytesLimit int64) int64 {
	if bytesLimit <= 0 {
		return 0
	}
	if bytesLimit < bytesInGB {
		return bytesInGB
	}
	return bytesLimit
}

func (c *Client) login() error {
	data := map[string]string{
		"username": c.username,
		"password": c.password,
	}

	body, err := json.Marshal(data)
	if err != nil {
		return err
	}

	log.Printf("[XUI] Logging in to %s...", c.baseURL)
	resp, err := c.client.Post(c.baseURL+"/login", "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("login request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("login failed: status=%d, body=%s", resp.StatusCode, string(respBody))
	}

	var result Response
	if err := json.Unmarshal(respBody, &result); err != nil {
		return fmt.Errorf("failed to decode login response (status=%d): %w", resp.StatusCode, err)
	}

	if !result.Success {
		return fmt.Errorf("login failed: %s", result.Msg)
	}

	c.loggedIn = true
	log.Println("[XUI] Login successful")
	return nil
}

// Login is exposed for tests/diagnostics. Production code should rely on
// automatic login through doRequest.
func (c *Client) Login() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.login()
}

func (c *Client) ensureLoggedIn() error {
	if c.loggedIn {
		return nil
	}
	return c.login()
}

// looksLikeAuthIssue heuristically detects when 3x-ui served us the login HTML
// instead of a JSON response (cookie session expired, restart, etc).
//
// 404 is treated as a legitimate "not found" — 3x-ui returns it (often with
// an empty body) when the targeted client/inbound doesn't exist, and the
// callers handle that semantically via isClientNotFound. Re-logging in
// wouldn't change the outcome there.
func looksLikeAuthIssue(status int, body []byte) bool {
	if status == http.StatusNotFound {
		return false
	}
	if status == http.StatusUnauthorized || status == http.StatusForbidden {
		return true
	}
	if status >= 300 && status < 400 {
		return true
	}
	if len(body) == 0 {
		return true
	}
	prefix := bytes.TrimSpace(body)
	if len(prefix) > 0 && prefix[0] != '{' && prefix[0] != '[' {
		return true
	}
	return false
}

// doRequest sends an HTTP request and transparently re-logs in once if the
// response looks like an authentication failure. It always returns the final
// response body and status (not the auth-failure ones).
func (c *Client) doRequest(method, path string, payload interface{}) (int, []byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.ensureLoggedIn(); err != nil {
		return 0, nil, err
	}

	url := c.baseURL + path

	send := func() (int, []byte, error) {
		var body io.Reader
		if payload != nil {
			b, err := json.Marshal(payload)
			if err != nil {
				return 0, nil, fmt.Errorf("marshal request: %w", err)
			}
			body = bytes.NewReader(b)
		}

		req, err := http.NewRequest(method, url, body)
		if err != nil {
			return 0, nil, err
		}
		if payload != nil {
			req.Header.Set("Content-Type", "application/json")
		}

		resp, err := c.client.Do(req)
		if err != nil {
			return 0, nil, fmt.Errorf("%s %s: %w", method, path, err)
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return resp.StatusCode, nil, fmt.Errorf("read body: %w", err)
		}
		return resp.StatusCode, respBody, nil
	}

	status, body, err := send()
	if err != nil {
		return status, body, err
	}

	if looksLikeAuthIssue(status, body) {
		log.Printf("[XUI] %s %s: auth issue (status=%d, body_len=%d), re-logging in", method, path, status, len(body))
		c.loggedIn = false
		if err := c.login(); err != nil {
			return status, body, fmt.Errorf("re-login failed: %w", err)
		}
		return send()
	}

	return status, body, nil
}

// callJSON is a thin convenience wrapper around doRequest that decodes the
// 3x-ui-style envelope and returns the raw obj as bytes for the caller to
// further parse.
func (c *Client) callJSON(method, path string, payload interface{}) (json.RawMessage, error) {
	status, body, err := c.doRequest(method, path, payload)
	if err != nil {
		return nil, err
	}

	var env struct {
		Success bool            `json:"success"`
		Msg     string          `json:"msg"`
		Obj     json.RawMessage `json:"obj"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, fmt.Errorf("decode response (status=%d, body=%s): %w", status, string(body), err)
	}

	if !env.Success {
		return env.Obj, fmt.Errorf("%s", env.Msg)
	}
	return env.Obj, nil
}

// isDuplicateEmail returns true for the various phrasings 3x-ui uses when
// a client with the same email already exists in any inbound.
func isDuplicateEmail(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "already exists") ||
		strings.Contains(msg, "duplicate") ||
		strings.Contains(msg, "уже существует") ||
		strings.Contains(msg, "exists")
}

// isClientNotFound matches 3x-ui's "client not found" / 404-style errors.
// Includes the "Inbound Not Found For Email" phrasing returned by recent
// 3x-ui builds when querying traffic for a non-existent email.
func isClientNotFound(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "status=404") ||
		strings.Contains(msg, "not found") ||
		strings.Contains(msg, "не найден") ||
		strings.Contains(msg, "object not found") ||
		strings.Contains(msg, "inbound not found")
}

func clampMaxDevices(maxDevices int) int {
	if maxDevices <= 0 {
		return 3
	}
	return maxDevices
}

func (c *Client) buildClientConfig(clientID, email string, trafficBytes int64, expiryTime int64, maxDevices int) ClientConfig {
	return ClientConfig{
		ID:         clientID,
		Email:      email,
		Enable:     true,
		Flow:       "xtls-rprx-vision",
		LimitIP:    clampMaxDevices(maxDevices),
		TotalGB:    bytesToTotalGB(trafficBytes),
		ExpiryTime: expiryTime,
	}
}

// AddClient creates a new VLESS client. trafficLimitGB is in gigabytes
// (0 = unlimited); expiryDays is in days from now (0 = no expiry).
func (c *Client) AddClient(email string, trafficLimitGB int64, expiryDays int, maxDevices int) (*ClientConfig, error) {
	clientID := uuid.New().String()

	var expiryTime int64
	if expiryDays > 0 {
		expiryTime = time.Now().Add(time.Duration(expiryDays) * 24 * time.Hour).UnixMilli()
	}

	var trafficBytes int64
	if trafficLimitGB > 0 {
		trafficBytes = trafficLimitGB * bytesInGB
	}

	cfg := c.buildClientConfig(clientID, email, trafficBytes, expiryTime, maxDevices)
	if err := c.addClientRaw(cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// addClientWithUUIDBytes re-creates a client with a specific UUID (used by
// the recovery path in UpdateClientTraffic when the client has disappeared
// from the panel). trafficLimitBytes is the raw byte budget (0 = unlimited).
func (c *Client) addClientWithUUIDBytes(clientUUID, email string, trafficLimitBytes, expiryTime int64, maxDevices int) error {
	cfg := c.buildClientConfig(clientUUID, email, trafficLimitBytes, expiryTime, maxDevices)
	return c.addClientRaw(cfg)
}

func (c *Client) addClientRaw(cfg ClientConfig) error {
	settingsJSON, err := json.Marshal(map[string]interface{}{
		"clients": []ClientConfig{cfg},
	})
	if err != nil {
		return err
	}

	payload := map[string]interface{}{
		"id":       c.inboundID,
		"settings": string(settingsJSON),
	}

	if _, err := c.callJSON(http.MethodPost, "/panel/api/inbounds/addClient", payload); err != nil {
		return fmt.Errorf("add client failed: %w", err)
	}
	log.Printf("[XUI] Client created: id=%s, email=%s", cfg.ID, cfg.Email)
	return nil
}

func (c *Client) DeleteClient(clientID string) error {
	path := fmt.Sprintf("/panel/api/inbounds/%d/delClient/%s", c.inboundID, clientID)
	if _, err := c.callJSON(http.MethodPost, path, nil); err != nil {
		return fmt.Errorf("delete client failed: %w", err)
	}
	return nil
}

// DeleteClientByEmail finds a client by email and deletes it by UUID. Returns
// nil (not an error) when the client doesn't exist.
func (c *Client) DeleteClientByEmail(email string) error {
	inbound, err := c.GetInbound()
	if err != nil {
		return fmt.Errorf("get inbound: %w", err)
	}

	var settings InboundSettings
	if err := json.Unmarshal([]byte(inbound.Settings), &settings); err != nil {
		return fmt.Errorf("parse inbound settings: %w", err)
	}

	var clientUUID string
	for _, cl := range settings.Clients {
		if cl.Email == email {
			clientUUID = cl.ID
			break
		}
	}

	if clientUUID == "" {
		log.Printf("[XUI] DeleteClientByEmail: client %s not found, skipping", email)
		return nil
	}

	log.Printf("[XUI] DeleteClientByEmail: deleting %s (uuid=%s)", email, clientUUID)
	return c.DeleteClient(clientUUID)
}

// GetClientTraffic returns nil, nil when the client doesn't exist on the
// panel. Different 3x-ui versions handle this differently:
//   - older: success=true with obj=null
//   - newer: success=false with msg "Inbound Not Found For Email: ..."
//
// Both shapes are normalized to (nil, nil) so callers can treat "not found"
// as a soft signal instead of an error.
func (c *Client) GetClientTraffic(email string) (*Traffic, error) {
	path := fmt.Sprintf("/panel/api/inbounds/getClientTraffics/%s", email)
	obj, err := c.callJSON(http.MethodGet, path, nil)
	if err != nil {
		if isClientNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("get traffic failed: %w", err)
	}
	if len(obj) == 0 || string(obj) == "null" {
		return nil, nil
	}
	var t Traffic
	if err := json.Unmarshal(obj, &t); err != nil {
		return nil, fmt.Errorf("decode traffic: %w", err)
	}
	return &t, nil
}

// UpdateClientTraffic updates a client's traffic limit and expiry.
// trafficLimitBytes is the raw byte budget (0 = unlimited). If the client has
// been removed from the panel, it is recreated transparently. If the email
// collides with another inbound, a unique suffix is appended.
func (c *Client) UpdateClientTraffic(clientUUID, email string, trafficLimitBytes, expiryTime int64, maxDevices int) error {
	err := c.updateClientRaw(clientUUID, email, trafficLimitBytes, expiryTime, maxDevices)
	if err == nil {
		return nil
	}

	if isClientNotFound(err) {
		log.Printf("[XUI] Client %s missing on panel, recreating", clientUUID)
		// Best-effort cleanup of any stale entry with the same email.
		_ = c.DeleteClientByEmail(email)
		createErr := c.addClientWithUUIDBytes(clientUUID, email, trafficLimitBytes, expiryTime, maxDevices)
		if createErr != nil && isDuplicateEmail(createErr) {
			newEmail := fmt.Sprintf("%s_%d", email, time.Now().Unix())
			log.Printf("[XUI] Duplicate email on recreate, retrying as %s", newEmail)
			return c.addClientWithUUIDBytes(clientUUID, newEmail, trafficLimitBytes, expiryTime, maxDevices)
		}
		return createErr
	}

	if isDuplicateEmail(err) {
		newEmail := fmt.Sprintf("%s_%d", email, time.Now().Unix())
		log.Printf("[XUI] Duplicate email on update, retrying as %s", newEmail)
		_ = c.DeleteClientByEmail(email)
		return c.addClientWithUUIDBytes(clientUUID, newEmail, trafficLimitBytes, expiryTime, maxDevices)
	}

	return err
}

func (c *Client) updateClientRaw(clientUUID, email string, trafficLimitBytes, expiryTime int64, maxDevices int) error {
	cfg := c.buildClientConfig(clientUUID, email, trafficLimitBytes, expiryTime, maxDevices)

	settingsJSON, err := json.Marshal(map[string]interface{}{
		"clients": []ClientConfig{cfg},
	})
	if err != nil {
		return err
	}

	payload := map[string]interface{}{
		"id":       c.inboundID,
		"settings": string(settingsJSON),
	}

	// 3x-ui's updateClient endpoint does NOT take an inbound ID in the path
	// (only the client UUID). The inbound ID is supplied in the body instead.
	// Including it in the path returns 404.
	path := fmt.Sprintf("/panel/api/inbounds/updateClient/%s", clientUUID)
	if _, err := c.callJSON(http.MethodPost, path, payload); err != nil {
		return fmt.Errorf("update client failed: %w", err)
	}
	return nil
}

func (c *Client) ResetClientTraffic(email string) error {
	path := fmt.Sprintf("/panel/api/inbounds/%d/resetClientTraffic/%s", c.inboundID, email)
	if _, err := c.callJSON(http.MethodPost, path, nil); err != nil {
		return fmt.Errorf("reset traffic failed: %w", err)
	}
	return nil
}

func (c *Client) GetInbound() (*Inbound, error) {
	path := fmt.Sprintf("/panel/api/inbounds/get/%d", c.inboundID)
	obj, err := c.callJSON(http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("get inbound failed: %w", err)
	}
	var inbound Inbound
	if err := json.Unmarshal(obj, &inbound); err != nil {
		return nil, fmt.Errorf("decode inbound: %w", err)
	}
	return &inbound, nil
}

// RealitySettings represents Reality security settings
type RealitySettings struct {
	Show        bool     `json:"show"`
	Dest        string   `json:"dest"`
	Xver        int      `json:"xver"`
	ServerNames []string `json:"serverNames"`
	PrivateKey  string   `json:"privateKey"`
	ShortIds    []string `json:"shortIds"`
	Settings    struct {
		PublicKey   string `json:"publicKey"`
		Fingerprint string `json:"fingerprint"`
		ServerName  string `json:"serverName"`
		SpiderX     string `json:"spiderX"`
	} `json:"settings"`
}

// StreamSettings represents inbound stream settings
type StreamSettings struct {
	Network         string          `json:"network"`
	Security        string          `json:"security"`
	RealitySettings RealitySettings `json:"realitySettings"`
}

// InboundInfo contains parsed inbound information for generating keys
type InboundInfo struct {
	Port       int
	PublicKey  string
	ShortID    string
	ServerName string
}

func (c *Client) GetInboundInfo() (*InboundInfo, error) {
	inbound, err := c.GetInbound()
	if err != nil {
		return nil, err
	}

	var streamSettings StreamSettings
	if err := json.Unmarshal([]byte(inbound.StreamSettings), &streamSettings); err != nil {
		return nil, fmt.Errorf("failed to parse stream settings: %w", err)
	}

	info := &InboundInfo{Port: inbound.Port}

	if streamSettings.RealitySettings.Settings.PublicKey != "" {
		info.PublicKey = streamSettings.RealitySettings.Settings.PublicKey
	}
	if len(streamSettings.RealitySettings.ServerNames) > 0 {
		info.ServerName = streamSettings.RealitySettings.ServerNames[0]
	} else if streamSettings.RealitySettings.Settings.ServerName != "" {
		info.ServerName = streamSettings.RealitySettings.Settings.ServerName
	}
	if len(streamSettings.RealitySettings.ShortIds) > 0 {
		info.ShortID = streamSettings.RealitySettings.ShortIds[0]
	}

	return info, nil
}
