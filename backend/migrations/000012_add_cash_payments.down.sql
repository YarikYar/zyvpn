ALTER TABLE promo_codes
    DROP COLUMN IF EXISTS plan_id,
    DROP COLUMN IF EXISTS cash_amount_rub;
