package handler

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/zyvpn/backend/internal/middleware"
	"github.com/zyvpn/backend/internal/model"
	"github.com/zyvpn/backend/internal/service"
)

// AdminHandler handles admin panel requests
type AdminHandler struct {
	adminSvc        *service.AdminService
	paymentSvc      *service.PaymentService
	subscriptionSvc *service.SubscriptionService
}

// NewAdminHandler creates a new admin handler
func NewAdminHandler(adminSvc *service.AdminService, paymentSvc *service.PaymentService, subscriptionSvc *service.SubscriptionService) *AdminHandler {
	return &AdminHandler{
		adminSvc:        adminSvc,
		paymentSvc:      paymentSvc,
		subscriptionSvc: subscriptionSvc,
	}
}

// --- Stats ---

// GetStats returns admin dashboard counters.
//
//	@Summary	Admin dashboard stats
//	@Tags		admin
//	@Produce	json
//	@Success	200	{object}	map[string]interface{}
//	@Router		/api/admin/stats [get]
//	@Security	TelegramInitData
func (h *AdminHandler) GetStats(c *fiber.Ctx) error {
	adminID := middleware.GetAdminID(c)
	stats, err := h.adminSvc.GetStats(c.Context(), adminID)
	if err != nil {
		return respondInternalError(c, err)
	}
	return c.JSON(stats)
}

// --- User Management ---

type ListUsersResponse struct {
	Users []model.User `json:"users"`
	Total int          `json:"total"`
}

// ListUsers lists users with pagination + search.
//
//	@Summary	List users
//	@Tags		admin
//	@Produce	json
//	@Param		limit	query		int		false	"page size"	default(50)
//	@Param		offset	query		int		false	"offset"	default(0)
//	@Param		search	query		string	false	"username/name search"
//	@Success	200		{object}	ListUsersResponse
//	@Router		/api/admin/users [get]
//	@Security	TelegramInitData
func (h *AdminHandler) ListUsers(c *fiber.Ctx) error {
	adminID := middleware.GetAdminID(c)
	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))
	search := c.Query("search", "")

	users, total, err := h.adminSvc.ListUsers(c.Context(), adminID, limit, offset, search)
	if err != nil {
		return respondInternalError(c, err)
	}

	return c.JSON(ListUsersResponse{
		Users: users,
		Total: total,
	})
}

// GetUser returns detailed user info with active subscription.
//
//	@Summary	Get user
//	@Tags		admin
//	@Produce	json
//	@Param		user_id	path		int	true	"user id (telegram)"
//	@Success	200		{object}	model.UserWithSubscription
//	@Failure	404		{object}	map[string]string
//	@Router		/api/admin/users/{user_id} [get]
//	@Security	TelegramInitData
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

// SetBalance sets user balance to an absolute value.
//
//	@Summary	Set user balance
//	@Tags		admin
//	@Accept		json
//	@Produce	json
//	@Param		user_id	path		int					true	"user id"
//	@Param		body	body		SetBalanceRequest	true	"new balance"
//	@Success	200		{object}	map[string]interface{}
//	@Router		/api/admin/users/{user_id}/balance/set [post]
//	@Security	TelegramInitData
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
		return respondInternalError(c, err)
	}

	return c.JSON(fiber.Map{"success": true})
}

type AddBalanceRequest struct {
	Amount float64 `json:"amount"`
}

// AddBalance adds delta to user balance.
//
//	@Summary	Add to user balance
//	@Tags		admin
//	@Accept		json
//	@Produce	json
//	@Param		user_id	path		int					true	"user id"
//	@Param		body	body		AddBalanceRequest	true	"delta"
//	@Success	200		{object}	map[string]interface{}
//	@Router		/api/admin/users/{user_id}/balance/add [post]
//	@Security	TelegramInitData
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
		return respondInternalError(c, err)
	}

	return c.JSON(fiber.Map{"success": true})
}

