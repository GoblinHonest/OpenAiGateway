-- PostgreSQL version of init schema

CREATE TABLE IF NOT EXISTS schema_migrations (
    version VARCHAR(64) PRIMARY KEY,
    description VARCHAR(255),
    applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    checksum VARCHAR(64)
);

CREATE TABLE IF NOT EXISTS providers (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    provider_type VARCHAR(64),
    base_url VARCHAR(512),
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    format_endpoints JSONB DEFAULT '[]',
    models JSONB DEFAULT '[]',
    supported_formats JSONB DEFAULT '[]',
    endpoints JSONB DEFAULT '{}',
    rate_limit_config JSONB,
    timeout_config JSONB,
    retry_config JSONB,
    metadata JSONB,
    version INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_providers_status ON providers(status);
CREATE INDEX IF NOT EXISTS idx_providers_name ON providers(name);

CREATE TABLE IF NOT EXISTS tokens (
    id VARCHAR(64) PRIMARY KEY,
    provider_id VARCHAR(64) NOT NULL REFERENCES providers(id) ON DELETE CASCADE,
    name VARCHAR(255),
    token_value TEXT NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    quota_total BIGINT,
    quota_used BIGINT DEFAULT 0,
    quota_remaining BIGINT,
    quota_reset_at TIMESTAMP,
    rate_limited BOOLEAN DEFAULT FALSE,
    rate_limit_reset_at TIMESTAMP,
    failure_count INT DEFAULT 0,
    consecutive_failures INT DEFAULT 0,
    last_failure_at TIMESTAMP,
    last_success_at TIMESTAMP,
    last_used_at TIMESTAMP,
    total_requests BIGINT DEFAULT 0,
    success_requests BIGINT DEFAULT 0,
    success_rate DECIMAL(5,2),
    version INT NOT NULL DEFAULT 0,
    metadata JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_tokens_provider_status ON tokens(provider_id, status);
CREATE INDEX IF NOT EXISTS idx_tokens_status ON tokens(status);
CREATE INDEX IF NOT EXISTS idx_tokens_last_used ON tokens(last_used_at);

CREATE TABLE IF NOT EXISTS models (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    display_name VARCHAR(255),
    description TEXT,
    model_type VARCHAR(64),
    context_window INT,
    max_output_tokens INT,
    input_price_per_1k DECIMAL(10, 6),
    output_price_per_1k DECIMAL(10, 6),
    enabled BOOLEAN DEFAULT TRUE,
    version INT NOT NULL DEFAULT 0,
    metadata JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_models_name ON models(name);
CREATE INDEX IF NOT EXISTS idx_models_enabled ON models(enabled);

CREATE TABLE IF NOT EXISTS model_provider_bindings (
    id VARCHAR(64) PRIMARY KEY,
    model_id VARCHAR(64) NOT NULL REFERENCES models(id) ON DELETE CASCADE,
    provider_id VARCHAR(64) NOT NULL REFERENCES providers(id) ON DELETE CASCADE,
    upstream_model_name VARCHAR(255) DEFAULT '',
    weight INT DEFAULT 1,
    priority INT DEFAULT 0,
    enabled BOOLEAN DEFAULT TRUE,
    version INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(model_id, provider_id)
);

CREATE INDEX IF NOT EXISTS idx_bindings_model ON model_provider_bindings(model_id);
CREATE INDEX IF NOT EXISTS idx_bindings_provider ON model_provider_bindings(provider_id);
CREATE INDEX IF NOT EXISTS idx_bindings_enabled ON model_provider_bindings(enabled);

CREATE TABLE IF NOT EXISTS groups (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    load_balance_strategy VARCHAR(64) NOT NULL DEFAULT 'round_robin',
    rate_limit_config JSONB,
    quota_config JSONB,
    enabled BOOLEAN DEFAULT TRUE,
    version INT NOT NULL DEFAULT 0,
    metadata JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_groups_name ON groups(name);
CREATE INDEX IF NOT EXISTS idx_groups_enabled ON groups(enabled);

CREATE TABLE IF NOT EXISTS request_logs (
    id VARCHAR(64) PRIMARY KEY,
    request_id VARCHAR(64) NOT NULL UNIQUE,
    timestamp TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    client_ip VARCHAR(64),
    user_agent TEXT,
    api_key_id VARCHAR(64),
    api_key_name VARCHAR(255),
    group_id VARCHAR(64),
    route VARCHAR(128),
    model_id VARCHAR(64),
    model_name VARCHAR(255),
    provider_id VARCHAR(64),
    provider_name VARCHAR(255),
    token_id VARCHAR(64),
    upstream_model_name VARCHAR(255),
    interface_type VARCHAR(32),
    source_format VARCHAR(32),
    target_format VARCHAR(32),
    protocol_converted BOOLEAN DEFAULT FALSE,
    is_streaming BOOLEAN DEFAULT FALSE,
    request_body_size INT,
    max_tokens INT,
    request_headers TEXT,
    request_body TEXT,
    response_headers TEXT,
    response_body TEXT,
    input_tokens INT DEFAULT 0,
    output_tokens INT DEFAULT 0,
    total_tokens INT DEFAULT 0,
    cache_hit BOOLEAN DEFAULT FALSE,
    cache_type VARCHAR(16),
    cache_key VARCHAR(64),
    cache_read_input_tokens INT DEFAULT 0,
    cache_saved_tokens INT DEFAULT 0,
    first_byte_latency_ms INT,
    total_latency_ms INT,
    provider_latency_ms INT,
    status_code INT,
    success BOOLEAN DEFAULT FALSE,
    error_code VARCHAR(64),
    error_message TEXT,
    retry_count INT DEFAULT 0,
    fallback_used BOOLEAN DEFAULT FALSE,
    estimated_cost DECIMAL(10, 6),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_logs_timestamp ON request_logs(timestamp);
CREATE INDEX IF NOT EXISTS idx_logs_api_key_time ON request_logs(api_key_id, timestamp);
CREATE INDEX IF NOT EXISTS idx_logs_provider_model ON request_logs(provider_id, model_id, timestamp);
CREATE INDEX IF NOT EXISTS idx_logs_success ON request_logs(success);
CREATE INDEX IF NOT EXISTS idx_logs_interface ON request_logs(interface_type);
CREATE INDEX IF NOT EXISTS idx_logs_cache ON request_logs(cache_hit);
CREATE INDEX IF NOT EXISTS idx_logs_api_key_name ON request_logs(api_key_name, timestamp);

CREATE TABLE IF NOT EXISTS api_keys (
    id VARCHAR(64) PRIMARY KEY,
    key_hash VARCHAR(128) NOT NULL UNIQUE,
    key_prefix VARCHAR(16) NOT NULL,
    plain_key VARCHAR(128) DEFAULT NULL,
    name VARCHAR(255),
    group_id VARCHAR(64) REFERENCES groups(id) ON DELETE SET NULL,
    rate_limit_config JSONB,
    quota_config JSONB,
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    expires_at TIMESTAMP,
    last_used_at TIMESTAMP,
    version INT NOT NULL DEFAULT 0,
    metadata JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_api_keys_hash ON api_keys(key_hash);
CREATE INDEX IF NOT EXISTS idx_api_keys_status ON api_keys(status);
CREATE INDEX IF NOT EXISTS idx_api_keys_group ON api_keys(group_id);

CREATE TABLE IF NOT EXISTS provider_health_checks (
    id BIGSERIAL PRIMARY KEY,
    provider_id VARCHAR(64) NOT NULL REFERENCES providers(id) ON DELETE CASCADE,
    token_id VARCHAR(64),
    status VARCHAR(20) NOT NULL,
    latency_ms INT,
    error_rate DECIMAL(5,2),
    error_message TEXT,
    checked_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_health_provider_time ON provider_health_checks(provider_id, checked_at);
CREATE INDEX IF NOT EXISTS idx_health_status ON provider_health_checks(status);

CREATE TABLE IF NOT EXISTS circuit_breaker_states (
    provider_id VARCHAR(64) PRIMARY KEY REFERENCES providers(id) ON DELETE CASCADE,
    state VARCHAR(20) NOT NULL,
    failure_count INT DEFAULT 0,
    success_count INT DEFAULT 0,
    last_failure_at TIMESTAMP,
    last_success_at TIMESTAMP,
    next_retry_at TIMESTAMP,
    half_open_requests INT DEFAULT 0,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_circuit_state ON circuit_breaker_states(state);
CREATE INDEX IF NOT EXISTS idx_circuit_retry ON circuit_breaker_states(next_retry_at);

CREATE TABLE IF NOT EXISTS reconciliation_records (
    id BIGSERIAL PRIMARY KEY,
    type VARCHAR(64) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    before_state JSONB,
    after_state JSONB,
    difference JSONB,
    corrected_at TIMESTAMP,
    notes TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_recon_type ON reconciliation_records(type);
CREATE INDEX IF NOT EXISTS idx_recon_status ON reconciliation_records(status);

CREATE TABLE IF NOT EXISTS admin_audit_logs (
    id BIGSERIAL PRIMARY KEY,
    admin_token_prefix VARCHAR(16),
    action VARCHAR(64) NOT NULL,
    resource_type VARCHAR(64) NOT NULL,
    resource_id VARCHAR(64) NOT NULL,
    changes JSONB,
    ip_address VARCHAR(64),
    user_agent TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_audit_resource ON admin_audit_logs(resource_type, resource_id);
CREATE INDEX IF NOT EXISTS idx_audit_action ON admin_audit_logs(action);
CREATE INDEX IF NOT EXISTS idx_audit_created ON admin_audit_logs(created_at);
