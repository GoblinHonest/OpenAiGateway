# 部署与配置设计

## 1. 项目结构

```
aigateway/
├── cmd/
│   └── gateway/
│       └── main.go                 # 程序入口
├── internal/
│   ├── config/
│   │   ├── config.go               # 配置加载
│   │   └── types.go                # 配置类型定义
│   ├── server/
│   │   ├── server.go               # HTTP服务器
│   │   ├── router.go               # 路由注册
│   │   └── middleware/             # 中间件
│   │       ├── auth.go             # 认证
│   │       ├── ratelimit.go        # 速率限制
│   │       ├── logging.go          # 日志
│   │       ├── recovery.go         # Panic恢复
│   │       └── cors.go             # CORS
│   ├── handler/
│   │   ├── chat.go                 # Chat请求处理
│   │   ├── admin/
│   │   │   ├── provider.go         # Provider管理
│   │   │   ├── token.go            # Token管理
│   │   │   ├── model.go            # Model管理
│   │   │   ├── group.go            # Group管理
│   │   │   ├── apikey.go           # API Key管理
│   │   │   ├── stats.go            # 统计查询
│   │   │   └── logs.go             # 日志查询
│   │   └── health.go               # 健康检查
│   ├── service/
│   │   ├── gateway.go              # 网关核心逻辑
│   │   ├── provider.go             # Provider服务
│   │   ├── token.go                # Token服务
│   │   ├── model.go                # Model服务
│   │   ├── group.go                # Group服务
│   │   ├── apikey.go               # API Key服务
│   │   ├── stats.go                # 统计服务
│   │   └── reconciliation.go       # 对账服务
│   ├── domain/
│   │   ├── provider.go             # Provider模型
│   │   ├── token.go                # Token模型
│   │   ├── model.go                # Model模型
│   │   ├── group.go                # Group模型
│   │   ├── apikey.go               # API Key模型
│   │   └── request_log.go          # 请求日志模型
│   ├── repository/
│   │   ├── provider.go             # Provider数据访问
│   │   ├── token.go                # Token数据访问
│   │   ├── model.go                # Model数据访问
│   │   ├── group.go                # Group数据访问
│   │   ├── apikey.go               # API Key数据访问
│   │   ├── request_log.go          # 请求日志数据访问
│   │   └── circuit_breaker.go      # 熔断器状态数据访问
│   ├── protocol/
│   │   ├── converter.go            # 协议转换器接口
│   │   ├── openai.go               # OpenAI协议
│   │   ├── anthropic.go            # Anthropic协议
│   │   └── gemini.go               # Gemini协议
│   ├── routing/
│   │   ├── strategy.go             # 负载均衡策略接口
│   │   ├── roundrobin.go           # 轮询
│   │   ├── weighted.go             # 加权轮询
│   │   ├── leastconn.go            # 最少连接
│   │   ├── priority.go             # 优先级
│   │   └── adaptive.go             # 自适应
│   ├── resilience/
│   │   ├── circuitbreaker.go       # 熔断器
│   │   ├── retry.go                # 重试
│   │   └── fallback.go             # 降级
│   ├── client/
│   │   ├── httpclient.go           # HTTP客户端池
│   │   └── stream.go               # 流式处理
│   ├── health/
│   │   ├── checker.go              # 健康检查器
│   │   └── state.go                # 健康状态
│   ├── cache/
│   │   └── cache.go                # 缓存层
│   └── event/
│       └── eventbus.go             # 事件总线
├── pkg/
│   ├── logger/
│   │   └── logger.go               # 日志库
│   ├── metrics/
│   │   └── metrics.go              # Metrics库
│   ├── tracing/
│   │   └── tracing.go              # Tracing库
│   └── utils/
│       ├── hash.go                 # 哈希工具
│       ├── crypto.go               # 加密工具
│       └── time.go                 # 时间工具
├── migrations/
│   ├── 001_init_schema.sql         # 初始数据库迁移
│   └── 002_add_indexes.sql         # 索引迁移
├── config/
│   ├── config.yaml                 # 默认配置
│   ├── config.dev.yaml             # 开发环境
│   ├── config.staging.yaml         # 预发布环境
│   └── config.prod.yaml            # 生产环境
├── scripts/
│   ├── init-db.sql                 # 数据库初始化
│   └── seed-data.sql               # 初始数据
├── docker/
│   ├── Dockerfile                  # Docker镜像
│   ├── docker-compose.yaml         # Docker Compose
│   └── docker-compose.prod.yaml    # 生产环境Compose
├── k8s/
│   ├── deployment.yaml             # K8s部署
│   ├── service.yaml                # K8s服务
│   ├── configmap.yaml              # K8s配置
│   ├── secret.yaml                 # K8s密钥
│   ├── ingress.yaml                # K8s入口
│   ├── hpa.yaml                    # 自动扩缩容
│   └── pvc.yaml                    # 持久化存储
├── .github/
│   └── workflows/
│       ├── ci.yaml                 # CI流水线
│       └── cd.yaml                 # CD流水线
├── go.mod
├── go.sum
├── Makefile
├── .gitignore
└── README.md
```

