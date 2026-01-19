-- Add balance column to users (in TON)
ALTER TABLE users ADD COLUMN balance DECIMAL(18,9) NOT NULL DEFAULT 0;

-- Balance transactions history
CREATE TABLE balance_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id BIGINT NOT NULL REFERENCES users(id),
    amount DECIMAL(18,9) NOT NULL,  -- positive = credit, negative = debit
    type VARCHAR(50) NOT NULL,       -- referral_bonus, giveaway, subscription_payment, refund
    description TEXT,
    reference_id UUID,               -- payment_id, referral_id, etc
    balance_before DECIMAL(18,9) NOT NULL,
    balance_after DECIMAL(18,9) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_balance_transactions_user_id ON balance_transactions(user_id);
CREATE INDEX idx_balance_transactions_type ON balance_transactions(type);
CREATE INDEX idx_balance_transactions_created_at ON balance_transactions(created_at);

-- Update referrals table to store TON bonus instead of days
ALTER TABLE referrals ADD COLUMN bonus_ton DECIMAL(18,9) DEFAULT 0;
