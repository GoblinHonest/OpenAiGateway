-- Rollback: Drop all tables (SQLite)
DROP TABLE IF EXISTS admin_audit_logs;
DROP TABLE IF EXISTS reconciliation_records;
DROP TABLE IF EXISTS circuit_breaker_states;
DROP TABLE IF EXISTS provider_health_checks;
DROP TABLE IF EXISTS api_keys;
DROP TABLE IF EXISTS request_logs;
DROP TABLE IF EXISTS group_models;
DROP TABLE IF EXISTS groups;
DROP TABLE IF EXISTS model_provider_bindings;
DROP TABLE IF EXISTS models;
DROP TABLE IF EXISTS tokens;
DROP TABLE IF EXISTS providers;
DROP TABLE IF EXISTS schema_migrations;
