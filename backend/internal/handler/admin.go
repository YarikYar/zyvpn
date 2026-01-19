package handler

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/zyvpn/backend/internal/middleware"
	"github.com/zyvpn/backend/internal/model"
	"github.com/zyvpn/backend/internal/service"
)

// AdminHandler handles admin panel requests
type AdminHandler struct {
	adminSvc *service.AdminService
}

// NewAdminHandler creates a new admin handler
func NewAdminHandler(adminSvc *service.AdminService) *AdminHandler {
	return &AdminHandler{adminSvc: adminSvc}
}

// --- Stats ---

// GetStats returns admin dashboard statistics
func (h *AdminHandler) GetStats(c *fiber.Ctx) error {
	adminID := middleware.GetAdminID(c)
	stats, err := h.adminSvc.GetStats(c.Context(), adminID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	return c.JSON(stats)
}

// --- User Management ---

type ListUsersResponse struct {
	Users []model.User `json:"users"`
	Total int          `json:"total"`
}

// ListUsers lists users with pagination
func (h *AdminHandler) ListUsers(c *fiber.Ctx) error {
	adminID := middleware.GetAdminID(c)
	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))
	search := c.Query("search", "")

	users, total, err := h.adminSvc.ListUsers(c.Context(), adminID, limit, offset, search)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(ListUsersResponse{
		Users: users,
		Total: total,
	})
}

// GetUser gets detailed user info
func (h *AdminHandler) GetUser(c *fiber.Ctx) error {
	adminID := middleware.GetAdminID(c)
	targetUserID, err := strconv.ParseInt(c.Params("user_id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid user_id",
		})
	}

	user, err := h.adminSvc.GetUser(c.Context(), adminID, targetUserID)
	if err != nil {
		status := fiber.StatusInternalServerError
		if err == service.ErrUserNotFound {
			status = fiber.StatusNotFound
		}
		return c.Status(status).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(user)
}

// --- Balance Management ---

type SetBalanceRequest struct {
	Balance float64 `json:"balance"`
}

// SetBalance sets user balance to a specific value
func (h *AdminHandler) SetBalance(c *fiber.Ctx) error {
	adminID := middleware.GetAdminID(c)
	targetUserID, err := strconv.ParseInt(c.Params("user_id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid user_id",
		})
	}

	var req SetBalanceRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if err := h.adminSvc.SetBalance(c.Context(), adminID, targetUserID, req.Balance); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{"success": true})
}

type AddBalanceRequest struct {
	Amount float64 `json:"amount"`
}

// AddBalance adds amount to user balance
func (h *AdminHandler) AddBalance(c *fiber.Ctx) error {
	adminID := middleware.GetAdminID(c)
	targetUserID, err := strconv.ParseInt(c.Params("user_id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid user_id",
		})
	}

	var req AddBalanceRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if err := h.adminSvc.AddBalance(c.Context(), adminID, targetUserID, req.Amount); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{"success": true})
}

// --- Subscription Management ---

type ExtendSubscriptionRequest struct {
	Days int `json:"days"`
}

// ExtendSubscription extends user subscription
func (h *AdminHandler) ExtendSubscription(c *fiber.Ctx) error {
	adminID := middleware.GetAdminID(c)
	targetUserID, err := strconv.ParseInt(c.Params("user_id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid user_id",
		})
	}

	var req ExtendSubscriptionRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if req.Days <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "days must be positive",
		})
	}

	if err := h.adminSvc.ExtendSubscription(c.Context(), adminID, targetUserID, req.Days); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{"success": true})
}

// CancelSubscription cancels user subscription
func (h *AdminHandler) CancelSubscription(c *fiber.Ctx) error {
	adminID := middleware.GetAdminID(c)
	targetUserID, err := strconv.ParseInt(c.Params("user_id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid user_id",
		})
	}

	if err := h.adminSvc.CancelSubscription(c.Context(), adminID, targetUserID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{"success": true})
}

// --- Ban Management ---

