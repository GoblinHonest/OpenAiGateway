-- Rollback: Remove plain_key column (PostgreSQL)
ALTER TABLE api_keys DROP COLUMN IF EXISTS plain_key;
