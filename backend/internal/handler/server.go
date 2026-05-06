package handler

import (
	"log"
	"time"

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

// Response wrapper types — needed for swag to generate typed schemas
// instead of generic map[string]interface{}.

type ServersResponse struct {
	Servers []model.ServerPublic `json:"servers"`
}

type IncidentsResponse struct {
	Incidents []model.Incident `json:"incidents"`
}

type AdminServersResponse struct {
	Servers []model.ServerAdmin `json:"servers"`
}

// NewServerHandler creates a new server handler
func NewServerHandler(serverSvc *service.ServerService) *ServerHandler {
	return &ServerHandler{serverSvc: serverSvc}
}

// --- User Endpoints ---

// GetIncidents returns recent offline intervals across servers.
//
//	@Summary	Recent server incidents
//	@Tags		servers
//	@Produce	json
//	@Param		hours	query		int	false	"window in hours"	default(168)
//	@Param		limit	query		int	false	"max items"			default(50)
//	@Success	200		{object}	IncidentsResponse
//	@Router		/api/servers/incidents [get]
//	@Security	TelegramInitData
func (h *ServerHandler) GetIncidents(c *fiber.Ctx) error {
	hours := 168
	if v := c.QueryInt("hours", 0); v > 0 {
		hours = v
	}
	if hours > 24*30 {
		hours = 24 * 30
	}
	limit := c.QueryInt("limit", 50)

	since := time.Now().Add(-time.Duration(hours) * time.Hour)
	incidents, err := h.serverSvc.ListIncidents(c.Context(), since, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Не удалось получить инциденты",
		})
	}
	return c.JSON(fiber.Map{"incidents": incidents})
}

// GetServers returns active servers visible to end users with monitoring
// fields (uptime, traffic, status_since).
//
//	@Summary	List active servers
//	@Tags		servers
//	@Produce	json
//	@Success	200	{object}	ServersResponse
//	@Router		/api/servers [get]
//	@Security	TelegramInitData
func (h *ServerHandler) GetServers(c *fiber.Ctx) error {
	servers, err := h.serverSvc.GetActiveServers(c.Context())
	if err != nil {
		return respondInternalError(c, err)
	}
	return c.JSON(fiber.Map{"servers": servers})
}

// --- Admin Endpoints ---

// GetAllServers returns every server with private fields (XUI creds) — admin only.
//
//	@Summary	List all servers (admin)
//	@Tags		admin
//	@Produce	json
//	@Success	200	{object}	AdminServersResponse
//	@Router		/api/admin/servers [get]
//	@Security	TelegramInitData
func (h *ServerHandler) GetAllServers(c *fiber.Ctx) error {
	_ = middleware.GetAdminID(c)
	servers, err := h.serverSvc.GetAllServers(c.Context())
	if err != nil {
		return respondInternalError(c, err)
	}
	return c.JSON(fiber.Map{"servers": servers})
}

// GetServer returns one server with private fields.
//
//	@Summary	Get server (admin)
//	@Tags		admin
//	@Produce	json
//	@Param		server_id	path		string	true	"server uuid"
//	@Success	200			{object}	model.Server
//	@Failure	404			{object}	map[string]string
//	@Router		/api/admin/servers/{server_id} [get]
//	@Security	TelegramInitData
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

// CreateServer registers a new VPN server.
//
//	@Summary	Create server
//	@Tags		admin
//	@Accept		json
//	@Produce	json
//	@Param		body	body		CreateServerRequest	true	"server config"
//	@Success	200		{object}	model.Server
//	@Failure	400		{object}	map[string]string
//	@Router		/api/admin/servers [post]
//	@Security	TelegramInitData
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
		return respondInternalError(c, err)
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

// UpdateServer patches a server config.
//
//	@Summary	Update server
//	@Tags		admin
//	@Accept		json
//	@Produce	json
//	@Param		server_id	path		string				true	"server uuid"
//	@Param		body		body		UpdateServerRequest	true	"updates"
//	@Success	200			{object}	model.Server
//	@Failure	404			{object}	map[string]string
//	@Router		/api/admin/servers/{server_id} [put]
//	@Security	TelegramInitData
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
		return respondInternalError(c, err)
	}

	return c.JSON(server.ToAdmin())
}

// DeleteServer removes a server from rotation.
//
//	@Summary	Delete server
//	@Tags		admin
//	@Produce	json
//	@Param		server_id	path		string	true	"server uuid"
//	@Success	200			{object}	map[string]interface{}
//	@Router		/api/admin/servers/{server_id} [delete]
//	@Security	TelegramInitData
func (h *ServerHandler) DeleteServer(c *fiber.Ctx) error {
	_ = middleware.GetAdminID(c)
	serverID, err := uuid.Parse(c.Params("server_id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid server_id",
		})
	}

	if err := h.serverSvc.DeleteServer(c.Context(), serverID); err != nil {
		return respondInternalError(c, err)
	}

	return c.JSON(fiber.Map{"success": true})
}

// TestServerConnection probes XUI panel connectivity.
//
//	@Summary	Test server XUI connection
//	@Tags		admin
//	@Produce	json
//	@Param		server_id	path		string	true	"server uuid"
//	@Success	200			{object}	map[string]interface{}
//	@Router		/api/admin/servers/{server_id}/test [post]
//	@Security	TelegramInitData
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
		log.Printf("[TestServerConnection] GetXUIClient %s: %v", serverID, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":     "Не удалось подключиться к серверу",
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
