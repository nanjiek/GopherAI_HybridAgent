# Software Design Document

## 0. Document Metadata
- Project: `GopherMind`
- Version: `v0.1`
- Date: `2026-03-12`
- Author: `Codex + User`

## 1. Requirements Summary
### 1.1 Business Goals
- Build a production-ready backend for multi-session AI Q&A with optional knowledge-grounded answers.
- Support both cloud and local model providers to balance quality, cost, and availability.
- Provide real-time streaming UX for chat and robust async processing for ingestion/index tasks.
- Ensure observability, reliability, and secure handling of user and document data.

### 1.2 Functional Requirements
- FR-1: User authentication and authorization for all protected APIs.
- FR-2: Session lifecycle management (create/list/get history/archive).
- FR-3: Synchronous and streaming chat (SSE primary, WebSocket optional/upgrade path).
- FR-4: Multi-model routing across `OpenAI`, `Ollama`, and `BGE` reranker.
- FR-5: Document ingestion pipeline (upload -> parse -> recursive chunk -> embed -> index -> status).
- FR-6: RAG query flow (`Qdrant` retrieval + `BGE` rerank + KG placeholder + generation).
- FR-7: Async job execution via RabbitMQ with retry/DLQ/idempotency.
- FR-8: Persistent storage of users, sessions, messages, documents, and jobs in MySQL.
- FR-9: Hot-path cache and short-lived state in Redis Cluster.
- FR-10: Structured logging, tracing, metrics, dashboards, and alerting.

### 1.3 Non-Functional Requirements
- NFR-1: P95 chat first token latency <= 2.0s for non-RAG prompts.
- NFR-2: P95 full response latency <= 8.0s (text <= 700 tokens).
- NFR-3: API availability >= 99.9% monthly.
- NFR-4: At-least-once async job processing with idempotent consumers.
- NFR-5: Horizontal scaling for API and workers.
- NFR-6: Security baseline with JWT auth, secret isolation, audit logging, and PII minimization.

### 1.4 Assumptions and Constraints
Required inputs were not explicitly provided, so the following defaults are applied:
- `project_name`: `GopherMind`
- `requirement_description`: "Backend AI assistant with session chat, streaming output, document-grounded RAG, and async ingestion."
- `models_config`: `OpenAI / Ollama / BGE`
- `redis_cluster`:
  - `redis-node1:6379`
  - `redis-node2:6379`
  - `redis-node3:6379`
- `mysql_dsn`: `root:password@tcp(mysql:3306)/gophermind`
- `rabbitmq_url`: `amqp://guest:guest@rabbitmq:5672/`

Constraints:
- Backend language: Go.
- RAG indexing/retrieval service may invoke Python components where needed.
- Streaming compatibility required for modern browsers and reverse proxies.

Requirement-to-module/interface mapping:

| Requirement | Primary Module | Interface Contract |
|---|---|---|
| FR-1 | API Gateway + Auth | `POST /api/v1/auth/login`, JWT middleware |
| FR-2 | Session Service | `POST /api/v1/sessions`, `GET /api/v1/sessions` |
| FR-3 | Chat Service | `POST /api/v1/chat/send`, `POST /api/v1/chat/stream` |
| FR-4 | Model Orchestrator | `Generate(ctx, req)`, provider adapters |
| FR-5 | Ingestion Worker | `doc.ingest.request.v1` RabbitMQ message |
| FR-6 | RAG Service | `RAGQuery(ctx, query, userID, sessionID)` |
| FR-7 | Async Queue | `x.gophermind.doc`, retry + DLQ queues |
| FR-8 | Persistence Layer | MySQL schema and transactional DAO APIs |
| FR-9 | Cache Layer | Redis keyspace contract (`sess:*`, `rate:*`) |
| FR-10 | Observability | OTel traces/metrics + Zap structured logs |

## 2. System Architecture and Module Decomposition
### 2.1 High-Level Architecture
Text diagram:

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

