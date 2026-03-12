# 附件上传功能说明

## 设计思路
附件功能采用“鉴权后本地对象存储”最小落地：
1. 上传路径按 `user/date` 分层，避免单目录文件过多。
2. 后端严格限制大小与后缀白名单，优先防误用与滥用。
3. 返回 `file_key` 作为后续业务引用键，而不是暴露绝对磁盘路径。
4. 下载接口按当前登录用户校验 `file_key` 前缀，防止越权读取。

## 接口
- `POST /attachments`（需要 Bearer Token）
  - `Content-Type: multipart/form-data`
  - 文件字段名：`file`
- `GET /attachments/file?key=<file_key>`（需要 Bearer Token）

## 上传成功响应字段
- `attachment_id`：文件唯一 ID
- `file_key`：业务引用键（例如 `u1/2026/03/12/xxx.pdf`）
- `download_url`：下载地址
- `original_name` / `content_type` / `size_bytes` / `sha256`

## 示例
```bash
curl -X POST "http://localhost:9090/attachments" \
  -H "Authorization: Bearer <access_token>" \
  -F "file=@./sample.pdf"
```

```bash
curl -L "http://localhost:9090/attachments/file?key=u1/2026/03/12/xxx.pdf" \
  -H "Authorization: Bearer <access_token>" \
  -o downloaded.pdf
```
