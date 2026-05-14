package telegram

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	tele "gopkg.in/telebot.v3"

	"github.com/zyvpn/backend/internal/config"
	"github.com/zyvpn/backend/internal/repository"
	"github.com/zyvpn/backend/internal/service"
)

type Bot struct {
	bot             *tele.Bot
	cfg             *config.Config
	repo            *repository.Repository
	userService     *service.UserService
	subscriptionSvc *service.SubscriptionService
	referralSvc     *service.ReferralService
	paymentSvc      *service.PaymentService
}

func NewBot(
	cfg *config.Config,
	repo *repository.Repository,
	userService *service.UserService,
	subscriptionSvc *service.SubscriptionService,
	referralSvc *service.ReferralService,
) (*Bot, error) {
	pref := tele.Settings{
		Token:  cfg.Telegram.BotToken,
		Poller: &tele.LongPoller{Timeout: 60 * time.Second},
	}

	bot, err := tele.NewBot(pref)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot: %w", err)
	}

	b := &Bot{
		bot:             bot,
		cfg:             cfg,
		repo:            repo,
		userService:     userService,
		subscriptionSvc: subscriptionSvc,
		referralSvc:     referralSvc,
	}

	b.registerHandlers()

	return b, nil
}

func (b *Bot) registerHandlers() {
	b.bot.Handle("/start", b.handleStart)
	b.bot.Handle("/status", b.handleStatus)
	b.bot.Handle("/key", b.handleKey)
	b.bot.Handle("/help", b.handleHelp)
	b.bot.Handle("/support", b.handleSupport)
	b.bot.Handle("/referral", b.handleReferral)
	b.bot.Handle("/trial", b.handleTrial)
	b.bot.Handle("/apps", b.handleApps)

	b.bot.Handle(tele.OnCallback, b.handleCallback)
	b.bot.Handle(tele.OnCheckout, b.handlePreCheckout)
	b.bot.Handle(tele.OnPayment, b.handleSuccessfulPayment)
}

func (b *Bot) SetPaymentService(svc *service.PaymentService) {
	b.paymentSvc = svc
}

func (b *Bot) StartPolling(ctx context.Context) {
	go func() {
		<-ctx.Done()
		b.bot.Stop()
	}()
	b.bot.Start()
}

func (b *Bot) handlePreCheckout(c tele.Context) error {
	// Accept all pre-checkout queries
	return c.Accept()
}

func (b *Bot) handleSuccessfulPayment(c tele.Context) error {
	payment := c.Message().Payment
	if payment == nil {
		return nil
	}

	log.Printf("Received successful payment: %+v", payment)

	// Payment payload contains our payment ID
	paymentID, err := uuid.Parse(payment.Payload)
	if err != nil {
		log.Printf("Invalid payment payload: %s", payment.Payload)
		return nil
	}

	if b.paymentSvc == nil {
		log.Printf("Payment service not configured")
		return nil
	}

	// Save Telegram charge ID for potential refunds
	if payment.TelegramChargeID != "" {
		if err := b.paymentSvc.UpdateExternalID(context.Background(), paymentID, payment.TelegramChargeID); err != nil {
			log.Printf("Failed to save telegram charge ID: %v", err)
		}
	}

	// Complete the payment
	if err := b.paymentSvc.CompletePayment(context.Background(), paymentID); err != nil {
		log.Printf("Failed to complete payment %s: %v", paymentID, err)
		return c.Send("Ошибка при обработке платежа. Обратитесь в поддержку.")
	}

	// Get subscription for notification
	sub, err := b.subscriptionSvc.GetActiveSubscription(context.Background(), c.Sender().ID)
	if err == nil && sub != nil {
		return b.SendSubscriptionActivated(c.Sender().ID, sub.ExpiresAt.Format("02.01.2006"))
	}

	return c.Send("✅ Оплата прошла успешно! Используйте /key для получения ключа.")
}

func (b *Bot) GetBotUsername() string {
	return b.bot.Me.Username
}