## 2. 配置文件设计

### 2.1 主配置文件 (config.yaml)

```yaml
# AI Gateway 配置文件

# 服务器配置
server:
  host: "0.0.0.0"
  port: 8080
  read_timeout: 30s
  write_timeout: 60s
  idle_timeout: 120s
  max_header_bytes: 1048576  # 1MB
  graceful_shutdown_timeout: 30s

# 数据库配置
database:
  driver: "sqlite"           # sqlite, mysql, postgres
  dsn: "./data/gateway.db"

  # MySQL/PostgreSQL配置
  # driver: "mysql"
  # dsn: "user:password@tcp(localhost:3306)/aigateway?charset=utf8mb4&parseTime=True&loc=Local"

  # driver: "postgres"
  # dsn: "host=localhost user=postgres password=postgres dbname=aigateway port=5432 sslmode=disable"

  max_open_conns: 100
  max_idle_conns: 10
  conn_max_lifetime: 1h
  conn_max_idle_time: 30m

  # SQLite特有配置
  sqlite:
    journal_mode: "WAL"
    busy_timeout: 5000
    cache_size: -20000  # 20MB

# Redis配置
redis:
  addr: "localhost:6379"
  password: ""
  db: 0
  pool_size: 100
  min_idle_conns: 10
  dial_timeout: 5s
  read_timeout: 3s
  write_timeout: 3s
  pool_timeout: 4s

# 认证配置
auth:
  admin_token: "admin-xxxxxxxxxxxxxxxx"  # 管理API Token
  api_key_header: "Authorization"
  api_key_prefix: "Bearer "

# 速率限制配置
rate_limit:
  enabled: true
  default_rpm: 60           # 默认每分钟请求数
  default_tpm: 100000       # 默认每分钟Token数
  burst_size: 10            # 突发大小

# 熔断器配置
circuit_breaker:
  enabled: true
  failure_threshold: 5      # 失败次数阈值
  success_threshold: 2      # 成功次数阈值
  cooldown_duration: 60s    # 冷却时间

# 重试配置
retry:
  max_attempts: 3
  initial_backoff: 1s
  max_backoff: 10s
  backoff_multiplier: 2.0
  retryable_status_codes: [408, 429, 500, 502, 503, 504]

# 健康检查配置
health_check:
  enabled: true
  interval: 5m              # 检查间隔
  timeout: 10s              # 超时时间
  healthy_threshold: 2      # 健康阈值
  unhealthy_threshold: 3    # 不健康阈值

# 日志配置
log:
  level: "info"             # debug, info, warn, error
  format: "json"            # json, text
  output: "stdout"          # stdout, file
  file:
    path: "./logs/gateway.log"
    max_size: 100           # MB
    max_backups: 7
    max_age: 30             # days
    compress: true

# Metrics配置
metrics:
  enabled: true
  path: "/metrics"
  port: 9090                # 独立端口

# Tracing配置
tracing:
  enabled: false
  provider: "jaeger"        # jaeger, zipkin, otlp
  endpoint: "http://localhost:14268/api/traces"
  sample_rate: 0.1

# 对账配置
reconciliation:
  enabled: true
  schedule: "0 2 * * *"    # 每天凌晨2点
  types:
    - "token_quota"
    - "usage_stats"
    - "cost"

# 事件总线配置
event_bus:
  buffer_size: 1000
  workers: 4

# HTTP客户端配置
http_client:
  max_idle_conns: 100
  max_idle_conns_per_host: 10
  idle_conn_timeout: 90s
  tls_handshake_timeout: 10s
  expect_continue_timeout: 1s

# 流式配置
streaming:
  buffer_size: 4096
  max_chunk_size: 1024
  flush_interval: 100ms
  keepalive_interval: 15s

# 缓存配置
cache:
  enabled: true
  ttl: 5m
  max_size: 1000

# 协议配置
protocol:
  default_format: "openai"  # 默认响应格式
  auto_detect: true         # 自动检测请求格式
```

