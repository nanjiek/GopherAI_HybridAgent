# GopherAI Hybrid Agent

面向生产的 Agentic AI 平台，重点解决两类问题：

1. 长文本/知识库检索质量不稳定（仅 embedding 容易漏召回）
2. 新闻信息同质化严重（信息茧房）

本项目采用 **文件系统友好多级索引 + 混合检索**：

- 文件系统保存正文（source of truth）
- PageIndex 负责长文页级召回
- Qdrant + BGE 负责语义补召回
- BGE Reranker 负责重排序
- 主 Agent + 受限 bash sub-agent 协同检索

---

## 我们的规划（Roadmap）

> 说明：Phase H（质量与上线）当前暂缓，先完成 A~G。

### Phase A：底座收口（已完成）

- 回滚基线 tag：`phase-a-baseline-ec4563d`
- 编译基线确认（非图像链路可编译）
- 职责边界确认：
  - MySQL：业务与元数据
  - 文件系统：正文真相源
  - Qdrant：向量检索基础设施
  - PageIndex：长文主检索层

基线文档：`GopherAI_HybirdAgent/docs/PHASE_A_BASELINE.md`

### Phase B：统一数据与协议规范

- 统一 ID 规范：`doc_id / page_id / chunk_id`
- MySQL 表契约（news/article/event/index_version）
- PageIndex JSON Schema
- Qdrant Collection + Payload Schema
- 统一检索返回协议（证据路径/页号/分数）

### Phase C：摄取与索引流水线

- 递归分块（Recursive Splitter）
- BGE Embedding 写入 Qdrant
- PageIndex 构建与版本化
- 增量索引（hash + mtime）
- 失败回滚与双版本切换

### Phase D：在线检索执行链路

- Query Planner（问题类型路由）
- L1 文档级召回
- L2 PageIndex 页级召回
- L3 向量补召回
- BGE Reranker 精排
- 分文档证据聚合回答

### Phase E：Agentic 检索（bash sub-agent）

- 受限命令白名单（rg/find/tree/head/sed/cat）
- 沙箱与超时控制
- 结构化 JSON 输出
- 与主检索链融合排序

### Phase F：新闻专线（反茧房）

- 来源白名单 + 审核状态管理
- 分级抓取频率（5/15/60 分钟）
- 事件聚类 + 观点抽取
- 反向观点注入与跨地区平衡

### Phase G：模型与工具整合

- LangChain + ChatGLM3 检索服务
- Kimi/OpenAI/本地模型并行 Provider
- MCP 工具扩展（news/weather/...）

---

## 技术架构

- Backend：Go + Gin + GORM
- DB：MySQL（业务数据）
- Cache/Infra：Redis、RabbitMQ
- Retrieval：PageIndex、Qdrant、BGE Embedding/Reranker
- Agent：主 Agent + 受限 bash sub-agent
- Frontend：Vue3

---

## 目录说明

```text
GopherAI_HybirdAgent/
  common/        # 基础组件（aihelper/rag/redis/rabbitmq/...）
  config/        # 配置与校验
  controller/    # API 控制层
  dao/           # 数据访问层
  model/         # 数据模型
  router/        # 路由
  service/       # 业务层
  utils/         # 工具函数
  vue-frontend/  # 前端
  docs/          # 规划与阶段文档
```

---

## 开发原则

- 文件系统优先：正文留痕、可追溯
- 证据优先：回答必须可定位到文档片段
- 分层清晰：业务层与检索层解耦
- 可回滚：索引与服务变更必须有版本基线
- 渐进演进：先稳定，再扩展

---

## 当前下一步

从 **Phase B** 开始，先固化 schema 和协议，再进入索引流水线实施。
