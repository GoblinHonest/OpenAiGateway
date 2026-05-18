# 数据库Schema设计

## 设计原则

1. **双引擎支持**: 开发/测试使用SQLite，生产环境使用MySQL/PostgreSQL
2. **乐观锁**: 所有核心表添加 `version` 字段
3. **审计追踪**: 所有管理操作记录审计日志
4. **明文存储**: Token明文存储（按用户要求），通过访问控制保护

## 主键策略说明

| 表类型 | 主键类型 | 原因 |
|--------|----------|------|
| 业务实体表 (providers, tokens, models等) | VARCHAR(64) | 业务ID，支持分布式生成，可预测 |
| 日志/历史表 (request_logs, health_checks等) | BIGINT AUTO_INCREMENT | 自增ID性能好，无业务含义 |

## 配额更新策略

`quota_used` 和 `quota_remaining` 冗余存储，避免并发计算问题：
- 更新时使用原子操作: `quota_remaining = quota_remaining - amount`
- `quota_used` 通过对账任务定期同步
- 乐观锁通过 `version` 字段防止并发冲突

---

## 第一部分：MySQL DDL

### 1. schema_migrations - 数据库迁移版本表

```sql
CREATE TABLE schema_migrations (
    version VARCHAR(64) PRIMARY KEY,
    description VARCHAR(255),
    applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    checksum VARCHAR(64)  -- 迁移文件校验和
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

### 2. providers - 服务商表

```sql
CREATE TABLE providers (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    base_url VARCHAR(512),
    status VARCHAR(32) NOT NULL DEFAULT 'active',  -- active, inactive, maintenance
    supported_formats JSON NOT NULL,  -- ["openai", "anthropic", "gemini"]
    endpoints JSON NOT NULL,  -- {"chat": "/v1/chat/completions", ...}
    rate_limit_config JSON,  -- {"requests_per_minute": 60, "tokens_per_minute": 90000}
    timeout_config JSON,  -- {"connect_timeout": 10, "read_timeout": 60}
    retry_config JSON,  -- {"max_attempts": 3, "initial_backoff": 1}
    metadata JSON,
    version INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_status (status),
    INDEX idx_name (name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

### 3. tokens - Token表

```sql
CREATE TABLE tokens (
    id VARCHAR(64) PRIMARY KEY,
    provider_id VARCHAR(64) NOT NULL,
    name VARCHAR(255),
    token_value TEXT NOT NULL,  -- 明文存储
    status VARCHAR(32) NOT NULL DEFAULT 'active',  -- active, inactive, exhausted, disabled

    -- 额度管理
    quota_total BIGINT,  -- 总配额
    quota_used BIGINT DEFAULT 0,  -- 已使用（对账同步）
    quota_remaining BIGINT,  -- 剩余配额（原子递减）
    quota_reset_at TIMESTAMP,  -- 配额重置时间

    -- 限速状态
    rate_limited BOOLEAN DEFAULT FALSE,
    rate_limit_reset_at TIMESTAMP,

    -- 失败追踪
    failure_count INT DEFAULT 0,
    consecutive_failures INT DEFAULT 0,
    last_failure_at TIMESTAMP,
    last_success_at TIMESTAMP,
    last_used_at TIMESTAMP,

    -- 统计信息
    total_requests BIGINT DEFAULT 0,
    success_requests BIGINT DEFAULT 0,
    success_rate DECIMAL(5,2),

    -- 并发控制
    version INT NOT NULL DEFAULT 0,  -- 乐观锁
    metadata JSON,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    FOREIGN KEY (provider_id) REFERENCES providers(id) ON DELETE CASCADE,
    INDEX idx_provider_status (provider_id, status),
    INDEX idx_provider_status_available (provider_id, status, rate_limited),
    INDEX idx_status (status),
    INDEX idx_last_used (last_used_at),
    INDEX idx_rate_limit_reset (rate_limit_reset_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

### 4. models - 模型表

```sql
CREATE TABLE models (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    display_name VARCHAR(255),
    description TEXT,
    model_type VARCHAR(64),  -- chat, embeddings, image, audio
    context_window INT,
    max_output_tokens INT,
    input_price_per_1k DECIMAL(10, 6),
    output_price_per_1k DECIMAL(10, 6),
    enabled BOOLEAN DEFAULT TRUE,
    version INT NOT NULL DEFAULT 0,
    metadata JSON,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_name (name),
    INDEX idx_enabled (enabled)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

### 5. model_provider_bindings - 模型服务商绑定表

```sql
CREATE TABLE model_provider_bindings (
    id VARCHAR(64) PRIMARY KEY,
    model_id VARCHAR(64) NOT NULL,
    provider_id VARCHAR(64) NOT NULL,
    weight INT DEFAULT 1,  -- 权重（加权轮询）
    priority INT DEFAULT 0,  -- 优先级（越大越优先）
    enabled BOOLEAN DEFAULT TRUE,
    version INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (model_id) REFERENCES models(id) ON DELETE CASCADE,
    FOREIGN KEY (provider_id) REFERENCES providers(id) ON DELETE CASCADE,
    UNIQUE KEY uk_model_provider (model_id, provider_id),
    INDEX idx_model (model_id),
    INDEX idx_provider (provider_id),
    INDEX idx_enabled (enabled),
    INDEX idx_model_enabled (model_id, enabled)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

### 6. groups - 分组表

```sql
CREATE TABLE groups (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    load_balance_strategy VARCHAR(64) NOT NULL DEFAULT 'round_robin',
    rate_limit_config JSON,  -- {"requests_per_minute": 60, "tokens_per_minute": 100000}
    quota_config JSON,  -- {"total_tokens": 10000000, "reset_period": "monthly"}
    enabled BOOLEAN DEFAULT TRUE,
    version INT NOT NULL DEFAULT 0,
    metadata JSON,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_name (name),
    INDEX idx_enabled (enabled)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

### 7. request_logs - 请求日志表

```sql
CREATE TABLE request_logs (
    id VARCHAR(64) PRIMARY KEY,
    request_id VARCHAR(64) NOT NULL UNIQUE,
    timestamp TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),

    -- 客户端信息
    client_ip VARCHAR(64),
    user_agent TEXT,
    api_key_id VARCHAR(64),
    api_key_name VARCHAR(255),  -- API Key名称（冗余，便于查询）

    -- 路由信息
    group_id VARCHAR(64),
    route VARCHAR(128),  -- 请求路由，如 /chat/completions

    -- 客户端请求的模型（用户视角）
    model_id VARCHAR(64),
    model_name VARCHAR(255),  -- 客户端请求的模型名，如 "deepseek-v4-flash"

    -- 上游服务商信息（实际调用的服务商）
    provider_id VARCHAR(64),
    provider_name VARCHAR(255),  -- 实际调用的服务商名，如 "sensenova", "modelscope"
    token_id VARCHAR(64),
    upstream_model_name VARCHAR(255),  -- 上游实际模型名（可能与客户端请求不同）

    -- 协议信息
    interface_type VARCHAR(32),  -- 接口类型: openai, anthropic, gemini
    source_format VARCHAR(32),  -- 客户端请求格式
    target_format VARCHAR(32),  -- 上游服务商格式
    protocol_converted BOOLEAN DEFAULT FALSE,  -- 是否进行了协议转换

    -- 请求内容
    is_streaming BOOLEAN DEFAULT FALSE,
    request_body_size INT,
    max_tokens INT,  -- 请求中的max_tokens参数

    -- 请求/响应体（完整存储）
    request_headers TEXT,  -- JSON格式的请求头
    request_body TEXT,     -- JSON格式的请求体
    response_headers TEXT, -- JSON格式的响应头
    response_body TEXT,    -- JSON格式的响应体

    -- 使用量
    input_tokens INT DEFAULT 0,
    output_tokens INT DEFAULT 0,
    total_tokens INT DEFAULT 0,

    -- 缓存相关
    cache_hit BOOLEAN DEFAULT FALSE,
    cache_type VARCHAR(16),  -- local, redis, none
    cache_key VARCHAR(64),
    cache_read_input_tokens INT DEFAULT 0,  -- 从缓存读取的input tokens
    cache_saved_tokens INT DEFAULT 0,  -- 因缓存节省的token数

    -- 性能指标
    first_byte_latency_ms INT,
    total_latency_ms INT,
    provider_latency_ms INT,

    -- 结果
    status_code INT,
    success BOOLEAN DEFAULT FALSE,
    error_code VARCHAR(64),
    error_message TEXT,

    -- 重试和降级
    retry_count INT DEFAULT 0,
    fallback_used BOOLEAN DEFAULT FALSE,

    -- 成本
    estimated_cost DECIMAL(10, 6),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    INDEX idx_timestamp (timestamp),
    INDEX idx_api_key_time (api_key_id, timestamp),
    INDEX idx_provider_model (provider_id, model_id, timestamp),
    INDEX idx_success (success),
    INDEX idx_interface_type (interface_type),
    INDEX idx_cache_hit (cache_hit),
    INDEX idx_api_key_name (api_key_name, timestamp)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

**MySQL分区策略**（生产环境）:
```sql
-- 按月分区
ALTER TABLE request_logs PARTITION BY RANGE (UNIX_TIMESTAMP(timestamp)) (
    PARTITION p202601 VALUES LESS THAN (UNIX_TIMESTAMP('2026-02-01')),
    PARTITION p202602 VALUES LESS THAN (UNIX_TIMESTAMP('2026-03-01')),
    PARTITION p202603 VALUES LESS THAN (UNIX_TIMESTAMP('2026-04-01')),
    PARTITION p_future VALUES LESS THAN MAXVALUE
);
```

### 8. api_keys - API密钥表

```sql
CREATE TABLE api_keys (
    id VARCHAR(64) PRIMARY KEY,
    key_hash VARCHAR(128) NOT NULL UNIQUE,  -- SHA256哈希
    key_prefix VARCHAR(16) NOT NULL,  -- 前缀用于展示
    name VARCHAR(255),
    group_id VARCHAR(64),
    rate_limit_config JSON,  -- {"requests_per_minute": 60, "tokens_per_minute": 100000}
    quota_config JSON,  -- {"total_tokens": 10000000, "reset_period": "monthly"}
    status VARCHAR(32) NOT NULL DEFAULT 'active',  -- active, inactive, revoked
    expires_at TIMESTAMP,
    last_used_at TIMESTAMP,
    version INT NOT NULL DEFAULT 0,
    metadata JSON,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (group_id) REFERENCES groups(id) ON DELETE SET NULL,
    INDEX idx_key_hash (key_hash),
    INDEX idx_status (status),
    INDEX idx_group (group_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

### 9. provider_health_checks - 健康检查历史表

```sql
CREATE TABLE provider_health_checks (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    provider_id VARCHAR(64) NOT NULL,
    token_id VARCHAR(64),
    status VARCHAR(20) NOT NULL,  -- healthy, degraded, down
    latency_ms INT,
    error_rate DECIMAL(5,2),
    error_message TEXT,
    checked_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (provider_id) REFERENCES providers(id) ON DELETE CASCADE,
    INDEX idx_provider_time (provider_id, checked_at),
    INDEX idx_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

### 10. circuit_breaker_states - 熔断器状态表

```sql
CREATE TABLE circuit_breaker_states (
    provider_id VARCHAR(64) PRIMARY KEY,
    state VARCHAR(20) NOT NULL,  -- closed, open, half_open
    failure_count INT DEFAULT 0,
    success_count INT DEFAULT 0,
    last_failure_at TIMESTAMP,
    last_success_at TIMESTAMP,
    next_retry_at TIMESTAMP,
    half_open_requests INT DEFAULT 0,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (provider_id) REFERENCES providers(id) ON DELETE CASCADE,
    INDEX idx_state (state),
    INDEX idx_next_retry (next_retry_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

### 11. reconciliation_records - 对账记录表

```sql
CREATE TABLE reconciliation_records (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    reconciliation_type VARCHAR(32) NOT NULL,  -- token_quota, usage_stats, cost
    target_id VARCHAR(64) NOT NULL,
    expected_value DECIMAL(20, 6),
    actual_value DECIMAL(20, 6),
    difference DECIMAL(20, 6),
    status VARCHAR(20) NOT NULL,  -- matched, mismatched, corrected
    reconciled_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    corrected_at TIMESTAMP,
    notes TEXT,
    INDEX idx_type_status (reconciliation_type, status),
    INDEX idx_target (target_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

### 12. admin_audit_logs - 管理操作审计日志表

```sql
CREATE TABLE admin_audit_logs (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    admin_token_prefix VARCHAR(16),  -- 管理员token前缀（脱敏）
    action VARCHAR(64) NOT NULL,  -- create, update, delete
    resource_type VARCHAR(64) NOT NULL,  -- provider, token, model, group, api_key
    resource_id VARCHAR(64) NOT NULL,
    changes JSON,  -- {"field": {"old": "xxx", "new": "yyy"}}
    ip_address VARCHAR(64),
    user_agent TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_resource (resource_type, resource_id),
    INDEX idx_action (action),
    INDEX idx_created (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

---

## 第二部分：SQLite DDL

SQLite不支持MySQL的部分特性，需要适配：
- JSON类型 → TEXT
- 分区表 → 定期归档到历史表
- ON UPDATE CURRENT_TIMESTAMP → 触发器
- 条件索引 → 标准索引

### 1. schema_migrations

```sql
CREATE TABLE schema_migrations (
    version TEXT PRIMARY KEY,
    description TEXT,
    applied_at TEXT NOT NULL DEFAULT (datetime('now')),
    checksum TEXT
);
```

### 2. providers

```sql
CREATE TABLE providers (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    base_url TEXT,
    status TEXT NOT NULL DEFAULT 'active',
    supported_formats TEXT NOT NULL,  -- JSON string
    endpoints TEXT NOT NULL,  -- JSON string
    rate_limit_config TEXT,  -- JSON string
    timeout_config TEXT,  -- JSON string
    retry_config TEXT,  -- JSON string
    metadata TEXT,  -- JSON string
    version INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX idx_providers_status ON providers(status);
CREATE INDEX idx_providers_name ON providers(name);

-- 触发器：自动更新 updated_at
CREATE TRIGGER trg_providers_updated
AFTER UPDATE ON providers
BEGIN
    UPDATE providers SET updated_at = datetime('now') WHERE id = NEW.id;
END;
```

### 3. tokens

```sql
CREATE TABLE tokens (
    id TEXT PRIMARY KEY,
    provider_id TEXT NOT NULL,
    name TEXT,
    token_value TEXT NOT NULL,  -- 明文存储
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
    metadata TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now')),

    FOREIGN KEY (provider_id) REFERENCES providers(id) ON DELETE CASCADE
);

CREATE INDEX idx_tokens_provider_status ON tokens(provider_id, status);
CREATE INDEX idx_tokens_provider_available ON tokens(provider_id, status, rate_limited);
CREATE INDEX idx_tokens_status ON tokens(status);
CREATE INDEX idx_tokens_last_used ON tokens(last_used_at);
CREATE INDEX idx_tokens_rate_limit ON tokens(rate_limit_reset_at) WHERE rate_limited = 1;

CREATE TRIGGER trg_tokens_updated
AFTER UPDATE ON tokens
BEGIN
    UPDATE tokens SET updated_at = datetime('now') WHERE id = NEW.id;
END;
```

### 4. models

```sql
CREATE TABLE models (
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
    metadata TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX idx_models_name ON models(name);
CREATE INDEX idx_models_enabled ON models(enabled);

CREATE TRIGGER trg_models_updated
AFTER UPDATE ON models
BEGIN
    UPDATE models SET updated_at = datetime('now') WHERE id = NEW.id;
END;
```

### 5. model_provider_bindings

```sql
CREATE TABLE model_provider_bindings (
    id TEXT PRIMARY KEY,
    model_id TEXT NOT NULL,
    provider_id TEXT NOT NULL,
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

CREATE INDEX idx_bindings_model ON model_provider_bindings(model_id);
CREATE INDEX idx_bindings_provider ON model_provider_bindings(provider_id);
CREATE INDEX idx_bindings_enabled ON model_provider_bindings(enabled);
CREATE INDEX idx_bindings_model_enabled ON model_provider_bindings(model_id, enabled);

CREATE TRIGGER trg_bindings_updated
AFTER UPDATE ON model_provider_bindings
BEGIN
    UPDATE model_provider_bindings SET updated_at = datetime('now') WHERE id = NEW.id;
END;
```

### 6. groups

```sql
CREATE TABLE groups (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    description TEXT,
    load_balance_strategy TEXT NOT NULL DEFAULT 'round_robin',
    rate_limit_config TEXT,  -- JSON string
    quota_config TEXT,  -- JSON string
    enabled INTEGER DEFAULT 1,
    version INTEGER NOT NULL DEFAULT 0,
    metadata TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX idx_groups_name ON groups(name);
CREATE INDEX idx_groups_enabled ON groups(enabled);

CREATE TRIGGER trg_groups_updated
AFTER UPDATE ON groups
BEGIN
    UPDATE groups SET updated_at = datetime('now') WHERE id = NEW.id;
END;
```

### 7. request_logs

```sql
CREATE TABLE request_logs (
    id TEXT PRIMARY KEY,
    request_id TEXT NOT NULL UNIQUE,
    timestamp TEXT NOT NULL DEFAULT (datetime('now', 'subsecond')),

    -- 客户端信息
    client_ip TEXT,
    user_agent TEXT,
    api_key_id TEXT,
    api_key_name TEXT,

    -- 路由信息
    group_id TEXT,
    route TEXT,

    -- 客户端请求的模型
    model_id TEXT,
    model_name TEXT,

    -- 上游服务商信息
    provider_id TEXT,
    provider_name TEXT,
    token_id TEXT,
    upstream_model_name TEXT,

    -- 协议信息
    interface_type TEXT,
    source_format TEXT,
    target_format TEXT,
    protocol_converted INTEGER DEFAULT 0,

    is_streaming INTEGER DEFAULT 0,
    request_body_size INTEGER,
    max_tokens INTEGER,

    -- 请求/响应体（完整存储）
    request_headers TEXT,  -- JSON格式的请求头
    request_body TEXT,     -- JSON格式的请求体
    response_headers TEXT, -- JSON格式的响应头
    response_body TEXT,    -- JSON格式的响应体

    input_tokens INTEGER DEFAULT 0,
    output_tokens INTEGER DEFAULT 0,
    total_tokens INTEGER DEFAULT 0,

    -- 缓存相关
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

CREATE INDEX idx_logs_timestamp ON request_logs(timestamp);
CREATE INDEX idx_logs_api_key_time ON request_logs(api_key_id, timestamp);
CREATE INDEX idx_logs_provider_model ON request_logs(provider_id, model_id, timestamp);
CREATE INDEX idx_logs_success ON request_logs(success);
CREATE INDEX idx_logs_interface ON request_logs(interface_type);
CREATE INDEX idx_logs_cache ON request_logs(cache_hit);
CREATE INDEX idx_logs_api_key_name ON request_logs(api_key_name, timestamp);
```

**SQLite归档策略**:
```sql
-- 创建历史表（结构相同）
CREATE TABLE request_logs_archive AS SELECT * FROM request_logs WHERE 0;

-- 每月归档（保留最近3个月数据）
INSERT INTO request_logs_archive
SELECT * FROM request_logs
WHERE timestamp < datetime('now', '-3 months');

DELETE FROM request_logs
WHERE timestamp < datetime('now', '-3 months');

-- VACUUM回收空间
VACUUM;
```

### 8. api_keys

```sql
CREATE TABLE api_keys (
    id TEXT PRIMARY KEY,
    key_hash TEXT NOT NULL UNIQUE,
    key_prefix TEXT NOT NULL,
    name TEXT,
    group_id TEXT,
    rate_limit_config TEXT,  -- JSON string
    quota_config TEXT,  -- JSON string
    status TEXT NOT NULL DEFAULT 'active',
    expires_at TEXT,
    last_used_at TEXT,
    version INTEGER NOT NULL DEFAULT 0,
    metadata TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now')),

    FOREIGN KEY (group_id) REFERENCES groups(id) ON DELETE SET NULL
);

CREATE INDEX idx_api_keys_hash ON api_keys(key_hash);
CREATE INDEX idx_api_keys_status ON api_keys(status);
CREATE INDEX idx_api_keys_group ON api_keys(group_id);

CREATE TRIGGER trg_api_keys_updated
AFTER UPDATE ON api_keys
BEGIN
    UPDATE api_keys SET updated_at = datetime('now') WHERE id = NEW.id;
END;
```

### 9. provider_health_checks

```sql
CREATE TABLE provider_health_checks (
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

CREATE INDEX idx_health_provider_time ON provider_health_checks(provider_id, checked_at);
CREATE INDEX idx_health_status ON provider_health_checks(status);
```

### 10. circuit_breaker_states

```sql
CREATE TABLE circuit_breaker_states (
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

CREATE INDEX idx_circuit_state ON circuit_breaker_states(state);
CREATE INDEX idx_circuit_retry ON circuit_breaker_states(next_retry_at);
```

### 11. reconciliation_records

```sql
CREATE TABLE reconciliation_records (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    reconciliation_type TEXT NOT NULL,
    target_id TEXT NOT NULL,
    expected_value REAL,
    actual_value REAL,
    difference REAL,
    status TEXT NOT NULL,
    reconciled_at TEXT NOT NULL DEFAULT (datetime('now')),
    corrected_at TEXT,
    notes TEXT
);

CREATE INDEX idx_recon_type_status ON reconciliation_records(reconciliation_type, status);
CREATE INDEX idx_recon_target ON reconciliation_records(target_id);
```

### 12. admin_audit_logs

```sql
CREATE TABLE admin_audit_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    admin_token_prefix TEXT,
    action TEXT NOT NULL,
    resource_type TEXT NOT NULL,
    resource_id TEXT NOT NULL,
    changes TEXT,  -- JSON string
    ip_address TEXT,
    user_agent TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX idx_audit_resource ON admin_audit_logs(resource_type, resource_id);
CREATE INDEX idx_audit_action ON admin_audit_logs(action);
CREATE INDEX idx_audit_created ON admin_audit_logs(created_at);
```

---

## 第三部分：数据库归档策略

### MySQL归档

```sql
-- 按月自动分区管理
-- 添加新分区
ALTER TABLE request_logs ADD PARTITION (
    PARTITION p202604 VALUES LESS THAN (UNIX_TIMESTAMP('2026-05-01'))
);

-- 删除旧分区（数据会丢失，先备份）
ALTER TABLE request_logs DROP PARTITION p202601;
```

### SQLite归档

```bash
#!/bin/bash
# archive_logs.sh - 每月执行

DB_PATH="/app/data/gateway.db"
ARCHIVE_DATE=$(date -d "-3 months" +%Y-%m-%d)

# 导出旧数据到CSV
sqlite3 $DB_PATH <<EOF
.mode csv
.output /backup/logs_$(date +%Y%m).csv
SELECT * FROM request_logs WHERE timestamp < '$ARCHIVE_DATE';
.output stdout

-- 删除已归档数据
DELETE FROM request_logs WHERE timestamp < '$ARCHIVE_DATE';
VACUUM;
EOF

# 压缩归档文件
gzip /backup/logs_$(date +%Y%m).csv
```

---

## 第四部分：迁移工具

使用 [golang-migrate](https://github.com/golang-migrate/migrate) 管理数据库迁移。

### 迁移文件命名规范

```
migrations/
├── 000001_init_schema.up.sql
├── 000001_init_schema.down.sql
├── 000002_add_indexes.up.sql
├── 000002_add_indexes.down.sql
└── ...
```

### 迁移命令

```bash
# 创建新迁移
migrate create -ext sql -dir migrations -seq add_audit_logs

# 执行迁移
migrate -path migrations -database "sqlite3:///app/data/gateway.db" up

# 回滚
migrate -path migrations -database "sqlite3:///app/data/gateway.db" down 1

# 查看当前版本
migrate -path migrations -database "sqlite3:///app/data/gateway.db" version
```

---

## 第五部分：备份策略

### SQLite备份

```bash
#!/bin/bash
# backup_sqlite.sh

DB_PATH="/app/data/gateway.db"
BACKUP_DIR="/backup/sqlite"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)

# 使用SQLite在线备份
sqlite3 $DB_PATH ".backup '${BACKUP_DIR}/gateway_${TIMESTAMP}.db'"

# 压缩
gzip ${BACKUP_DIR}/gateway_${TIMESTAMP}.db

# 保留最近30天
find $BACKUP_DIR -name "*.gz" -mtime +30 -delete
```

### MySQL备份

```bash
#!/bin/bash
# backup_mysql.sh

BACKUP_DIR="/backup/mysql"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)

mysqldump \
    --host=localhost \
    --user=root \
    --password=${MYSQL_ROOT_PASSWORD} \
    --single-transaction \
    --routines \
    --triggers \
    aigateway | gzip > ${BACKUP_DIR}/aigateway_${TIMESTAMP}.sql.gz

# 保留最近30天
find $BACKUP_DIR -name "*.gz" -mtime +30 -delete
```
