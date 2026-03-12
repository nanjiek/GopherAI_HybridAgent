# internal/obs/otel 设计说明

## 设计定位
该模块负责 tracing 初始化，让链路从入口到外部依赖可追踪。

## 核心设计思路
1. 初始化 tracer provider 并在进程生命周期内统一管理。
2. 采样策略默认保守，平衡可观测性成本与问题定位能力。
3. 与 HTTP 中间件、repo、queue 协作形成端到端 trace。

## 边界与依赖
边界在于只做 OTel 基础设施引导，不定义业务指标阈值。

## 演进策略
后续可补 metric/exporter/propagator 的生产配置与告警联动。