### 2.2 环境变量覆盖

所有配置都可以通过环境变量覆盖：

```bash
# 服务器
AIGATEWAY_SERVER_HOST=0.0.0.0
AIGATEWAY_SERVER_PORT=8080

# 数据库
AIGATEWAY_DATABASE_DRIVER=sqlite
AIGATEWAY_DATABASE_DSN=./data/gateway.db

# Redis
AIGATEWAY_REDIS_ADDR=localhost:6379
AIGATEWAY_REDIS_PASSWORD=secret

# 认证
AIGATEWAY_AUTH_ADMIN_TOKEN=admin-xxxxxxxxxxxxxxxx

# 日志
AIGATEWAY_LOG_LEVEL=info
AIGATEWAY_LOG_FORMAT=json
```

## 3. Docker部署

### 3.1 Dockerfile

```dockerfile
# 多阶段构建
FROM golang:1.21-alpine AS builder

# 安装SQLite依赖
RUN apk add --no-cache gcc musl-dev

WORKDIR /app

# 下载依赖
COPY go.mod go.sum ./
RUN go mod download

# 编译
COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -o gateway ./cmd/gateway

# 运行镜像
FROM alpine:3.18

# 安装SQLite运行时
RUN apk add --no-cache sqlite-libs ca-certificates tzdata

# 创建非root用户
RUN adduser -D -g '' appuser

WORKDIR /app

# 复制二进制文件
COPY --from=builder /app/gateway .
COPY --from=builder /app/config ./config

# 创建数据目录
RUN mkdir -p /app/data /app/logs && chown -R appuser:appuser /app

USER appuser

# 暴露端口
EXPOSE 8080 9090

# 健康检查
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD wget -qO- http://localhost:8080/health || exit 1

# 启动命令
ENTRYPOINT ["./gateway"]
CMD ["--config", "config/config.yaml"]
```

### 3.2 Docker Compose

