# 软件设计文档

## 0. 文档元信息
- 项目：`GopherMind`
- 版本：`v0.1`
- 日期：`2026-03-12`
- 作者：`Codex + User`

## 1. 需求摘要
### 1.1 业务目标
- 构建一个可生产落地的后端，支持多会话 AI 问答，并可选基于知识库的回答。
- 同时支持云端与本地模型提供方，在质量、成本与可用性之间取得平衡。
- 为聊天提供实时流式体验，并为文档导入/索引任务提供稳健的异步处理能力。
- 确保可观测性、可靠性，以及用户与文档数据的安全处理。

### 1.2 功能需求
- FR-1：所有受保护 API 需具备用户认证与授权能力。
- FR-2：会话生命周期管理（创建/列表/历史查询/归档）。
- FR-3：同步与流式聊天（SSE 为主，WebSocket 可选/升级路径）。
- FR-4：在 `OpenAI`、`Ollama`、`BGE` 重排器之间进行多模型路由。
- FR-5：文档导入流水线（上传 -> 解析 -> 递归切分 -> 向量化 -> 建索引 -> 状态更新）。
- FR-6：RAG 查询流程（`Qdrant` 检索 + `BGE` 重排 + KG 占位 + 生成）。
- FR-7：通过 RabbitMQ 执行异步任务，支持重试/DLQ/幂等。
- FR-8：在 MySQL 中持久化用户、会话、消息、文档与作业数据。
- FR-9：在 Redis Cluster 中维护热点缓存与短生命周期状态。
- FR-10：结构化日志、链路追踪、指标、看板与告警。

### 1.3 非功能需求
- NFR-1：非 RAG 场景下聊天首 Token 延迟 P95 <= 2.0s。
- NFR-2：完整响应延迟 P95 <= 8.0s（文本 <= 700 tokens）。
- NFR-3：API 月可用性 >= 99.9%。
- NFR-4：异步任务至少一次处理语义，消费者需保证幂等。
- NFR-5：API 与 Worker 支持水平扩展。
- NFR-6：安全基线包含 JWT 鉴权、密钥隔离、审计日志、PII 最小化。

### 1.4 假设与约束
由于未显式提供全部输入，采用以下默认值：
- `project_name`：`GopherMind`
- `requirement_description`："Backend AI assistant with session chat, streaming output, document-grounded RAG, and async ingestion."
- `models_config`：`OpenAI / Ollama / BGE`
- `redis_cluster`：
  - `redis-node1:6379`
  - `redis-node2:6379`
  - `redis-node3:6379`
- `mysql_dsn`：`root:password@tcp(mysql:3306)/gophermind`
- `rabbitmq_url`：`amqp://guest:guest@rabbitmq:5672/`

约束：
- 后端语言：Go。
- RAG 索引/检索服务在需要时可调用 Python 组件。
- 需保证流式输出在现代浏览器与反向代理环境下兼容。

需求到模块/接口映射：

| 需求 | 主模块 | 接口契约 |
|---|---|---|
| FR-1 | API 网关 + 认证 | `POST /api/v1/auth/login`，JWT 中间件 |
| FR-2 | 会话服务 | `POST /api/v1/sessions`，`GET /api/v1/sessions` |
| FR-3 | 聊天服务 | `POST /api/v1/chat/send`，`POST /api/v1/chat/stream` |
| FR-4 | 模型编排器 | `Generate(ctx, req)`，provider 适配器 |
| FR-5 | 导入 Worker | `doc.ingest.request.v1` RabbitMQ 消息 |
| FR-6 | RAG 服务 | `RAGQuery(ctx, query, userID, sessionID)` |
| FR-7 | 异步队列 | `x.gophermind.doc`，重试 + DLQ 队列 |
| FR-8 | 持久化层 | MySQL Schema 与事务 DAO API |
| FR-9 | 缓存层 | Redis 键空间契约（`sess:*`，`rate:*`） |
| FR-10 | 可观测性 | OTel traces/metrics + Zap 结构化日志 |

