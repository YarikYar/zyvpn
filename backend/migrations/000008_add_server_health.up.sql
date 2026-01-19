-- Server health and capacity fields
ALTER TABLE servers ADD COLUMN IF NOT EXISTS capacity INT NOT NULL DEFAULT 100;
ALTER TABLE servers ADD COLUMN IF NOT EXISTS current_load INT NOT NULL DEFAULT 0;
ALTER TABLE servers ADD COLUMN IF NOT EXISTS ping_ms INT;
ALTER TABLE servers ADD COLUMN IF NOT EXISTS status VARCHAR(20) NOT NULL DEFAULT 'unknown';
ALTER TABLE servers ADD COLUMN IF NOT EXISTS last_check_at TIMESTAMP;

-- Index for finding available servers
CREATE INDEX IF NOT EXISTS idx_servers_status_load ON servers(status, is_active, current_load);

COMMENT ON COLUMN servers.capacity IS 'Max number of clients this server can handle';
COMMENT ON COLUMN servers.current_load IS 'Current number of active clients';
COMMENT ON COLUMN servers.ping_ms IS 'Last measured ping in milliseconds';
COMMENT ON COLUMN servers.status IS 'online, offline, or unknown';