```yaml
# docker-compose.yaml
version: '3.8'

services:
  gateway:
    build:
      context: .
      dockerfile: docker/Dockerfile
    container_name: aigateway
    ports:
      - "8080:8080"     # API端口
      - "9090:9090"     # Metrics端口
    volumes:
      - gateway-data:/app/data
      - gateway-logs:/app/logs
      - ./config/config.yaml:/app/config/config.yaml:ro
    environment:
      - AIGATEWAY_DATABASE_DSN=/app/data/gateway.db
      - AIGATEWAY_REDIS_ADDR=redis:6379
      - AIGATEWAY_AUTH_ADMIN_TOKEN=${ADMIN_TOKEN}
      - AIGATEWAY_LOG_LEVEL=info
    depends_on:
      redis:
        condition: service_healthy
    restart: unless-stopped
    networks:
      - gateway-net

  redis:
    image: redis:7-alpine
    container_name: aigateway-redis
    ports:
      - "6379:6379"
    volumes:
      - redis-data:/data
    # volatile-lru: 仅为设置了TTL的key执行淘汰，保护熔断器状态和分布式锁
    command: redis-server --appendonly yes --maxmemory 256mb --maxmemory-policy volatile-lru
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5
    restart: unless-stopped
    networks:
      - gateway-net

  # Prometheus (可选)
  prometheus:
    image: prom/prometheus:latest
    container_name: aigateway-prometheus
    ports:
      - "9091:9090"
    volumes:
      - ./docker/prometheus.yaml:/etc/prometheus/prometheus.yml:ro
      - prometheus-data:/prometheus
    restart: unless-stopped
    networks:
      - gateway-net

  # Grafana (可选)
  grafana:
    image: grafana/grafana:latest
    container_name: aigateway-grafana
    ports:
      - "3000:3000"
    volumes:
      - grafana-data:/var/lib/grafana
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=${GRAFANA_PASSWORD:-admin}
    restart: unless-stopped
    networks:
      - gateway-net

volumes:
  gateway-data:
  gateway-logs:
  redis-data:
  prometheus-data:
  grafana-data:

networks:
  gateway-net:
    driver: bridge
```

## 4. Kubernetes部署

### 4.1 ConfigMap

```yaml
# k8s/configmap.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: aigateway-config
  namespace: aigateway
data:
  config.yaml: |
    server:
      host: "0.0.0.0"
      port: 8080

    # 生产环境使用MySQL/PostgreSQL，支持多副本
    database:
      driver: "mysql"
      dsn: "aigateway:${MYSQL_PASSWORD}@tcp(mysql-service:3306)/aigateway?charset=utf8mb4&parseTime=True&loc=Local"
      max_open_conns: 100
      max_idle_conns: 10

    redis:
      addr: "redis-service:6379"
      password: "${REDIS_PASSWORD}"

    log:
      level: "info"
      format: "json"

    metrics:
      enabled: true
      path: "/metrics"

    health_check:
      enabled: true
      interval: 5m
```

### 4.2 Secret

```yaml
# k8s/secret.yaml
apiVersion: v1
kind: Secret
metadata:
  name: aigateway-secret
  namespace: aigateway
type: Opaque
stringData:
  admin-token: "admin-xxxxxxxxxxxxxxxx"
  redis-password: ""
```

### 4.3 Deployment

```yaml
# k8s/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: aigateway
  namespace: aigateway
  labels:
    app: aigateway
spec:
  replicas: 2
  selector:
    matchLabels:
      app: aigateway
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 0
  template:
    metadata:
      labels:
        app: aigateway
      annotations:
        prometheus.io/scrape: "true"
        prometheus.io/port: "9090"
        prometheus.io/path: "/metrics"
    spec:
      serviceAccountName: aigateway
      securityContext:
        runAsNonRoot: true
        runAsUser: 1000
        fsGroup: 1000
      containers:
        - name: gateway
          image: aigateway:latest
          imagePullPolicy: Always
          ports:
            - containerPort: 8080
              name: http
              protocol: TCP
            - containerPort: 9090
              name: metrics
              protocol: TCP
          env:
            - name: AIGATEWAY_AUTH_ADMIN_TOKEN
              valueFrom:
                secretKeyRef:
                  name: aigateway-secret
                  key: admin-token
            - name: AIGATEWAY_REDIS_ADDR
              value: "redis-service:6379"
            - name: AIGATEWAY_REDIS_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: aigateway-secret
                  key: redis-password
            - name: AIGATEWAY_LOG_LEVEL
              value: "info"
          volumeMounts:
            - name: config
              mountPath: /app/config
              readOnly: true
            - name: data
              mountPath: /app/data
            - name: logs
              mountPath: /app/logs
          resources:
            requests:
              cpu: "500m"
              memory: "512Mi"
            limits:
              cpu: "2000m"
              memory: "2Gi"
          livenessProbe:
            httpGet:
              path: /health
              port: 8080
            initialDelaySeconds: 10
            periodSeconds: 30
            timeoutSeconds: 5
            failureThreshold: 3
          readinessProbe:
            httpGet:
              path: /health
              port: 8080
            initialDelaySeconds: 5
            periodSeconds: 10
            timeoutSeconds: 3
            failureThreshold: 3
          startupProbe:
            httpGet:
              path: /health
              port: 8080
            initialDelaySeconds: 0
            periodSeconds: 5
            failureThreshold: 30
      volumes:
        - name: config
          configMap:
            name: aigateway-config
        - name: logs
          emptyDir: {}
      terminationGracePeriodSeconds: 30
```

