-- Servers table for multi-server support
CREATE TABLE IF NOT EXISTS servers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    country VARCHAR(50) NOT NULL,
    city VARCHAR(50),
    flag_emoji VARCHAR(10) DEFAULT '',

    -- XUI Panel connection
    xui_base_url VARCHAR(255) NOT NULL,
    xui_username VARCHAR(100) NOT NULL,
    xui_password VARCHAR(255) NOT NULL,
    xui_inbound_id INT NOT NULL DEFAULT 1,

    -- Server connection details
    server_address VARCHAR(255) NOT NULL,
    server_port INT NOT NULL DEFAULT 443,
    public_key VARCHAR(255),
    short_id VARCHAR(50),
    server_name VARCHAR(255) DEFAULT 'www.google.com',

    -- Status
    is_active BOOLEAN DEFAULT true,
    sort_order INT DEFAULT 0,

    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Add server_id to subscriptions
ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS server_id UUID REFERENCES servers(id);

-- Create index for active servers
CREATE INDEX IF NOT EXISTS idx_servers_active ON servers(is_active, sort_order);
