-- Add free region switches counter to users
ALTER TABLE users ADD COLUMN IF NOT EXISTS free_region_switches INT NOT NULL DEFAULT 0;

-- Add region switch price setting (default 0.1 TON)
INSERT INTO settings (key, value) VALUES ('region_switch_price', '0.1')
ON CONFLICT (key) DO NOTHING;

-- Add region_switch promo code type support (value = number of free switches)
-- No schema change needed, just new type value 'region_switch'