**重要说明**: 生产环境K8s部署必须使用MySQL/PostgreSQL，不支持SQLite。SQLite仅用于开发/测试环境。

### 4.4 Service

```yaml
# k8s/service.yaml
apiVersion: v1
kind: Service
metadata:
  name: aigateway-service
  namespace: aigateway
  labels:
    app: aigateway
spec:
  type: ClusterIP
  ports:
    - port: 8080
      targetPort: http
      protocol: TCP
      name: http
    - port: 9090
      targetPort: metrics
      protocol: TCP
      name: metrics
  selector:
    app: aigateway
```

### 4.5 Ingress

```yaml
# k8s/ingress.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: aigateway-ingress
  namespace: aigateway
  annotations:
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
    nginx.ingress.kubernetes.io/proxy-body-size: "10m"
    nginx.ingress.kubernetes.io/proxy-read-timeout: "60"
    nginx.ingress.kubernetes.io/proxy-send-timeout: "60"
    cert-manager.io/cluster-issuer: "letsencrypt-prod"
spec:
  ingressClassName: nginx
  tls:
    - hosts:
        - api.example.com
      secretName: aigateway-tls
  rules:
    - host: api.example.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: aigateway-service
                port:
                  number: 8080
```

### 4.6 HPA (自动扩缩容)

```yaml
# k8s/hpa.yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: aigateway-hpa
  namespace: aigateway
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: aigateway
  minReplicas: 2
  maxReplicas: 10
  metrics:
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 70
    - type: Resource
      resource:
        name: memory
        target:
          type: Utilization
          averageUtilization: 80
    - type: Pods
      pods:
        metric:
          name: http_requests_per_second
        target:
          type: AverageValue
          averageValue: "1000"
  behavior:
    scaleDown:
      stabilizationWindowSeconds: 300
      policies:
        - type: Percent
          value: 10
          periodSeconds: 60
    scaleUp:
      stabilizationWindowSeconds: 60
      policies:
        - type: Percent
          value: 50
          periodSeconds: 60
```

### 4.7 MySQL (生产环境数据库)

```yaml
# k8s/mysql.yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: mysql
  namespace: aigateway
spec:
  serviceName: mysql-service
  replicas: 1
  selector:
    matchLabels:
      app: mysql
  template:
    metadata:
      labels:
        app: mysql
    spec:
      containers:
        - name: mysql
          image: mysql:8.0
          ports:
            - containerPort: 3306
          env:
            - name: MYSQL_ROOT_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: mysql-secret
                  key: root-password
            - name: MYSQL_DATABASE
              value: aigateway
            - name: MYSQL_USER
              value: aigateway
            - name: MYSQL_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: mysql-secret
                  key: user-password
          volumeMounts:
            - name: mysql-data
              mountPath: /var/lib/mysql
          resources:
            requests:
              cpu: "500m"
              memory: "1Gi"
            limits:
              cpu: "2000m"
              memory: "4Gi"
  volumeClaimTemplates:
    - metadata:
        name: mysql-data
      spec:
        accessModes: ["ReadWriteOnce"]
        storageClassName: standard
        resources:
          requests:
            storage: 50Gi
---
apiVersion: v1
kind: Service
metadata:
  name: mysql-service
  namespace: aigateway
spec:
  clusterIP: None
  ports:
    - port: 3306
      targetPort: 3306
  selector:
    app: mysql
```

