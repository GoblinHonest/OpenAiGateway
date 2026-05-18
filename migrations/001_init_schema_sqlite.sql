-- SQLite版本的初始化Schema

CREATE TABLE IF NOT EXISTS schema_migrations (
    version TEXT PRIMARY KEY,
    description TEXT,
    applied_at TEXT NOT NULL DEFAULT (datetime('now')),
    checksum TEXT
);

CREATE TABLE IF NOT EXISTS providers (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    provider_type TEXT,
    base_url TEXT,
    status TEXT NOT NULL DEFAULT 'active',
    format_endpoints TEXT DEFAULT '[]',   -- JSON array of {format,url,path}
    models TEXT DEFAULT '[]',             -- JSON array of model names
    supported_formats TEXT DEFAULT '[]',  -- JSON array of format strings
    endpoints TEXT DEFAULT '{}',          -- JSON object of endpoint mappings
    rate_limit_config TEXT DEFAULT '{}',  -- JSON
    timeout_config TEXT DEFAULT '{}',     -- JSON
    retry_config TEXT DEFAULT '{}',       -- JSON
    metadata TEXT DEFAULT '{}',           -- JSON
    version INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_providers_status ON providers(status);
CREATE INDEX IF NOT EXISTS idx_providers_name ON providers(name);

CREATE TRIGGER IF NOT EXISTS trg_providers_updated
AFTER UPDATE ON providers
BEGIN
    UPDATE providers SET updated_at = datetime('now') WHERE id = NEW.id;
END;

CREATE TABLE IF NOT EXISTS tokens (
    id TEXT PRIMARY KEY,
    provider_id TEXT NOT NULL,
    name TEXT,
    token_value TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active',
    quota_total INTEGER,
    quota_used INTEGER DEFAULT 0,
    quota_remaining INTEGER,
    quota_reset_at TEXT,
    rate_limited INTEGER DEFAULT 0,
    rate_limit_reset_at TEXT,
    failure_count INTEGER DEFAULT 0,
    consecutive_failures INTEGER DEFAULT 0,
    last_failure_at TEXT,
    last_success_at TEXT,
    last_used_at TEXT,
    total_requests INTEGER DEFAULT 0,
    success_requests INTEGER DEFAULT 0,
    success_rate REAL,
    version INTEGER NOT NULL DEFAULT 0,
    metadata TEXT,  -- JSON
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (provider_id) REFERENCES providers(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_tokens_provider_status ON tokens(provider_id, status);
CREATE INDEX IF NOT EXISTS idx_tokens_status ON tokens(status);
CREATE INDEX IF NOT EXISTS idx_tokens_last_used ON tokens(last_used_at);

CREATE TRIGGER IF NOT EXISTS trg_tokens_updated
AFTER UPDATE ON tokens
BEGIN
    UPDATE tokens SET updated_at = datetime('now') WHERE id = NEW.id;
END;

CREATE TABLE IF NOT EXISTS models (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    display_name TEXT,
    description TEXT,
    model_type TEXT,
    context_window INTEGER,
    max_output_tokens INTEGER,
    input_price_per_1k REAL,
    output_price_per_1k REAL,
    enabled INTEGER DEFAULT 1,
    version INTEGER NOT NULL DEFAULT 0,
    metadata TEXT,  -- JSON
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_models_name ON models(name);
CREATE INDEX IF NOT EXISTS idx_models_enabled ON models(enabled);

CREATE TRIGGER IF NOT EXISTS trg_models_updated
AFTER UPDATE ON models
BEGIN
    UPDATE models SET updated_at = datetime('now') WHERE id = NEW.id;
END;

CREATE TABLE IF NOT EXISTS model_provider_bindings (
    id TEXT PRIMARY KEY,
    model_id TEXT NOT NULL,
    provider_id TEXT NOT NULL,
    upstream_model_name TEXT DEFAULT '',
    weight INTEGER DEFAULT 1,
    priority INTEGER DEFAULT 0,
    enabled INTEGER DEFAULT 1,
    version INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (model_id) REFERENCES models(id) ON DELETE CASCADE,
    FOREIGN KEY (provider_id) REFERENCES providers(id) ON DELETE CASCADE,
    UNIQUE(model_id, provider_id)
);

CREATE INDEX IF NOT EXISTS idx_bindings_model ON model_provider_bindings(model_id);
CREATE INDEX IF NOT EXISTS idx_bindings_provider ON model_provider_bindings(provider_id);
CREATE INDEX IF NOT EXISTS idx_bindings_enabled ON model_provider_bindings(enabled);

CREATE TRIGGER IF NOT EXISTS trg_bindings_updated
AFTER UPDATE ON model_provider_bindings
BEGIN
    UPDATE model_provider_bindings SET updated_at = datetime('now') WHERE id = NEW.id;
END;

CREATE TABLE IF NOT EXISTS groups (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    description TEXT,
    load_balance_strategy TEXT NOT NULL DEFAULT 'round_robin',
    rate_limit_config TEXT,  -- JSON
    quota_config TEXT,  -- JSON
    enabled INTEGER DEFAULT 1,
    version INTEGER NOT NULL DEFAULT 0,
    metadata TEXT,  -- JSON
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_groups_name ON groups(name);
CREATE INDEX IF NOT EXISTS idx_groups_enabled ON groups(enabled);

CREATE TRIGGER IF NOT EXISTS trg_groups_updated
AFTER UPDATE ON groups
BEGIN
    UPDATE groups SET updated_at = datetime('now') WHERE id = NEW.id;
END;

CREATE TABLE IF NOT EXISTS request_logs (
    id TEXT PRIMARY KEY,
    request_id TEXT NOT NULL UNIQUE,
    timestamp TEXT NOT NULL DEFAULT (datetime('now', 'subsecond')),
    client_ip TEXT,
    user_agent TEXT,
    api_key_id TEXT,
    api_key_name TEXT,
    group_id TEXT,
    route TEXT,
    model_id TEXT,
    model_name TEXT,
    provider_id TEXT,
    provider_name TEXT,
    token_id TEXT,
    upstream_model_name TEXT,
    interface_type TEXT,
    source_format TEXT,
    target_format TEXT,
    protocol_converted INTEGER DEFAULT 0,
    is_streaming INTEGER DEFAULT 0,
    request_body_size INTEGER,
    max_tokens INTEGER,
    request_headers TEXT,  -- JSON
    request_body TEXT,
    response_headers TEXT,  -- JSON
    response_body TEXT,
    input_tokens INTEGER DEFAULT 0,
    output_tokens INTEGER DEFAULT 0,
    total_tokens INTEGER DEFAULT 0,
    cache_hit INTEGER DEFAULT 0,
    cache_type TEXT,
    cache_key TEXT,
    cache_read_input_tokens INTEGER DEFAULT 0,
    cache_saved_tokens INTEGER DEFAULT 0,
    first_byte_latency_ms INTEGER,
    total_latency_ms INTEGER,
    provider_latency_ms INTEGER,
    status_code INTEGER,
    success INTEGER DEFAULT 0,
    error_code TEXT,
    error_message TEXT,
    retry_count INTEGER DEFAULT 0,
    fallback_used INTEGER DEFAULT 0,
    estimated_cost REAL,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_logs_timestamp ON request_logs(timestamp);
CREATE INDEX IF NOT EXISTS idx_logs_api_key_time ON request_logs(api_key_id, timestamp);
CREATE INDEX IF NOT EXISTS idx_logs_provider_model ON request_logs(provider_id, model_id, timestamp);
CREATE INDEX IF NOT EXISTS idx_logs_success ON request_logs(success);
CREATE INDEX IF NOT EXISTS idx_logs_interface ON request_logs(interface_type);
CREATE INDEX IF NOT EXISTS idx_logs_cache ON request_logs(cache_hit);
CREATE INDEX IF NOT EXISTS idx_logs_api_key_name ON request_logs(api_key_name, timestamp);

CREATE TABLE IF NOT EXISTS api_keys (
    id TEXT PRIMARY KEY,
    key_hash TEXT NOT NULL UNIQUE,
    key_prefix TEXT NOT NULL,
    plain_key TEXT DEFAULT NULL,
    name TEXT,
    group_id TEXT,
    rate_limit_config TEXT,  -- JSON
    quota_config TEXT,  -- JSON
    status TEXT NOT NULL DEFAULT 'active',
    expires_at TEXT,
    last_used_at TEXT,
    version INTEGER NOT NULL DEFAULT 0,
    metadata TEXT,  -- JSON
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (group_id) REFERENCES groups(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_api_keys_hash ON api_keys(key_hash);
CREATE INDEX IF NOT EXISTS idx_api_keys_status ON api_keys(status);
CREATE INDEX IF NOT EXISTS idx_api_keys_group ON api_keys(group_id);

CREATE TRIGGER IF NOT EXISTS trg_api_keys_updated
AFTER UPDATE ON api_keys
BEGIN
    UPDATE api_keys SET updated_at = datetime('now') WHERE id = NEW.id;
END;

CREATE TABLE IF NOT EXISTS provider_health_checks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    provider_id TEXT NOT NULL,
    token_id TEXT,
    status TEXT NOT NULL,
    latency_ms INTEGER,
    error_rate REAL,
    error_message TEXT,
    checked_at TEXT NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (provider_id) REFERENCES providers(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_health_provider_time ON provider_health_checks(provider_id, checked_at);
CREATE INDEX IF NOT EXISTS idx_health_status ON provider_health_checks(status);

CREATE TABLE IF NOT EXISTS circuit_breaker_states (
    provider_id TEXT PRIMARY KEY,
    state TEXT NOT NULL,
    failure_count INTEGER DEFAULT 0,
    success_count INTEGER DEFAULT 0,
    last_failure_at TEXT,
    last_success_at TEXT,
    next_retry_at TEXT,
    half_open_requests INTEGER DEFAULT 0,
    updated_at TEXT NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (provider_id) REFERENCES providers(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_circuit_state ON circuit_breaker_states(state);
CREATE INDEX IF NOT EXISTS idx_circuit_retry ON circuit_breaker_states(next_retry_at);

CREATE TABLE IF NOT EXISTS reconciliation_records (
    id TEXT PRIMARY KEY,
    type TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    before_state TEXT,
    after_state TEXT,
    difference TEXT,
    corrected_at TEXT,
    notes TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_recon_type ON reconciliation_records(type);
CREATE INDEX IF NOT EXISTS idx_recon_status ON reconciliation_records(status);

CREATE TABLE IF NOT EXISTS admin_audit_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    admin_token_prefix TEXT,
    action TEXT NOT NULL,
    resource_type TEXT NOT NULL,
    resource_id TEXT NOT NULL,
    changes TEXT,  -- JSON
    ip_address TEXT,
    user_agent TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_audit_resource ON admin_audit_logs(resource_type, resource_id);
CREATE INDEX IF NOT EXISTS idx_audit_action ON admin_audit_logs(action);
CREATE INDEX IF NOT EXISTS idx_audit_created ON admin_audit_logs(created_at);
