-- Insert legacy server from config (Germany, Frankfurt)
INSERT INTO servers (
    id,
    name,
    country,
    city,
    flag_emoji,
    xui_base_url,
    xui_username,
    xui_password,
    xui_inbound_id,
    server_address,
    server_port,
    public_key,
    short_id,
    server_name,
    is_active,
    sort_order,
    capacity,
    current_load,
    status
) VALUES (
    '00000000-0000-0000-0000-000000000001',
    'Frankfurt',
    'Germany',
    'Frankfurt',
    '🇩🇪',
    'http://185.218.137.109:32768/ZQD3ySh4HinHwZy',
    'p39Dyk6db1',
    'o0RV7MZoqE',
    3,
    '185.218.137.109',
    43843,
    'FyHTbnC9T-q3AYNeDgh4XMkh4UXwTuIjhj9z8-2yBDg',
    'cf1a4e',
    'microsoft.com',
    true,
    1,
    100,
    0,
    'unknown'
) ON CONFLICT (id) DO UPDATE SET
    xui_base_url = EXCLUDED.xui_base_url,
    xui_username = EXCLUDED.xui_username,
    xui_password = EXCLUDED.xui_password,
    xui_inbound_id = EXCLUDED.xui_inbound_id,
    server_port = EXCLUDED.server_port,
    public_key = EXCLUDED.public_key,
    short_id = EXCLUDED.short_id,
    server_name = EXCLUDED.server_name;

-- Update existing subscriptions without server_id to use this server
UPDATE subscriptions
SET server_id = '00000000-0000-0000-0000-000000000001'
WHERE server_id IS NULL;
