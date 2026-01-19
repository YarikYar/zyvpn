ALTER TABLE referrals DROP COLUMN IF EXISTS bonus_ton;
DROP TABLE IF EXISTS balance_transactions;
ALTER TABLE users DROP COLUMN IF EXISTS balance;
