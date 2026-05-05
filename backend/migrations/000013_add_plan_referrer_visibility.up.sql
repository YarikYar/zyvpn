-- Plan visibility scoped to a specific inviter.
--
-- visible_to_referrer_id is the user_id of the person whose referrals can
-- see this plan. NULL means the plan is public to everyone (default).
--
-- A friend who came in via /start ref_<code> ends up with users.referred_by
-- set to that referrer's id. Plan visibility check on user listing becomes:
--   visible_to_referrer_id IS NULL OR visible_to_referrer_id = users.referred_by

ALTER TABLE plans
    ADD COLUMN visible_to_referrer_id BIGINT REFERENCES users(id) ON DELETE SET NULL;

CREATE INDEX idx_plans_visible_to_referrer_id ON plans(visible_to_referrer_id) WHERE visible_to_referrer_id IS NOT NULL;

COMMENT ON COLUMN plans.visible_to_referrer_id IS 'If set, plan is hidden from everyone except users referred by this user_id';
