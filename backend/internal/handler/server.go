package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/zyvpn/backend/internal/middleware"
	"github.com/zyvpn/backend/internal/model"
	"github.com/zyvpn/backend/internal/service"
)

// ServerHandler handles server-related requests
type ServerHandler struct {
	serverSvc *service.ServerService
}

// NewServerHandler creates a new server handler
func NewServerHandler(serverSvc *service.ServerService) *ServerHandler {
	return &ServerHandler{serverSvc: serverSvc}
}

// --- User Endpoints ---

// GetServers returns list of available servers for users
func (h *ServerHandler) GetServers(c *fiber.Ctx) error {
	servers, err := h.serverSvc.GetActiveServers(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	return c.JSON(fiber.Map{"servers": servers})
}

// --- Admin Endpoints ---

// GetAllServers returns all servers for admin
func (h *ServerHandler) GetAllServers(c *fiber.Ctx) error {
	_ = middleware.GetAdminID(c)
	servers, err := h.serverSvc.GetAllServers(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	return c.JSON(fiber.Map{"servers": servers})
}

// GetServer returns a single server for admin
func (h *ServerHandler) GetServer(c *fiber.Ctx) error {
	_ = middleware.GetAdminID(c)
	serverID, err := uuid.Parse(c.Params("server_id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid server_id",
		})
	}

	server, err := h.serverSvc.GetServer(c.Context(), serverID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "server not found",
		})
	}

	return c.JSON(server.ToAdmin())
}

type CreateServerRequest struct {
	Name          string  `json:"name"`
	Country       string  `json:"country"`
	City          *string `json:"city,omitempty"`
	FlagEmoji     string  `json:"flag_emoji"`
	XUIBaseURL    string  `json:"xui_base_url"`
	XUIUsername   string  `json:"xui_username"`
	XUIPassword   string  `json:"xui_password"`
	XUIInboundID  int     `json:"xui_inbound_id"`
	ServerAddress string  `json:"server_address"`
	ServerPort    int     `json:"server_port"`
	PublicKey     string  `json:"public_key"`
	ShortID       string  `json:"short_id"`
	ServerName    string  `json:"server_name"`
	IsActive      bool    `json:"is_active"`
	SortOrder     int     `json:"sort_order"`
	Capacity      int     `json:"capacity"`
}

// CreateServer creates a new server
func (h *ServerHandler) CreateServer(c *fiber.Ctx) error {
	_ = middleware.GetAdminID(c)

	var req CreateServerRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if req.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "name is required",
		})
	}

	if req.XUIBaseURL == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "xui_base_url is required",
		})
	}

	if req.ServerAddress == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "server_address is required",
		})
	}

	if req.ServerPort <= 0 {
		req.ServerPort = 443
	}

	if req.XUIInboundID <= 0 {
		req.XUIInboundID = 1
	}

	if req.Capacity <= 0 {
		req.Capacity = 100
	}

	server := &model.Server{
		Name:          req.Name,
		Country:       req.Country,
		City:          req.City,
		FlagEmoji:     req.FlagEmoji,
		XUIBaseURL:    req.XUIBaseURL,
		XUIUsername:   req.XUIUsername,
		XUIPassword:   req.XUIPassword,
		XUIInboundID:  req.XUIInboundID,
		ServerAddress: req.ServerAddress,
		ServerPort:    req.ServerPort,
		PublicKey:     req.PublicKey,
		ShortID:       req.ShortID,
		ServerName:    req.ServerName,
		IsActive:      req.IsActive,
		SortOrder:     req.SortOrder,
		Capacity:      req.Capacity,
	}

	if err := h.serverSvc.CreateServer(c.Context(), server); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(server.ToAdmin())
}

type UpdateServerRequest struct {
	Name          *string `json:"name,omitempty"`
	Country       *string `json:"country,omitempty"`
	City          *string `json:"city,omitempty"`
	FlagEmoji     *string `json:"flag_emoji,omitempty"`
	XUIBaseURL    *string `json:"xui_base_url,omitempty"`
	XUIUsername   *string `json:"xui_username,omitempty"`
	XUIPassword   *string `json:"xui_password,omitempty"`
	XUIInboundID  *int    `json:"xui_inbound_id,omitempty"`
	ServerAddress *string `json:"server_address,omitempty"`
	ServerPort    *int    `json:"server_port,omitempty"`
	PublicKey     *string `json:"public_key,omitempty"`
	ShortID       *string `json:"short_id,omitempty"`
	ServerName    *string `json:"server_name,omitempty"`
	IsActive      *bool   `json:"is_active,omitempty"`
	SortOrder     *int    `json:"sort_order,omitempty"`
	Capacity      *int    `json:"capacity,omitempty"`
}

// UpdateServer updates a server
func (h *ServerHandler) UpdateServer(c *fiber.Ctx) error {
	_ = middleware.GetAdminID(c)
	serverID, err := uuid.Parse(c.Params("server_id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid server_id",
		})
	}

	server, err := h.serverSvc.GetServer(c.Context(), serverID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "server not found",
		})
	}

	var req UpdateServerRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	// Update only provided fields
	if req.Name != nil {
		server.Name = *req.Name
	}
	if req.Country != nil {
		server.Country = *req.Country
	}
	if req.City != nil {
		server.City = req.City
	}
	if req.FlagEmoji != nil {
		server.FlagEmoji = *req.FlagEmoji
	}
	if req.XUIBaseURL != nil {
		server.XUIBaseURL = *req.XUIBaseURL
	}
	if req.XUIUsername != nil {
		server.XUIUsername = *req.XUIUsername
	}
	if req.XUIPassword != nil {
		server.XUIPassword = *req.XUIPassword
	}
	if req.XUIInboundID != nil {
		server.XUIInboundID = *req.XUIInboundID
	}
	if req.ServerAddress != nil {
		server.ServerAddress = *req.ServerAddress
	}
	if req.ServerPort != nil {
		server.ServerPort = *req.ServerPort
	}
	if req.PublicKey != nil {
		server.PublicKey = *req.PublicKey
	}
	if req.ShortID != nil {
		server.ShortID = *req.ShortID
	}
	if req.ServerName != nil {
		server.ServerName = *req.ServerName
	}
	if req.IsActive != nil {
		server.IsActive = *req.IsActive
	}
	if req.SortOrder != nil {
		server.SortOrder = *req.SortOrder
	}
	if req.Capacity != nil {
		server.Capacity = *req.Capacity
	}

	if err := h.serverSvc.UpdateServer(c.Context(), server); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(server.ToAdmin())
}

// DeleteServer deletes a server
func (h *ServerHandler) DeleteServer(c *fiber.Ctx) error {
	_ = middleware.GetAdminID(c)
	serverID, err := uuid.Parse(c.Params("server_id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid server_id",
		})
	}

	if err := h.serverSvc.DeleteServer(c.Context(), serverID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{"success": true})
}

// TestServerConnection tests connection to a server's XUI panel
func (h *ServerHandler) TestServerConnection(c *fiber.Ctx) error {
	_ = middleware.GetAdminID(c)
	serverID, err := uuid.Parse(c.Params("server_id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid server_id",
		})
	}

	client, _, err := h.serverSvc.GetXUIClient(c.Context(), serverID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
			"connected": false,
		})
	}

	// Try to get inbound info to verify connection
	info, err := client.GetInboundInfo()
	if err != nil {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"connected": false,
			"error":     err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"connected":   true,
		"port":        info.Port,
		"public_key":  info.PublicKey,
		"short_id":    info.ShortID,
		"server_name": info.ServerName,
	})
}
