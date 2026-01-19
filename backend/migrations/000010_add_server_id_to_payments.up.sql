-- Add server_id to payments table for server selection during purchase
ALTER TABLE payments ADD COLUMN IF NOT EXISTS server_id UUID REFERENCES servers(id);
