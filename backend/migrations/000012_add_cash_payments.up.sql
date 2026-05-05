-- Cash payments + cash_plan promo type.
--
-- Cash payments use Provider='cash', Currency='RUB' on the existing payments
-- table. They sit in status='pending' until an admin approves via the bot or
-- admin panel; approval triggers the same subscription provisioning path as
-- TON/Stars completions.
--
-- The cash_plan promo type lets an admin pre-bind a promo code to a specific
-- plan + cash amount, so a user redeeming the code automatically gets a
-- pending cash payment of the right size. This is the "give friends a year
-- for 100 RUB" flow.

ALTER TABLE promo_codes
    ADD COLUMN plan_id UUID REFERENCES plans(id) ON DELETE SET NULL,
    ADD COLUMN cash_amount_rub NUMERIC(10, 2);

COMMENT ON COLUMN promo_codes.plan_id IS 'For cash_plan type: which plan the promo unlocks';
COMMENT ON COLUMN promo_codes.cash_amount_rub IS 'For cash_plan type: cash amount in RUB the user pays the representative';
