ALTER TABLE users DROP COLUMN IF EXISTS free_region_switches;
DELETE FROM settings WHERE key = 'region_switch_price';
