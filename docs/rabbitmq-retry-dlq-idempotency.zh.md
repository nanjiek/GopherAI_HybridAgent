# RabbitMQ 重试 + DLQ + 幂等落库一致性说明

## 目标
保证消息在失败场景下可恢复、可追踪、可止损，避免重复副作用与静默丢失。

## 设计思路
1. 重试分级：主队列失败后进入分级重试队列（短延迟到长延迟），控制突发故障抖动。
2. 死信隔离：超过最大重试次数或不可恢复错误直接进入 DLQ，避免毒消息拖垮主链路。
3. 幂等主线：以 `idempotency_key` 为唯一处理键，先判重再执行业务。
4. 持久一致：MySQL `consumer_inbox` 保存处理状态机（processing/failed/succeeded/dead），作为一致性真相源。

## 一致性策略
- 消费语义采用“至少一次 + 幂等”，不追求昂贵的分布式精确一次。
- 当处理记录卡在 `processing` 且超过租约窗口时，允许接管处理，避免僵尸状态阻塞。
- 失败消息先完成重试或死信路由，再更新 inbox 状态，避免“已标失败但未成功转移”的丢失窗口。

## 可运维性
- 指标暴露 `gophermind_mq_retry_total`、`gophermind_mq_dlq_total`、`gophermind_mq_idempotent_hit_total`。
- 告警对 DLQ 增量进行触发，快速定位消费者或下游异常。
