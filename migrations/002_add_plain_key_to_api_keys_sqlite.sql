-- Migration: 002_add_plain_key_to_api_keys.sql (SQLite)
-- Adds plain_key column to store the full API key for admin viewing (小眼睛查看)

ALTER TABLE api_keys ADD COLUMN plain_key TEXT DEFAULT NULL;
