-- Admins table (Telegram user IDs with admin privileges)
CREATE TABLE admins (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id BIGINT UNIQUE NOT NULL REFERENCES users(id),
    role VARCHAR(20) DEFAULT 'admin',  -- 'admin' or 'superadmin'
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_by BIGINT REFERENCES users(id)
);

-- Banned users table
CREATE TABLE banned_users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id BIGINT REFERENCES users(id),
    ip_address VARCHAR(45),  -- IPv4 or IPv6
    reason TEXT,
    banned_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    banned_by BIGINT REFERENCES users(id),
    expires_at TIMESTAMP WITH TIME ZONE DEFAULT NULL,  -- NULL = permanent
    is_active BOOLEAN DEFAULT true
);

-- Admin action logs
CREATE TABLE admin_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    admin_id BIGINT NOT NULL REFERENCES users(id),
    action VARCHAR(50) NOT NULL,
    target_user_id BIGINT REFERENCES users(id),
    details JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_banned_users_user_id ON banned_users(user_id) WHERE is_active = true;
CREATE INDEX idx_banned_users_ip ON banned_users(ip_address) WHERE is_active = true;
CREATE INDEX idx_admin_logs_admin ON admin_logs(admin_id);
CREATE INDEX idx_admin_logs_target ON admin_logs(target_user_id);
