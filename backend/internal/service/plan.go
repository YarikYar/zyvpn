package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/zyvpn/backend/internal/model"
	"github.com/zyvpn/backend/internal/repository"
)

type PlanService struct {
	repo *repository.Repository
}

func NewPlanService(repo *repository.Repository) *PlanService {
	return &PlanService{repo: repo}
}

func (s *PlanService) GetPlan(ctx context.Context, id uuid.UUID) (*model.Plan, error) {
	plan, err := s.repo.GetPlan(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := s.repo.HydratePlanWithServers(ctx, plan); err != nil {
		return nil, err
	}
	return plan, nil
}

func (s *PlanService) GetActivePlans(ctx context.Context) ([]model.Plan, error) {
	plans, err := s.repo.GetActivePlans(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.repo.HydratePlansWithServers(ctx, plans); err != nil {
		return nil, err
	}
	return plans, nil
}

// GetActivePlansForUser returns plans visible to a specific user (public
// plans plus any plans whose visible_to_referrer_id matches the user's
// referrer). userID=0 falls back to the public list (unauth callers).
func (s *PlanService) GetActivePlansForUser(ctx context.Context, userID int64) ([]model.Plan, error) {
	var plans []model.Plan
	var err error
	if userID == 0 {
		plans, err = s.repo.GetActivePlans(ctx)
	} else {
		plans, err = s.repo.GetActivePlansForUser(ctx, userID)
	}
	if err != nil {
		return nil, err
	}
	if err := s.repo.HydratePlansWithServers(ctx, plans); err != nil {
		return nil, err
	}
	return plans, nil
}

func (s *PlanService) GetAllPlans(ctx context.Context) ([]model.Plan, error) {
	plans, err := s.repo.GetAllPlans(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.repo.HydratePlansWithServers(ctx, plans); err != nil {
		return nil, err
	}
	return plans, nil
}

func (s *PlanService) DeletePlan(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeletePlanByID(ctx, id)
}