func (b *Bot) handleStart(c tele.Context) error {
	user := c.Sender()

	var referredBy *int64
	args := c.Message().Payload
	if strings.HasPrefix(args, "ref_") {
		code := strings.TrimPrefix(args, "ref_")
		referrer, err := b.userService.GetUserByReferralCode(context.Background(), code)
		if err == nil && referrer.ID != user.ID {
			referredBy = &referrer.ID
		}
	}

	firstName := user.FirstName
	lastName := user.LastName
	username := user.Username
	langCode := user.LanguageCode

	telegramUser := service.TelegramUser{
		ID:           user.ID,
		Username:     &username,
		FirstName:    &firstName,
		LastName:     &lastName,
		LanguageCode: &langCode,
		ReferredBy:   referredBy,
	}

	_, isNew, err := b.userService.GetOrCreateUser(context.Background(), telegramUser)
	if err != nil {
		return err
	}

	if isNew && referredBy != nil {
		_ = b.referralSvc.CreateReferral(context.Background(), *referredBy, user.ID)
	}

	text := fmt.Sprintf(`Привет, %s! 👋

🔐 <b>CoreVPN</b> — быстрый и безопасный VPN

✅ Протокол VLESS + Reality
✅ Высокая скорость
✅ Безлимитные тарифы
✅ Простое подключение

Нажми кнопку ниже, чтобы выбрать тариф и оплатить подписку.`, user.FirstName)

	if isNew && referredBy != nil {
		text += "\n\n🎁 Тебя пригласил друг! Твой друг получит бонус при твоей первой оплате."
	}

	// Check if user has used trial
	hasUsedTrial, _ := b.subscriptionSvc.HasUsedTrial(context.Background(), user.ID)

	keyboard := &tele.ReplyMarkup{}
	var rows []tele.Row

	// Only show trial button if user hasn't used it yet
	if !hasUsedTrial {
		rows = append(rows, keyboard.Row(
			keyboard.Data("🎁 Попробовать бесплатно (3 дня)", "trial"),
		))
	}

	rows = append(rows,
		keyboard.Row(
			keyboard.WebApp("📱 Открыть магазин", &tele.WebApp{URL: b.cfg.Telegram.WebAppURL}),
		),
		keyboard.Row(
			keyboard.Data("📊 Статус подписки", "status"),
			keyboard.Data("🔑 Получить ключ", "key"),
		),
		keyboard.Row(
			keyboard.Data("📲 Скачать приложение", "apps"),
		),
	)
	keyboard.Inline(rows...)

	return c.Send(text, keyboard, tele.ModeHTML)
}

func (b *Bot) handleStatus(c tele.Context) error {
	user := c.Sender()
	sub, err := b.subscriptionSvc.GetActiveSubscription(context.Background(), user.ID)
	if err != nil {
		text := `❌ <b>У вас нет активной подписки</b>

Нажмите кнопку ниже, чтобы выбрать тариф.`

		keyboard := &tele.ReplyMarkup{}
		keyboard.Inline(
			keyboard.Row(
				keyboard.WebApp("📱 Выбрать тариф", &tele.WebApp{URL: b.cfg.Telegram.WebAppURL}),
			),
		)

		return c.Send(text, keyboard, tele.ModeHTML)
	}

	_ = b.subscriptionSvc.SyncTraffic(context.Background(), sub.ID)
	sub, _ = b.subscriptionSvc.GetSubscription(context.Background(), sub.ID)

	var trafficText string
	if sub.TrafficLimit > 0 {
		trafficGB := float64(sub.TrafficUsed) / (1024 * 1024 * 1024)
		limitGB := float64(sub.TrafficLimit) / (1024 * 1024 * 1024)
		trafficText = fmt.Sprintf("%.2f / %.0f ГБ", trafficGB, limitGB)
	} else {
		trafficGB := float64(sub.TrafficUsed) / (1024 * 1024 * 1024)
		trafficText = fmt.Sprintf("%.2f ГБ (безлимит)", trafficGB)
	}

	text := fmt.Sprintf(`✅ <b>Подписка активна</b>

📅 Действует до: %s
📊 Трафик: %s
⏳ Осталось: %d дней`,
		sub.ExpiresAt.Format("02.01.2006"),
		trafficText,
		sub.DaysRemaining(),
	)

	keyboard := &tele.ReplyMarkup{}
	keyboard.Inline(
		keyboard.Row(
			keyboard.Data("🔑 Получить ключ", "key"),
		),
		keyboard.Row(
			keyboard.WebApp("📱 Продлить подписку", &tele.WebApp{URL: b.cfg.Telegram.WebAppURL}),
		),
	)

	return c.Send(text, keyboard, tele.ModeHTML)
}