// --- Subscription Management ---

type ExtendSubscriptionRequest struct {
	Days int `json:"days"`
}

// ExtendSubscription extends user subscription by N days.
//
//	@Summary	Extend subscription
//	@Tags		admin
//	@Accept		json
//	@Produce	json
//	@Param		user_id	path		int							true	"user id"
//	@Param		body	body		ExtendSubscriptionRequest	true	"days"
//	@Success	200		{object}	map[string]interface{}
//	@Router		/api/admin/users/{user_id}/subscription/extend [post]
//	@Security	TelegramInitData
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
		return respondInternalError(c, err)
	}

	return c.JSON(fiber.Map{"success": true})
}

// CancelSubscription cancels user's active subscription.
//
//	@Summary	Cancel subscription
//	@Tags		admin
//	@Produce	json
//	@Param		user_id	path		int	true	"user id"
//	@Success	200		{object}	map[string]interface{}
//	@Router		/api/admin/users/{user_id}/subscription/cancel [post]
//	@Security	TelegramInitData
func (h *AdminHandler) CancelSubscription(c *fiber.Ctx) error {
	adminID := middleware.GetAdminID(c)
	targetUserID, err := strconv.ParseInt(c.Params("user_id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid user_id",
		})
	}

	if err := h.adminSvc.CancelSubscription(c.Context(), adminID, targetUserID); err != nil {
		return respondInternalError(c, err)
	}

	return c.JSON(fiber.Map{"success": true})
}

// --- Ban Management ---