## 5. CI/CD流水线

### 5.1 GitHub Actions CI

```yaml
# .github/workflows/ci.yaml
name: CI

on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main]

jobs:
  test:
    name: Test
    runs-on: ubuntu-latest

    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.21'

      - name: Cache Go modules
        uses: actions/cache@v3
        with:
          path: ~/go/pkg/mod
          key: ${{ runner.os }}-go-${{ hashFiles('**/go.sum') }}
          restore-keys: |
            ${{ runner.os }}-go-

      - name: Install dependencies
        run: |
          sudo apt-get update
          sudo apt-get install -y gcc  # SQLite需要CGO
          go mod download

      - name: Run linter
        uses: golangci/golangci-lint-action@v3
        with:
          version: latest

      - name: Run tests
        run: go test -v -race -coverprofile=coverage.out ./...

      - name: Upload coverage
        uses: codecov/codecov-action@v3
        with:
          file: ./coverage.out

  build:
    name: Build
    needs: test
    runs-on: ubuntu-latest

    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.21'

      - name: Build
        run: |
          CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o gateway ./cmd/gateway

      - name: Upload artifact
        uses: actions/upload-artifact@v3
        with:
          name: gateway
          path: gateway

  docker:
    name: Docker Build
    needs: build
    runs-on: ubuntu-latest
    if: github.event_name == 'push' && github.ref == 'refs/heads/main'

    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Set up Docker Buildx
        uses: docker/setup-buildx-action@v3

      - name: Login to DockerHub
        uses: docker/login-action@v3
        with:
          username: ${{ secrets.DOCKERHUB_USERNAME }}
          password: ${{ secrets.DOCKERHUB_TOKEN }}

      - name: Build and push
        uses: docker/build-push-action@v5
        with:
          context: .
          push: true
          tags: |
            ${{ secrets.DOCKERHUB_USERNAME }}/aigateway:latest
            ${{ secrets.DOCKERHUB_USERNAME }}/aigateway:${{ github.sha }}
          cache-from: type=gha
          cache-to: type=gha,mode=max
```

### 5.2 GitHub Actions CD

```yaml
# .github/workflows/cd.yaml
name: CD

on:
  push:
    tags:
      - 'v*'

jobs:
  release:
    name: Create Release
    runs-on: ubuntu-latest

    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Create Release
        uses: softprops/action-gh-release@v1
        with:
          generate_release_notes: true
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}

  deploy:
    name: Deploy
    needs: release
    runs-on: ubuntu-latest

    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Setup kubectl
        uses: azure/setup-kubectl@v3

      - name: Configure kubeconfig
        run: |
          mkdir -p $HOME/.kube
          echo "${{ secrets.KUBECONFIG }}" | base64 -d > $HOME/.kube/config

      - name: Update image tag
        run: |
          sed -i "s|image: aigateway:.*|image: ${{ secrets.DOCKERHUB_USERNAME }}/aigateway:${{ github.ref_name }}|g" k8s/deployment.yaml

      - name: Deploy to Kubernetes
        run: |
          kubectl apply -f k8s/
          kubectl rollout status deployment/aigateway -n aigateway --timeout=300s
```

## 6. 监控配置

### 6.1 Prometheus配置

```yaml
# docker/prometheus.yaml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: 'aigateway'
    static_configs:
      - targets: ['gateway:9090']
    metrics_path: /metrics
    scrape_interval: 10s

  - job_name: 'redis'
    static_configs:
      - targets: ['redis-exporter:9121']

  - job_name: 'node'
    static_configs:
      - targets: ['node-exporter:9100']
```

