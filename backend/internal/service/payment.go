package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/zyvpn/backend/internal/config"
	"github.com/zyvpn/backend/internal/model"
	"github.com/zyvpn/backend/internal/repository"
	"github.com/zyvpn/backend/internal/ton"
)

var (
	ErrInvalidPaymentProvider = errors.New("Неверный способ оплаты")
	ErrPaymentAlreadyComplete = errors.New("Платёж уже завершён")
	ErrPaymentNotPending      = errors.New("Платёж не ожидает оплаты")
)

// Notifier interface for sending notifications (implemented by telegram.Bot)
type Notifier interface {
	SendReferralBonus(chatID int64, bonusTON float64, bonusDays int) error
	SendBalanceTopUp(chatID int64, amount float64, newBalance float64) error
	// NotifyAdminsCashPayment is called when a user creates a pending cash
	// payment. Bot sends an alert to all admins with inline approve/reject
	// buttons. amountRUB is the original cash amount in roubles (the
	// payment row stores amount in TON for accounting consistency).
	NotifyAdminsCashPayment(payment *model.Payment, plan *model.Plan, user *model.User, amountRUB float64) error
	// NotifyCashApproved/Rejected goes back to the buyer once the admin
	// makes a decision.
	NotifyCashApproved(chatID int64, plan *model.Plan) error
	NotifyCashRejected(chatID int64, plan *model.Plan, reason string) error
}

type PaymentService struct {
	repo            *repository.Repository
	subscriptionSvc *SubscriptionService
	referralSvc     *ReferralService
	balanceSvc      *BalanceService
	tonVerifier     *ton.Verifier
	cfg             *config.Config
	notifier        Notifier
	ratesSvc        *RatesService
}

func NewPaymentService(
	repo *repository.Repository,
	subscriptionSvc *SubscriptionService,
	referralSvc *ReferralService,
	cfg *config.Config,
) *PaymentService {
	// Create TON verifier (connects to TON network via lite servers)
	tonVerifier := ton.NewVerifier(cfg.TON.Testnet, cfg.TON.WalletAddress)

	return &PaymentService{
		repo:            repo,
		subscriptionSvc: subscriptionSvc,
		referralSvc:     referralSvc,
		tonVerifier:     tonVerifier,
		cfg:             cfg,
	}
}

// SetBalanceService sets the balance service (to avoid circular dependency)
func (s *PaymentService) SetBalanceService(balanceSvc *BalanceService) {
	s.balanceSvc = balanceSvc
}

// SetNotifier sets the notifier for sending notifications
func (s *PaymentService) SetNotifier(notifier Notifier) {
	s.notifier = notifier
}

// SetRatesService wires the rates service so cash-RUB payments can be
// converted to a TON-equivalent amount at creation time. Without rates the
// cash provider will refuse to create the payment.
func (s *PaymentService) SetRatesService(rs *RatesService) {
	s.ratesSvc = rs
}

func (s *PaymentService) CreatePayment(ctx context.Context, userID int64, planID uuid.UUID, provider model.PaymentProvider) (*model.Payment, error) {
	return s.CreatePaymentWithServer(ctx, userID, planID, nil, provider)
}

func (s *PaymentService) CreatePaymentWithServer(ctx context.Context, userID int64, planID uuid.UUID, serverID *uuid.UUID, provider model.PaymentProvider) (*model.Payment, error) {
	plan, err := s.repo.GetPlan(ctx, planID)
	if err != nil {
		return nil, err
	}

	var amount float64
	var currency string

	switch provider {
	case model.PaymentProviderTON, model.PaymentProviderBalance:
		amount = plan.PriceTON
		currency = "TON"
	case model.PaymentProviderStars:
		amount = float64(plan.PriceStars)
		currency = "XTR" // Telegram Stars
	case model.PaymentProviderCash:
		// Cash payments use plan.PriceUSD as the default RUB amount the
		// representative collects. For discounted cash purchases the caller
		// goes through CreateCashPaymentWithAmount instead.
		amount = plan.PriceUSD
		currency = "RUB"
	default:
		return nil, ErrInvalidPaymentProvider
	}

	payment := &model.Payment{
		UserID:      userID,
		PlanID:      &planID,
		ServerID:    serverID,
		PaymentType: model.PaymentTypeSubscription,
		Provider:    provider,
		Amount:      amount,
		Currency:    currency,
		Status:      model.PaymentStatusPending,
	}

	if err := s.repo.CreatePayment(ctx, payment); err != nil {
		return nil, err
	}

	return payment, nil
}

