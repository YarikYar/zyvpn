package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"github.com/zyvpn/backend/internal/config"
	"github.com/zyvpn/backend/internal/handler"
	"github.com/zyvpn/backend/internal/middleware"
	"github.com/zyvpn/backend/internal/repository"
	"github.com/zyvpn/backend/internal/service"
	"github.com/zyvpn/backend/internal/telegram"
	"github.com/zyvpn/backend/internal/ton"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Connect to database
	repo, err := repository.New(cfg.Database.DSN())
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer repo.Close()

	// Create services
	userService := service.NewUserService(repo)
	planService := service.NewPlanService(repo)
	referralSvc := service.NewReferralService(repo)
	serverSvc := service.NewServerService(repo)
	subscriptionSvc := service.NewSubscriptionService(repo, serverSvc, cfg)
	subscriptionSvc.SetServerService(serverSvc) // Link server service
	paymentSvc := service.NewPaymentService(repo, subscriptionSvc, referralSvc, cfg)
	ratesSvc := service.NewRatesService()
	balanceSvc := service.NewBalanceService(repo)
	promoCodeSvc := service.NewPromoCodeService(repo)
	adminSvc := service.NewAdminService(repo)

	// Set balance service on payment service (to avoid circular dependency)
	paymentSvc.SetBalanceService(balanceSvc)

	// Set dependencies on promo code service (to avoid circular dependency)
	promoCodeSvc.SetBalanceService(balanceSvc)
	promoCodeSvc.SetSubscriptionService(subscriptionSvc)

	// Set dependencies on admin service (to avoid circular dependency)
	adminSvc.SetBalanceService(balanceSvc)
	adminSvc.SetSubscriptionService(subscriptionSvc)
	adminSvc.SetPromoCodeService(promoCodeSvc)

	// Create TON verifier and worker
	tonVerifier := ton.NewVerifier(cfg.TON.Testnet, cfg.TON.WalletAddress)
	tonWorker := service.NewTonWorker(repo, tonVerifier, balanceSvc, paymentSvc)

	// Create Telegram bot
	var bot *telegram.Bot
	if cfg.Telegram.BotToken != "" {
		bot, err = telegram.NewBot(cfg, userService, subscriptionSvc, referralSvc)
		if err != nil {
			log.Printf("Warning: Failed to create Telegram bot: %v", err)
		} else {
			bot.SetPaymentService(paymentSvc)
			paymentSvc.SetNotifier(bot)
			log.Printf("Telegram bot @%s initialized", bot.GetBotUsername())
		}
	}

	// Create handlers
	h := handler.New(cfg, userService, planService, subscriptionSvc, paymentSvc, referralSvc, ratesSvc, balanceSvc, promoCodeSvc, adminSvc, bot)
	adminHandler := handler.NewAdminHandler(adminSvc)
	serverHandler := handler.NewServerHandler(serverSvc)

	// Create Fiber app
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{
				"error": err.Error(),
			})
		},
	})

	// Middleware
	app.Use(recover.New())
	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: cfg.Server.AllowOrigins,
		AllowHeaders: "Origin, Content-Type, Accept, Authorization, X-Telegram-Init-Data",
	}))

	// Health check
	app.Get("/health", h.Health)
	app.Get("/internal/health", h.Health)

	// Public API (no auth required)
	app.Get("/api/rates", h.GetRates)

	// Webhooks (no auth required) - TON payment callbacks
	app.Post("/webhook/ton", h.TONWebhook)
	app.Post("/webhook/stars", h.StarsWebhook)

	// API routes with Telegram authentication
	api := app.Group("/api", middleware.TelegramAuth(cfg))

	// Plans
	api.Get("/plans", h.GetPlans)

	// User
	api.Get("/user/me", h.GetMe)

	// Subscription
	api.Post("/subscription/buy", h.BuySubscription)
	api.Get("/subscription/key", h.GetSubscriptionKey)
	api.Get("/subscription/status", h.GetSubscriptionStatus)
	api.Post("/subscription/trial", h.ActivateTrial)
	api.Get("/subscription/switch-server/info", h.GetSwitchServerInfo)
	api.Post("/subscription/switch-server", h.SwitchServer)

	// Payments
	api.Get("/payment/ton/init", h.InitTONPayment)
	api.Post("/payment/ton/check", h.VerifyTONPayment)
	api.Get("/payment/stars/init", h.InitStarsPayment)
	api.Post("/payment/stars/refund", h.RefundStarsPayment)
	api.Get("/payment/status", h.GetPaymentStatus)

	// Referrals
	api.Get("/referral/stats", h.GetReferralStats)
	api.Get("/referral/link", h.GetReferralLink)
	api.Post("/referral/apply", h.ApplyReferralCode)
	api.Get("/referral/users", h.GetReferredUsers)

	// Balance
	api.Get("/balance", h.GetBalance)
	api.Get("/balance/transactions", h.GetBalanceTransactions)
	api.Post("/balance/pay", h.PayFromBalance)
	api.Post("/balance/topup", h.InitTopUp)
	api.Get("/balance/topup/ton", h.GetTopUpTONInfo)
	api.Get("/balance/topup/stars", h.InitTopUpStars)
	api.Post("/balance/topup/verify", h.VerifyTopUp)

	// Promo codes
	api.Post("/promo/apply", h.ApplyPromoCode)
	api.Get("/promo/validate", h.ValidatePromoCode)

	// Servers (for users)
	api.Get("/servers", serverHandler.GetServers)

	// Admin panel routes (requires Telegram auth + admin check)
	admin := app.Group("/api/admin", middleware.TelegramAuth(cfg), middleware.AdminAuth(adminSvc))
	admin.Get("/stats", adminHandler.GetStats)

	// Admin - User management
	admin.Get("/users", adminHandler.ListUsers)
	admin.Get("/users/:user_id", adminHandler.GetUser)
	admin.Post("/users/:user_id/balance/set", adminHandler.SetBalance)
	admin.Post("/users/:user_id/balance/add", adminHandler.AddBalance)
	admin.Post("/users/:user_id/subscription/extend", adminHandler.ExtendSubscription)
	admin.Post("/users/:user_id/subscription/cancel", adminHandler.CancelSubscription)

	// Admin - Ban management
	admin.Get("/bans", adminHandler.ListBans)
	admin.Post("/users/:user_id/ban", adminHandler.BanUser)
	admin.Post("/users/:user_id/unban", adminHandler.UnbanUser)
	admin.Post("/bans/ip", adminHandler.BanIP)
	admin.Post("/bans/ip/unban", adminHandler.UnbanIP)

	// Admin - Promo codes
	admin.Get("/promo", adminHandler.ListPromoCodes)
	admin.Post("/promo", adminHandler.CreatePromoCode)
	admin.Post("/promo/bulk", adminHandler.CreateBulkPromoCodes)
	admin.Post("/promo/deactivate", adminHandler.DeactivatePromoCode)

	// Admin - Plans
	admin.Get("/plans", adminHandler.ListPlans)
	admin.Post("/plans", adminHandler.CreatePlan)
	admin.Put("/plans/:plan_id", adminHandler.UpdatePlan)
	admin.Delete("/plans/:plan_id", adminHandler.DeletePlan)

	// Admin - Logs
	admin.Get("/logs", adminHandler.GetLogs)

	// Admin - Settings
	admin.Get("/settings", adminHandler.GetSettings)
	admin.Get("/settings/topup-bonus", adminHandler.GetTopupBonus)
	admin.Post("/settings/topup-bonus", adminHandler.SetTopupBonus)
	admin.Get("/settings/referral-bonus", adminHandler.GetReferralBonus)
	admin.Post("/settings/referral-bonus", adminHandler.SetReferralBonus)
	admin.Get("/settings/referral-bonus-days", adminHandler.GetReferralBonusDays)
	admin.Post("/settings/referral-bonus-days", adminHandler.SetReferralBonusDays)
	admin.Get("/settings/region-switch-price", adminHandler.GetRegionSwitchPrice)
	admin.Post("/settings/region-switch-price", adminHandler.SetRegionSwitchPrice)

	// Admin - Servers
	admin.Get("/servers", serverHandler.GetAllServers)
	admin.Get("/servers/:server_id", serverHandler.GetServer)
	admin.Post("/servers", serverHandler.CreateServer)
	admin.Put("/servers/:server_id", serverHandler.UpdateServer)
	admin.Delete("/servers/:server_id", serverHandler.DeleteServer)
	admin.Post("/servers/:server_id/test", serverHandler.TestServerConnection)

	// Internal endpoints (for cron jobs)
	internal := app.Group("/internal")
	internal.Post("/cron/expire", func(c *fiber.Ctx) error {
		if err := subscriptionSvc.ProcessExpiredSubscriptions(c.Context()); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		return c.JSON(fiber.Map{"status": "ok"})
	})

	internal.Post("/cron/notify", func(c *fiber.Ctx) error {
		if bot == nil {
			return c.JSON(fiber.Map{"status": "bot not configured"})
		}

		// Notify about subscriptions expiring in 3 days
		expiring3Days, _ := subscriptionSvc.GetExpiringSubscriptions(c.Context(), 72)
		for _, sub := range expiring3Days {
			_ = bot.SendSubscriptionExpiring(sub.UserID, sub.DaysRemaining())
		}

		// Notify about subscriptions expiring in 1 day
		expiring1Day, _ := subscriptionSvc.GetExpiringSubscriptions(c.Context(), 24)
		for _, sub := range expiring1Day {
			_ = bot.SendSubscriptionExpiring(sub.UserID, sub.DaysRemaining())
		}

		return c.JSON(fiber.Map{"status": "ok"})
	})

	// Start background jobs
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start Telegram bot long polling
	if bot != nil {
		go bot.StartPolling(ctx)
		log.Println("Telegram bot started with long polling")
	}

	// Start TON transaction verification worker
	go tonWorker.Start(ctx)

	// Start server health checker
	healthWorker := service.NewHealthWorker(repo, serverSvc)
	go healthWorker.Start(ctx)

	go runSubscriptionChecker(ctx, subscriptionSvc, bot)

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		log.Println("Shutting down server...")
		cancel()
		_ = app.Shutdown()
	}()

	// Start server
	log.Printf("Server starting on port %s", cfg.Server.Port)
	if err := app.Listen(":" + cfg.Server.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func runSubscriptionChecker(ctx context.Context, subscriptionSvc *service.SubscriptionService, bot *telegram.Bot) {
	ticker := time.NewTicker(config.SubscriptionCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Process expired subscriptions
			if err := subscriptionSvc.ProcessExpiredSubscriptions(ctx); err != nil {
				log.Printf("Error processing expired subscriptions: %v", err)
			}

			// Send expiration notifications
			if bot != nil {
				// 3 days before
				expiring3Days, _ := subscriptionSvc.GetExpiringSubscriptions(ctx, 72)
				for _, sub := range expiring3Days {
					if sub.DaysRemaining() == 3 {
						_ = bot.SendSubscriptionExpiring(sub.UserID, 3)
					}
				}

				// 1 day before
				expiring1Day, _ := subscriptionSvc.GetExpiringSubscriptions(ctx, 24)
				for _, sub := range expiring1Day {
					if sub.DaysRemaining() == 1 {
						_ = bot.SendSubscriptionExpiring(sub.UserID, 1)
					}
				}
			}
		}
	}
}
