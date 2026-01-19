package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/zyvpn/backend/internal/model"
	"github.com/zyvpn/backend/internal/repository"
)

var (
	ErrNotAdmin          = errors.New("Пользователь не является администратором")
	ErrUserNotFound      = errors.New("Пользователь не найден")
	ErrAlreadyBanned     = errors.New("Пользователь уже заблокирован")
	ErrNotBanned         = errors.New("Пользователь не заблокирован")
	ErrInsufficientPerms = errors.New("Недостаточно прав")
)

type AdminService struct {
	repo            *repository.Repository
	balanceSvc      *BalanceService
	subscriptionSvc *SubscriptionService
	promoCodeSvc    *PromoCodeService
}

func NewAdminService(repo *repository.Repository) *AdminService {
	return &AdminService{repo: repo}
}

// SetBalanceService sets the balance service (to avoid circular deps)
func (s *AdminService) SetBalanceService(balanceSvc *BalanceService) {
	s.balanceSvc = balanceSvc
}

// SetSubscriptionService sets the subscription service (to avoid circular deps)
func (s *AdminService) SetSubscriptionService(subscriptionSvc *SubscriptionService) {
	s.subscriptionSvc = subscriptionSvc
}

// SetPromoCodeService sets the promo code service (to avoid circular deps)
func (s *AdminService) SetPromoCodeService(promoCodeSvc *PromoCodeService) {
	s.promoCodeSvc = promoCodeSvc
}

// IsAdmin checks if user is an admin
func (s *AdminService) IsAdmin(ctx context.Context, userID int64) (bool, error) {
	return s.repo.IsAdmin(ctx, userID)
}

// GetAdmin gets admin info
func (s *AdminService) GetAdmin(ctx context.Context, userID int64) (*model.Admin, error) {
	return s.repo.GetAdmin(ctx, userID)
}

// --- User Management ---

// ListUsers lists users with pagination
func (s *AdminService) ListUsers(ctx context.Context, adminID int64, limit, offset int, search string) ([]model.User, int, error) {
	if ok, _ := s.IsAdmin(ctx, adminID); !ok {
		return nil, 0, ErrNotAdmin
	}
	if limit <= 0 {
		limit = 50
	}
	return s.repo.ListUsers(ctx, limit, offset, search)
}

// GetUser gets user info
func (s *AdminService) GetUser(ctx context.Context, adminID, targetUserID int64) (*model.UserWithSubscription, error) {
	if ok, _ := s.IsAdmin(ctx, adminID); !ok {
		return nil, ErrNotAdmin
	}
	return s.repo.GetUserWithSubscription(ctx, targetUserID)
}

// --- Balance Management ---

// SetBalance sets user balance to a specific value
func (s *AdminService) SetBalance(ctx context.Context, adminID, targetUserID int64, balance float64) error {
	if ok, _ := s.IsAdmin(ctx, adminID); !ok {
		return ErrNotAdmin
	}

	user, err := s.repo.GetUser(ctx, targetUserID)
	if err != nil {
		return ErrUserNotFound
	}

	oldBalance := user.Balance
	if err := s.repo.SetUserBalance(ctx, targetUserID, balance); err != nil {
		return err
	}

	// Log action
	_ = s.repo.LogAdminAction(ctx, adminID, model.AdminActionSetBalance, &targetUserID, map[string]interface{}{
		"old_balance": oldBalance,
		"new_balance": balance,
	})

	return nil
}

// AddBalance adds amount to user balance
func (s *AdminService) AddBalance(ctx context.Context, adminID, targetUserID int64, amount float64) error {
	if ok, _ := s.IsAdmin(ctx, adminID); !ok {
		return ErrNotAdmin
	}

	if s.balanceSvc == nil {
		return errors.New("balance service not configured")
	}

	user, err := s.repo.GetUser(ctx, targetUserID)
	if err != nil {
		return ErrUserNotFound
	}

	oldBalance := user.Balance
	_, err = s.balanceSvc.CreditManual(ctx, targetUserID, amount, fmt.Sprintf("Admin credit by %d", adminID))
	if err != nil {
		return err
	}

	// Log action
	_ = s.repo.LogAdminAction(ctx, adminID, model.AdminActionAddBalance, &targetUserID, map[string]interface{}{
		"amount":      amount,
		"old_balance": oldBalance,
		"new_balance": oldBalance + amount,
	})

	return nil
}

