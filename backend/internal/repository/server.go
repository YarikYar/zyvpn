package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/zyvpn/backend/internal/model"
)

var ErrServerNotFound = errors.New("server not found")

// GetServer returns a server by ID
func (r *Repository) GetServer(ctx context.Context, id uuid.UUID) (*model.Server, error) {
	var server model.Server
	err := r.db.GetContext(ctx, &server, `
		SELECT * FROM servers WHERE id = $1
	`, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrServerNotFound
		}
		return nil, err
	}
	return &server, nil
}

// GetActiveServers returns all active servers ordered by sort_order
func (r *Repository) GetActiveServers(ctx context.Context) ([]model.Server, error) {
	var servers []model.Server
	err := r.db.SelectContext(ctx, &servers, `
		SELECT * FROM servers
		WHERE is_active = true
		ORDER BY sort_order, name
	`)
	if err != nil {
		return nil, err
	}
	return servers, nil
}

// GetAllServers returns all servers (for admin)
func (r *Repository) GetAllServers(ctx context.Context) ([]model.Server, error) {
	var servers []model.Server
	err := r.db.SelectContext(ctx, &servers, `
		SELECT * FROM servers
		ORDER BY sort_order, name
	`)
	if err != nil {
		return nil, err
	}
	return servers, nil
}

// CreateServer creates a new server
func (r *Repository) CreateServer(ctx context.Context, server *model.Server) error {
	server.ID = uuid.New()
	_, err := r.db.NamedExecContext(ctx, `
		INSERT INTO servers (id, name, country, city, flag_emoji,
			xui_base_url, xui_username, xui_password, xui_inbound_id,
			server_address, server_port, public_key, short_id, server_name,
			is_active, sort_order)
		VALUES (:id, :name, :country, :city, :flag_emoji,
			:xui_base_url, :xui_username, :xui_password, :xui_inbound_id,
			:server_address, :server_port, :public_key, :short_id, :server_name,
			:is_active, :sort_order)
	`, server)
	return err
}

// UpdateServer updates a server
func (r *Repository) UpdateServer(ctx context.Context, server *model.Server) error {
	result, err := r.db.NamedExecContext(ctx, `
		UPDATE servers SET
			name = :name,
			country = :country,
			city = :city,
			flag_emoji = :flag_emoji,
			xui_base_url = :xui_base_url,
			xui_username = :xui_username,
			xui_password = :xui_password,
			xui_inbound_id = :xui_inbound_id,
			server_address = :server_address,
			server_port = :server_port,
			public_key = :public_key,
			short_id = :short_id,
			server_name = :server_name,
			is_active = :is_active,
			sort_order = :sort_order,
			updated_at = NOW()
		WHERE id = :id
	`, server)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrServerNotFound
	}
	return nil
}

// DeleteServer deletes a server
func (r *Repository) DeleteServer(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM servers WHERE id = $1`, id)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrServerNotFound
	}
	return nil
}

// GetDefaultServer returns the first active server (used when no server is specified)
func (r *Repository) GetDefaultServer(ctx context.Context) (*model.Server, error) {
	var server model.Server
	err := r.db.GetContext(ctx, &server, `
		SELECT * FROM servers
		WHERE is_active = true
		ORDER BY sort_order, name
		LIMIT 1
	`)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrServerNotFound
		}
		return nil, err
	}
	return &server, nil
}

// GetBestServer returns the best available server based on load and capacity
// Prioritizes: online status, lower load percentage, higher capacity
func (r *Repository) GetBestServer(ctx context.Context) (*model.Server, error) {
	var server model.Server
	err := r.db.GetContext(ctx, &server, `
		SELECT * FROM servers
		WHERE is_active = true AND status = 'online'
		ORDER BY
			(CAST(current_load AS FLOAT) / NULLIF(capacity, 0)) ASC,
			capacity DESC,
			sort_order ASC
		LIMIT 1
	`)
	if err != nil {
		if err == sql.ErrNoRows {
			// Fall back to any active server
			return r.GetDefaultServer(ctx)
		}
		return nil, err
	}
	return &server, nil
}

// GetOnlineServers returns all online active servers
func (r *Repository) GetOnlineServers(ctx context.Context) ([]model.Server, error) {
	var servers []model.Server
	err := r.db.SelectContext(ctx, &servers, `
		SELECT * FROM servers
		WHERE is_active = true AND status = 'online'
		ORDER BY sort_order, name
	`)
	if err != nil {
		return nil, err
	}
	return servers, nil
}

// UpdateServerHealth updates server ping and status
func (r *Repository) UpdateServerHealth(ctx context.Context, id uuid.UUID, pingMs *int, status string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE servers
		SET ping_ms = $2, status = $3, last_check_at = NOW()
		WHERE id = $1
	`, id, pingMs, status)
	return err
}

// UpdateServerLoad updates server current load
func (r *Repository) UpdateServerLoad(ctx context.Context, id uuid.UUID, currentLoad int) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE servers SET current_load = $2 WHERE id = $1
	`, id, currentLoad)
	return err
}

// IncrementServerLoad increments server current load by 1
func (r *Repository) IncrementServerLoad(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE servers SET current_load = current_load + 1 WHERE id = $1
	`, id)
	return err
}

// DecrementServerLoad decrements server current load by 1
func (r *Repository) DecrementServerLoad(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE servers SET current_load = GREATEST(current_load - 1, 0) WHERE id = $1
	`, id)
	return err
}

// CountActiveSubscriptionsByServer counts active subscriptions for a server
func (r *Repository) CountActiveSubscriptionsByServer(ctx context.Context, serverID uuid.UUID) (int, error) {
	var count int
	err := r.db.GetContext(ctx, &count, `
		SELECT COUNT(*) FROM subscriptions
		WHERE server_id = $1 AND status = 'active'
	`, serverID)
	return count, err
}

// SyncAllServerLoads updates current_load for all servers based on actual subscriptions
func (r *Repository) SyncAllServerLoads(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE servers s
		SET current_load = COALESCE((
			SELECT COUNT(*) FROM subscriptions sub
			WHERE sub.server_id = s.id AND sub.status = 'active'
		), 0)
	`)
	return err
}
