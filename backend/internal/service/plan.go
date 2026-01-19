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
	return s.repo.GetPlan(ctx, id)
}

func (s *PlanService) GetActivePlans(ctx context.Context) ([]model.Plan, error) {
	return s.repo.GetActivePlans(ctx)
}

func (s *PlanService) GetAllPlans(ctx context.Context) ([]model.Plan, error) {
	return s.repo.GetAllPlans(ctx)
}

func (s *PlanService) DeletePlan(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeletePlanByID(ctx, id)
}