// --- Subscription Management ---

// ExtendSubscription extends user subscription
func (s *AdminService) ExtendSubscription(ctx context.Context, adminID, targetUserID int64, days int) error {
	if ok, _ := s.IsAdmin(ctx, adminID); !ok {
		return ErrNotAdmin
	}

	if s.subscriptionSvc == nil {
		return errors.New("subscription service not configured")
	}

	sub, err := s.subscriptionSvc.GetActiveSubscription(ctx, targetUserID)
	if err != nil {
		return err
	}
	if sub == nil {
		return ErrNoActiveSubscription
	}

	if err := s.subscriptionSvc.ExtendSubscription(ctx, sub.ID, days); err != nil {
		return err
	}

	// Log action
	_ = s.repo.LogAdminAction(ctx, adminID, model.AdminActionExtendSub, &targetUserID, map[string]interface{}{
		"days":            days,
		"subscription_id": sub.ID,
	})

	return nil
}

// CancelSubscription cancels user subscription
func (s *AdminService) CancelSubscription(ctx context.Context, adminID, targetUserID int64) error {
	if ok, _ := s.IsAdmin(ctx, adminID); !ok {
		return ErrNotAdmin
	}

	if s.subscriptionSvc == nil {
		return errors.New("subscription service not configured")
	}

	sub, err := s.subscriptionSvc.GetActiveSubscription(ctx, targetUserID)
	if err != nil {
		return err
	}
	if sub == nil {
		return ErrNoActiveSubscription
	}

	if err := s.subscriptionSvc.CancelSubscription(ctx, sub.ID); err != nil {
		return err
	}

	// Log action
	_ = s.repo.LogAdminAction(ctx, adminID, model.AdminActionCancelSub, &targetUserID, map[string]interface{}{
		"subscription_id": sub.ID,
	})

	return nil
}

// --- Ban Management ---

// BanUser bans a user
func (s *AdminService) BanUser(ctx context.Context, adminID, targetUserID int64, reason string, expiresAt *time.Time) error {
	if ok, _ := s.IsAdmin(ctx, adminID); !ok {
		return ErrNotAdmin
	}

	// Check if already banned
	banned, err := s.repo.IsUserBanned(ctx, targetUserID)
	if err != nil {
		return err
	}
	if banned {
		return ErrAlreadyBanned
	}

	ban := &model.BannedUser{
		UserID:    &targetUserID,
		Reason:    &reason,
		BannedBy:  &adminID,
		ExpiresAt: expiresAt,
	}

	if err := s.repo.BanUser(ctx, ban); err != nil {
		return err
	}

	// Log action
	_ = s.repo.LogAdminAction(ctx, adminID, model.AdminActionBanUser, &targetUserID, map[string]interface{}{
		"reason":     reason,
		"expires_at": expiresAt,
	})

	return nil
}

// BanIP bans an IP address
func (s *AdminService) BanIP(ctx context.Context, adminID int64, ip string, reason string, expiresAt *time.Time) error {
	if ok, _ := s.IsAdmin(ctx, adminID); !ok {
		return ErrNotAdmin
	}

	ban := &model.BannedUser{
		IPAddress: &ip,
		Reason:    &reason,
		BannedBy:  &adminID,
		ExpiresAt: expiresAt,
	}

	if err := s.repo.BanUser(ctx, ban); err != nil {
		return err
	}

	// Log action
	_ = s.repo.LogAdminAction(ctx, adminID, model.AdminActionBanIP, nil, map[string]interface{}{
		"ip":         ip,
		"reason":     reason,
		"expires_at": expiresAt,
	})

	return nil
}

// UnbanUser unbans a user
func (s *AdminService) UnbanUser(ctx context.Context, adminID, targetUserID int64) error {
	if ok, _ := s.IsAdmin(ctx, adminID); !ok {
		return ErrNotAdmin
	}

	banned, err := s.repo.IsUserBanned(ctx, targetUserID)
	if err != nil {
		return err
	}
	if !banned {
		return ErrNotBanned
	}

	if err := s.repo.UnbanUser(ctx, targetUserID); err != nil {
		return err
	}

	// Log action
	_ = s.repo.LogAdminAction(ctx, adminID, model.AdminActionUnbanUser, &targetUserID, nil)

	return nil
}

