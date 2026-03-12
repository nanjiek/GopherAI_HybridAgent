# python/rag_service/app 设计说明

## 设计定位
该目录是 Python RAG 服务应用层，负责协议入口与算法引擎衔接。

## 核心设计思路
1. API 层只做请求校验和响应封装，算法逻辑下沉到 engine。
2. schemas 显式定义契约，避免动态 JSON 带来的隐式错误。
3. engine 提供可替换策略，方便逐步接入真实向量库与重排模型。

## 边界与依赖
边界在于应用编排与契约执行，不承担 Go 侧调用策略。

## 演进策略
后续可拆分为 api/service/repository 分层并增加异步任务处理。