func (b *Bot) handleKey(c tele.Context) error {
	user := c.Sender()
	subURL, err := b.subscriptionSvc.GetSubscriptionURLForUser(context.Background(), user.ID)
	if err != nil {
		text := `❌ <b>Подписка не активна</b>

У вас нет активной подписки. Оформите подписку, чтобы получить ссылку подключения.`

		keyboard := &tele.ReplyMarkup{}
		keyboard.Inline(
			keyboard.Row(
				keyboard.WebApp("📱 Выбрать тариф", &tele.WebApp{URL: b.cfg.Telegram.WebAppURL}),
			),
		)

		return c.Send(text, keyboard, tele.ModeHTML)
	}

	text := fmt.Sprintf(`🔑 <b>Ваша ссылка подписки:</b>

<code>%s</code>

📱 Вставьте её в VPN-клиент — он сам подтянет все серверы:
• iOS: Streisand, V2Box, Shadowrocket
• Android: v2rayNG, Hiddify, NekoBox
• Windows/Mac: Nekoray, v2rayN, Hiddify`, subURL)

	keyboard := &tele.ReplyMarkup{}
	keyboard.Inline(
		keyboard.Row(
			keyboard.WebApp("📱 QR-код", &tele.WebApp{URL: b.cfg.Telegram.WebAppURL + "/key"}),
		),
	)

	return c.Send(text, keyboard, tele.ModeHTML)
}

func (b *Bot) handleHelp(c tele.Context) error {
	text := `📖 <b>Помощь по CoreVPN</b>

<b>🔧 Настройка VPN:</b>

1️⃣ Установи приложение под свою систему — нажми /apps

2️⃣ Получи ключ: /key или в Mini App

3️⃣ Импортируй ключ в приложение и подключись

<b>🎁 Промокоды:</b>
Mini App → Баланс → Введи промокод

<b>📱 Команды:</b>
/start — Главное меню
/apps — Скачать приложение
/status — Статус подписки
/key — Получить ключ
/trial — Бесплатный период
/referral — Реферальная программа
/support — Связаться с поддержкой

❓ Вопросы? /support`

	keyboard := &tele.ReplyMarkup{}
	keyboard.Inline(
		keyboard.Row(
			keyboard.WebApp("📱 Открыть Mini App", &tele.WebApp{URL: b.cfg.Telegram.WebAppURL}),
		),
		keyboard.Row(
			keyboard.Data("📲 Скачать приложение", "apps"),
		),
	)

	return c.Send(text, keyboard, tele.ModeHTML)
}

// App download links per platform. Apps support VLESS Reality and import
// vless:// keys, which is what we hand out via /key.
const (
	appV2rayNG = "https://github.com/2dust/v2rayNG/releases/latest"
	appHapp    = "https://apps.apple.com/app/happ-proxy-utility/id6504287215"
	appThrone  = "https://github.com/throneproj/Throne/releases/latest"
)

func (b *Bot) handleApps(c tele.Context) error {
	return c.Send(appsText(), appsKeyboard(), tele.ModeHTML, tele.NoPreview)
}

func appsText() string {
	return `📲 <b>Приложения для подключения</b>

Выбери под свою платформу. Все три понимают наш VLESS Reality ключ.

🤖 <b>Android — v2rayNG</b>
Свежий APK с GitHub. Скачай, открой ключ из бота, импортируй.

🍏 <b>iPhone / iPad — Happ</b>
Из App Store. После установки вставь ключ.

💻 <b>Windows / macOS / Linux — Throne</b>
Десктоп-клиент с GitHub. Релизы под все платформы.`
}

func appsKeyboard() *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{}
	kb.Inline(
		kb.Row(
			kb.URL("🤖 v2rayNG", appV2rayNG),
			kb.Data("📥 в чат", "appfile|v2rayng_arm64"),
		),
		kb.Row(kb.URL("🍏 Happ (iOS)", appHapp)),
		kb.Row(
			kb.URL("💻 Throne", appThrone),
			kb.Data("📥 Win", "appfile|throne_win"),
		),
		kb.Row(
			kb.Data("📥 Mac (arm64)", "appfile|throne_mac"),
			kb.Data("📥 Linux .deb", "appfile|throne_linux"),
		),
	)
	return kb
}

func (b *Bot) handleSupport(c tele.Context) error {
	text := `💬 <b>Поддержка</b>

Если у вас возникли вопросы или проблемы, напишите нам:

📧 support@corevpn.com
💬 @corevpn_support

Мы ответим в течение 24 часов.`

	return c.Send(text, tele.ModeHTML)
}