Online Q&A sequence flow:

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

### 2.2 Core Modules
- API Gateway / HTTP Layer
  - Request validation, authn/authz, rate limiting, protocol adaptation (REST/SSE/WebSocket).
- Session & Conversation Service
  - Session state, message append, history retrieval, conversation policies.
- RAG Pipeline Service
  - Query rewrite, retrieval, rerank, context pack, citation metadata.
- Model Orchestration Layer
  - Provider routing, timeout/retry/circuit-breaker, token budget enforcement.
- Cache Layer (Redis Cluster)
  - Session snapshots, short-lived context cache, idempotency keys, rate-limit counters.
- Persistence Layer (MySQL)
  - Users/sessions/messages/documents/jobs as durable system of record.
- Async Worker Layer (RabbitMQ Consumers)
  - Ingestion/index tasks, retry, DLQ, recovery and replay.
- Observability Layer
  - Zap logs, OTel traces/metrics, dashboards and alerting.

### 2.3 Module Responsibility Matrix
| Module | Responsibility | Input | Output | Depends On |
|---|---|---|---|---|
| API Gateway | Validate/auth/route/stream | HTTP requests | JSON/SSE/WS events | JWT, Session Service |
| Session Service | Session and history lifecycle | user/session/message ops | persisted messages, session metadata | MySQL, Redis |
| RAG Service | Grounding context assembly | query + user scope | ranked passages + citations | Qdrant, BGE |
| Model Orchestrator | Generation dispatch | prompt + context + policy | completion/token stream | OpenAI/Ollama |
| Ingestion Worker | Document indexing | RabbitMQ ingest message | Qdrant vectors + status | Python pipeline, MySQL |
| Cache Layer | Low-latency ephemeral state | key/value ops | cached values | Redis Cluster |
| Observability | Telemetry and alerting | logs/traces/metrics | dashboards/alerts | OTel backend |

## 3. Redis Cluster Design
### 3.1 Keyspace and Data Model
- `sess:{user_id}:{session_id}:summary` (String/JSON): rolling summary for prompt compression.
- `sess:{user_id}:{session_id}:ctx` (Hash): last turn metadata, model, token usage.
- `chat:stream:{request_id}` (String): stream cursor/state for reconnect.
- `idempotency:{consumer}:{msg_id}` (String): async dedupe key.
- `rate:{user_id}:{minute_bucket}` (String counter): per-minute quota.
- `model:fallback:state:{provider}` (String): provider health/circuit state.

### 3.2 TTL and Eviction Policy
- Session summary/context keys: `TTL 24h`.
- Stream state keys: `TTL 10m`.
- Idempotency keys: `TTL 48h`.
- Rate-limit counters: `TTL 2m`.
- Maxmemory policy: `allkeys-lru` for cache-only nodes; no business-critical data only in Redis.

### 3.3 Persistence and Data Safety
- AOF enabled (`appendonly yes`) with `everysec`.
- RDB snapshots every 15 minutes for faster restarts.
- Redis used as accelerator; MySQL remains source of truth.

### 3.4 Cache-Aside / Write Strategy
- Read path: try Redis -> fallback MySQL -> populate Redis with TTL.
- Write path: transaction commit in MySQL first, then async cache invalidation/update.
- Streaming metadata updates are write-through to Redis and eventually flushed to MySQL.

### 3.5 Degradation and Fallback Plan
- Redis unavailable:
  - Continue with MySQL-only mode for session/history.
  - Disable non-critical features: reconnectable stream cursors and hot summaries.
  - Increase API latency budget and emit `degraded_mode=true` metric tag.

### 3.6 Example Redis Config and Pseudocode
Config example:

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

Go-like pseudocode:

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

## 4. MySQL Design
### 4.1 Entity and Table Overview
- `users`: identity and account status.
- `sessions`: conversation container per user.
- `messages`: ordered chat turns and metadata.
- `documents`: uploaded docs and indexing lifecycle.
- `jobs`: async job state machine.
- `model_usage`: token/cost accounting for governance.

