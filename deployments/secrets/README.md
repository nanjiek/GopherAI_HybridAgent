# JWT 密钥托管说明

## 设计思路
项目将 JWT 密钥与应用配置解耦：代码只读取 `JWT_ACCESS_SECRET_FILE` / `JWT_REFRESH_SECRET_FILE` 指向的文件内容，密钥本身不写入源码和镜像层。

## 本地最小落地
- `jwt_access_secret.txt` 与 `jwt_refresh_secret.txt` 由 Compose 作为 secret 挂载到容器 `/run/secrets/`。
- 后端容器通过文件路径读取密钥，避免在 `docker inspect` 的环境变量里直接暴露明文。

## 生产建议
- 使用 KMS/Vault/云 Secret Manager 动态下发。
- Access 与 Refresh 使用不同密钥，并设置定期轮转窗口。
- 轮转期间保留短暂双 Key 验签兼容窗口，再下线旧密钥。