func (s *PaymentService) GetPayment(ctx context.Context, id uuid.UUID) (*model.Payment, error) {
	return s.repo.GetPayment(ctx, id)
}

func (s *PaymentService) GetTONPaymentInfo(ctx context.Context, paymentID uuid.UUID) (*model.TONPaymentInfo, error) {
	payment, err := s.repo.GetPayment(ctx, paymentID)
	if err != nil {
		return nil, err
	}

	if payment.PlanID == nil {
		return nil, errors.New("payment has no plan")
	}

	plan, err := s.repo.GetPlan(ctx, *payment.PlanID)
	if err != nil {
		return nil, err
	}

	// Format amount for TON (9 decimals)
	amountNano := fmt.Sprintf("%.0f", plan.PriceTON*1e9)
	comment := payment.ID.String()

	deepLink := fmt.Sprintf("ton://transfer/%s?amount=%s&text=%s",
		s.cfg.TON.WalletAddress,
		amountNano,
		comment,
	)

	return &model.TONPaymentInfo{
		PaymentID:     payment.ID,
		WalletAddress: s.cfg.TON.WalletAddress,
		Amount:        fmt.Sprintf("%.9f", plan.PriceTON),
		Comment:       comment,
		DeepLink:      deepLink,
	}, nil
}

func (s *PaymentService) VerifyTONPayment(ctx context.Context, paymentID uuid.UUID, boc string) error {
	payment, err := s.repo.GetPayment(ctx, paymentID)
	if err != nil {
		return err
	}

	if payment.Status != model.PaymentStatusPending && payment.Status != model.PaymentStatusAwaitingTx {
		return ErrPaymentNotPending
	}

	// Set status to awaiting_tx - background worker will verify
	if payment.Status == model.PaymentStatusPending {
		if err := s.repo.UpdatePaymentStatus(ctx, paymentID, model.PaymentStatusAwaitingTx); err != nil {
			return err
		}
		fmt.Printf("[TON] Payment %s set to awaiting_tx, worker will verify\n", paymentID)
	}

	// Try immediate verification (optional - worker will also check)
	expectedAmountNano := int64(payment.Amount * 1e9)
	txInfo, err := s.tonVerifier.VerifyTransaction(boc, expectedAmountNano, "")
	if err != nil {
		// Not found yet - that's ok, worker will keep checking
		fmt.Printf("[TON] Payment %s: transaction not confirmed yet, worker will retry\n", paymentID)
		return nil // Return success - payment is being processed
	}

	fmt.Printf("[TON] Transaction verified immediately: hash=%s, amount=%d nanoTON, from=%s\n",
		txInfo.Hash, txInfo.Amount, txInfo.FromAddress)

	// Update external ID with verified transaction hash
	if err := s.repo.UpdatePaymentExternalID(ctx, paymentID, txInfo.Hash); err != nil {
		return err
	}

	return s.CompletePayment(ctx, paymentID)
}

// GetPaymentStatus returns payment status for polling
func (s *PaymentService) GetPaymentStatus(ctx context.Context, paymentID uuid.UUID) (*model.Payment, error) {
	return s.repo.GetPayment(ctx, paymentID)
}

