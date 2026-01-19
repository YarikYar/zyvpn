-- Add max_devices to plans (how many IPs can connect simultaneously)
ALTER TABLE plans ADD COLUMN IF NOT EXISTS max_devices INTEGER NOT NULL DEFAULT 3;

-- Add max_devices to subscriptions (inherited from plan at purchase time)
ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS max_devices INTEGER NOT NULL DEFAULT 3;

-- Update existing plans with sensible defaults
UPDATE plans SET max_devices = 3 WHERE name LIKE '%1 мес%' OR name LIKE '%Месяц%';
UPDATE plans SET max_devices = 5 WHERE name LIKE '%3 мес%';
UPDATE plans SET max_devices = 10 WHERE name LIKE '%6 мес%' OR name LIKE '%12 мес%' OR name LIKE '%Год%';
