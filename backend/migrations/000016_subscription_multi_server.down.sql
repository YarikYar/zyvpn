-- Откат: восстанавливаем per-сервер поля subscriptions и счётчик смен регионов.
-- Внимание: rollback теряет данные multi-server подписок (только один server_id
-- из subscription_clients можно восстановить, остальные клиенты — потеряются).

ALTER TABLE subscriptions
    ADD COLUMN IF NOT EXISTS server_id      UUID REFERENCES servers(id),
    ADD COLUMN IF NOT EXISTS xui_client_id  VARCHAR(255),
    ADD COLUMN IF NOT EXISTS xui_email      VARCHAR(255),
    ADD COLUMN IF NOT EXISTS connection_key TEXT;

ALTER TABLE subscriptions DROP COLUMN IF EXISTS sub_token;

DROP TABLE IF EXISTS subscription_clients;
DROP TABLE IF EXISTS plan_servers;

ALTER TABLE users ADD COLUMN IF NOT EXISTS free_region_switches INT NOT NULL DEFAULT 0;
INSERT INTO settings (key, value) VALUES ('region_switch_price', '0.1')
ON CONFLICT (key) DO NOTHING;