func (b *Bot) handleReferral(c tele.Context) error {
	user := c.Sender()
	stats, err := b.referralSvc.GetReferralStats(context.Background(), user.ID)
	if err != nil {
		return err
	}

	link, err := b.referralSvc.GetReferralLink(context.Background(), user.ID, b.bot.Me.Username)
	if err != nil {
		return err
	}

	text := fmt.Sprintf(`🎁 <b>Реферальная программа</b>

Приглашай друзей и получай TON на баланс!

Когда друг оплатит первую подписку:
• Ты получишь +0.1 TON на баланс

📊 <b>Твоя статистика:</b>
👥 Приглашено: %d
⏳ Ожидают оплаты: %d
💎 Получено TON: %.4f

🔗 <b>Твоя ссылка:</b>
<code>%s</code>`,
		stats.TotalReferrals,
		stats.PendingReferrals,
		stats.CreditedBonusTON,
		link,
	)

	return c.Send(text, tele.ModeHTML)
}

func (b *Bot) handleCallback(c tele.Context) error {
	data := c.Callback().Data
	fmt.Printf("[Bot] Callback received: %q from user %d\n", data, c.Sender().ID)

	// Acknowledge callback to remove loading state
	defer c.Respond()

	// telebot adds \f prefix to callback data
	clean := strings.TrimPrefix(data, "\f")

	// «Скачать в чат» buttons in /apps: "appfile|<app_key>"
	if rest, ok := strings.CutPrefix(clean, "appfile|"); ok {
		return b.sendAppDocument(context.Background(), c, rest)
	}

	switch clean {
	case "status":
		return b.handleStatus(c)
	case "key":
		return b.handleKey(c)
	case "trial":
		return b.handleTrial(c)
	case "apps":
		return b.handleApps(c)
	default:
		fmt.Printf("[Bot] Unknown callback data: %q\n", data)
	}
	return nil
}

func (b *Bot) handleTrial(c tele.Context) error {
	user := c.Sender()
	fmt.Printf("[Bot] handleTrial called for user %d\n", user.ID)

	sub, err := b.subscriptionSvc.ActivateTrial(context.Background(), user.ID)
	if err != nil {
		fmt.Printf("[Bot] Trial activation error for user %d: %v\n", user.ID, err)
		var text string
		if err.Error() == "trial subscription already used" {
			text = `❌ <b>Trial уже использован</b>

Вы уже использовали бесплатный пробный период. Выберите тариф для продления.`
		} else if err.Error() == "user already has an active subscription" {
			text = `❌ <b>Подписка уже активна</b>

У вас уже есть активная подписка. Используйте /status для проверки статуса.`
		} else {
			text = fmt.Sprintf("❌ Ошибка: %s", err.Error())
		}

		keyboard := &tele.ReplyMarkup{}
		keyboard.Inline(
			keyboard.Row(
				keyboard.WebApp("📱 Выбрать тариф", &tele.WebApp{URL: b.cfg.Telegram.WebAppURL}),
			),
		)

		return c.Send(text, keyboard, tele.ModeHTML)
	}

	text := fmt.Sprintf(`✅ <b>Trial подписка активирована!</b>

🎁 Вам предоставлен бесплатный доступ на 3 дня.

📅 Действует до: %s
📊 Трафик: 10 ГБ

Используйте команду /key чтобы получить ключ подключения.`, sub.ExpiresAt.Format("02.01.2006 15:04"))

	keyboard := &tele.ReplyMarkup{}
	keyboard.Inline(
		keyboard.Row(
			keyboard.Data("🔑 Получить ключ", "key"),
		),
	)

	return c.Send(text, keyboard, tele.ModeHTML)
}

func (b *Bot) SendMessage(chatID int64, text string) error {
	_, err := b.bot.Send(&tele.User{ID: chatID}, text, tele.ModeHTML)
	return err
}

func (b *Bot) SendSubscriptionExpiring(chatID int64, daysLeft int) error {
	text := fmt.Sprintf(`⏰ <b>Подписка скоро закончится!</b>

Осталось дней: %d

Продлите подписку, чтобы не потерять доступ к VPN.`, daysLeft)

	keyboard := &tele.ReplyMarkup{}
	keyboard.Inline(
		keyboard.Row(
			keyboard.WebApp("📱 Продлить подписку", &tele.WebApp{URL: b.cfg.Telegram.WebAppURL}),
		),
	)

	_, err := b.bot.Send(&tele.User{ID: chatID}, text, keyboard, tele.ModeHTML)
	return err
}

