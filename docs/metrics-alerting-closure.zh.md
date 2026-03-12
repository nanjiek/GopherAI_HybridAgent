# 业务指标埋点与基础告警闭环说明

## 目标
把“监控可见”升级为“故障可响应”，形成最小可运行闭环。

## 指标分层
1. 接口层：`gophermind_http_requests_total`、`gophermind_http_request_duration_seconds`
2. 认证层：`gophermind_auth_login_total`、`gophermind_auth_refresh_total`
3. MQ 可靠性层：`gophermind_mq_retry_total`、`gophermind_mq_dlq_total`、`gophermind_mq_idempotent_hit_total`
4. 业务层：`gophermind_query_latency_seconds`、`gophermind_query_requests_total`、`gophermind_stream_first_token_seconds`、`gophermind_stream_requests_total`

## 告警闭环
1. Prometheus 抓取 `/metrics` 并评估规则。
2. 规则触发后发送到 Alertmanager。
3. Alertmanager 聚合并路由到 `alert-webhook`。
4. `alert-webhook` 接收告警并落日志，作为本地联调闭环终点。

## 当前规则覆盖
- HTTP 5xx 错误率持续超阈值
- Query P95 延迟持续超阈值
- DLQ 增量告警
- Refresh 失败激增告警

## 取舍说明
- 先用 webhook 形成可验证闭环，再按团队实际接入企业微信/钉钉/飞书/PagerDuty。
- 优先覆盖“高风险且可行动”的规则，避免早期告警噪音。
