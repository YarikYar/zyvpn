package repository

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
)

var ErrSettingNotFound = errors.New("setting not found")

func (r *Repository) GetSetting(ctx context.Context, key string) (string, error) {
	var value string
	err := r.db.GetContext(ctx, &value, "SELECT value FROM settings WHERE key = $1", key)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrSettingNotFound
		}
		return "", err
	}
	return value, nil
}

func (r *Repository) SetSetting(ctx context.Context, key, value string) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO settings (key, value, updated_at) VALUES ($1, $2, NOW())
		ON CONFLICT (key) DO UPDATE SET value = $2, updated_at = NOW()
	`, key, value)
	return err
}

func (r *Repository) GetSettingFloat(ctx context.Context, key string) (float64, error) {
	value, err := r.GetSetting(ctx, key)
	if err != nil {
		return 0, err
	}
	return strconv.ParseFloat(value, 64)
}

func (r *Repository) GetAllSettings(ctx context.Context) (map[string]string, error) {
	rows, err := r.db.QueryxContext(ctx, "SELECT key, value FROM settings")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	settings := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, err
		}
		settings[key] = value
	}
	return settings, nil
}
