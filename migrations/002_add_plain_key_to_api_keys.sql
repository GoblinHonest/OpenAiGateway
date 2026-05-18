-- Migration: 002_add_plain_key_to_api_keys.sql
-- Adds plain_key column to store the full API key for admin viewing (小眼睛查看)
-- Run: mysql -u root -p aigateway < migrations/002_add_plain_key_to_api_keys.sql

ALTER TABLE api_keys ADD COLUMN plain_key VARCHAR(128) DEFAULT NULL AFTER key_prefix;
