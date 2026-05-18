-- PostgreSQL: Add plain_key column to api_keys
ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS plain_key VARCHAR(128) DEFAULT NULL;
