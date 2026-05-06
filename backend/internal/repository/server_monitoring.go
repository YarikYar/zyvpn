package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/zyvpn/backend/internal/model"
)

// RecordHealthEvent inserts a status row only if the latest event for this
// server has a different status. This keeps the table compact: one row per
// status change, not one per probe.
func (r *Repository) RecordHealthEvent(ctx context.Context, serverID uuid.UUID, status string) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO server_health_events (server_id, status)
		SELECT $1, $2
		WHERE NOT EXISTS (
			SELECT 1 FROM server_health_events
			WHERE server_id = $1
			ORDER BY started_at DESC
			LIMIT 1
		) OR (
			SELECT status FROM server_health_events
			WHERE server_id = $1
			ORDER BY started_at DESC
			LIMIT 1
		) <> $2`, serverID, status)
	return err
}

// GetCurrentStatusSince returns when the current status started (timestamp
// of the latest status-change event). Nil if no events recorded yet.
func (r *Repository) GetCurrentStatusSince(ctx context.Context, serverID uuid.UUID) (*time.Time, error) {
	var t time.Time
	err := r.db.GetContext(ctx, &t, `
		SELECT started_at FROM server_health_events
		WHERE server_id = $1
		ORDER BY started_at DESC LIMIT 1`, serverID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &t, nil
}

// ComputeUptime returns the fraction (0..1) of online time between `since`
// and now, based on health events. Returns nil if there's not enough data.
func (r *Repository) ComputeUptime(ctx context.Context, serverID uuid.UUID, since time.Time) (*float64, error) {
	type row struct {
		Status    string    `db:"status"`
		StartedAt time.Time `db:"started_at"`
	}
	var rows []row
	// Get the last event before `since` (to know status at window start)
	// plus all events in the window.
	err := r.db.SelectContext(ctx, &rows, `
		(SELECT status, $2::timestamptz AS started_at
		 FROM server_health_events
		 WHERE server_id = $1 AND started_at < $2
		 ORDER BY started_at DESC LIMIT 1)
		UNION ALL
		(SELECT status, started_at
		 FROM server_health_events
		 WHERE server_id = $1 AND started_at >= $2
		 ORDER BY started_at)
		ORDER BY started_at`, serverID, since)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}

	now := time.Now()
	totalSeconds := now.Sub(since).Seconds()
	if totalSeconds <= 0 {
		return nil, nil
	}

	var onlineSeconds float64
	for i, ev := range rows {
		var end time.Time
		if i+1 < len(rows) {
			end = rows[i+1].StartedAt
		} else {
			end = now
		}
		if ev.Status == "online" {
			onlineSeconds += end.Sub(ev.StartedAt).Seconds()
		}
	}
	ratio := onlineSeconds / totalSeconds
	if ratio < 0 {
		ratio = 0
	} else if ratio > 1 {
		ratio = 1
	}
	return &ratio, nil
}

// RecordTrafficSnapshot stores a snapshot of cumulative XUI inbound counters.
func (r *Repository) RecordTrafficSnapshot(ctx context.Context, snap model.ServerTrafficSnapshot) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO server_traffic_snapshots (server_id, up_bytes, down_bytes, all_time_bytes)
		VALUES ($1, $2, $3, $4)`,
		snap.ServerID, snap.UpBytes, snap.DownBytes, snap.AllTimeBytes)
	return err
}

// GetLatestTrafficSnapshot returns the freshest snapshot or nil.
func (r *Repository) GetLatestTrafficSnapshot(ctx context.Context, serverID uuid.UUID) (*model.ServerTrafficSnapshot, error) {
	var snap model.ServerTrafficSnapshot
	err := r.db.GetContext(ctx, &snap, `
		SELECT * FROM server_traffic_snapshots
		WHERE server_id = $1
		ORDER BY taken_at DESC LIMIT 1`, serverID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &snap, nil
}

// ComputeTrafficSince returns (up,down) bytes consumed between `since` and
// now using the snapshot delta. Returns nil if not enough snapshots in
// window. Naive: if the inbound counters were reset between snapshots, the
// delta would go negative — we floor at 0.
func (r *Repository) ComputeTrafficSince(ctx context.Context, serverID uuid.UUID, since time.Time) (up *int64, down *int64, err error) {
	type row struct {
		UpBytes   int64 `db:"up_bytes"`
		DownBytes int64 `db:"down_bytes"`
	}
	var first row
	err = r.db.GetContext(ctx, &first, `
		SELECT up_bytes, down_bytes FROM server_traffic_snapshots
		WHERE server_id = $1 AND taken_at <= $2
		ORDER BY taken_at DESC LIMIT 1`, serverID, since)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil, nil
		}
		return nil, nil, err
	}
	var last row
	err = r.db.GetContext(ctx, &last, `
		SELECT up_bytes, down_bytes FROM server_traffic_snapshots
		WHERE server_id = $1
		ORDER BY taken_at DESC LIMIT 1`, serverID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil, nil
		}
		return nil, nil, err
	}

	upDelta := last.UpBytes - first.UpBytes
	if upDelta < 0 {
		upDelta = 0
	}
	downDelta := last.DownBytes - first.DownBytes
	if downDelta < 0 {
		downDelta = 0
	}
	return &upDelta, &downDelta, nil
}

// ListIncidents returns recent offline intervals across all (or one) server.
// Pairs each offline event with the next event for that server; if it's the
// most recent, ended_at is NULL (incident still ongoing).
func (r *Repository) ListIncidents(ctx context.Context, since time.Time, limit int) ([]model.Incident, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	var incidents []model.Incident
	err := r.db.SelectContext(ctx, &incidents, `
		WITH paired AS (
			SELECT
				e.server_id,
				e.started_at,
				LEAD(e.started_at) OVER (PARTITION BY e.server_id ORDER BY e.started_at) AS ended_at,
				e.status
			FROM server_health_events e
			WHERE e.started_at >= $1
		)
		SELECT
			p.server_id,
			s.name AS server_name,
			s.country,
			s.flag_emoji,
			p.started_at,
			p.ended_at,
			EXTRACT(EPOCH FROM COALESCE(p.ended_at, NOW()) - p.started_at)::bigint AS duration_sec
		FROM paired p
		JOIN servers s ON s.id = p.server_id
		WHERE p.status = 'offline'
		ORDER BY p.started_at DESC
		LIMIT $2`, since, limit)
	return incidents, err
}