type BanUserRequest struct {
	Reason    string     `json:"reason"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

// BanUser bans a user
func (h *AdminHandler) BanUser(c *fiber.Ctx) error {
	adminID := middleware.GetAdminID(c)
	targetUserID, err := strconv.ParseInt(c.Params("user_id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid user_id",
		})
	}

	var req BanUserRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if err := h.adminSvc.BanUser(c.Context(), adminID, targetUserID, req.Reason, req.ExpiresAt); err != nil {
		status := fiber.StatusInternalServerError
		if err == service.ErrAlreadyBanned {
			status = fiber.StatusConflict
		}
		return c.Status(status).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{"success": true})
}

type BanIPRequest struct {
	IP        string     `json:"ip"`
	Reason    string     `json:"reason"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

// BanIP bans an IP address
func (h *AdminHandler) BanIP(c *fiber.Ctx) error {
	adminID := middleware.GetAdminID(c)

	var req BanIPRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if req.IP == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ip is required",
		})
	}

	if err := h.adminSvc.BanIP(c.Context(), adminID, req.IP, req.Reason, req.ExpiresAt); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{"success": true})
}

// UnbanUser unbans a user
func (h *AdminHandler) UnbanUser(c *fiber.Ctx) error {
	adminID := middleware.GetAdminID(c)
	targetUserID, err := strconv.ParseInt(c.Params("user_id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid user_id",
		})
	}

	if err := h.adminSvc.UnbanUser(c.Context(), adminID, targetUserID); err != nil {
		status := fiber.StatusInternalServerError
		if err == service.ErrNotBanned {
			status = fiber.StatusNotFound
		}
		return c.Status(status).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{"success": true})
}

type UnbanIPRequest struct {
	IP string `json:"ip"`
}

// UnbanIP unbans an IP address
func (h *AdminHandler) UnbanIP(c *fiber.Ctx) error {
	adminID := middleware.GetAdminID(c)

	var req UnbanIPRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if req.IP == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ip is required",
		})
	}

	if err := h.adminSvc.UnbanIP(c.Context(), adminID, req.IP); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{"success": true})
}

// ListBans lists all active bans
func (h *AdminHandler) ListBans(c *fiber.Ctx) error {
	adminID := middleware.GetAdminID(c)
	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))

	bans, err := h.adminSvc.ListBannedUsers(c.Context(), adminID, limit, offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{"bans": bans})
}

// --- Promo Code Management ---

type CreatePromoCodeRequest struct {
	Type        model.PromoCodeType `json:"type"`
	Value       float64             `json:"value"`
	MaxUses     *int                `json:"max_uses,omitempty"`
	ExpiresAt   *time.Time          `json:"expires_at,omitempty"`
	Description string              `json:"description,omitempty"`
}

// CreatePromoCode creates a new promo code
func (h *AdminHandler) CreatePromoCode(c *fiber.Ctx) error {
	adminID := middleware.GetAdminID(c)

	var req CreatePromoCodeRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if req.Type != model.PromoCodeTypeBalance && req.Type != model.PromoCodeTypeDays && req.Type != model.PromoCodeTypeRegionSwitch {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid promo code type, must be 'balance', 'days' or 'region_switch'",
		})
	}

	if req.Value <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "value must be positive",
		})
	}

	promo, err := h.adminSvc.GeneratePromoCode(c.Context(), adminID, req.Type, req.Value, req.MaxUses, req.ExpiresAt, req.Description)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(promo)
}

type BulkPromoCodeRequest struct {
	Count     int                 `json:"count"`
	Type      model.PromoCodeType `json:"type"`
	Value     float64             `json:"value"`
	MaxUses   *int                `json:"max_uses,omitempty"`
	ExpiresAt *time.Time          `json:"expires_at,omitempty"`
	Prefix    string              `json:"prefix,omitempty"`
}

// CreateBulkPromoCodes creates multiple promo codes
func (h *AdminHandler) CreateBulkPromoCodes(c *fiber.Ctx) error {
	adminID := middleware.GetAdminID(c)

	var req BulkPromoCodeRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if req.Count <= 0 || req.Count > 100 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "count must be between 1 and 100",
		})
	}

	if req.Type != model.PromoCodeTypeBalance && req.Type != model.PromoCodeTypeDays && req.Type != model.PromoCodeTypeRegionSwitch {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid promo code type",
		})
	}

	if req.Value <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "value must be positive",
		})
	}

	codes, err := h.adminSvc.GenerateBulkPromoCodes(c.Context(), adminID, req.Count, req.Type, req.Value, req.MaxUses, req.ExpiresAt, req.Prefix)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{"codes": codes, "count": len(codes)})
}

