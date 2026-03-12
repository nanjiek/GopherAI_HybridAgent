# python/rag_service 设计说明

## 设计定位
该模块是独立 RAG 子服务，先于 Go 侧演进，提供真实的 `LangChain + Qdrant + BGE` 最小可用链路。

## 核心设计思路
1. 对 Go 保持稳定接口：`/embed`、`/retrieve`、`/rerank`、`/kg/placeholder`。
2. 检索链路采用两阶段：先向量召回（Qdrant），再交叉编码重排（BGE）。
3. 通过新增 `/ingest` 提供最小入库能力，确保链路可独立闭环验证。

## 边界与依赖
边界在于该服务只处理 RAG 算法和向量存储，不处理用户鉴权、会话事务、消息队列。
核心依赖：
- LangChain（embeddings + chunking）
- Qdrant（向量检索）
- FlagEmbedding BGE（重排）

## 运行配置
最小环境变量（建议）：
- `QDRANT_URL=http://localhost:6333`
- `QDRANT_API_KEY=`
- `QDRANT_COLLECTION=gophermind_docs`
- `QDRANT_VECTOR_SIZE=1024`
- `QDRANT_DISTANCE=cosine`
- `EMBEDDING_PROVIDER=huggingface` 或 `openai`
- `EMBEDDING_MODEL=BAAI/bge-m3`（或你的模型）
- `OPENAI_API_KEY=`（当 provider=openai 时）
- `OPENAI_BASE_URL=`（可选）
- `BGE_RERANK_MODEL=BAAI/bge-reranker-v2-m3`
- `BGE_USE_FP16=true`
- `CHUNK_SIZE=600`
- `CHUNK_OVERLAP=120`

## 运行方式
```bash
pip install -r requirements.txt
uvicorn app.main:app --host 0.0.0.0 --port 8000
```

## 最小联调顺序
1. 先调用 `/ingest` 写入测试文本。
2. 再调用 `/retrieve` 验证向量召回。
3. 调用 `/rerank` 验证 BGE 重排。
4. 最后由 Go 侧按原接口链路调用。