## 2. 系统架构与模块拆分
### 2.1 高层架构
文本示意图：

```text
Client(Web/App)
  -> API Gateway(Gin, JWT, rate limit)
    -> Session/Chat Service
      -> Model Orchestrator
        -> OpenAI Adapter
        -> Ollama Adapter
      -> RAG Service
        -> Query Rewriter(optional)
        -> Qdrant Retriever
        -> BGE Reranker
        -> KG Placeholder
      -> Redis Cluster (cache/state)
      -> MySQL (system of record)
      -> RabbitMQ (async ingestion/index jobs)
        -> Worker Pool
          -> Python Ingestion Pipeline
```

在线问答时序流程：

```mermaid
sequenceDiagram
  participant C as Client
  participant API as API Gateway
  participant S as Session Service
  participant R as RAG Service
  participant Q as Qdrant
  participant M as Model Orchestrator
  participant O as OpenAI/Ollama
  participant DB as MySQL
  C->>API: POST /chat/stream {sessionId, query}
  API->>S: Validate JWT + load session
  S->>R: Retrieve(query,userId)
  R->>Q: topK vector search
  Q-->>R: candidates
  R-->>S: reranked context + citations
  S->>M: Generate(stream=true, context)
  M->>O: provider request
  O-->>M: token stream
  M-->>API: chunks
  API-->>C: SSE data events
  S->>DB: persist user+assistant messages (async-safe write)
```

### 2.2 核心模块
- API 网关 / HTTP 层
  - 请求校验、认证鉴权、限流、协议适配（REST/SSE/WebSocket）。
- 会话与对话服务
  - 会话状态、消息追加、历史查询、对话策略。
- RAG 流水线服务
  - 查询改写、检索、重排、上下文打包、引用元数据。
- 模型编排层
  - Provider 路由、超时/重试/熔断、Token 预算控制。
- 缓存层（Redis Cluster）
  - 会话快照、短时上下文缓存、幂等键、限流计数。
- 持久化层（MySQL）
  - 用户/会话/消息/文档/作业作为持久化系统记录。
- 异步 Worker 层（RabbitMQ Consumers）
  - 导入/索引任务、重试、DLQ、故障恢复与回放。
- 可观测性层
  - Zap 日志、OTel 链路与指标、看板和告警。

### 2.3 模块职责矩阵
| 模块 | 职责 | 输入 | 输出 | 依赖 |
|---|---|---|---|---|
| API Gateway | 校验/鉴权/路由/流式输出 | HTTP 请求 | JSON/SSE/WS 事件 | JWT，会话服务 |
| Session Service | 会话与历史生命周期管理 | 用户/会话/消息操作 | 持久化消息，会话元数据 | MySQL，Redis |
| RAG Service | 构建可溯源上下文 | 查询 + 用户作用域 | 排序片段 + 引用 | Qdrant，BGE |
| Model Orchestrator | 生成请求分发 | prompt + context + policy | completion/token stream | OpenAI/Ollama |
| Ingestion Worker | 文档索引 | RabbitMQ 导入消息 | Qdrant 向量 + 状态 | Python pipeline，MySQL |
| Cache Layer | 低延迟临时状态 | key/value 操作 | 缓存值 | Redis Cluster |
| Observability | 遥测与告警 | logs/traces/metrics | dashboards/alerts | OTel backend |

## 3. Redis Cluster 设计
### 3.1 键空间与数据模型
- `sess:{user_id}:{session_id}:summary`（String/JSON）：用于 prompt 压缩的滚动摘要。
- `sess:{user_id}:{session_id}:ctx`（Hash）：最近轮次元数据、模型信息、token 使用量。
- `chat:stream:{request_id}`（String）：重连时使用的流游标/状态。
- `idempotency:{consumer}:{msg_id}`（String）：异步去重键。
- `rate:{user_id}:{minute_bucket}`（String counter）：分钟级配额计数。
- `model:fallback:state:{provider}`（String）：Provider 健康/熔断状态。

