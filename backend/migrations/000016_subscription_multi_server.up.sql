-- Multi-server subscriptions: один тариф включает N серверов, доступ юзер
-- получает через subscription URL. Платная смена региона удаляется как
-- концепция.

-- m:n plan ↔ servers
CREATE TABLE plan_servers (
    plan_id   UUID NOT NULL REFERENCES plans(id)   ON DELETE CASCADE,
    server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    PRIMARY KEY (plan_id, server_id)
);

CREATE INDEX idx_plan_servers_plan   ON plan_servers(plan_id);
CREATE INDEX idx_plan_servers_server ON plan_servers(server_id);

-- per-server xui-клиенты внутри одной подписки (1:N)
CREATE TABLE subscription_clients (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    subscription_id UUID NOT NULL REFERENCES subscriptions(id) ON DELETE CASCADE,
    server_id       UUID NOT NULL REFERENCES servers(id)       ON DELETE CASCADE,
    xui_client_id   TEXT NOT NULL,
    xui_email       TEXT NOT NULL,
    connection_key  TEXT NOT NULL,
    traffic_used    BIGINT NOT NULL DEFAULT 0,
    enabled         BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (subscription_id, server_id)
);

CREATE INDEX idx_subscription_clients_sub    ON subscription_clients(subscription_id);
CREATE INDEX idx_subscription_clients_server ON subscription_clients(server_id);
CREATE INDEX idx_subscription_clients_email  ON subscription_clients(xui_email);

-- стабильный токен подписки для /sub/<token>
ALTER TABLE subscriptions ADD COLUMN sub_token TEXT UNIQUE;

-- Удаление per-сервер полей с subscriptions — теперь живут в subscription_clients
ALTER TABLE subscriptions
    DROP COLUMN IF EXISTS server_id,
    DROP COLUMN IF EXISTS xui_client_id,
    DROP COLUMN IF EXISTS xui_email,
    DROP COLUMN IF EXISTS connection_key;

-- Удаляем платную смену региона как концепцию
ALTER TABLE users DROP COLUMN IF EXISTS free_region_switches;
DELETE FROM settings WHERE key = 'region_switch_price';