// UnbanIP unbans an IP address
func (s *AdminService) UnbanIP(ctx context.Context, adminID int64, ip string) error {
	if ok, _ := s.IsAdmin(ctx, adminID); !ok {
		return ErrNotAdmin
	}

	if err := s.repo.UnbanIP(ctx, ip); err != nil {
		return err
	}

	// Log action
	_ = s.repo.LogAdminAction(ctx, adminID, model.AdminActionUnbanUser, nil, map[string]interface{}{
		"ip": ip,
	})

	return nil
}

// IsUserBanned checks if user is banned
func (s *AdminService) IsUserBanned(ctx context.Context, userID int64) (bool, error) {
	return s.repo.IsUserBanned(ctx, userID)
}

// IsIPBanned checks if IP is banned
func (s *AdminService) IsIPBanned(ctx context.Context, ip string) (bool, error) {
	return s.repo.IsIPBanned(ctx, ip)
}

// ListBannedUsers lists all active bans
func (s *AdminService) ListBannedUsers(ctx context.Context, adminID int64, limit, offset int) ([]model.BannedUser, error) {
	if ok, _ := s.IsAdmin(ctx, adminID); !ok {
		return nil, ErrNotAdmin
	}
	if limit <= 0 {
		limit = 50
	}
	return s.repo.ListBannedUsers(ctx, limit, offset)
}

// --- Promo Code Management ---

// GeneratePromoCode generates a random promo code
func (s *AdminService) GeneratePromoCode(ctx context.Context, adminID int64, promoType model.PromoCodeType, value float64, maxUses *int, expiresAt *time.Time, description string) (*model.PromoCode, error) {
	if ok, _ := s.IsAdmin(ctx, adminID); !ok {
		return nil, ErrNotAdmin
	}

	code := generateRandomCode(8)

	promo := &model.PromoCode{
		Code:        code,
		Type:        promoType,
		Value:       value,
		MaxUses:     maxUses,
		ExpiresAt:   expiresAt,
		IsActive:    true,
		Description: &description,
	}

	if err := s.repo.CreatePromoCode(ctx, promo); err != nil {
		return nil, err
	}

	// Log action
	_ = s.repo.LogAdminAction(ctx, adminID, model.AdminActionCreatePromoCode, nil, map[string]interface{}{
		"code":        code,
		"type":        promoType,
		"value":       value,
		"max_uses":    maxUses,
		"expires_at":  expiresAt,
		"description": description,
	})

	// Get full promo with ID
	return s.repo.GetPromoCodeByCode(ctx, code)
}

// GenerateBulkPromoCodes generates multiple promo codes
func (s *AdminService) GenerateBulkPromoCodes(ctx context.Context, adminID int64, count int, promoType model.PromoCodeType, value float64, maxUses *int, expiresAt *time.Time, prefix string) ([]string, error) {
	if ok, _ := s.IsAdmin(ctx, adminID); !ok {
		return nil, ErrNotAdmin
	}

	if count <= 0 || count > 100 {
		return nil, errors.New("count must be between 1 and 100")
	}

	var codes []string
	for i := 0; i < count; i++ {
		code := prefix + generateRandomCode(8)

		promo := &model.PromoCode{
			Code:     code,
			Type:     promoType,
			Value:    value,
			MaxUses:  maxUses,
			ExpiresAt: expiresAt,
			IsActive: true,
		}

		if err := s.repo.CreatePromoCode(ctx, promo); err != nil {
			continue // Skip duplicates
		}
		codes = append(codes, code)
	}

	// Log action
	_ = s.repo.LogAdminAction(ctx, adminID, model.AdminActionCreatePromoCode, nil, map[string]interface{}{
		"bulk_count": count,
		"type":       promoType,
		"value":      value,
		"prefix":     prefix,
	})

	return codes, nil
}

// ListPromoCodes lists all promo codes
func (s *AdminService) ListPromoCodes(ctx context.Context, adminID int64, limit, offset int) ([]model.PromoCode, error) {
	if ok, _ := s.IsAdmin(ctx, adminID); !ok {
		return nil, ErrNotAdmin
	}
	if s.promoCodeSvc == nil {
		return nil, errors.New("promo code service not configured")
	}
	return s.promoCodeSvc.ListPromoCodes(ctx, limit, offset)
}