### 4.2 Table DDL Examples
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

### 4.3 Transaction Boundaries
- `CreateSession + first user message`: single transaction.
- `Append user message + reserve request_id`: single transaction before model call.
- `Persist assistant response`: transaction with optimistic checks on `request_id`.
- `Document state updates`: transaction per state transition with monotonic guard.

### 4.4 Indexing and Query Optimization
- Session listing by user: covering index `(user_id, updated_at)`.
- Message history by session/time: index `(session_id, created_at)`.
- Job replay/recovery: `(status, updated_at)` composite index recommended.
- Document readiness checks: `(user_id, status)`.

### 4.5 Migration and Backward Compatibility
- Use additive migrations first (new nullable columns/tables).
- Backfill async and switch reads after verification.
- Keep previous API fields until at least one release cycle passes.

### 4.6 Data Retention and Archival
- Messages retained for 180 days by default (configurable).
- Jobs retained 30 days for debugging.
- Soft-delete for sessions/documents where compliance requires delayed purge.

## 5. RabbitMQ Async Queue Design
### 5.1 Exchange/Queue/Binding Topology
- Exchange: `x.gophermind.doc` (topic).
- Primary queue: `q.doc.ingest`.
- Retry queue: `q.doc.ingest.retry` with TTL and dead-letter back to primary.
- DLQ: `q.doc.ingest.dlq`.
- Routing keys:
  - `doc.ingest.request.v1`
  - `doc.ingest.retry.v1`
  - `doc.ingest.dead.v1`

### 5.2 Message Schema and Versioning
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

### 5.3 Retry, DLQ, and Idempotency
- Consumer checks Redis/MySQL idempotency before processing.
- Retry policy: exponential backoff (`5s`, `30s`, `2m`, `10m`, max 5 attempts).
- After max attempts, route to DLQ with failure reason and alert.

### 5.4 Consumer Concurrency and Scaling
- Worker concurrency starts at `N=8` per instance.
- HPA scaling signal: queue depth + processing latency.
- Prefetch tuned to avoid head-of-line blocking (`prefetch=16` baseline).

### 5.5 Operational Metrics and Alerts
- Queue depth, consumer lag, retry rate, DLQ inflow, success ratio, processing latency.
- Alert examples:
  - DLQ inflow > 20/min for 5 min.
  - Success ratio < 95% for 10 min.
  - Oldest message age > 15 min.

## 6. Model Invocation Governance
### 6.1 Supported Models and Routing Rules
- `OpenAI`:
  - Default for high quality and complex reasoning.
- `Ollama`:
  - Fallback when OpenAI unavailable or budget threshold exceeded.
- `BGE`:
  - Dedicated reranker for retrieved passages, not final generator.

Routing policy:
- If request requires grounding and rerank: run retrieval + BGE then generate.
- If provider health is degraded: fail over OpenAI -> Ollama.
- If user policy requires local-only: force Ollama.

### 6.2 Prompt and Context Assembly
- Prompt segments:
  - system policy
  - session summary
  - latest user turns
  - RAG citations/context blocks
- Max context budget with truncation priority: system > retrieved evidence > latest turns > old turns.

### 6.3 Timeout, Circuit Breaker, and Fallback
- Provider timeout: `10s` for first token, `45s` total.
- Circuit breaker opens after 5 consecutive failures within 60s.
- Fallback ladder:
  - retry same provider once (transient)
  - switch provider
  - degrade to non-RAG concise response if retrieval path fails

### 6.4 Cost and Token Budget Controls
- Per-request max input/output token caps.
- Per-user daily quota and anomaly checks.
- Usage record persisted in `model_usage`.

### 6.5 Safety/Policy Controls
- Prompt injection guard patterns before retrieval and generation.
- Output moderation hook (configurable severity).
- Redact sensitive data in logs and traces.

