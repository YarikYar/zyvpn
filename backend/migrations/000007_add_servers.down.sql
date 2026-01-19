-- Remove server_id from subscriptions
ALTER TABLE subscriptions DROP COLUMN IF EXISTS server_id;

-- Drop servers table
DROP TABLE IF EXISTS servers;