### 3.2 TTL 与淘汰策略
- 会话摘要/上下文键：`TTL 24h`。
- 流状态键：`TTL 10m`。
- 幂等键：`TTL 48h`。
- 限流计数键：`TTL 2m`。
- Maxmemory 策略：纯缓存节点使用 `allkeys-lru`；关键业务数据不只保存在 Redis。

### 3.3 持久化与数据安全
- 启用 AOF（`appendonly yes`），刷盘策略 `everysec`。
- 每 15 分钟做一次 RDB 快照，加速重启恢复。
- Redis 作为加速层，MySQL 仍是事实来源（source of truth）。

### 3.4 Cache-Aside / 写入策略
- 读路径：先读 Redis -> 回退 MySQL -> 回填 Redis（带 TTL）。
- 写路径：先提交 MySQL 事务，再异步失效/更新缓存。
- 流式元数据写透 Redis，随后异步刷入 MySQL。

### 3.5 降级与回退方案
- Redis 不可用时：
  - 会话/历史退化为 MySQL-only 模式。
  - 关闭非关键能力：可重连流游标与热点摘要。
  - 放宽 API 延迟预算，并上报 `degraded_mode=true` 指标标签。

### 3.6 Redis 配置与伪代码示例
配置示例：

```yaml
redis_cluster:
  addrs:
    - redis-node1:6379
    - redis-node2:6379
    - redis-node3:6379
  pool_size: 200
  dial_timeout_ms: 100
  read_timeout_ms: 150
  write_timeout_ms: 150
```

Go 风格伪代码：

```go
func LoadSessionSummary(ctx context.Context, userID, sessionID string) (Summary, error) {
  key := fmt.Sprintf("sess:%s:%s:summary", userID, sessionID)
  raw, err := redis.Get(ctx, key)
  if err == nil {
    return decodeSummary(raw), nil
  }
  summary, err := repo.GetLatestSummary(ctx, userID, sessionID)
  if err != nil { return Summary{}, err }
  _ = redis.SetEX(ctx, key, encode(summary), 24*time.Hour)
  return summary, nil
}
```

## 4. MySQL 设计
### 4.1 实体与表概览
- `users`：身份与账号状态。
- `sessions`：用户级会话容器。
- `messages`：有序聊天轮次与元数据。
- `documents`：上传文档及其索引生命周期。
- `jobs`：异步作业状态机。
- `model_usage`：Token/成本统计，用于治理。

### 4.2 表 DDL 示例
```sql
CREATE TABLE users (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  username VARCHAR(64) NOT NULL UNIQUE,
  email VARCHAR(128) NOT NULL UNIQUE,
  password_hash VARCHAR(255) NOT NULL,
  status TINYINT NOT NULL DEFAULT 1,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

CREATE TABLE sessions (
  id CHAR(36) PRIMARY KEY,
  user_id BIGINT NOT NULL,
  title VARCHAR(255) NOT NULL,
  model_pref VARCHAR(32) NULL,
  last_message_at DATETIME NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_sessions_user_updated (user_id, updated_at DESC),
  CONSTRAINT fk_sessions_user FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE TABLE messages (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  session_id CHAR(36) NOT NULL,
  user_id BIGINT NOT NULL,
  role ENUM('system','user','assistant','tool') NOT NULL,
  content MEDIUMTEXT NOT NULL,
  token_count INT NULL,
  provider VARCHAR(32) NULL,
  model_name VARCHAR(64) NULL,
  request_id CHAR(36) NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_messages_session_time (session_id, created_at),
  INDEX idx_messages_request (request_id),
  CONSTRAINT fk_messages_session FOREIGN KEY (session_id) REFERENCES sessions(id)
);

CREATE TABLE documents (
  id CHAR(36) PRIMARY KEY,
  user_id BIGINT NOT NULL,
  filename VARCHAR(255) NOT NULL,
  storage_uri VARCHAR(512) NOT NULL,
  status ENUM('uploaded','queued','indexing','ready','failed') NOT NULL,
  checksum CHAR(64) NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_documents_user_status (user_id, status)
);

CREATE TABLE jobs (
  id CHAR(36) PRIMARY KEY,
  job_type VARCHAR(64) NOT NULL,
  dedupe_key VARCHAR(128) NOT NULL,
  status ENUM('queued','running','succeeded','failed','dead') NOT NULL,
  retry_count INT NOT NULL DEFAULT 0,
  payload JSON NOT NULL,
  error_msg VARCHAR(1000) NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uk_jobs_dedupe (job_type, dedupe_key)
);
```

