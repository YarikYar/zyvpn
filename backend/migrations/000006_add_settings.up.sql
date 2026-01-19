-- Settings table for admin-configurable values
CREATE TABLE IF NOT EXISTS settings (
    key VARCHAR(100) PRIMARY KEY,
    value TEXT NOT NULL,
    description TEXT,
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Insert default settings
INSERT INTO settings (key, value, description) VALUES
    ('topup_bonus_percent', '0', 'Бонус % при пополнении баланса через TON (0-10)'),
    ('referral_bonus_ton', '0.1', 'Бонус TON реферреру при первой оплате приглашённого')
ON CONFLICT (key) DO NOTHING;
