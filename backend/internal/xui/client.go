package xui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Client struct {
	baseURL   string
	username  string
	password  string
	inboundID int
	client    *http.Client
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
	Email    string `json:"email"`
	Enable   bool   `json:"enable"`
	Up       int64  `json:"up"`
	Down     int64  `json:"down"`
	Total    int64  `json:"total"`
	ExpiryTime int64 `json:"expiryTime"`
}

type InboundSettings struct {
	Clients []ClientConfig `json:"clients"`
}

type Inbound struct {
	ID          int             `json:"id"`
	Remark      string          `json:"remark"`
	Enable      bool            `json:"enable"`
	Protocol    string          `json:"protocol"`
	Settings    string          `json:"settings"`
	StreamSettings string       `json:"streamSettings"`
	Port        int             `json:"port"`
	Tag         string          `json:"tag"`
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

	return &Client{
		baseURL:   baseURL,
		username:  username,
		password:  password,
		inboundID: inboundID,
		client: &http.Client{
			Timeout: 30 * time.Second,
			Jar:     jar,
		},
	}, nil
}

func (c *Client) Login() error {
	data := map[string]string{
		"username": c.username,
		"password": c.password,
	}

	body, err := json.Marshal(data)
	if err != nil {
		return err
	}

	fmt.Printf("[XUI] Logging in to %s...\n", c.baseURL)
	resp, err := c.client.Post(c.baseURL+"/login", "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("login request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	fmt.Printf("[XUI] Login response status: %d, body: %s\n", resp.StatusCode, string(respBody))

	var result Response
	if err := json.Unmarshal(respBody, &result); err != nil {
		return fmt.Errorf("failed to decode login response: %w", err)
	}

	if !result.Success {
		return fmt.Errorf("login failed: %s", result.Msg)
	}

	c.loggedIn = true
	fmt.Println("[XUI] Login successful")
	return nil
}

func (c *Client) ensureLoggedIn() error {
	if !c.loggedIn {
		return c.Login()
	}
	return nil
}

func (c *Client) AddClient(email string, trafficLimitGB int64, expiryDays int, maxDevices int) (*ClientConfig, error) {
	if err := c.ensureLoggedIn(); err != nil {
		return nil, err
	}

	clientID := uuid.New().String()

	var expiryTime int64 = 0
	if expiryDays > 0 {
		expiryTime = time.Now().Add(time.Duration(expiryDays) * 24 * time.Hour).UnixMilli()
	}

	var totalGB int64 = 0
	if trafficLimitGB > 0 {
		totalGB = trafficLimitGB * 1024 * 1024 * 1024
	}

	if maxDevices <= 0 {
		maxDevices = 3 // Default to 3 devices
	}

	client := ClientConfig{
		ID:         clientID,
		Email:      email,
		Enable:     true,
		Flow:       "xtls-rprx-vision",
		LimitIP:    maxDevices,
		TotalGB:    totalGB,
		ExpiryTime: expiryTime,
	}

	settings := map[string]interface{}{
		"clients": []ClientConfig{client},
	}

	settingsJSON, err := json.Marshal(settings)
	if err != nil {
		return nil, err
	}

	data := map[string]interface{}{
		"id":       c.inboundID,
		"settings": string(settingsJSON),
	}

	body, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	log.Printf("[XUI] AddClient URL: %s", c.baseURL+"/panel/api/inbounds/addClient")
	log.Printf("[XUI] AddClient Body: %s", string(body))

	resp, err := c.client.Post(c.baseURL+"/panel/api/inbounds/addClient", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("add client request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body for logging
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	log.Printf("[XUI] AddClient Response Status: %d", resp.StatusCode)
	log.Printf("[XUI] AddClient Response: %s", string(bodyBytes))

	var result Response
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return nil, fmt.Errorf("failed to decode response (status=%d, body=%s): %w", resp.StatusCode, string(bodyBytes), err)
	}

	if !result.Success {
		return nil, fmt.Errorf("add client failed: %s", result.Msg)
	}

	return &client, nil
}

func (c *Client) DeleteClient(clientID string) error {
	if err := c.ensureLoggedIn(); err != nil {
		return err
	}

	url := fmt.Sprintf("%s/panel/api/inbounds/%d/delClient/%s", c.baseURL, c.inboundID, clientID)

	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		return err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("delete client request failed: %w", err)
	}
	defer resp.Body.Close()

	var result Response
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if !result.Success {
		return fmt.Errorf("delete client failed: %s", result.Msg)
	}

	return nil
}

// DeleteClientByEmail finds a client by email and deletes it by UUID
func (c *Client) DeleteClientByEmail(email string) error {
	if err := c.ensureLoggedIn(); err != nil {
		return err
	}

	// First, get the inbound to find the client UUID by email
	inbound, err := c.GetInbound()
	if err != nil {
		fmt.Printf("[XUI] DeleteClientByEmail: failed to get inbound: %v\n", err)
		return err
	}

	// Parse settings to get clients list
	var settings InboundSettings
	if err := json.Unmarshal([]byte(inbound.Settings), &settings); err != nil {
		fmt.Printf("[XUI] DeleteClientByEmail: failed to parse settings: %v\n", err)
		return err
	}

	// Find client by email
	var clientUUID string
	for _, client := range settings.Clients {
		if client.Email == email {
			clientUUID = client.ID
			break
		}
	}

	if clientUUID == "" {
		fmt.Printf("[XUI] DeleteClientByEmail: client with email %s not found\n", email)
		return nil // Not an error - client doesn't exist
	}

	fmt.Printf("[XUI] DeleteClientByEmail: found client %s with UUID %s, deleting...\n", email, clientUUID)

	// Delete by UUID
	url := fmt.Sprintf("%s/panel/api/inbounds/%d/delClient/%s", c.baseURL, c.inboundID, clientUUID)
	fmt.Printf("[XUI] DeleteClientByEmail URL: %s\n", url)

	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		return err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("delete client request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	fmt.Printf("[XUI] DeleteClientByEmail Response: %d - %s\n", resp.StatusCode, string(respBody))

	var result Response
	if err := json.Unmarshal(respBody, &result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if !result.Success {
		return fmt.Errorf("delete client failed: %s", result.Msg)
	}

	fmt.Printf("[XUI] DeleteClientByEmail: successfully deleted client %s\n", email)
	return nil
}

func (c *Client) GetClientTraffic(email string) (*Traffic, error) {
	if err := c.ensureLoggedIn(); err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/panel/api/inbounds/getClientTraffics/%s", c.baseURL, email)

	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("get traffic request failed: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Success bool     `json:"success"`
		Msg     string   `json:"msg"`
		Obj     *Traffic `json:"obj"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !result.Success {
		return nil, fmt.Errorf("get traffic failed: %s", result.Msg)
	}

	return result.Obj, nil
}

func (c *Client) UpdateClientTraffic(clientUUID string, email string, totalGB int64, expiryTime int64, maxDevices int) error {
	if maxDevices <= 0 {
		maxDevices = 3
	}
	err := c.updateClientTrafficWithRetry(clientUUID, email, totalGB, expiryTime, maxDevices, true)

	// If client not found (404), try to recreate it
	if err != nil && (strings.Contains(err.Error(), "status=404") || strings.Contains(err.Error(), "not found")) {
		fmt.Printf("[XUI] Client %s not found, recreating...\n", clientUUID)

		// First try to delete any existing client with this email (cleanup)
		_ = c.DeleteClientByEmail(email)

		// Try to create with original email first
		createErr := c.addClientWithUUID(clientUUID, email, totalGB, expiryTime, maxDevices)

		// If duplicate email (exists in another inbound), use new unique email
		if createErr != nil && strings.Contains(createErr.Error(), "duplicate") {
			newEmail := fmt.Sprintf("%s_%d", email, time.Now().Unix())
			fmt.Printf("[XUI] Duplicate email in another inbound, using new email: %s\n", newEmail)
			return c.addClientWithUUID(clientUUID, newEmail, totalGB, expiryTime, maxDevices)
		}

		return createErr
	}

	// If duplicate email error on update, try with new email
	if err != nil && strings.Contains(err.Error(), "duplicate") {
		fmt.Printf("[XUI] Duplicate email detected, trying with new email...\n")
		_ = c.DeleteClientByEmail(email)
		newEmail := fmt.Sprintf("%s_%d", email, time.Now().Unix())
		return c.addClientWithUUID(clientUUID, newEmail, totalGB, expiryTime, maxDevices)
	}

	return err
}

// addClientWithUUID adds a client with a specific UUID (for recreating deleted clients)
func (c *Client) addClientWithUUID(clientUUID string, email string, totalGB int64, expiryTime int64, maxDevices int) error {
	if err := c.ensureLoggedIn(); err != nil {
		return err
	}

	if maxDevices <= 0 {
		maxDevices = 3
	}

	client := ClientConfig{
		ID:         clientUUID,
		Email:      email,
		Enable:     true,
		Flow:       "xtls-rprx-vision",
		LimitIP:    maxDevices,
		TotalGB:    totalGB * 1024 * 1024 * 1024,
		ExpiryTime: expiryTime,
	}

	settings := map[string]interface{}{
		"clients": []ClientConfig{client},
	}

	settingsJSON, err := json.Marshal(settings)
	if err != nil {
		return err
	}

	data := map[string]interface{}{
		"id":       c.inboundID,
		"settings": string(settingsJSON),
	}

	body, err := json.Marshal(data)
	if err != nil {
		return err
	}

	fmt.Printf("[XUI] AddClient URL: %s/panel/api/inbounds/addClient\n", c.baseURL)
	fmt.Printf("[XUI] AddClient Body: %s\n", string(body))

	resp, err := c.client.Post(c.baseURL+"/panel/api/inbounds/addClient", "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("add client request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	fmt.Printf("[XUI] AddClient Response Status: %d\n", resp.StatusCode)
	fmt.Printf("[XUI] AddClient Response: %s\n", string(respBody))

	var result Response
	if err := json.Unmarshal(respBody, &result); err != nil {
		return fmt.Errorf("failed to decode response: %w, body: %s", err, string(respBody))
	}

	if !result.Success {
		return fmt.Errorf("add client failed: %s", result.Msg)
	}

	fmt.Printf("[XUI] Client %s created successfully\n", clientUUID)
	return nil
}

func (c *Client) updateClientTrafficWithRetry(clientUUID string, email string, totalGB int64, expiryTime int64, maxDevices int, canRetry bool) error {
	if err := c.ensureLoggedIn(); err != nil {
		return err
	}

	if maxDevices <= 0 {
		maxDevices = 3
	}

	client := ClientConfig{
		ID:         clientUUID,
		Email:      email,
		Enable:     true,
		Flow:       "xtls-rprx-vision",
		LimitIP:    maxDevices,
		TotalGB:    totalGB * 1024 * 1024 * 1024,
		ExpiryTime: expiryTime,
	}

	settings := map[string]interface{}{
		"clients": []ClientConfig{client},
	}

	settingsJSON, err := json.Marshal(settings)
	if err != nil {
		return err
	}

	data := map[string]interface{}{
		"id":       c.inboundID,
		"settings": string(settingsJSON),
	}

	body, err := json.Marshal(data)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/panel/api/inbounds/%d/updateClient/%s", c.baseURL, c.inboundID, clientUUID)
	fmt.Printf("[XUI] UpdateClient URL: %s\n", url)
	fmt.Printf("[XUI] UpdateClient Body: %s\n", string(body))

	resp, err := c.client.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("update client request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	fmt.Printf("[XUI] UpdateClient Response Status: %d\n", resp.StatusCode)
	fmt.Printf("[XUI] UpdateClient Response: %s\n", string(respBody))

	// Check status code - 3xx or non-200 with empty body means auth issue
	if resp.StatusCode != 200 || (len(respBody) == 0 && canRetry) {
		fmt.Printf("[XUI] Bad response (status=%d, body_len=%d), attempting re-login...\n", resp.StatusCode, len(respBody))
		c.loggedIn = false
		// Force re-login
		if err := c.Login(); err != nil {
			return fmt.Errorf("re-login failed: %w", err)
		}
		if canRetry {
			return c.updateClientTrafficWithRetry(clientUUID, email, totalGB, expiryTime, maxDevices, false)
		}
		return fmt.Errorf("update client failed after re-login: status=%d, body=%s", resp.StatusCode, string(respBody))
	}

	var result Response
	if err := json.Unmarshal(respBody, &result); err != nil {
		return fmt.Errorf("failed to decode response: %w, body: %s", err, string(respBody))
	}

	if !result.Success {
		return fmt.Errorf("update client failed: %s", result.Msg)
	}

	return nil
}

func (c *Client) ResetClientTraffic(email string) error {
	if err := c.ensureLoggedIn(); err != nil {
		return err
	}

	url := fmt.Sprintf("%s/panel/api/inbounds/%d/resetClientTraffic/%s", c.baseURL, c.inboundID, email)

	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		return err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("reset traffic request failed: %w", err)
	}
	defer resp.Body.Close()

	var result Response
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if !result.Success {
		return fmt.Errorf("reset traffic failed: %s", result.Msg)
	}

	return nil
}

func (c *Client) GetInbound() (*Inbound, error) {
	if err := c.ensureLoggedIn(); err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/panel/api/inbounds/get/%d", c.baseURL, c.inboundID)

	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("get inbound request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var result struct {
		Success bool     `json:"success"`
		Msg     string   `json:"msg"`
		Obj     *Inbound `json:"obj"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !result.Success {
		return nil, fmt.Errorf("get inbound failed: %s", result.Msg)
	}

	return result.Obj, nil
}

// GenerateVLESSLink generates a VLESS connection link for a client
func (c *Client) GenerateVLESSLink(clientID, email, serverAddress string, port int, publicKey, shortID, serverName string) string {
	// VLESS + Reality format
	return fmt.Sprintf("vless://%s@%s:%d?type=tcp&security=reality&pbk=%s&fp=chrome&sni=%s&sid=%s&spx=%%2F&flow=xtls-rprx-vision#%s",
		clientID,
		serverAddress,
		port,
		publicKey,
		serverName,
		shortID,
		email,
	)
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

// GetInboundInfo retrieves and parses inbound settings
func (c *Client) GetInboundInfo() (*InboundInfo, error) {
	inbound, err := c.GetInbound()
	if err != nil {
		return nil, err
	}

	var streamSettings StreamSettings
	if err := json.Unmarshal([]byte(inbound.StreamSettings), &streamSettings); err != nil {
		return nil, fmt.Errorf("failed to parse stream settings: %w", err)
	}

	info := &InboundInfo{
		Port: inbound.Port,
	}

	// Get public key from reality settings
	if streamSettings.RealitySettings.Settings.PublicKey != "" {
		info.PublicKey = streamSettings.RealitySettings.Settings.PublicKey
	}

	// Get server name
	if len(streamSettings.RealitySettings.ServerNames) > 0 {
		info.ServerName = streamSettings.RealitySettings.ServerNames[0]
	} else if streamSettings.RealitySettings.Settings.ServerName != "" {
		info.ServerName = streamSettings.RealitySettings.Settings.ServerName
	}

	// Get short ID
	if len(streamSettings.RealitySettings.ShortIds) > 0 {
		info.ShortID = streamSettings.RealitySettings.ShortIds[0]
	}

	return info, nil
}