### 4.3 事务边界
- `CreateSession + first user message`：单事务。
- `Append user message + reserve request_id`：模型调用前单事务。
- `Persist assistant response`：基于 `request_id` 乐观校验的事务。
- `Document state updates`：每次状态迁移单事务，并带单调性保护。

### 4.4 索引与查询优化
- 用户维度会话列表：覆盖索引 `(user_id, updated_at)`。
- 会话消息历史：索引 `(session_id, created_at)`。
- 作业回放/恢复：建议复合索引 `(status, updated_at)`。
- 文档就绪检查：`(user_id, status)`。

### 4.5 迁移与向后兼容
- 先做增量迁移（新增可空字段/表）。
- 异步回填，验证后切读。
- 至少保留一整个发布周期的旧 API 字段。

### 4.6 数据保留与归档
- 消息默认保留 180 天（可配置）。
- 作业数据保留 30 天用于排障。
- 合规要求下，会话/文档支持软删除与延迟清理。

## 5. RabbitMQ 异步队列设计
### 5.1 Exchange/Queue/Binding 拓扑
- Exchange：`x.gophermind.doc`（topic）。
- 主队列：`q.doc.ingest`。
- 重试队列：`q.doc.ingest.retry`（带 TTL，死信回主队列）。
- 死信队列（DLQ）：`q.doc.ingest.dlq`。
- 路由键：
  - `doc.ingest.request.v1`
  - `doc.ingest.retry.v1`
  - `doc.ingest.dead.v1`

### 5.2 消息结构与版本化
```json
{
  "event_type": "doc.ingest.request",
  "version": "v1",
  "job_id": "0d86c8d7-1de5-43bc-aab1-6a2ea9d1f21f",
  "dedupe_key": "user123:file_sha256",
  "user_id": 123,
  "document_id": "9b2d8fd8-9b56-4f93-9cf6-266d0506803a",
  "storage_uri": "s3://bucket/u123/a.pdf",
  "created_at": "2026-03-12T10:00:00Z"
}
```

### 5.3 重试、DLQ 与幂等
- 消费前先检查 Redis/MySQL 幂等状态。
- 重试策略：指数退避（`5s`、`30s`、`2m`、`10m`，最多 5 次）。
- 超过最大次数后携带失败原因进入 DLQ 并触发告警。

### 5.4 消费并发与扩缩容
- 每实例 Worker 初始并发 `N=8`。
- HPA 扩缩容信号：队列深度 + 处理延迟。
- 调优 prefetch 避免队头阻塞（基线 `prefetch=16`）。

### 5.5 运行指标与告警
- 队列深度、消费延迟、重试率、DLQ 流入、成功率、处理时延。
- 告警示例：
  - DLQ 流入 > 20/min，持续 5 分钟。
  - 成功率 < 95%，持续 10 分钟。
  - 最老消息年龄 > 15 分钟。

## 6. 模型调用治理
### 6.1 支持模型与路由规则
- `OpenAI`：
  - 高质量、复杂推理场景默认使用。
- `Ollama`：
  - OpenAI 不可用或预算超阈值时回退。
- `BGE`：
  - 专用于检索片段重排，不作为最终生成模型。

