-- Откат: возвращаем cash_plan-поля в promo_codes. Отменённые pending
-- cash-payment'ы при откате НЕ восстанавливаются (нечем).

ALTER TABLE promo_codes
    ADD COLUMN IF NOT EXISTS plan_id UUID REFERENCES plans(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS cash_amount_rub NUMERIC(10, 2);

COMMENT ON COLUMN promo_codes.plan_id IS 'For cash_plan type: which plan the promo unlocks';
COMMENT ON COLUMN promo_codes.cash_amount_rub IS 'For cash_plan type: cash amount in RUB the user pays the representative';
