package model

import (
	"time"

	"github.com/google/uuid"
)

type Plan struct {
	ID           uuid.UUID `json:"id" db:"id"`
	Name         string    `json:"name" db:"name"`
	Description  string    `json:"description" db:"description"`
	DurationDays int       `json:"duration_days" db:"duration_days"`
	TrafficGB    int       `json:"traffic_gb" db:"traffic_gb"`
	MaxDevices   int       `json:"max_devices" db:"max_devices"`
	PriceTON     float64   `json:"price_ton" db:"price_ton"`
	PriceStars   int       `json:"price_stars" db:"price_stars"`
	PriceUSD     float64   `json:"price_usd" db:"price_usd"`
	IsActive     bool      `json:"is_active" db:"is_active"`
	SortOrder    int       `json:"sort_order" db:"sort_order"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

// TrafficBytes returns traffic limit in bytes
func (p *Plan) TrafficBytes() int64 {
	return int64(p.TrafficGB) * 1024 * 1024 * 1024
}