func (s *PaymentService) CompletePayment(ctx context.Context, paymentID uuid.UUID) error {
	payment, err := s.repo.GetPayment(ctx, paymentID)
	if err != nil {
		return err
	}

	if payment.Status == model.PaymentStatusCompleted {
		return ErrPaymentAlreadyComplete
	}

	// For top-up payments, use CompleteTopUpPayment instead
	if payment.PaymentType == model.PaymentTypeTopUp {
		return s.CompleteTopUpPayment(ctx, paymentID, "")
	}

	if payment.PlanID == nil {
		return errors.New("subscription payment has no plan")
	}

	plan, err := s.repo.GetPlan(ctx, *payment.PlanID)
	if err != nil {
		return err
	}

	// Создаём подписку — серверы определяются тарифом (plan_servers),
	// payment.server_id больше не учитывается.
	sub, err := s.subscriptionSvc.CreateSubscription(ctx, payment.UserID, plan)
	if err != nil {
		return fmt.Errorf("failed to create subscription: %w", err)
	}

	// Update payment
	if err := s.repo.UpdatePaymentSubscription(ctx, paymentID, sub.ID); err != nil {
		return err
	}

	if err := s.repo.UpdatePaymentStatus(ctx, paymentID, model.PaymentStatusCompleted); err != nil {
		return err
	}

	// Process referral bonus (percentage of payment in TON) - for every payment
	if err := s.creditReferralBonus(ctx, payment); err != nil {
		// Log error but don't fail the payment
		fmt.Printf("Failed to credit referral bonus for user %d: %v\n", payment.UserID, err)
	}

	return nil
}

// creditReferralBonus credits percentage of payment (converted to TON) to referrer's balance
// and adds bonus days to referrer's subscription (only on first payment)
func (s *PaymentService) creditReferralBonus(ctx context.Context, payment *model.Payment) error {
	if s.balanceSvc == nil {
		return fmt.Errorf("balance service not configured")
	}

	// Get referral (any status - we credit on every payment)
	referral, err := s.repo.GetReferralByReferredID(ctx, payment.UserID)
	if err != nil {
		if err == repository.ErrReferralNotFound {
			return nil // No referral relationship
		}
		return err
	}

	isFirstPayment := referral.Status == model.ReferralStatusPending

	// Convert payment amount to TON equivalent
	var paymentAmountTON float64
	switch payment.Currency {
	case "TON":
		paymentAmountTON = payment.Amount
	case "XTR": // Stars - convert to TON (100 Stars = 1 TON)
		paymentAmountTON = payment.Amount / 100
	default:
		return fmt.Errorf("unsupported currency: %s", payment.Currency)
	}

	// Track bonuses for notification
	var creditedTON float64
	var creditedDays int

	// Get bonus percentage from settings
	bonusPercent, err := s.repo.GetSettingFloat(ctx, "referral_bonus_percent")
	if err != nil {
		bonusPercent = 5 // Default 5%
	}

	// Credit TON bonus if percentage > 0
	if bonusPercent > 0 {
		bonusAmount := paymentAmountTON * bonusPercent / 100
		if bonusAmount >= 0.0001 {
			_, err = s.balanceSvc.CreditReferralBonus(ctx, referral.ReferrerID, bonusAmount, referral.ID)
			if err != nil {
				fmt.Printf("Failed to credit TON bonus: %v\n", err)
			} else {
				creditedTON = bonusAmount
				fmt.Printf("Credited %.4f TON (%.1f%% of %.4f TON) to user %d for referral of user %d\n",
					bonusAmount, bonusPercent, paymentAmountTON, referral.ReferrerID, payment.UserID)
			}
		}
	}

	// Add bonus days to referrer's subscription (only on first payment from referred user)
	if isFirstPayment {
		bonusDays, err := s.repo.GetSettingFloat(ctx, "referral_bonus_days")
		if err != nil {
			bonusDays = 0
		}
		if bonusDays > 0 {
			// Get referrer's active subscription
			referrerSub, err := s.subscriptionSvc.GetActiveSubscription(ctx, referral.ReferrerID)
			if err != nil {
				fmt.Printf("Referrer %d has no active subscription, skipping bonus days\n", referral.ReferrerID)
			} else {
				// Extend referrer's subscription
				err = s.subscriptionSvc.ExtendSubscription(ctx, referrerSub.ID, int(bonusDays))
				if err != nil {
					fmt.Printf("Failed to add %d bonus days to user %d: %v\n", int(bonusDays), referral.ReferrerID, err)
				} else {
					creditedDays = int(bonusDays)
					fmt.Printf("Added %d bonus days to user %d for referral of user %d\n",
						int(bonusDays), referral.ReferrerID, payment.UserID)
				}
			}
		}
	}

	// Mark referral as credited if it was pending (first payment)
	if isFirstPayment {
		_ = s.referralSvc.MarkReferralCredited(ctx, referral.ID)
	}

	// Send notification to referrer
	if s.notifier != nil && (creditedTON > 0 || creditedDays > 0) {
		if err := s.notifier.SendReferralBonus(referral.ReferrerID, creditedTON, creditedDays); err != nil {
			fmt.Printf("Failed to send referral bonus notification: %v\n", err)
		}
	}

	return nil
}

