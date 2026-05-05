package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/zyvpn/backend/internal/model"
)

var ErrPlanNotFound = errors.New("plan not found")

func (r *Repository) GetPlan(ctx context.Context, id uuid.UUID) (*model.Plan, error) {
	var plan model.Plan
	err := r.db.GetContext(ctx, &plan, "SELECT * FROM plans WHERE id = $1", id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPlanNotFound
		}
		return nil, err
	}
	return &plan, nil
}

func (r *Repository) GetActivePlans(ctx context.Context) ([]model.Plan, error) {
	var plans []model.Plan
	// Public plans only (no per-referrer visibility) — used internally where
	// no specific viewer is known.
	query := `
		SELECT * FROM plans
		WHERE is_active = true
		  AND price_stars > 0
		  AND visible_to_referrer_id IS NULL
		ORDER BY sort_order ASC`
	err := r.db.SelectContext(ctx, &plans, query)
	return plans, err
}

// GetActivePlansForUser returns plans visible to the given user: all public
// plans plus any plans whose visible_to_referrer_id matches that user's
// referred_by. Use this for the user-facing /api/plans endpoint.
func (r *Repository) GetActivePlansForUser(ctx context.Context, userID int64) ([]model.Plan, error) {
	var plans []model.Plan
	query := `
		SELECT p.* FROM plans p
		LEFT JOIN users u ON u.id = $1
		WHERE p.is_active = true
		  AND p.price_stars > 0
		  AND (
		    p.visible_to_referrer_id IS NULL
		    OR p.visible_to_referrer_id = u.referred_by
		  )
		ORDER BY p.sort_order ASC`
	err := r.db.SelectContext(ctx, &plans, query, userID)
	return plans, err
}

func (r *Repository) GetAllPlans(ctx context.Context) ([]model.Plan, error) {
	var plans []model.Plan
	query := "SELECT * FROM plans ORDER BY sort_order ASC"
	err := r.db.SelectContext(ctx, &plans, query)
	return plans, err
}

func (r *Repository) DeletePlanByID(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM plans WHERE id = $1", id)
	return err
}

func (r *Repository) GetTrialPlan(ctx context.Context) (*model.Plan, error) {
	var plan model.Plan
	err := r.db.GetContext(ctx, &plan, "SELECT * FROM plans WHERE price_usd = 0 AND is_active = true LIMIT 1")
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPlanNotFound
		}
		return nil, err
	}
	return &plan, nil
}

// CreatePlan creates a new plan with parameters
func (r *Repository) CreatePlan(ctx context.Context, name, description string, durationDays, trafficGB, maxDevices int, priceTON float64, priceStars int, priceUSD float64, sortOrder int, visibleToReferrerID *int64) (*model.Plan, error) {
	var plan model.Plan
	query := `
		INSERT INTO plans (name, description, duration_days, traffic_gb, max_devices, price_ton, price_stars, price_usd, is_active, sort_order, visible_to_referrer_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, true, $9, $10)
		RETURNING *`

	err := r.db.QueryRowxContext(ctx, query, name, description, durationDays, trafficGB, maxDevices, priceTON, priceStars, priceUSD, sortOrder, visibleToReferrerID).StructScan(&plan)
	if err != nil {
		return nil, err
	}
	return &plan, nil
}

// UpdatePlan updates a plan with optional parameters. visibleToReferrerID
// is a tri-state: nil = leave as is, *value = set (or clear if value points
// to a sentinel — we use a dedicated `clearVisibility` flag instead, see
// callers for usage).
func (r *Repository) UpdatePlan(ctx context.Context, planID string, name, description *string, durationDays, trafficGB, maxDevices *int, priceTON *float64, priceStars *int, priceUSD *float64, isActive *bool, sortOrder *int, visibleToReferrerID *int64, clearVisibility bool) (*model.Plan, error) {
	id, err := uuid.Parse(planID)
	if err != nil {
		return nil, err
	}

	// Get current plan
	plan, err := r.GetPlan(ctx, id)
	if err != nil {
		return nil, err
	}

	// Apply updates
	if name != nil {
		plan.Name = *name
	}
	if description != nil {
		plan.Description = *description
	}
	if durationDays != nil {
		plan.DurationDays = *durationDays
	}
	if trafficGB != nil {
		plan.TrafficGB = *trafficGB
	}
	if maxDevices != nil {
		plan.MaxDevices = *maxDevices
	}
	if priceTON != nil {
		plan.PriceTON = *priceTON
	}
	if priceStars != nil {
		plan.PriceStars = *priceStars
	}
	if priceUSD != nil {
		plan.PriceUSD = *priceUSD
	}
	if isActive != nil {
		plan.IsActive = *isActive
	}
	if sortOrder != nil {
		plan.SortOrder = *sortOrder
	}
	if clearVisibility {
		plan.VisibleToReferrerID = nil
	} else if visibleToReferrerID != nil {
		plan.VisibleToReferrerID = visibleToReferrerID
	}

	query := `
		UPDATE plans SET
			name = $2,
			description = $3,
			duration_days = $4,
			traffic_gb = $5,
			max_devices = $6,
			price_ton = $7,
			price_stars = $8,
			price_usd = $9,
			is_active = $10,
			sort_order = $11,
			visible_to_referrer_id = $12
		WHERE id = $1
		RETURNING *`

	err = r.db.QueryRowxContext(ctx, query,
		plan.ID,
		plan.Name,
		plan.Description,
		plan.DurationDays,
		plan.TrafficGB,
		plan.MaxDevices,
		plan.PriceTON,
		plan.PriceStars,
		plan.PriceUSD,
		plan.IsActive,
		plan.SortOrder,
		plan.VisibleToReferrerID,
	).StructScan(plan)

	return plan, err
}
