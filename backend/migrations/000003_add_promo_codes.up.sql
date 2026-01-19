-- Promo codes table
CREATE TABLE promo_codes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(50) UNIQUE NOT NULL,
    type VARCHAR(20) NOT NULL,  -- 'balance' or 'days'
    value DECIMAL(18,9) NOT NULL,  -- TON amount or days count
    max_uses INT DEFAULT NULL,  -- NULL = unlimited
    used_count INT DEFAULT 0,
    expires_at TIMESTAMP WITH TIME ZONE DEFAULT NULL,  -- NULL = no expiration
    is_active BOOLEAN DEFAULT true,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Track which users used which promo codes
CREATE TABLE promo_code_uses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    promo_code_id UUID NOT NULL REFERENCES promo_codes(id),
    user_id BIGINT NOT NULL REFERENCES users(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(promo_code_id, user_id)  -- Each user can use each code only once
);

-- Add top_up transaction type support
ALTER TABLE balance_transactions
    ALTER COLUMN type TYPE VARCHAR(50);

-- Add payment type for top-ups (not tied to subscription)
ALTER TABLE payments
    ADD COLUMN payment_type VARCHAR(50) DEFAULT 'subscription',
    ALTER COLUMN plan_id DROP NOT NULL,
    ALTER COLUMN subscription_id DROP NOT NULL;

CREATE INDEX idx_promo_codes_code ON promo_codes(code);
CREATE INDEX idx_promo_code_uses_user ON promo_code_uses(user_id);