type BanUserRequest struct {
	Reason    string     `json:"reason"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

// BanUser bans a user.
//
//	@Summary	Ban user
//	@Tags		admin
//	@Accept		json
//	@Produce	json
//	@Param		user_id	path		int				true	"user id"
//	@Param		body	body		BanUserRequest	true	"reason + optional expiry"
//	@Success	200		{object}	map[string]interface{}
//	@Failure	409		{object}	map[string]string	"already banned"
//	@Router		/api/admin/users/{user_id}/ban [post]
//	@Security	TelegramInitData
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

// BanIP bans an IP address.
//
//	@Summary	Ban IP
//	@Tags		admin
//	@Accept		json
//	@Produce	json
//	@Param		body	body		BanIPRequest	true	"ip + reason"
//	@Success	200		{object}	map[string]interface{}
//	@Router		/api/admin/bans/ip [post]
//	@Security	TelegramInitData
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
		return respondInternalError(c, err)
	}

	return c.JSON(fiber.Map{"success": true})
}

// UnbanUser unbans a user.
//
//	@Summary	Unban user
//	@Tags		admin
//	@Produce	json
//	@Param		user_id	path		int	true	"user id"
//	@Success	200		{object}	map[string]interface{}
//	@Router		/api/admin/users/{user_id}/unban [post]
//	@Security	TelegramInitData
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

// UnbanIP unbans an IP.
//
//	@Summary	Unban IP
//	@Tags		admin
//	@Accept		json
//	@Produce	json
//	@Param		body	body		map[string]string	true	"{ip}"
//	@Success	200		{object}	map[string]interface{}
//	@Router		/api/admin/bans/ip/unban [post]
//	@Security	TelegramInitData
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
		return respondInternalError(c, err)
	}

	return c.JSON(fiber.Map{"success": true})
}

// ListBans lists active user/IP bans.
//
//	@Summary	List bans
//	@Tags		admin
//	@Produce	json
//	@Param		limit	query		int	false	"page size"	default(50)
//	@Param		offset	query		int	false	"offset"	default(0)
//	@Success	200		{object}	map[string]interface{}
//	@Router		/api/admin/bans [get]
//	@Security	TelegramInitData
func (h *AdminHandler) ListBans(c *fiber.Ctx) error {
	adminID := middleware.GetAdminID(c)
	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))

	bans, err := h.adminSvc.ListBannedUsers(c.Context(), adminID, limit, offset)
	if err != nil {
		return respondInternalError(c, err)
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

// CreatePromoCode creates a single promo code.
//
//	@Summary	Create promo code
//	@Tags		admin
//	@Accept		json
//	@Produce	json
//	@Param		body	body		CreatePromoCodeRequest	true	"promo params"
//	@Success	200		{object}	model.PromoCode
//	@Failure	400		{object}	map[string]string
//	@Router		/api/admin/promo [post]
//	@Security	TelegramInitData
func (h *AdminHandler) CreatePromoCode(c *fiber.Ctx) error {
	adminID := middleware.GetAdminID(c)

	var req CreatePromoCodeRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	switch req.Type {
	case model.PromoCodeTypeBalance, model.PromoCodeTypeDays:
		// ok
	default:
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid promo code type, must be 'balance' or 'days'",
		})
	}

	if req.Value <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "value must be positive",
		})
	}

	promo, err := h.adminSvc.GeneratePromoCode(c.Context(), adminID, req.Type, req.Value, req.MaxUses, req.ExpiresAt, req.Description)
	if err != nil {
		return respondInternalError(c, err)
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

// CreateBulkPromoCodes generates N promo codes at once.
//
//	@Summary	Bulk-create promo codes
//	@Tags		admin
//	@Accept		json
//	@Produce	json
//	@Param		body	body		BulkPromoCodeRequest	true	"params + count"
//	@Success	200		{object}	map[string]interface{}
//	@Router		/api/admin/promo/bulk [post]
//	@Security	TelegramInitData
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

	switch req.Type {
	case model.PromoCodeTypeBalance, model.PromoCodeTypeDays:
		// ok
	default:
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
		return respondInternalError(c, err)
	}

	return c.JSON(fiber.Map{"codes": codes, "count": len(codes)})
}

// ListPromoCodes lists all promo codes (paginated).
//
//	@Summary	List promo codes
//	@Tags		admin
//	@Produce	json
//	@Param		limit	query		int	false	"page size"	default(50)
//	@Param		offset	query		int	false	"offset"	default(0)
//	@Success	200		{object}	map[string]interface{}
//	@Router		/api/admin/promo [get]
//	@Security	TelegramInitData
func (h *AdminHandler) ListPromoCodes(c *fiber.Ctx) error {
	adminID := middleware.GetAdminID(c)
	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))

	promos, err := h.adminSvc.ListPromoCodes(c.Context(), adminID, limit, offset)
	if err != nil {
		return respondInternalError(c, err)
	}

	return c.JSON(fiber.Map{"promo_codes": promos})
}

type DeactivatePromoCodeRequest struct {
	Code string `json:"code"`
}

// DeactivatePromoCode disables a promo code.
//
//	@Summary	Deactivate promo code
//	@Tags		admin
//	@Accept		json
//	@Produce	json
//	@Param		body	body		map[string]string	true	"{code}"
//	@Success	200		{object}	map[string]interface{}
//	@Router		/api/admin/promo/deactivate [post]
//	@Security	TelegramInitData
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
		return respondInternalError(c, err)
	}

	return c.JSON(fiber.Map{"success": true})
}

// --- Admin Logs ---

// GetLogs returns admin action audit log.
//
//	@Summary	Admin audit log
//	@Tags		admin
//	@Produce	json
//	@Param		limit	query		int	false	"page size"	default(50)
//	@Param		offset	query		int	false	"offset"	default(0)
//	@Success	200		{object}	map[string]interface{}
//	@Router		/api/admin/logs [get]
//	@Security	TelegramInitData
func (h *AdminHandler) GetLogs(c *fiber.Ctx) error {
	adminID := middleware.GetAdminID(c)
	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))

	logs, err := h.adminSvc.GetAdminLogs(c.Context(), adminID, limit, offset)
	if err != nil {
		return respondInternalError(c, err)
	}

	return c.JSON(fiber.Map{"logs": logs})
}

// --- Plan Management ---

// ListPlans lists all plans (incl. inactive).
//
//	@Summary	List all plans
//	@Tags		admin
//	@Produce	json
//	@Success	200	{array}	model.Plan
//	@Router		/api/admin/plans [get]
//	@Security	TelegramInitData
func (h *AdminHandler) ListPlans(c *fiber.Ctx) error {
	plans, err := h.adminSvc.ListAllPlans(c.Context())
	if err != nil {
		return respondInternalError(c, err)
	}
	return c.JSON(fiber.Map{"plans": plans})
}

type UpdatePlanRequest struct {
	Name                *string  `json:"name,omitempty"`
	Description         *string  `json:"description,omitempty"`
	DurationDays        *int     `json:"duration_days,omitempty"`
	TrafficGB           *int     `json:"traffic_gb,omitempty"`
	MaxDevices          *int     `json:"max_devices,omitempty"`
	PriceTON            *float64 `json:"price_ton,omitempty"`
	PriceStars          *int     `json:"price_stars,omitempty"`
	PriceUSD            *float64 `json:"price_usd,omitempty"`
	IsActive            *bool    `json:"is_active,omitempty"`
	SortOrder           *int     `json:"sort_order,omitempty"`
	VisibleToReferrerID *int64   `json:"visible_to_referrer_id,omitempty"`
	// ClearVisibility=true forces plan back to public regardless of
	// VisibleToReferrerID (which would otherwise mean "leave as-is").
	ClearVisibility bool `json:"clear_visibility,omitempty"`
	// ServerIDs — UUID серверов, входящих в тариф. nil = не трогаем,
	// [] = очистить (тариф станет неактивируемым).
	ServerIDs *[]string `json:"server_ids,omitempty"`
}

// UpdatePlan updates a plan; nil-fields are unchanged.
//
//	@Summary	Update plan
//	@Tags		admin
//	@Accept		json
//	@Produce	json
//	@Param		plan_id	path		string			true	"plan uuid"
//	@Param		body	body		UpdatePlanRequest	true	"updates"
//	@Success	200		{object}	model.Plan
//	@Router		/api/admin/plans/{plan_id} [put]
//	@Security	TelegramInitData
func (h *AdminHandler) UpdatePlan(c *fiber.Ctx) error {
	adminID := middleware.GetAdminID(c)
	planID := c.Params("plan_id")

	var req UpdatePlanRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Неверный формат запроса",
		})
	}

	var serverIDsParsed *[]uuid.UUID
	if req.ServerIDs != nil {
		parsed := make([]uuid.UUID, 0, len(*req.ServerIDs))
		for _, sid := range *req.ServerIDs {
			id, err := uuid.Parse(sid)
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": "Невалидный server_id: " + sid,
				})
			}
			parsed = append(parsed, id)
		}
		serverIDsParsed = &parsed
	}

	plan, err := h.adminSvc.UpdatePlan(c.Context(), adminID, planID, service.UpdatePlanParams{
		Name:                req.Name,
		Description:         req.Description,
		DurationDays:        req.DurationDays,
		TrafficGB:           req.TrafficGB,
		MaxDevices:          req.MaxDevices,
		PriceTON:            req.PriceTON,
		PriceStars:          req.PriceStars,
		PriceUSD:            req.PriceUSD,
		IsActive:            req.IsActive,
		SortOrder:           req.SortOrder,
		VisibleToReferrerID: req.VisibleToReferrerID,
		ClearVisibility:     req.ClearVisibility,
		ServerIDs:           serverIDsParsed,
	})
	if err != nil {
		return respondInternalError(c, err)
	}

	return c.JSON(plan)
}

type CreatePlanRequest struct {
	Name                string   `json:"name"`
	Description         string   `json:"description"`
	DurationDays        int      `json:"duration_days"`
	TrafficGB           int      `json:"traffic_gb"`
	MaxDevices          int      `json:"max_devices"`
	PriceTON            float64  `json:"price_ton"`
	PriceStars          int      `json:"price_stars"`
	PriceUSD            float64  `json:"price_usd"`
	SortOrder           int      `json:"sort_order"`
	VisibleToReferrerID *int64   `json:"visible_to_referrer_id,omitempty"`
	ServerIDs           []string `json:"server_ids"`
}

// CreatePlan creates a new plan.
//
//	@Summary	Create plan
//	@Tags		admin
//	@Accept		json
//	@Produce	json
//	@Param		body	body		CreatePlanRequest	true	"new plan"
//	@Success	200		{object}	model.Plan
//	@Router		/api/admin/plans [post]
//	@Security	TelegramInitData
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
	if len(req.ServerIDs) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Нужно выбрать хотя бы один сервер",
		})
	}
	serverIDs := make([]uuid.UUID, 0, len(req.ServerIDs))
	for _, sid := range req.ServerIDs {
		id, err := uuid.Parse(sid)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Невалидный server_id: " + sid,
			})
		}
		serverIDs = append(serverIDs, id)
	}

	plan, err := h.adminSvc.CreatePlan(c.Context(), adminID, service.CreatePlanParams{
		Name:                req.Name,
		Description:         req.Description,
		DurationDays:        req.DurationDays,
		TrafficGB:           req.TrafficGB,
		MaxDevices:          req.MaxDevices,
		PriceTON:            req.PriceTON,
		PriceStars:          req.PriceStars,
		PriceUSD:            req.PriceUSD,
		SortOrder:           req.SortOrder,
		VisibleToReferrerID: req.VisibleToReferrerID,
		ServerIDs:           serverIDs,
	})
	if err != nil {
		return respondInternalError(c, err)
	}

	return c.JSON(plan)
}

// DeletePlan soft-deletes a plan (sets is_active=false).
//
//	@Summary	Delete plan
//	@Tags		admin
//	@Produce	json
//	@Param		plan_id	path		string	true	"plan uuid"
//	@Success	200		{object}	map[string]interface{}
//	@Router		/api/admin/plans/{plan_id} [delete]
//	@Security	TelegramInitData
func (h *AdminHandler) DeletePlan(c *fiber.Ctx) error {
	adminID := middleware.GetAdminID(c)
	planID := c.Params("plan_id")

	if err := h.adminSvc.DeletePlan(c.Context(), adminID, planID); err != nil {
		return respondInternalError(c, err)
	}

	return c.JSON(fiber.Map{"success": true})
}

// --- Settings Management ---

// GetSettings returns all dynamic settings.
//
//	@Summary	List settings
//	@Tags		admin
//	@Produce	json
//	@Success	200	{object}	map[string]interface{}
//	@Router		/api/admin/settings [get]
//	@Security	TelegramInitData
func (h *AdminHandler) GetSettings(c *fiber.Ctx) error {
	adminID := middleware.GetAdminID(c)

	settings, err := h.adminSvc.GetSettings(c.Context(), adminID)
	if err != nil {
		return respondInternalError(c, err)
	}

	return c.JSON(fiber.Map{"settings": settings})
}

// GetTopupBonus returns the topup-bonus percent.
//
//	@Summary	Get topup bonus
//	@Tags		admin
//	@Produce	json
//	@Success	200	{object}	map[string]interface{}
//	@Router		/api/admin/settings/topup-bonus [get]
//	@Security	TelegramInitData
func (h *AdminHandler) GetTopupBonus(c *fiber.Ctx) error {
	percent, err := h.adminSvc.GetTopupBonusPercent(c.Context())
	if err != nil {
		return respondInternalError(c, err)
	}

	return c.JSON(fiber.Map{"topup_bonus_percent": percent})
}

type SetTopupBonusRequest struct {
	Percent float64 `json:"percent"`
}

// SetTopupBonus sets the topup-bonus percent.
//
//	@Summary	Set topup bonus
//	@Tags		admin
//	@Accept		json
//	@Produce	json
//	@Param		body	body		map[string]float64	true	"{percent}"
//	@Success	200		{object}	map[string]interface{}
//	@Router		/api/admin/settings/topup-bonus [post]
//	@Security	TelegramInitData
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

// GetReferralBonus returns referral percent.
//
//	@Summary	Get referral bonus percent
//	@Tags		admin
//	@Produce	json
//	@Success	200	{object}	map[string]interface{}
//	@Router		/api/admin/settings/referral-bonus [get]
//	@Security	TelegramInitData
func (h *AdminHandler) GetReferralBonus(c *fiber.Ctx) error {
	percent, err := h.adminSvc.GetReferralBonusPercent(c.Context())
	if err != nil {
		return respondInternalError(c, err)
	}

	return c.JSON(fiber.Map{"referral_bonus_percent": percent})
}

type SetReferralBonusRequest struct {
	Percent float64 `json:"percent"`
}

// SetReferralBonus sets referral percent.
//
//	@Summary	Set referral bonus percent
//	@Tags		admin
//	@Accept		json
//	@Produce	json
//	@Param		body	body		map[string]float64	true	"{percent}"
//	@Success	200		{object}	map[string]interface{}
//	@Router		/api/admin/settings/referral-bonus [post]
//	@Security	TelegramInitData
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

// GetReferralBonusDays returns bonus days for referrer.
//
//	@Summary	Get referral bonus days
//	@Tags		admin
//	@Produce	json
//	@Success	200	{object}	map[string]interface{}
//	@Router		/api/admin/settings/referral-bonus-days [get]
//	@Security	TelegramInitData
func (h *AdminHandler) GetReferralBonusDays(c *fiber.Ctx) error {
	days, err := h.adminSvc.GetReferralBonusDays(c.Context())
	if err != nil {
		return respondInternalError(c, err)
	}

	return c.JSON(fiber.Map{"referral_bonus_days": days})
}

type SetReferralBonusDaysRequest struct {
	Days int `json:"days"`
}

// SetReferralBonusDays sets bonus days for referrer.
//
//	@Summary	Set referral bonus days
//	@Tags		admin
//	@Accept		json
//	@Produce	json
//	@Param		body	body		map[string]int	true	"{days}"
//	@Success	200		{object}	map[string]interface{}
//	@Router		/api/admin/settings/referral-bonus-days [post]
//	@Security	TelegramInitData
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

// --- Subscription admin actions ---

// RotateUserSubToken — выпустить новый sub_token для активной подписки юзера.
// Старый URL сразу перестаёт работать. Используется при компрометации
// ссылки или жалобе юзера на «слил кому-то».
//
//	@Summary	Rotate subscription token
//	@Tags		admin
//	@Produce	json
//	@Param		user_id	path	int	true	"telegram user id"
//	@Success	200	{object}	map[string]interface{}
//	@Failure	404	{object}	map[string]string
//	@Router		/api/admin/users/{user_id}/subscription/rotate-token [post]
//	@Security	TelegramInitData
func (h *AdminHandler) RotateUserSubToken(c *fiber.Ctx) error {
	userIDStr := c.Params("user_id")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid user_id"})
	}
	sub, err := h.subscriptionSvc.GetActiveSubscription(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "no active subscription"})
	}
	newToken, err := h.subscriptionSvc.RotateSubToken(c.Context(), sub.ID)
	if err != nil {
		return respondInternalError(c, err)
	}
	sub.SubToken = newToken
	return c.JSON(fiber.Map{
		"success":          true,
		"subscription_url": h.subscriptionSvc.BuildSubscriptionURL(sub),
	})
}

