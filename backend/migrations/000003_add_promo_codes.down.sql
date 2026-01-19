DROP INDEX IF EXISTS idx_promo_code_uses_user;
DROP INDEX IF EXISTS idx_promo_codes_code;
DROP TABLE IF EXISTS promo_code_uses;
DROP TABLE IF EXISTS promo_codes;

ALTER TABLE payments
    DROP COLUMN IF EXISTS payment_type;
