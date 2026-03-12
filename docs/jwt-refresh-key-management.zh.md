# JWT（Access + Refresh）与密钥托管落地说明

## 目标
在多轮会话场景下，保证认证安全性、可续期能力和最小暴露面。

## 设计思路
1. 令牌分层：Access（短 TTL）只用于接口访问；Refresh（长 TTL）只用于换发。
2. 密钥分离：Access/Refresh 使用不同密钥，降低单点泄漏影响半径。
3. 持久化策略：Refresh token 仅保存哈希值，避免明文落库。
4. 轮换策略：Refresh 每次成功使用后立即吊销旧 token 并签发新 token（rotation）。

## 密钥托管实践
- 代码读取 `JWT_ACCESS_SECRET_FILE` / `JWT_REFRESH_SECRET_FILE`，优先从容器 secret 文件加载。
- Compose 中通过 `secrets` 挂载密钥文件到 `/run/secrets/*`。
- 启动时校验密钥最小强度（长度、Access/Refresh 不同），防止弱配置上线。

## 风险与取舍
- 采用 HMAC HS256，部署与运维成本低；若要跨团队/跨系统验签，可演进到非对称签名。
- 当前为单密钥生效模型，后续可扩展 KID + 双 Key 兼容窗口支持平滑轮换。