路由策略：
- 若请求需要 grounding 与重排：先检索 + BGE，再生成。
- 若 Provider 健康降级：OpenAI -> Ollama 自动切换。
- 若用户策略要求本地模型：强制使用 Ollama。

### 6.2 Prompt 与上下文组装
- Prompt 分段：
  - system policy
  - session summary
  - 最新用户轮次
  - RAG 引用/上下文块
- 最大上下文预算裁剪优先级：system > 检索证据 > 最近轮次 > 历史轮次。

### 6.3 超时、熔断与回退
- Provider 超时：首 Token `10s`，总时长 `45s`。
- 60 秒内连续 5 次失败触发熔断。
- 回退阶梯：
  - 同一 Provider 重试一次（瞬时错误）
  - 切换 Provider
  - 若检索链路失败，降级为非 RAG 精简回答

### 6.4 成本与 Token 预算控制
- 单请求输入/输出 Token 上限。
- 用户日配额与异常检测。
- 使用记录持久化到 `model_usage`。

### 6.5 安全/策略控制
- 检索前与生成前执行 Prompt 注入防护规则。
- 输出审核钩子（严重度可配置）。
- 在日志和追踪中脱敏敏感数据。

## 7. RAG 问答流水线
### 7.1 导入流程
- 第 1 步：解析上传文件为标准化文本。
- 第 2 步：递归切分（`chunk_size=600`，`overlap=120`，分隔符层级策略）。
- 第 3 步：生成稠密向量。
- 第 4 步：向 `Qdrant` upsert 向量与元数据。
- 第 5 步：文档状态置为 `ready`。

### 7.2 查询流程
- 对歧义输入可选执行查询改写。
- 从 Qdrant 检索候选（`topK=20`）。
- 使用 BGE 重排（最终上下文块 `topN=5`）。
- 知识图谱占位模块在可用时补充实体/关系信息。
- 组装 grounding prompt，经 OpenAI/Ollama 生成。

### 7.3 准确性与护栏策略
- 事实性陈述需具备引用覆盖。
- 若检索置信度低于阈值，返回不确定性模板。
- 过滤不符合策略的上下文块（PII/毒性约束）。

### 7.4 分阶段时延预算
- 查询改写：<= 100ms
- Qdrant 检索：<= 150ms
- BGE 重排：<= 250ms
- Prompt 组装：<= 50ms
- 模型首 Token：<= 1200ms
- 端到端 P95：首 Token <= 2000ms / 完整响应 <= 8000ms

### 7.5 流水线伪代码
Python 风格伪代码：

```python
def answer_query(user_id, session_id, query):
    rewritten = maybe_rewrite(query)
    candidates = qdrant.search(namespace=user_id, text=rewritten, top_k=20)
    ranked = bge_rerank(query=rewritten, docs=candidates)[:5]
    kg_ctx = kg_placeholder.lookup_entities(rewritten)  # optional
    prompt = compose_prompt(system_policy(), session_summary(user_id, session_id), ranked, kg_ctx, query)
    provider = route_provider(user_id, prompt)
    return provider.generate_stream(prompt)
```

## 8. 流式输出设计
### 8.1 WebSocket 路径
- 使用 `/api/v1/chat/ws` 提供双向实时会话。
- 支持 tool-calls 与交互控制信号。
- 适用于需要全双工能力的高级客户端。

### 8.2 SSE 路径
- 使用 `/api/v1/chat/stream`，`Content-Type: text/event-stream`。
- 代理兼容性与浏览器支持更简单。
- 当前 Web 聊天默认采用 SSE。

### 8.3 背压、取消与恢复
- 背压：
  - Provider 适配层与 HTTP Writer 之间使用有界通道。
- 取消：
  - 客户端断开或超时时触发 context cancel。
- 恢复：
  - 使用 `request_id` 重连；若 Redis 中存在缓冲片段则回放。

### 8.4 消息/事件契约示例
SSE 事件：

