package model

import "time"

// AppRelease caches a single GitHub release asset that we redistribute as
// a Telegram document. See migration 000014.
type AppRelease struct {
	ID             int       `json:"id" db:"id"`
	AppKey         string    `json:"app_key" db:"app_key"`
	Tag            string    `json:"tag" db:"tag"`
	AssetName      string    `json:"asset_name" db:"asset_name"`
	AssetURL       string    `json:"asset_url" db:"asset_url"`
	FileSize       *int64    `json:"file_size,omitempty" db:"file_size"`
	TelegramFileID *string   `json:"telegram_file_id,omitempty" db:"telegram_file_id"`
	UpdatedAt      time.Time `json:"updated_at" db:"updated_at"`
}
