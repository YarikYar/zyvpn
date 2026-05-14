package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/zyvpn/backend/internal/model"
)

// SetPlanServers заменяет набор серверов плана целиком. Транзакция:
// DELETE + INSERT batch — никаких смешанных состояний.
func (r *Repository) SetPlanServers(ctx context.Context, planID uuid.UUID, serverIDs []uuid.UUID) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx,
		"DELETE FROM plan_servers WHERE plan_id = $1", planID); err != nil {
		return err
	}

	if len(serverIDs) == 0 {
		return tx.Commit()
	}

	type pair struct {
		PlanID   uuid.UUID `db:"plan_id"`
		ServerID uuid.UUID `db:"server_id"`
	}
	pairs := make([]pair, len(serverIDs))
	for i, sid := range serverIDs {
		pairs[i] = pair{PlanID: planID, ServerID: sid}
	}
	if _, err := tx.NamedExecContext(ctx,
		"INSERT INTO plan_servers (plan_id, server_id) VALUES (:plan_id, :server_id)",
		pairs); err != nil {
		return err
	}
	return tx.Commit()
}

// GetServersForPlan возвращает список серверов одного плана.
func (r *Repository) GetServersForPlan(ctx context.Context, planID uuid.UUID) ([]model.Server, error) {
	var servers []model.Server
	err := r.db.SelectContext(ctx, &servers, `
		SELECT s.* FROM servers s
		JOIN plan_servers ps ON ps.server_id = s.id
		WHERE ps.plan_id = $1
		ORDER BY s.sort_order, s.name`, planID)
	return servers, err
}

// HydratePlansWithServers — batched fill .Servers без N+1: один SELECT
// IN (...) по всем plan_id, потом раскладываем по планам в Go.
func (r *Repository) HydratePlansWithServers(ctx context.Context, plans []model.Plan) error {
	if len(plans) == 0 {
		return nil
	}
	ids := make([]uuid.UUID, len(plans))
	for i := range plans {
		ids[i] = plans[i].ID
	}

	type row struct {
		PlanID uuid.UUID `db:"plan_id"`
		model.Server
	}
	q, args, err := sqlx.In(`
		SELECT ps.plan_id, s.*
		FROM servers s
		JOIN plan_servers ps ON ps.server_id = s.id
		WHERE ps.plan_id IN (?)
		ORDER BY s.sort_order, s.name`, ids)
	if err != nil {
		return err
	}
	q = r.db.Rebind(q)

	var rows []row
	if err := r.db.SelectContext(ctx, &rows, q, args...); err != nil {
		return err
	}

	byPlan := make(map[uuid.UUID][]model.Server, len(plans))
	for _, r := range rows {
		byPlan[r.PlanID] = append(byPlan[r.PlanID], r.Server)
	}
	for i := range plans {
		plans[i].Servers = byPlan[plans[i].ID]
	}
	return nil
}

// HydratePlanWithServers — single-plan flavor.
func (r *Repository) HydratePlanWithServers(ctx context.Context, plan *model.Plan) error {
	servers, err := r.GetServersForPlan(ctx, plan.ID)
	if err != nil {
		return err
	}
	plan.Servers = servers
	return nil
}
