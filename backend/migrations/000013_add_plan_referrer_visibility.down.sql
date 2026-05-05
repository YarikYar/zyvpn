DROP INDEX IF EXISTS idx_plans_visible_to_referrer_id;
ALTER TABLE plans DROP COLUMN IF EXISTS visible_to_referrer_id;