func (b *Bot) SendSubscriptionExpired(chatID int64) error {
	text := `❌ <b>Подписка закончилась</b>

Ваша подписка на VPN истекла. Продлите подписку, чтобы восстановить доступ.`

	keyboard := &tele.ReplyMarkup{}
	keyboard.Inline(
		keyboard.Row(
			keyboard.WebApp("📱 Продлить подписку", &tele.WebApp{URL: b.cfg.Telegram.WebAppURL}),
		),
	)

	_, err := b.bot.Send(&tele.User{ID: chatID}, text, keyboard, tele.ModeHTML)
	return err
}

func (b *Bot) SendSubscriptionActivated(chatID int64, expiresAt string) error {
	text := fmt.Sprintf(`✅ <b>Подписка активирована!</b>

Спасибо за покупку! Ваша подписка активна до %s.

Используйте команду /key чтобы получить ключ подключения.`, expiresAt)

	keyboard := &tele.ReplyMarkup{}
	keyboard.Inline(
		keyboard.Row(
			keyboard.Data("🔑 Получить ключ", "key"),
		),
	)

	_, err := b.bot.Send(&tele.User{ID: chatID}, text, keyboard, tele.ModeHTML)
	return err
}

// SendReferralBonus notifies referrer about received bonus
func (b *Bot) SendReferralBonus(chatID int64, bonusTON float64, bonusDays int) error {
	var text string
	if bonusTON > 0 && bonusDays > 0 {
		text = fmt.Sprintf(`🎁 <b>Реферальный бонус!</b>

Ваш друг оплатил подписку. Вы получили:
💎 <b>+%.4f TON</b> на баланс
📅 <b>+%d дней</b> к подписке

Спасибо, что приглашаете друзей!`, bonusTON, bonusDays)
	} else if bonusTON > 0 {
		text = fmt.Sprintf(`🎁 <b>Реферальный бонус!</b>

Ваш друг оплатил подписку. Вы получили:
💎 <b>+%.4f TON</b> на баланс

Спасибо, что приглашаете друзей!`, bonusTON)
	} else if bonusDays > 0 {
		text = fmt.Sprintf(`🎁 <b>Реферальный бонус!</b>

Ваш друг оплатил подписку. Вы получили:
📅 <b>+%d дней</b> к подписке

Спасибо, что приглашаете друзей!`, bonusDays)
	} else {
		return nil // No bonus to notify about
	}

	keyboard := &tele.ReplyMarkup{}
	keyboard.Inline(
		keyboard.Row(
			keyboard.WebApp("👥 Пригласить ещё", &tele.WebApp{URL: b.cfg.Telegram.WebAppURL + "/#/referral"}),
		),
	)

	_, err := b.bot.Send(&tele.User{ID: chatID}, text, keyboard, tele.ModeHTML)
	return err
}

// SendBalanceTopUp notifies user about balance top-up
func (b *Bot) SendBalanceTopUp(chatID int64, amount float64, newBalance float64) error {
	text := fmt.Sprintf(`💰 <b>Баланс пополнен!</b>

Зачислено: <b>+%.4f TON</b>
Текущий баланс: <b>%.4f TON</b>`, amount, newBalance)

	_, err := b.bot.Send(&tele.User{ID: chatID}, text, tele.ModeHTML)
	return err
}

// CreateStarsInvoice creates a Telegram Stars invoice link
func (b *Bot) CreateStarsInvoice(userID int64, title, description string, amount int, paymentID string) (string, error) {
	invoice := tele.Invoice{
		Title:       title,
		Description: description,
		Payload:     paymentID,
		Currency:    "XTR", // Telegram Stars
		Prices: []tele.Price{
			{Label: title, Amount: amount},
		},
	}

	link, err := b.bot.CreateInvoiceLink(invoice)
	if err != nil {
		return "", fmt.Errorf("failed to create invoice: %w", err)
	}

	return link, nil
}

// RefundStarsPayment refunds a Stars payment
func (b *Bot) RefundStarsPayment(userID int64, telegramPaymentChargeID string) error {
	params := map[string]interface{}{
		"user_id":                    userID,
		"telegram_payment_charge_id": telegramPaymentChargeID,
	}

	_, err := b.bot.Raw("refundStarPayment", params)
	if err != nil {
		return fmt.Errorf("failed to refund payment: %w", err)
	}

	return nil
}

