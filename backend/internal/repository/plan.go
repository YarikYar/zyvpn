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
	// Exclude trial/free plans (price_stars = 0) from purchasable list
	query := "SELECT * FROM plans WHERE is_active = true AND price_stars > 0 ORDER BY sort_order ASC"
	err := r.db.SelectContext(ctx, &plans, query)
	return plans, err
}

func (r *Repository) GetAllPlans(ctx context.Context) ([]model.Plan, error) {
	var plans []model.Plan
	query := "SELECT * FROM plans ORDER BY sort_order ASC"
	err := r.db.SelectContext(ctx, &plans, query)
	return plans, err
}

func (r *Repository) CreatePlan(ctx context.Context, plan *model.Plan) error {
	query := `
		INSERT INTO plans (name, description, duration_days, traffic_gb, price_ton, price_stars, price_usd, is_active, sort_order)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at`

	return r.db.QueryRowContext(ctx, query,
		plan.Name,
		plan.Description,
		plan.DurationDays,
		plan.TrafficGB,
		plan.PriceTON,
		plan.PriceStars,
		plan.PriceUSD,
		plan.IsActive,
		plan.SortOrder,
	).Scan(&plan.ID, &plan.CreatedAt)
}

func (r *Repository) UpdatePlan(ctx context.Context, plan *model.Plan) error {
	query := `
		UPDATE plans SET
			name = $2,
			description = $3,
			duration_days = $4,
			traffic_gb = $5,
			price_ton = $6,
			price_stars = $7,
			price_usd = $8,
			is_active = $9,
			sort_order = $10
		WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query,
		plan.ID,
		plan.Name,
		plan.Description,
		plan.DurationDays,
		plan.TrafficGB,
		plan.PriceTON,
		plan.PriceStars,
		plan.PriceUSD,
		plan.IsActive,
		plan.SortOrder,
	)
	return err
}

func (r *Repository) DeletePlan(ctx context.Context, id uuid.UUID) error {
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
