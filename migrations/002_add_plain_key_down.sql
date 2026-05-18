-- Rollback: Remove plain_key column (MySQL)
ALTER TABLE api_keys DROP COLUMN plain_key;