func (s *PaymentService) FailPayment(ctx context.Context, paymentID uuid.UUID) error {
	return s.repo.UpdatePaymentStatus(ctx, paymentID, model.PaymentStatusFailed)
}

// CreateCashPaymentWithAmount creates a pending cash payment for `plan`. The
// admin specifies amountRUB; we convert it to a TON-equivalent at the
// current rate so referral bonuses and accounting stay consistent across
// providers (everything ends up in TON internally). The original RUB amount
// and the rate used are stored in the payment metadata for the audit trail.
//
// Stores currency='TON' and amount=<TON value> so existing creditReferralBonus
// logic (and any future payment analytics) can treat the row as a TON payment
// without provider-specific handling.
func (s *PaymentService) CreateCashPaymentWithAmount(ctx context.Context, userID int64, planID uuid.UUID, serverID *uuid.UUID, amountRUB float64) (*model.Payment, error) {
	if amountRUB <= 0 {
		return nil, errors.New("amount_rub must be positive")
	}
	if s.ratesSvc == nil {
		return nil, errors.New("rates service not configured for cash payments")
	}
	rates, err := s.ratesSvc.GetRates()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch exchange rates: %w", err)
	}
	if rates.TONRUB <= 0 {
		return nil, errors.New("TON/RUB rate is unavailable")
	}

	if _, err := s.repo.GetPlan(ctx, planID); err != nil {
		return nil, err
	}

	amountTON := amountRUB / rates.TONRUB
	metadata := fmt.Sprintf(`{"amount_rub":%.2f,"ton_rub_rate":%.4f}`, amountRUB, rates.TONRUB)

	payment := &model.Payment{
		UserID:      userID,
		PlanID:      &planID,
		ServerID:    serverID,
		PaymentType: model.PaymentTypeSubscription,
		Provider:    model.PaymentProviderCash,
		Amount:      amountTON,
		Currency:    "TON",
		Status:      model.PaymentStatusPending,
		Metadata:    &metadata,
	}

	if err := s.repo.CreatePayment(ctx, payment); err != nil {
		return nil, err
	}

	plan, err := s.repo.GetPlan(ctx, planID)
	if err != nil {
		return nil, err
	}

	// Best-effort admin notification — payment is already in DB so failure
	// here doesn't fail the user-facing call.
	if s.notifier != nil {
		user, _ := s.repo.GetUser(ctx, userID)
		if err := s.notifier.NotifyAdminsCashPayment(payment, plan, user, amountRUB); err != nil {
			fmt.Printf("[cash] admin notify failed: %v\n", err)
		}
	}

	return payment, nil
}

// ApproveCashPayment marks a pending cash payment as completed and runs the
// regular subscription provisioning. Idempotent: re-approving a completed
// payment returns ErrPaymentAlreadyComplete.
func (s *PaymentService) ApproveCashPayment(ctx context.Context, paymentID uuid.UUID) error {
	payment, err := s.repo.GetPayment(ctx, paymentID)
	if err != nil {
		return err
	}
	if payment.Provider != model.PaymentProviderCash {
		return errors.New("payment is not a cash payment")
	}
	if payment.Status == model.PaymentStatusCompleted {
		return ErrPaymentAlreadyComplete
	}
	if payment.Status != model.PaymentStatusPending {
		return ErrPaymentNotPending
	}

	if err := s.CompletePayment(ctx, paymentID); err != nil {
		return err
	}

	if s.notifier != nil && payment.PlanID != nil {
		plan, perr := s.repo.GetPlan(ctx, *payment.PlanID)
		if perr == nil {
			_ = s.notifier.NotifyCashApproved(payment.UserID, plan)
		}
	}
	return nil
}

