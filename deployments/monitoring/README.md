# 监控与告警设计说明

## 设计目标
这一套监控不是为了“把指标打出来”，而是为了形成可执行的运维闭环：
1. 后端持续暴露业务与系统指标。
2. Prometheus 定时抓取并计算聚合指标。
3. 告警规则基于错误率、延迟、DLQ 等关键风险触发。
4. Alertmanager 统一路由到 webhook 接收器，保证告警可落地验证。

## 分层思路
- 指标层：在应用内埋点（HTTP、鉴权、MQ、Query/Stream）。
- 规则层：Prometheus 只做“检测与判定”，不承载通知逻辑。
- 通知层：Alertmanager 管理分组、抑制、重发周期。
- 接收层：`alert_webhook.py` 作为最小可用通知终点，便于本地联调。

## 关键取舍
- 先做“稳定可验证”的基础链路，再接企业 IM/邮件/PagerDuty。
- 告警规则以“可行动”为先，避免一次性放太多低价值噪音告警。
- 先覆盖核心风险：5xx、P95、DLQ 增长、Refresh 异常峰值。

## 本地闭环验证
1. `docker compose up -d`
2. 访问 `http://localhost:9091` 查看 Prometheus Targets 与 Rules。
3. 访问 `http://localhost:9093` 查看 Alertmanager 页面。
4. 触发异常场景后，查看 `docker logs gophermind_alert_webhook`，确认收到告警 JSON。
