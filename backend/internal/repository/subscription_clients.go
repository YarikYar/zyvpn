package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"

	"github.com/zyvpn/backend/internal/model"
)

// CreateSubscriptionClient вставляет per-server клиента подписки. Должен
// вызываться после успешного создания клиента в XUI-панели.
func (r *Repository) CreateSubscriptionClient(ctx context.Context, c *model.SubscriptionClient) error {
	query := `
		INSERT INTO subscription_clients (
			subscription_id, server_id, xui_client_id, xui_email, connection_key,
			traffic_used, enabled
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at`
	return r.db.QueryRowContext(ctx, query,
		c.SubscriptionID, c.ServerID, c.XUIClientID, c.XUIEmail, c.ConnectionKey,
		c.TrafficUsed, c.Enabled,
	).Scan(&c.ID, &c.CreatedAt)
}

// GetSubscriptionClients возвращает все клиенты подписки. С JOIN'нутыми
// данными сервера для отрисовки (ServerEntry) — порядок по sort_order сервера.
func (r *Repository) GetSubscriptionClients(ctx context.Context, subscriptionID uuid.UUID) ([]model.SubscriptionClient, error) {
	type row struct {
		model.SubscriptionClient
		ServerName       string `db:"server_name_col"`
		ServerCountry    string `db:"server_country_col"`
		ServerCity       string `db:"server_city_col"`
		ServerFlag       string `db:"server_flag_col"`
		ServerStatus     string `db:"server_status_col"`
		ServerPingMs     *int   `db:"server_ping_col"`
		ServerSortOrder  int    `db:"server_sort_col"`
		ServerIsActive   bool   `db:"server_is_active_col"`
	}
	query := `
		SELECT
			sc.id, sc.subscription_id, sc.server_id, sc.xui_client_id, sc.xui_email,
			sc.connection_key, sc.traffic_used, sc.enabled, sc.created_at,
			s.name        AS server_name_col,
			s.country     AS server_country_col,
			COALESCE(s.city, '')       AS server_city_col,
			COALESCE(s.flag_emoji, '') AS server_flag_col,
			COALESCE(s.status, 'unknown') AS server_status_col,
			s.ping_ms     AS server_ping_col,
			s.sort_order  AS server_sort_col,
			s.is_active   AS server_is_active_col
		FROM subscription_clients sc
		JOIN servers s ON s.id = sc.server_id
		WHERE sc.subscription_id = $1
		ORDER BY s.sort_order, s.name`
	var rows []row
	if err := r.db.SelectContext(ctx, &rows, query, subscriptionID); err != nil {
		return nil, err
	}
	out := make([]model.SubscriptionClient, 0, len(rows))
	for _, r := range rows {
		c := r.SubscriptionClient
		city := r.ServerCity
		c.Server = &model.Server{
			ID:        c.ServerID,
			Name:      r.ServerName,
			Country:   r.ServerCountry,
			City:      &city,
			FlagEmoji: r.ServerFlag,
			Status:    r.ServerStatus,
			PingMs:    r.ServerPingMs,
			SortOrder: r.ServerSortOrder,
			IsActive:  r.ServerIsActive,
		}
		out = append(out, c)
	}
	return out, nil
}

// GetSubscriptionClient — один клиент подписки на конкретном сервере.
func (r *Repository) GetSubscriptionClient(ctx context.Context, subscriptionID, serverID uuid.UUID) (*model.SubscriptionClient, error) {
	var c model.SubscriptionClient
	err := r.db.GetContext(ctx, &c, `
		SELECT * FROM subscription_clients
		WHERE subscription_id = $1 AND server_id = $2`, subscriptionID, serverID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSubscriptionNotFound
		}
		return nil, err
	}
	return &c, nil
}

// UpdateSubscriptionClientTraffic — для HealthWorker bulk-sync.
func (r *Repository) UpdateSubscriptionClientTraffic(ctx context.Context, id uuid.UUID, trafficUsed int64) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE subscription_clients SET traffic_used = $2 WHERE id = $1",
		id, trafficUsed)
	return err
}

// UpdateSubscriptionClientTrafficByEmail используется HealthWorker'ом —
// нам прилетает email из xui clientStats, а не client_id.
func (r *Repository) UpdateSubscriptionClientTrafficByEmail(ctx context.Context, serverID uuid.UUID, email string, trafficUsed int64) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE subscription_clients
		SET traffic_used = $3
		WHERE server_id = $1 AND xui_email = $2`,
		serverID, email, trafficUsed)
	return err
}

// GetSubscriptionClientByServerEmail — мап email→client_id для HealthWorker
// bulk-sync. Возвращает ErrSubscriptionNotFound если клиента у нас нет.
func (r *Repository) GetSubscriptionClientByServerEmail(ctx context.Context, serverID uuid.UUID, email string) (*model.SubscriptionClient, error) {
	var c model.SubscriptionClient
	err := r.db.GetContext(ctx, &c, `
		SELECT * FROM subscription_clients
		WHERE server_id = $1 AND xui_email = $2`, serverID, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSubscriptionNotFound
		}
		return nil, err
	}
	return &c, nil
}

// RecomputeSubscriptionTrafficUsed — пересчёт subscriptions.traffic_used как
// SUM(subscription_clients.traffic_used) по этой подписке. После HealthWorker
// bulk-sync клиентов даёт актуальный агрегат для enforcement и для отдачи
// фронту.
func (r *Repository) RecomputeSubscriptionTrafficUsed(ctx context.Context, subscriptionID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE subscriptions
		SET traffic_used = COALESCE((
			SELECT SUM(traffic_used)
			FROM subscription_clients
			WHERE subscription_id = $1
		), 0)
		WHERE id = $1`, subscriptionID)
	return err
}

// SetSubscriptionClientEnabled — для централизованного enforcement лимита:
// когда суммарный трафик подписки превысил traffic_limit, disable'им
// клиента и параллельно дёргаем xui API enable=false.
func (r *Repository) SetSubscriptionClientEnabled(ctx context.Context, id uuid.UUID, enabled bool) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE subscription_clients SET enabled = $2 WHERE id = $1",
		id, enabled)
	return err
}

// DeleteSubscriptionClient удаляет запись из БД (после успешного DeleteClient
// на xui-панели).
func (r *Repository) DeleteSubscriptionClient(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		"DELETE FROM subscription_clients WHERE id = $1", id)
	return err
}
