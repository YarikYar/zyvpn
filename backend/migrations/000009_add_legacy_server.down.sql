-- Remove server assignment from subscriptions
UPDATE subscriptions
SET server_id = NULL
WHERE server_id = '00000000-0000-0000-0000-000000000001';

-- Delete legacy server
DELETE FROM servers WHERE id = '00000000-0000-0000-0000-000000000001';
