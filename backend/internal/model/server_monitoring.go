package model

import (
	"time"

	"github.com/google/uuid"
)

// ServerHealthEvent — single status-change record, see migration 000015.
type ServerHealthEvent struct {
	ID        int       `db:"id"`
	ServerID  uuid.UUID `db:"server_id"`
	Status    string    `db:"status"`
	StartedAt time.Time `db:"started_at"`
}

// ServerTrafficSnapshot — periodic snapshot of cumulative XUI inbound
// counters; deltas between snapshots give per-window traffic.
type ServerTrafficSnapshot struct {
	ID            int       `db:"id"`
	ServerID      uuid.UUID `db:"server_id"`
	TakenAt       time.Time `db:"taken_at"`
	UpBytes       int64     `db:"up_bytes"`
	DownBytes     int64     `db:"down_bytes"`
	AllTimeBytes  int64     `db:"all_time_bytes"`
}

// Incident describes a continuous offline period for a server. EndedAt is
// nil for an ongoing incident.
type Incident struct {
	ServerID    uuid.UUID  `json:"server_id"`
	ServerName  string     `json:"server_name"`
	Country     string     `json:"country"`
	FlagEmoji   string     `json:"flag_emoji"`
	StartedAt   time.Time  `json:"started_at"`
	EndedAt     *time.Time `json:"ended_at,omitempty"`
	DurationSec int64      `json:"duration_seconds"`
}