// RejectCashPayment marks a pending cash payment as failed and notifies the
// buyer. Reason is optional and shown to the user.
func (s *PaymentService) RejectCashPayment(ctx context.Context, paymentID uuid.UUID, reason string) error {
	payment, err := s.repo.GetPayment(ctx, paymentID)
	if err != nil {
		return err
	}
	if payment.Provider != model.PaymentProviderCash {
		return errors.New("payment is not a cash payment")
	}
	if payment.Status != model.PaymentStatusPending {
		return ErrPaymentNotPending
	}
	if err := s.repo.UpdatePaymentStatus(ctx, paymentID, model.PaymentStatusFailed); err != nil {
		return err
	}
	if s.notifier != nil && payment.PlanID != nil {
		plan, perr := s.repo.GetPlan(ctx, *payment.PlanID)
		if perr == nil {
			_ = s.notifier.NotifyCashRejected(payment.UserID, plan, reason)
		}
	}
	return nil
}

// ListPendingCashPayments returns up to limit pending cash payments for the
// admin panel (sorted newest first).
func (s *PaymentService) ListPendingCashPayments(ctx context.Context, limit int) ([]model.Payment, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	return s.repo.GetPendingCashPayments(ctx, limit)
}

func (s *PaymentService) GetUserPayments(ctx context.Context, userID int64) ([]model.Payment, error) {
	return s.repo.GetUserPayments(ctx, userID)
}

func (s *PaymentService) UpdateExternalID(ctx context.Context, paymentID uuid.UUID, externalID string) error {
	return s.repo.UpdatePaymentExternalID(ctx, paymentID, externalID)
}

func (s *PaymentService) RefundPayment(ctx context.Context, paymentID uuid.UUID) error {
	payment, err := s.repo.GetPayment(ctx, paymentID)
	if err != nil {
		return err
	}

	if payment.Status != model.PaymentStatusCompleted {
		return errors.New("can only refund completed payments")
	}

	if payment.Provider != model.PaymentProviderStars {
		return errors.New("refunds only supported for Stars payments")
	}

	if payment.ExternalID == nil || *payment.ExternalID == "" {
		return errors.New("payment has no telegram charge ID")
	}

	// Update status to refunded
	return s.repo.UpdatePaymentStatus(ctx, paymentID, model.PaymentStatusRefunded)
}

func (s *PaymentService) GetTelegramChargeID(ctx context.Context, paymentID uuid.UUID) (int64, string, error) {
	payment, err := s.repo.GetPayment(ctx, paymentID)
	if err != nil {
		return 0, "", err
	}

	if payment.ExternalID == nil || *payment.ExternalID == "" {
		return 0, "", errors.New("payment has no telegram charge ID")
	}

	return payment.UserID, *payment.ExternalID, nil
}

// CreateTopUpPayment creates a payment for balance top-up
func (s *PaymentService) CreateTopUpPayment(ctx context.Context, userID int64, amountTON float64, provider model.PaymentProvider) (*model.Payment, error) {
	var amount float64
	var currency string

	switch provider {
	case model.PaymentProviderTON:
		amount = amountTON
		currency = "TON"
	case model.PaymentProviderStars:
		// Convert TON to Stars using current rate
		// Stars = TON * rate (e.g., 1 TON = ~100 Stars at $5/TON and $0.05/Star)
		// For now, use a fixed conversion: 1 TON = 100 Stars
		amount = amountTON * 100
		currency = "XTR"
	default:
		return nil, ErrInvalidPaymentProvider
	}

	payment := &model.Payment{
		UserID:      userID,
		PlanID:      nil, // No plan for top-up
		PaymentType: model.PaymentTypeTopUp,
		Provider:    provider,
		Amount:      amount,
		Currency:    currency,
		Status:      model.PaymentStatusPending,
	}

	if err := s.repo.CreateTopUpPayment(ctx, payment); err != nil {
		return nil, err
	}

	return payment, nil
}