```text
event: meta
data: {"request_id":"r123","session_id":"s123","model":"openai:gpt-4.1"}

event: token
data: {"delta":"Hello"}

event: done
data: {"finish_reason":"stop","usage":{"input_tokens":342,"output_tokens":211}}
```

WebSocket 帧载荷：

```json
{
  "type": "token",
  "request_id": "r123",
  "delta": "Hello",
  "seq": 18
}
```

权衡：
- SSE 作为默认方案，运维复杂度更低。
- WebSocket 预留给后续交互式工具与更丰富客户端控制能力。

## 9. 日志与可观测性
### 9.1 日志规范（Zap）
- 必填字段：`trace_id`、`request_id`、`user_id(hash)`、`session_id`、`provider`、`latency_ms`、`status_code`。
- 级别：
  - `INFO`：正常流程里程碑。
  - `WARN`：可重试的降级事件。
  - `ERROR`：请求/作业失败。
- 日志不得记录原始密钥或完整 Prompt；使用哈希/脱敏片段。

### 9.2 链路与指标（OpenTelemetry）
- Spans：
  - `http.request`
  - `session.load`
  - `rag.retrieve`
  - `rag.rerank`
  - `model.generate`
  - `mysql.query`
  - `redis.op`
  - `mq.consume`
- Metrics：
  - 请求延迟直方图
  - Token 吞吐
  - Provider 错误率
  - 队列深度与处理时长

### 9.3 看板与告警规则
- API 看板：RPS、P95/P99 延迟、错误率、首 Token 延迟。
- 模型看板：分 Provider 成功率与超时率。
- RAG 看板：检索延迟、重排延迟、引用命中率。
- 队列看板：积压、重试、DLQ。

告警阈值：
- HTTP 5xx > 2%，持续 5 分钟。
- 模型超时率 > 5%，持续 10 分钟。
- 首 Token P95 > 3s，持续 15 分钟。

### 9.4 SLO/SLI 定义
- SLO-1：月度 API 成功响应（非 4xx）>= 99.9%。
- SLO-2：95% 聊天请求首 Token <= 2s。
- SLO-3：99% 作业在 10 分钟内完成。

SLI：
- 可用性、延迟、作业完成率、DLQ 比率。

## 10. Go 后端项目结构
### 10.1 推荐目录树
```text
cmd/
  api-server/main.go
  worker/main.go
internal/
  api/
    handler/
    middleware/
    dto/
  core/
    session/
    chat/
    rag/
    modelrouter/
  repo/
    mysql/
    redis/
    qdrant/
    rabbitmq/
  worker/
    ingest/
    retry/
  config/
  observability/
  security/
pkg/
  errors/
  xcontext/
  util/
deployments/
docs/
```

### 10.2 包职责
- `internal/api`：仅处理传输层问题。
- `internal/core`：领域/应用逻辑。
- `internal/repo`：存储与外部 Provider 适配器。
- `internal/worker`：异步消费者与作业执行器。
- `internal/observability`：日志/链路/指标初始化。

### 10.3 横切中间件
- Request ID 注入。
- JWT 鉴权。
- 限流与配额检查。
- Panic 恢复与错误归一化。
- Trace 上下文传播。

## 11. API 与事件契约
### 11.1 HTTP API 契约（请求/响应）
`POST /api/v1/sessions`

请求：
```json
{"title":"How to onboard?"}
```

响应：
```json
{"code":0,"message":"ok","data":{"session_id":"6adf...","title":"How to onboard?"}}
```

`POST /api/v1/chat/send`

请求：
```json
{"session_id":"6adf...","question":"Summarize this policy","model_type":"auto","use_rag":true}
```

响应：
```json
{
  "code":0,
  "message":"ok",
  "data":{
    "answer":"...",
    "citations":[{"doc_id":"d1","chunk_id":"c7","score":0.91}],
    "usage":{"provider":"openai","input_tokens":512,"output_tokens":220}
  }
}
```