// ListPromoCodes lists all promo codes
func (h *AdminHandler) ListPromoCodes(c *fiber.Ctx) error {
	adminID := middleware.GetAdminID(c)
	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))

	promos, err := h.adminSvc.ListPromoCodes(c.Context(), adminID, limit, offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{"promo_codes": promos})
}

type DeactivatePromoCodeRequest struct {
	Code string `json:"code"`
}

// DeactivatePromoCode deactivates a promo code
func (h *AdminHandler) DeactivatePromoCode(c *fiber.Ctx) error {
	adminID := middleware.GetAdminID(c)

	var req DeactivatePromoCodeRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if req.Code == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "code is required",
		})
	}

	if err := h.adminSvc.DeactivatePromoCode(c.Context(), adminID, req.Code); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{"success": true})
}

// --- Admin Logs ---

// GetLogs retrieves admin action logs
func (h *AdminHandler) GetLogs(c *fiber.Ctx) error {
	adminID := middleware.GetAdminID(c)
	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))

	logs, err := h.adminSvc.GetAdminLogs(c.Context(), adminID, limit, offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{"logs": logs})
}

// --- Plan Management ---

// ListPlans lists all plans (including inactive)
func (h *AdminHandler) ListPlans(c *fiber.Ctx) error {
	plans, err := h.adminSvc.ListAllPlans(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	return c.JSON(fiber.Map{"plans": plans})
}

type UpdatePlanRequest struct {
	Name         *string  `json:"name,omitempty"`
	Description  *string  `json:"description,omitempty"`
	DurationDays *int     `json:"duration_days,omitempty"`
	TrafficGB    *int     `json:"traffic_gb,omitempty"`
	MaxDevices   *int     `json:"max_devices,omitempty"`
	PriceTON     *float64 `json:"price_ton,omitempty"`
	PriceStars   *int     `json:"price_stars,omitempty"`
	PriceUSD     *float64 `json:"price_usd,omitempty"`
	IsActive     *bool    `json:"is_active,omitempty"`
	SortOrder    *int     `json:"sort_order,omitempty"`
}

// UpdatePlan updates a plan
func (h *AdminHandler) UpdatePlan(c *fiber.Ctx) error {
	adminID := middleware.GetAdminID(c)
	planID := c.Params("plan_id")

	var req UpdatePlanRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Неверный формат запроса",
		})
	}

	plan, err := h.adminSvc.UpdatePlan(c.Context(), adminID, planID, service.UpdatePlanParams{
		Name:         req.Name,
		Description:  req.Description,
		DurationDays: req.DurationDays,
		TrafficGB:    req.TrafficGB,
		MaxDevices:   req.MaxDevices,
		PriceTON:     req.PriceTON,
		PriceStars:   req.PriceStars,
		PriceUSD:     req.PriceUSD,
		IsActive:     req.IsActive,
		SortOrder:    req.SortOrder,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(plan)
}

type CreatePlanRequest struct {
	Name         string  `json:"name"`
	Description  string  `json:"description"`
	DurationDays int     `json:"duration_days"`
	TrafficGB    int     `json:"traffic_gb"`
	MaxDevices   int     `json:"max_devices"`
	PriceTON     float64 `json:"price_ton"`
	PriceStars   int     `json:"price_stars"`
	PriceUSD     float64 `json:"price_usd"`
	SortOrder    int     `json:"sort_order"`
}

// CreatePlan creates a new plan
func (h *AdminHandler) CreatePlan(c *fiber.Ctx) error {
	adminID := middleware.GetAdminID(c)

	var req CreatePlanRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Неверный формат запроса",
		})
	}

	if req.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Название обязательно",
		})
	}

	if req.MaxDevices <= 0 {
		req.MaxDevices = 3
	}

	plan, err := h.adminSvc.CreatePlan(c.Context(), adminID, service.CreatePlanParams{
		Name:         req.Name,
		Description:  req.Description,
		DurationDays: req.DurationDays,
		TrafficGB:    req.TrafficGB,
		MaxDevices:   req.MaxDevices,
		PriceTON:     req.PriceTON,
		PriceStars:   req.PriceStars,
		PriceUSD:     req.PriceUSD,
		SortOrder:    req.SortOrder,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(plan)
}

// DeletePlan deactivates a plan (soft delete)
func (h *AdminHandler) DeletePlan(c *fiber.Ctx) error {
	adminID := middleware.GetAdminID(c)
	planID := c.Params("plan_id")

	if err := h.adminSvc.DeletePlan(c.Context(), adminID, planID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{"success": true})
}

// --- Settings Management ---

// GetSettings returns all admin settings
func (h *AdminHandler) GetSettings(c *fiber.Ctx) error {
	adminID := middleware.GetAdminID(c)

	settings, err := h.adminSvc.GetSettings(c.Context(), adminID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{"settings": settings})
}

// GetTopupBonus returns current topup bonus percentage
func (h *AdminHandler) GetTopupBonus(c *fiber.Ctx) error {
	percent, err := h.adminSvc.GetTopupBonusPercent(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{"topup_bonus_percent": percent})
}

type SetTopupBonusRequest struct {
	Percent float64 `json:"percent"`
}

// SetTopupBonus sets topup bonus percentage
func (h *AdminHandler) SetTopupBonus(c *fiber.Ctx) error {
	adminID := middleware.GetAdminID(c)

	var req SetTopupBonusRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Неверный формат запроса",
		})
	}

	if err := h.adminSvc.SetTopupBonusPercent(c.Context(), adminID, req.Percent); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{"success": true, "topup_bonus_percent": req.Percent})
}