// DeactivatePromoCode deactivates a promo code
func (s *AdminService) DeactivatePromoCode(ctx context.Context, adminID int64, code string) error {
	if ok, _ := s.IsAdmin(ctx, adminID); !ok {
		return ErrNotAdmin
	}
	if s.promoCodeSvc == nil {
		return errors.New("promo code service not configured")
	}

	if err := s.promoCodeSvc.DeactivatePromoCode(ctx, code); err != nil {
		return err
	}

	// Log action
	_ = s.repo.LogAdminAction(ctx, adminID, model.AdminActionDeactivatePromo, nil, map[string]interface{}{
		"code": code,
	})

	return nil
}

// --- Admin Logs ---

// GetAdminLogs retrieves admin action logs
func (s *AdminService) GetAdminLogs(ctx context.Context, adminID int64, limit, offset int) ([]model.AdminLog, error) {
	if ok, _ := s.IsAdmin(ctx, adminID); !ok {
		return nil, ErrNotAdmin
	}
	if limit <= 0 {
		limit = 50
	}
	return s.repo.GetAdminLogs(ctx, limit, offset)
}

// --- Stats ---

type AdminStats struct {
	TotalUsers         int `json:"total_users"`
	ActiveSubscriptions int `json:"active_subscriptions"`
	BannedUsers        int `json:"banned_users"`
	ActivePromoCodes   int `json:"active_promo_codes"`
}

// GetStats returns admin dashboard stats
func (s *AdminService) GetStats(ctx context.Context, adminID int64) (*AdminStats, error) {
	if ok, _ := s.IsAdmin(ctx, adminID); !ok {
		return nil, ErrNotAdmin
	}

	stats := &AdminStats{}

	// Count users
	_, total, _ := s.repo.ListUsers(ctx, 1, 0, "")
	stats.TotalUsers = total

	// Count active subscriptions
	var activeSubs int
	_ = s.repo.QueryRow(ctx, `SELECT COUNT(*) FROM subscriptions WHERE status = 'active'`).Scan(&activeSubs)
	stats.ActiveSubscriptions = activeSubs

	// Count banned users
	var bannedCount int
	_ = s.repo.QueryRow(ctx, `SELECT COUNT(*) FROM banned_users WHERE is_active = true`).Scan(&bannedCount)
	stats.BannedUsers = bannedCount

	// Count active promo codes
	var promoCount int
	_ = s.repo.QueryRow(ctx, `SELECT COUNT(*) FROM promo_codes WHERE is_active = true`).Scan(&promoCount)
	stats.ActivePromoCodes = promoCount

	return stats, nil
}

// generateRandomCode generates a random alphanumeric code
func generateRandomCode(length int) string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	var sb strings.Builder
	sb.Grow(length)

	for i := 0; i < length; i++ {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		sb.WriteByte(charset[n.Int64()])
	}

	return sb.String()
}

// --- Plan Management ---

// UpdatePlanParams holds parameters for updating a plan
type UpdatePlanParams struct {
	Name         *string
	Description  *string
	DurationDays *int
	TrafficGB    *int
	MaxDevices   *int
	PriceTON     *float64
	PriceStars   *int
	PriceUSD     *float64
	IsActive     *bool
	SortOrder    *int
}

// CreatePlanParams holds parameters for creating a plan
type CreatePlanParams struct {
	Name         string
	Description  string
	DurationDays int
	TrafficGB    int
	MaxDevices   int
	PriceTON     float64
	PriceStars   int
	PriceUSD     float64
	SortOrder    int
}

// ListAllPlans lists all plans including inactive
func (s *AdminService) ListAllPlans(ctx context.Context) ([]model.Plan, error) {
	return s.repo.GetAllPlans(ctx)
}

