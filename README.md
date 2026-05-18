# AI Gateway

企业级AI聚合网关，支持多服务商统一接入、智能路由、熔断降级、流式协议转换。

## 特性

- **多服务商支持**: OpenAI、Anthropic、Gemini、DeepSeek等
- **协议转换**: 自动转换不同服务商的API格式
- **智能路由**: 轮询、加权、最少连接、优先级、自适应等多种策略
- **高可用**: 熔断器、重试、降级、健康检查
- **流式支持**: SSE流式响应，支持客户端断开检测
- **配额管理**: Token配额跟踪、速率限制
- **可观测性**: 结构化日志、Prometheus指标、OpenTelemetry追踪
- **管理界面**: 现代化Web UI，液态玻璃设计风格

## 快速开始

### 本地开发

```bash
# 1. 克隆项目
git clone https://github.com/example/aigateway.git
cd aigateway

# 2. 安装依赖
go mod download

# 3. 启动Redis（必需）
docker run -d -p 6379:6379 redis:7-alpine

# 4. 初始化数据库
make migrate

# 5. 运行
make run
```

访问 http://localhost:8080/health 检查服务状态。

### Docker部署

```bash
# 复制环境变量文件并修改
cp .env.example .env

# 设置管理员Token
export ADMIN_TOKEN="admin-$(openssl rand -hex 16)"

# 启动服务
docker-compose -f docker/docker-compose.yaml up -d

# 查看日志
docker-compose logs -f gateway
```

## 配置

配置文件位于 `config/` 目录：

- `config.yaml` - 默认配置
- `config.dev.yaml` - 开发环境
- `config.staging.yaml` - 预发布环境
- `config.prod.yaml` - 生产环境

所有配置都可以通过环境变量覆盖，前缀为 `AIGATEWAY_`：

```bash
export AIGATEWAY_SERVER_PORT=8080
export AIGATEWAY_DATABASE_DSN=./data/gateway.db
export AIGATEWAY_REDIS_ADDR=localhost:6379
export AIGATEWAY_AUTH_ADMIN_TOKEN=admin-xxxxxxxx
```

## 数据库

### 开发/测试环境

使用SQLite（零配置）：

```yaml
database:
  driver: sqlite
  dsn: ./data/gateway.db
```

### 生产环境

**必须**使用MySQL或PostgreSQL（支持多实例）：

```yaml
database:
  driver: mysql
  dsn: "user:password@tcp(localhost:3306)/aigateway?charset=utf8mb4&parseTime=True"
```

## API文档

### 客户端API

```bash
# OpenAI格式
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer sk-xxxxxxxx" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4",
    "messages": [{"role": "user", "content": "Hello"}]
  }'
```

### 管理API

```bash
# 查看仪表盘
curl http://localhost:8080/admin/v1/dashboard/overview \
  -H "Authorization: Bearer admin-xxxxxxxx"

# 添加服务商
curl -X POST http://localhost:8080/admin/v1/providers \
  -H "Authorization: Bearer admin-xxxxxxxx" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "OpenAI",
    "base_url": "https://api.openai.com",
    "supported_formats": ["openai"]
  }'
```

完整API文档见 [docs/design/03-api-design.md](docs/design/03-api-design.md)

## 管理界面

访问 http://localhost:8080 打开Web管理界面。

默认管理员Token在配置文件中设置：

```yaml
auth:
  admin_token: "admin-xxxxxxxxxxxxxxxx"
```

## 架构

```
Client → Gateway → Protocol Converter → Router → Provider
         ↓
    [Auth, RateLimit, CircuitBreaker, Retry, Cache]
```

详细架构设计见 [docs/design/02-architecture-overview.md](docs/design/02-architecture-overview.md)

## 开发

### 项目结构

```
aigateway/
├── cmd/
│   ├── gateway/          # 程序入口
│   └── migrate/          # 数据库迁移工具
├── internal/
│   ├── handler/          # HTTP处理器
│   ├── service/          # 业务逻辑
│   ├── repository/       # 数据访问
│   ├── protocol/         # 协议转换
│   ├── routing/          # 负载均衡
│   ├── resilience/       # 熔断重试
│   └── ...
├── pkg/                  # 公共库
├── config/               # 配置文件
├── migrations/           # 数据库迁移（支持MySQL/PostgreSQL/SQLite）
├── docker/               # Docker配置
└── ui/                   # Web界面
```

### 运行测试

```bash
make test
```

### 代码检查

```bash
make lint
```

### 构建

```bash
make build
```

## 监控

### Prometheus指标

访问 http://localhost:9090/metrics

关键指标：
- `gateway_requests_total` - 总请求数
- `gateway_request_duration_seconds` - 请求延迟
- `gateway_tokens_total` - Token使用量

### Grafana仪表盘

使用 `docker-compose.yaml` 启动Grafana：

```bash
docker-compose up -d grafana
```

访问 http://localhost:3000（默认密码：admin/admin）

## 性能

在16核CPU、32GB内存的单实例上：

- 吞吐量: 10,000+ req/s
- 延迟P50: <10ms
- 延迟P99: <50ms
- 并发连接: 10,000+

*不含上游服务商响应时间*

## 安全

### Token存储

Provider Token明文存储（按设计要求），通过以下措施保护：

1. 数据库访问控制（最小权限）
2. 管理API独立认证
3. 所有操作记录审计日志
4. 日志中不记录Token明文
5. 数据库传输加密（TLS）
6. 定期备份，加密存储

### API Key管理

客户端API Key使用SHA256哈希存储，支持：

- 自动生成安全密钥
- 配额和速率限制
- 过期时间控制
- 撤销功能

## 故障排查

### Redis连接失败

网关会降级为单实例模式运行，但功能受限：

- 熔断器状态仅本地
- 分布式锁降级为本地锁
- 速率限制仅本地计数

**生产环境必须确保Redis可用**

### 数据库迁移

```bash
# 执行迁移（使用默认配置）
make migrate

# 或者手动运行迁移工具
go run ./cmd/migrate --config config/config.yaml up

# 回滚最近一次迁移
go run ./cmd/migrate --config config/config.yaml down --steps 1
```

### 查看日志

```bash
# Docker
docker-compose logs -f gateway

# 本地
tail -f logs/gateway.log
```

## 贡献

欢迎提交Issue和Pull Request。

## 许可证

MIT License

## 文档

- [数据库设计](docs/design/01-database-schema.md)
- [架构概览](docs/design/02-architecture-overview.md)
- [API设计](docs/design/03-api-design.md)
- [核心实现](docs/design/04-core-implementation-patterns.md)
- [部署配置](docs/design/05-deployment-config.md)
- [缓存设计](docs/design/06-llm-cache-design.md)
- [流式转换](docs/design/07-streaming-protocol-conversion.md)
- [UI设计](docs/design/08-ui-design.md)