// GetReferralBonus returns current referral bonus percentage
func (h *AdminHandler) GetReferralBonus(c *fiber.Ctx) error {
	percent, err := h.adminSvc.GetReferralBonusPercent(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{"referral_bonus_percent": percent})
}

type SetReferralBonusRequest struct {
	Percent float64 `json:"percent"`
}

// SetReferralBonus sets referral bonus percentage
func (h *AdminHandler) SetReferralBonus(c *fiber.Ctx) error {
	adminID := middleware.GetAdminID(c)

	var req SetReferralBonusRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Неверный формат запроса",
		})
	}

	if err := h.adminSvc.SetReferralBonusPercent(c.Context(), adminID, req.Percent); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{"success": true, "referral_bonus_percent": req.Percent})
}

// GetReferralBonusDays returns current referral bonus days
func (h *AdminHandler) GetReferralBonusDays(c *fiber.Ctx) error {
	days, err := h.adminSvc.GetReferralBonusDays(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{"referral_bonus_days": days})
}

type SetReferralBonusDaysRequest struct {
	Days int `json:"days"`
}

// SetReferralBonusDays sets referral bonus days
func (h *AdminHandler) SetReferralBonusDays(c *fiber.Ctx) error {
	adminID := middleware.GetAdminID(c)

	var req SetReferralBonusDaysRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Неверный формат запроса",
		})
	}

	if err := h.adminSvc.SetReferralBonusDays(c.Context(), adminID, req.Days); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{"success": true, "referral_bonus_days": req.Days})
}

// GetRegionSwitchPrice returns current region switch price
func (h *AdminHandler) GetRegionSwitchPrice(c *fiber.Ctx) error {
	price, err := h.adminSvc.GetRegionSwitchPrice(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{"region_switch_price": price})
}

type SetRegionSwitchPriceRequest struct {
	Price float64 `json:"price"`
}

// SetRegionSwitchPrice sets region switch price in TON
func (h *AdminHandler) SetRegionSwitchPrice(c *fiber.Ctx) error {
	adminID := middleware.GetAdminID(c)

	var req SetRegionSwitchPriceRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Неверный формат запроса",
		})
	}

	if err := h.adminSvc.SetRegionSwitchPrice(c.Context(), adminID, req.Price); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{"success": true, "region_switch_price": req.Price})
}
