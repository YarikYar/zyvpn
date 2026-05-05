package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/zyvpn/backend/internal/model"
)

// GetAppRelease returns the cached row for (app_key, tag) or nil if absent.
func (r *Repository) GetAppRelease(ctx context.Context, appKey, tag string) (*model.AppRelease, error) {
	var rel model.AppRelease
	err := r.db.GetContext(ctx, &rel, `
		SELECT * FROM app_releases WHERE app_key = $1 AND tag = $2`, appKey, tag)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &rel, nil
}

// UpsertAppRelease inserts or updates an app_releases row. Used both when
// caching a freshly uploaded file_id and when the monitor worker discovers
// a new tag.
func (r *Repository) UpsertAppRelease(ctx context.Context, rel *model.AppRelease) error {
	query := `
		INSERT INTO app_releases (app_key, tag, asset_name, asset_url, file_size, telegram_file_id)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (app_key, tag) DO UPDATE
		SET asset_name = EXCLUDED.asset_name,
		    asset_url = EXCLUDED.asset_url,
		    file_size = EXCLUDED.file_size,
		    telegram_file_id = COALESCE(EXCLUDED.telegram_file_id, app_releases.telegram_file_id),
		    updated_at = NOW()`
	_, err := r.db.ExecContext(ctx, query,
		rel.AppKey, rel.Tag, rel.AssetName, rel.AssetURL, rel.FileSize, rel.TelegramFileID)
	return err
}

// SetAppReleaseFileID writes only the cached file_id without touching the
// asset metadata. Called right after a successful sendDocument upload.
func (r *Repository) SetAppReleaseFileID(ctx context.Context, appKey, tag, fileID string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE app_releases SET telegram_file_id = $3, updated_at = NOW()
		WHERE app_key = $1 AND tag = $2`, appKey, tag, fileID)
	return err
}