// UpdatePlan updates a plan
func (s *AdminService) UpdatePlan(ctx context.Context, adminID int64, planID string, params UpdatePlanParams) (*model.Plan, error) {
	if ok, _ := s.IsAdmin(ctx, adminID); !ok {
		return nil, ErrNotAdmin
	}

	plan, err := s.repo.UpdatePlan(ctx, planID, params.Name, params.Description, params.DurationDays, params.TrafficGB, params.MaxDevices, params.PriceTON, params.PriceStars, params.PriceUSD, params.IsActive, params.SortOrder)
	if err != nil {
		return nil, err
	}

	// Log action
	_ = s.repo.LogAdminAction(ctx, adminID, model.AdminActionUpdatePlan, nil, map[string]interface{}{
		"plan_id": planID,
		"params":  params,
	})

	return plan, nil
}

// CreatePlan creates a new plan
func (s *AdminService) CreatePlan(ctx context.Context, adminID int64, params CreatePlanParams) (*model.Plan, error) {
	if ok, _ := s.IsAdmin(ctx, adminID); !ok {
		return nil, ErrNotAdmin
	}

	plan, err := s.repo.CreatePlan(ctx, params.Name, params.Description, params.DurationDays, params.TrafficGB, params.MaxDevices, params.PriceTON, params.PriceStars, params.PriceUSD, params.SortOrder)
	if err != nil {
		return nil, err
	}

	// Log action
	_ = s.repo.LogAdminAction(ctx, adminID, model.AdminActionCreatePlan, nil, map[string]interface{}{
		"plan_id": plan.ID,
		"name":    params.Name,
	})

	return plan, nil
}

// DeletePlan soft-deletes a plan (sets is_active = false)
func (s *AdminService) DeletePlan(ctx context.Context, adminID int64, planID string) error {
	if ok, _ := s.IsAdmin(ctx, adminID); !ok {
		return ErrNotAdmin
	}

	isActive := false
	if _, err := s.repo.UpdatePlan(ctx, planID, nil, nil, nil, nil, nil, nil, nil, nil, &isActive, nil); err != nil {
		return err
	}

	// Log action
	_ = s.repo.LogAdminAction(ctx, adminID, model.AdminActionDeletePlan, nil, map[string]interface{}{
		"plan_id": planID,
	})

	return nil
}

// --- Settings Management ---

// GetSettings returns all settings
func (s *AdminService) GetSettings(ctx context.Context, adminID int64) (map[string]string, error) {
	if ok, _ := s.IsAdmin(ctx, adminID); !ok {
		return nil, ErrNotAdmin
	}
	return s.repo.GetAllSettings(ctx)
}

// SetSetting sets a setting value
func (s *AdminService) SetSetting(ctx context.Context, adminID int64, key, value string) error {
	if ok, _ := s.IsAdmin(ctx, adminID); !ok {
		return ErrNotAdmin
	}
	return s.repo.SetSetting(ctx, key, value)
}

// GetTopupBonusPercent returns current topup bonus percentage
func (s *AdminService) GetTopupBonusPercent(ctx context.Context) (float64, error) {
	value, err := s.repo.GetSettingFloat(ctx, "topup_bonus_percent")
	if err != nil {
		return 0, nil // Default to 0 if not set
	}
	return value, nil
}

// SetTopupBonusPercent sets topup bonus percentage (0-10)
func (s *AdminService) SetTopupBonusPercent(ctx context.Context, adminID int64, percent float64) error {
	if ok, _ := s.IsAdmin(ctx, adminID); !ok {
		return ErrNotAdmin
	}
	if percent < 0 || percent > 10 {
		return errors.New("процент бонуса должен быть от 0 до 10")
	}
	return s.repo.SetSetting(ctx, "topup_bonus_percent", fmt.Sprintf("%.1f", percent))
}

// GetReferralBonusTON returns current referral bonus in TON
func (s *AdminService) GetReferralBonusTON(ctx context.Context) (float64, error) {
	value, err := s.repo.GetSettingFloat(ctx, "referral_bonus_ton")
	if err != nil {
		return 0.1, nil // Default to 0.1 TON if not set
	}
	return value, nil
}

// SetReferralBonusTON sets referral bonus in TON (0-1)
func (s *AdminService) SetReferralBonusTON(ctx context.Context, adminID int64, amount float64) error {
	if ok, _ := s.IsAdmin(ctx, adminID); !ok {
		return ErrNotAdmin
	}
	if amount < 0 || amount > 1 {
		return errors.New("бонус должен быть от 0 до 1 TON")
	}
	return s.repo.SetSetting(ctx, "referral_bonus_ton", fmt.Sprintf("%.2f", amount))
}