### 11.2 内部服务接口
```go
type ChatService interface {
  Send(ctx context.Context, req ChatRequest) (ChatResponse, error)
  Stream(ctx context.Context, req ChatRequest, onToken func(TokenChunk) error) error
}

type RAGService interface {
  BuildContext(ctx context.Context, req RAGRequest) (RAGContext, error)
}

type ModelRouter interface {
  Generate(ctx context.Context, req GenerateRequest) (GenerateResult, error)
  GenerateStream(ctx context.Context, req GenerateRequest, cb func(string) error) error
}
```

### 11.3 事件契约（RabbitMQ）
`doc.ingest.request.v1` 生产者契约：

```json
{
  "event_type":"doc.ingest.request",
  "version":"v1",
  "job_id":"uuid",
  "document_id":"uuid",
  "dedupe_key":"user_id:checksum",
  "storage_uri":"uri",
  "options":{"chunk_size":600,"overlap":120}
}
```

`doc.ingest.result.v1` 消费输出契约：

```json
{
  "event_type":"doc.ingest.result",
  "version":"v1",
  "job_id":"uuid",
  "document_id":"uuid",
  "status":"ready",
  "vector_count":342,
  "elapsed_ms":1732,
  "error":""
}
```

## 12. 安全与合规
### 12.1 认证与授权
- API 认证使用 JWT access token。
- 管理端接口执行角色校验。
- 每条会话读写路径都要校验会话归属权。

### 12.2 密钥管理
- API 密钥/DSN 从环境变量或密钥管理器加载。
- 周期性轮换 Provider 密钥与数据库凭据。
- 配置文件与日志中禁止存储明文密钥。

### 12.3 数据隐私与审计
- PII 最小化，敏感元数据可选字段级加密。
- 登录、权限变更、文档操作需记录不可变审计日志。
- 支持“删除权”流程与清理作业。

## 13. 风险清单与降级方案
| 风险 | 触发条件 | 影响 | 缓解措施 | 负责人 |
|---|---|---|---|---|
| Provider 故障 | OpenAI API 错误激增 | 聊天失败 | 熔断 + Ollama 回退 | AI Platform |
| 检索延迟飙升 | Qdrant 过载 | 响应变慢 | 自动扩容 + 热文档缓存 + 降低 topK | Search Platform |
| 重复导入 | 重投递/重试 | 计算浪费、数据噪音 | 幂等键 + 唯一去重索引 | Worker Team |
| Redis 集群异常 | 节点故障 | 延迟升高 | MySQL 回退模式 + 功能降级 | Backend Team |
| Schema 迁移回归 | 不兼容部署 | 服务不稳定 | expand-migrate-contract + 金丝雀发布 | Backend Team |

## 14. 发布计划
### 14.1 里程碑
1. M1：核心 API/会话/消息持久化 + JWT。
2. M2：模型路由与 SSE 流式输出。
3. M3：基于 RabbitMQ Worker 的异步导入。
4. M4：Qdrant + BGE 重排 + 引用输出。
5. M5：完整可观测性与 SLO 告警。

### 14.2 兼容性策略
- API 版本化路径 `/api/v1`。
- 新响应字段仅做追加，不破坏旧字段。
- 事件载荷中包含 schema 版本（`version: v1`）并固化路由键命名。

### 14.3 回滚方案
- 采用蓝绿或金丝雀部署，并通过功能开关控制：
  - `enable_rag`
  - `enable_ws`
  - `provider_ollama_fallback`
- 回滚流程：
  - 关闭新功能开关
  - 回退服务版本
  - 必要时从重试队列回放关键作业

## 15. 待确认问题
- v1 是否继续将知识图谱占位模块设为可选，还是在特定领域设为必选？
- Qdrant 是外部全托管，还是同集群自建？
- 不同区域最终的聊天/消息数据保留策略是什么？
- WebSocket 是否在 v1 启用，还是先在 SSE 稳定后分阶段上线？
- 生产环境应采用哪种输出审核策略与阈值？
