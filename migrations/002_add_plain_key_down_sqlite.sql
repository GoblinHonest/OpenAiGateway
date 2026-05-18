-- Rollback: SQLite does not support DROP COLUMN, recreate table
CREATE TABLE api_keys_backup AS SELECT id, key_hash, key_prefix, name, group_id, rate_limit_config, quota_config, status, expires_at, last_used_at, version, metadata, created_at, updated_at FROM api_keys;
DROP TABLE api_keys;
CREATE TABLE api_keys (
    id TEXT PRIMARY KEY,
    key_hash TEXT NOT NULL UNIQUE,
    key_prefix TEXT NOT NULL,
    name TEXT,
    group_id TEXT,
    rate_limit_config TEXT,
    quota_config TEXT,
    status TEXT NOT NULL DEFAULT 'active',
    expires_at TEXT,
    last_used_at TEXT,
    version INTEGER NOT NULL DEFAULT 0,
    metadata TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (group_id) REFERENCES groups(id) ON DELETE SET NULL
);
INSERT INTO api_keys SELECT * FROM api_keys_backup;
DROP TABLE api_keys_backup;
