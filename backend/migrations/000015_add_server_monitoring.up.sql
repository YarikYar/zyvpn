-- Server uptime + traffic monitoring.
--
-- server_health_events records every status change from the HealthWorker.
-- Each row marks the START of a continuous interval with `status`. The
-- interval ends when the next event for the same server arrives (or stays
-- open if it's the latest). Used for both uptime computation and incident
-- listing (incidents = offline intervals).
--
-- server_traffic_snapshots periodically captures the inbound's cumulative
-- up/down/all-time counters from 3x-ui. Traffic over a window is computed
-- as (snapshot at end) - (snapshot at start) to avoid storing per-second
-- deltas.

CREATE TABLE server_health_events (
    id          SERIAL                   PRIMARY KEY,
    server_id   UUID                     NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    status      VARCHAR(20)              NOT NULL,
    started_at  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_server_health_events_server_started
    ON server_health_events (server_id, started_at DESC);

CREATE TABLE server_traffic_snapshots (
    id              SERIAL                   PRIMARY KEY,
    server_id       UUID                     NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    taken_at        TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    up_bytes        BIGINT                   NOT NULL,
    down_bytes      BIGINT                   NOT NULL,
    all_time_bytes  BIGINT                   NOT NULL
);

CREATE INDEX idx_server_traffic_snapshots_server_taken
    ON server_traffic_snapshots (server_id, taken_at DESC);