## 7. RAG Q&A Pipeline
### 7.1 Ingestion Flow
- Step 1: Parse uploaded file to normalized text.
- Step 2: Recursive chunking (`chunk_size=600`, `overlap=120`, separator hierarchy).
- Step 3: Generate dense embeddings.
- Step 4: Upsert vectors + metadata into `Qdrant`.
- Step 5: Mark document status to `ready`.

### 7.2 Query Flow
- Optional query rewrite for ambiguous input.
- Retrieve candidates from Qdrant (`topK=20`).
- Rerank with BGE (`topN=5` final context blocks).
- Knowledge graph placeholder enriches entities/relations if available.
- Assemble grounded prompt and generate via OpenAI/Ollama.

### 7.3 Accuracy and Guardrail Strategy
- Require citation coverage for factual claims.
- If retrieval confidence below threshold, return uncertainty template.
- Block context chunks failing policy filters (PII/toxicity constraints).

### 7.4 Latency Budget by Stage
- Query rewrite: <= 100ms
- Qdrant retrieval: <= 150ms
- BGE rerank: <= 250ms
- Prompt assembly: <= 50ms
- Model first token: <= 1200ms
- End-to-end P95: <= 2000ms first token / <= 8000ms completion

### 7.5 Pipeline Pseudocode
Python-like pseudocode:

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

## 8. Streaming Output Design
### 8.1 WebSocket Path
- Use `/api/v1/chat/ws` for bidirectional real-time sessions.
- Supports tool-calls and interactive control signals.
- Preferred for advanced clients requiring full duplex.

### 8.2 SSE Path
- Use `/api/v1/chat/stream` with `text/event-stream`.
- Simpler proxy compatibility and browser support.
- Default for current web chat.

### 8.3 Backpressure, Cancellation, and Recovery
- Backpressure:
  - bounded channel between provider adapter and HTTP writer.
- Cancellation:
  - context cancellation on client disconnect or timeout.
- Recovery:
  - reconnect using `request_id`; replay buffered partial tokens from Redis if present.

### 8.4 Message/Event Contract Examples
SSE events:

```text
event: meta
data: {"request_id":"r123","session_id":"s123","model":"openai:gpt-4.1"}

event: token
data: {"delta":"Hello"}

event: done
data: {"finish_reason":"stop","usage":{"input_tokens":342,"output_tokens":211}}
```

WebSocket frame payload:

```json
{
  "type": "token",
  "request_id": "r123",
  "delta": "Hello",
  "seq": 18
}
```

Trade-off:
- SSE chosen as default for lower operational complexity.
- WebSocket retained for future interactive tooling and richer client control.

## 9. Logging and Observability
### 9.1 Logging Standards (Zap)
- Required fields: `trace_id`, `request_id`, `user_id(hash)`, `session_id`, `provider`, `latency_ms`, `status_code`.
- Levels:
  - `INFO`: normal flow milestones.
  - `WARN`: retriable degradation.
  - `ERROR`: failed request/job.
- No raw secrets/prompts in logs; use hash/masked snippets.

### 9.2 Tracing and Metrics (OpenTelemetry)
- Spans:
  - `http.request`
  - `session.load`
  - `rag.retrieve`
  - `rag.rerank`
  - `model.generate`
  - `mysql.query`
  - `redis.op`
  - `mq.consume`
- Metrics:
  - request latency histogram
  - token throughput
  - provider error ratio
  - queue depth and processing duration

### 9.3 Dashboards and Alert Rules
- API dashboard: RPS, P95/P99 latency, error rate, first-token latency.
- Model dashboard: per-provider success and timeout rates.
- RAG dashboard: retrieval latency, rerank latency, citation hit rate.
- Queue dashboard: backlog, retry, DLQ.

Alert thresholds:
- HTTP 5xx > 2% for 5 minutes.
- Model timeout rate > 5% for 10 minutes.
- First-token P95 > 3s for 15 minutes.

