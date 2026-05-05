-- Cache for executable files of recommended client apps.
--
-- Workflow: bot resolves the latest GitHub release for an app_key, finds
-- the matching asset, and uploads it to Telegram once. The returned
-- file_id is stored here so subsequent users get the file instantly via
-- file_id reference (no re-download from GitHub, no re-upload to TG).
--
-- A new release on GitHub will create a fresh row (different tag); the
-- monitoring worker (Phase 2) updates `tag` and clears `telegram_file_id`
-- to force a one-time re-upload.

CREATE TABLE app_releases (
    id              SERIAL PRIMARY KEY,
    app_key         VARCHAR(50) NOT NULL,
    tag             VARCHAR(100) NOT NULL,
    asset_name      TEXT NOT NULL,
    asset_url       TEXT NOT NULL,
    file_size       BIGINT,
    telegram_file_id TEXT,
    updated_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE (app_key, tag)
);

CREATE INDEX idx_app_releases_app_key ON app_releases(app_key);

COMMENT ON COLUMN app_releases.app_key IS 'Stable identifier (e.g. v2rayng_arm64, throne_win, throne_mac, throne_linux)';
COMMENT ON COLUMN app_releases.telegram_file_id IS 'Cached after first sendDocument; reuse for subsequent users';
