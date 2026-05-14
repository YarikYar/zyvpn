-- Удаление оплаты наличкой как опции.
-- В коде остаются TON и Stars; cash-провайдер и cash_plan промокоды
-- упраздняются.

-- 1) Любые висящие pending cash-payment'ы помечаем failed — иначе они
--    останутся «вечно pending» в БД и могут вылезти в админке.
UPDATE payments
SET status = 'failed'
WHERE provider = 'cash'
  AND status = 'pending';

-- 2) Удаляем cash_plan-промокоды целиком — без cash-flow они не нужны.
DELETE FROM promo_codes WHERE type = 'cash_plan';

-- 3) Снимаем расширения схемы под cash_plan.
ALTER TABLE promo_codes
    DROP COLUMN IF EXISTS plan_id,
    DROP COLUMN IF EXISTS cash_amount_rub;