### 9.4 SLO/SLI Definition
- SLO-1: 99.9% monthly successful API responses (non-4xx).
- SLO-2: 95% of chat requests first token <= 2s.
- SLO-3: 99% job completion within 10 minutes.

SLIs:
- Availability, latency, job completion ratio, and DLQ ratio.

## 10. Go Backend Project Structure
### 10.1 Recommended Folder Tree
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

### 10.2 Package Responsibility
- `internal/api`: transport concerns only.
- `internal/core`: domain/application logic.
- `internal/repo`: storage/provider adapters.
- `internal/worker`: async consumers and job runners.
- `internal/observability`: logging/tracing/metrics bootstrap.

### 10.3 Cross-Cutting Middleware
- Request ID injection.
- JWT auth.
- Rate limit and quota check.
- Panic recovery and error normalization.
- Trace context propagation.

## 11. API and Event Contracts
### 11.1 HTTP API Contracts (request/response)
`POST /api/v1/sessions`

Request:
```json
{"title":"How to onboard?"}
```

Response:
```json
{"code":0,"message":"ok","data":{"session_id":"6adf...","title":"How to onboard?"}}
```

`POST /api/v1/chat/send`

Request:
```json
{"session_id":"6adf...","question":"Summarize this policy","model_type":"auto","use_rag":true}
```

Response:
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

### 11.2 Internal Service Interfaces
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

### 11.3 Event Contracts (RabbitMQ)
`doc.ingest.request.v1` producer contract:

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

`doc.ingest.result.v1` consumer output contract:

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

## 12. Security and Compliance
### 12.1 Authentication and Authorization
- JWT access tokens for API authn.
- Role checks for admin endpoints.
- Session ownership enforced on every read/write path.

### 12.2 Secret Management
- Load API keys/DSNs from environment or secret manager.
- Rotate provider keys and DB credentials periodically.
- Do not store secrets in config files or logs.

### 12.3 Data Privacy and Audit
- PII minimization and optional field-level encryption for sensitive metadata.
- Immutable audit log for login, permission changes, and document operations.
- Support right-to-delete workflow with purge jobs.

## 13. Risk Register and Degradation Plan
| Risk | Trigger | Impact | Mitigation | Owner |
|---|---|---|---|---|
| Provider outage | OpenAI API errors spike | chat failures | circuit breaker + Ollama fallback | AI Platform |
| Retrieval latency spike | Qdrant overload | slow responses | autoscale + cache hot docs + reduced topK | Search Platform |
| Duplicate ingestion | redelivery/retry | wasted compute, data noise | idempotency keys + unique dedupe index | Worker Team |
| Redis cluster issue | node failures | higher latency | MySQL fallback mode + feature degrade | Backend Team |
| Schema migration regression | incompatible deploy | service instability | expand-migrate-contract + canary rollout | Backend Team |

## 14. Rollout Plan
### 14.1 Milestones
1. M1: Core API/session/message persistence + JWT.
2. M2: Model routing and SSE streaming.
3. M3: Async ingestion with RabbitMQ workers.
4. M4: Qdrant + BGE rerank + citation output.
5. M5: Full observability and SLO alerting.

### 14.2 Compatibility Strategy
- Version APIs under `/api/v1`.
- New response fields are additive.
- Event schema version in payload (`version: v1`) and routing key naming.

### 14.3 Rollback Plan
- Blue/green or canary deployment with feature flags:
  - `enable_rag`
  - `enable_ws`
  - `provider_ollama_fallback`
- Rollback procedure:
  - disable new flags
  - revert service version
  - replay critical jobs from retry queue if needed

## 15. Open Questions
- Should knowledge graph placeholder remain optional in v1, or be mandatory for selected domains?
- Is Qdrant fully managed externally, or self-hosted in the same cluster?
- What are final retention requirements for chat/message data by region?
- Should WebSocket be enabled in v1, or staged after SSE stabilization?
- Which moderation policy and threshold should gate model output in production?