### 6.2 Grafana Dashboard

```json
{
  "dashboard": {
    "title": "AI Gateway",
    "panels": [
      {
        "title": "Request Rate",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(gateway_requests_total[5m])",
            "legendFormat": "{{provider}} - {{model}}"
          }
        ]
      },
      {
        "title": "Latency P99",
        "type": "graph",
        "targets": [
          {
            "expr": "histogram_quantile(0.99, rate(gateway_request_duration_seconds_bucket[5m]))",
            "legendFormat": "{{provider}} - {{model}}"
          }
        ]
      },
      {
        "title": "Error Rate",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(gateway_requests_total{status!=\"success\"}[5m]) / rate(gateway_requests_total[5m])",
            "legendFormat": "{{provider}} - {{model}}"
          }
        ]
      },
      {
        "title": "Token Usage",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(gateway_tokens_total[5m])",
            "legendFormat": "{{provider}} - {{type}}"
          }
        ]
      }
    ]
  }
}
```

## 7. Makefile

```makefile
# Makefile

.PHONY: build test run clean docker

# 变量
APP_NAME := gateway
BUILD_DIR := ./bin
DOCKER_IMAGE := aigateway

# 构建
build:
	CGO_ENABLED=1 go build -o $(BUILD_DIR)/$(APP_NAME) ./cmd/gateway

# 测试
test:
	go test -v -race -coverprofile=coverage.out ./...

# 运行
run: build
	$(BUILD_DIR)/$(APP_NAME) --config config/config.yaml

# 清理
clean:
	rm -rf $(BUILD_DIR)
	rm -f coverage.out

# Docker构建
docker:
	docker build -t $(DOCKER_IMAGE) -f docker/Dockerfile .

# Docker运行
docker-run:
	docker-compose -f docker/docker-compose.yaml up -d

# 数据库迁移
migrate:
	go run ./cmd/migrate up

# 代码检查
lint:
	golangci-lint run ./...

# 格式化
fmt:
	go fmt ./...

# 依赖整理
tidy:
	go mod tidy

# 帮助
help:
	@echo "Available targets:"
	@echo "  build       - Build the application"
	@echo "  test        - Run tests"
	@echo "  run         - Build and run the application"
	@echo "  clean       - Clean build artifacts"
	@echo "  docker      - Build Docker image"
	@echo "  docker-run  - Run with Docker Compose"
	@echo "  migrate     - Run database migrations"
	@echo "  lint        - Run linter"
	@echo "  fmt         - Format code"
	@echo "  tidy        - Tidy dependencies"
```

## 8. 快速启动

### 8.1 本地开发

```bash
# 1. 克隆项目
git clone https://github.com/example/aigateway.git
cd aigateway

# 2. 安装依赖
go mod download

# 3. 创建配置
cp config/config.yaml config/config.local.yaml
# 编辑 config.local.yaml

# 4. 初始化数据库
make migrate

# 5. 运行
make run
```

### 8.2 Docker部署

```bash
# 1. 设置环境变量
export ADMIN_TOKEN="admin-$(openssl rand -hex 16)"

# 2. 启动服务
docker-compose -f docker/docker-compose.yaml up -d

# 3. 查看日志
docker-compose logs -f gateway

# 4. 健康检查
curl http://localhost:8080/health
```

### 8.3 Kubernetes部署

```bash
# 1. 创建命名空间
kubectl create namespace aigateway

# 2. 创建Secret
kubectl create secret generic aigateway-secret \
  --from-literal=admin-token=admin-$(openssl rand -hex 16) \
  -n aigateway

# 3. 部署
kubectl apply -f k8s/

# 4. 查看状态
kubectl get pods -n aigateway

# 5. 查看日志
kubectl logs -f deployment/aigateway -n aigateway
```
