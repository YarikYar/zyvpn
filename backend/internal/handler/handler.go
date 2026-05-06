package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/zyvpn/backend/internal/config"
	"github.com/zyvpn/backend/internal/service"
	"github.com/zyvpn/backend/internal/telegram"
)

type Handler struct {
	cfg             *config.Config
	userService     *service.UserService
	planService     *service.PlanService
	subscriptionSvc *service.SubscriptionService
	paymentSvc      *service.PaymentService
	referralSvc     *service.ReferralService
	ratesSvc        *service.RatesService
	balanceSvc      *service.BalanceService
	promoCodeSvc    *service.PromoCodeService
	adminSvc        *service.AdminService
	bot             *telegram.Bot
}

func New(
	cfg *config.Config,
	userService *service.UserService,
	planService *service.PlanService,
	subscriptionSvc *service.SubscriptionService,
	paymentSvc *service.PaymentService,
	referralSvc *service.ReferralService,
	ratesSvc *service.RatesService,
	balanceSvc *service.BalanceService,
	promoCodeSvc *service.PromoCodeService,
	adminSvc *service.AdminService,
	bot *telegram.Bot,
) *Handler {
	return &Handler{
		cfg:             cfg,
		userService:     userService,
		planService:     planService,
		subscriptionSvc: subscriptionSvc,
		paymentSvc:      paymentSvc,
		referralSvc:     referralSvc,
		ratesSvc:        ratesSvc,
		balanceSvc:      balanceSvc,
		promoCodeSvc:    promoCodeSvc,
		adminSvc:        adminSvc,
		bot:             bot,
	}
}

// Health returns liveness status.
//
//	@Summary	Liveness probe
//	@Tags		public
//	@Produce	json
//	@Success	200	{object}	map[string]string
//	@Router		/health [get]
func (h *Handler) Health(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status": "ok",
	})
}

// GetRates returns current TON/USD/RUB exchange rates (cached 5 min).
//
//	@Summary	Exchange rates
//	@Tags		public
//	@Produce	json
//	@Success	200	{object}	service.ExchangeRates
//	@Failure	503	{object}	map[string]string
//	@Router		/api/rates [get]
func (h *Handler) GetRates(c *fiber.Ctx) error {
	rates, err := h.ratesSvc.GetRates()
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "failed to get rates",
		})
	}

	return c.JSON(rates)
}
