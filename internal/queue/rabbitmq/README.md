# internal/queue/rabbitmq 设计说明

## 设计定位
该模块承载异步任务链路，目标是在不阻塞在线请求的前提下保证消息处理可靠性。

## 核心设计思路
1. 主队列 + 分级重试队列 + DLQ，形成可恢复且可观测的失败治理路径。
2. 以 `idempotency_key` 为主线，结合 Redis 快速判重与 MySQL inbox 持久判重，实现“重复可接受、重复副作用不可接受”。
3. 消费状态外显为 `processing/failed/succeeded/dead`，支持问题排查与人工重放。

## 一致性取舍
- 采用“至少一次投递 + 幂等消费”而非“精确一次”。
- 跨 RabbitMQ 与 MySQL 不做分布式事务，通过 inbox 状态机和重试语义保证最终一致。

## 演进策略
后续可引入 outbox relay、重放工具与失败分桶，进一步降低人工介入成本。