// GetTONTopUpInfo returns info for TON top-up payment
func (s *PaymentService) GetTONTopUpInfo(ctx context.Context, paymentID uuid.UUID) (*model.TONPaymentInfo, error) {
	payment, err := s.repo.GetPayment(ctx, paymentID)
	if err != nil {
		return nil, err
	}

	if payment.PaymentType != model.PaymentTypeTopUp {
		return nil, errors.New("payment is not a top-up")
	}

	// Format amount for TON (9 decimals)
	amountNano := fmt.Sprintf("%.0f", payment.Amount*1e9)
	comment := "topup_" + payment.ID.String()

	deepLink := fmt.Sprintf("ton://transfer/%s?amount=%s&text=%s",
		s.cfg.TON.WalletAddress,
		amountNano,
		comment,
	)

	return &model.TONPaymentInfo{
		PaymentID:     payment.ID,
		WalletAddress: s.cfg.TON.WalletAddress,
		Amount:        fmt.Sprintf("%.9f", payment.Amount),
		Comment:       comment,
		DeepLink:      deepLink,
	}, nil
}

// CompleteTopUpPayment completes a top-up payment and credits balance
func (s *PaymentService) CompleteTopUpPayment(ctx context.Context, paymentID uuid.UUID, boc string) error {
	if s.balanceSvc == nil {
		return errors.New("balance service not configured")
	}

	payment, err := s.repo.GetPayment(ctx, paymentID)
	if err != nil {
		return err
	}

	if payment.Status != model.PaymentStatusPending {
		return ErrPaymentNotPending
	}

	if payment.PaymentType != model.PaymentTypeTopUp {
		return errors.New("payment is not a top-up")
	}

	// Calculate TON amount based on currency
	var tonAmount float64
	switch payment.Currency {
	case "TON":
		tonAmount = payment.Amount

		// Set status to awaiting_tx - background worker will verify
		if payment.Status == model.PaymentStatusPending {
			if err := s.repo.UpdatePaymentStatus(ctx, paymentID, model.PaymentStatusAwaitingTx); err != nil {
				return err
			}
			fmt.Printf("[TON] Top-up %s set to awaiting_tx, worker will verify\n", paymentID)
		}

		// Try immediate verification
		expectedAmountNano := int64(payment.Amount * 1e9)
		txInfo, err := s.tonVerifier.VerifyTransaction(boc, expectedAmountNano, "")
		if err != nil {
			// Not found yet - worker will keep checking
			fmt.Printf("[TON] Top-up %s: transaction not confirmed yet, worker will retry\n", paymentID)
			return nil // Return success - payment is being processed
		}
		fmt.Printf("[TON] Top-up verified: hash=%s, amount=%d nanoTON\n", txInfo.Hash, txInfo.Amount)
		boc = txInfo.Hash
	case "XTR":
		// Convert Stars back to TON (1 TON = 100 Stars)
		// Stars verification is done via Telegram callback, no need to verify here
		tonAmount = payment.Amount / 100
	default:
		return errors.New("unsupported currency")
	}

	// Update external ID with transaction hash
	if boc != "" {
		if err := s.repo.UpdatePaymentExternalID(ctx, paymentID, boc); err != nil {
			return err
		}
	}

	// Credit balance
	_, err = s.balanceSvc.CreditTopUp(ctx, payment.UserID, tonAmount, paymentID)
	if err != nil {
		return fmt.Errorf("failed to credit balance: %w", err)
	}

	// Update payment status
	if err := s.repo.UpdatePaymentStatus(ctx, paymentID, model.PaymentStatusCompleted); err != nil {
		return err
	}

	return nil
}
