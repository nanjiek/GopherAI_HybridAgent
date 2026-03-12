from __future__ import annotations

import re
import uuid
from typing import Dict, List, Optional, Tuple

from langchain_huggingface import HuggingFaceEmbeddings
from langchain_openai import OpenAIEmbeddings
from langchain_text_splitters import RecursiveCharacterTextSplitter
from qdrant_client import QdrantClient, models

from .schemas import RetrieveDoc
from .settings import Settings

try:
    from FlagEmbedding import FlagReranker
except Exception:  # pragma: no cover - optional import guard for minimal runtime robustness
    FlagReranker = None


class RAGEngine:
    """
    Minimal production-ready pipeline:
    - Embedding: LangChain embeddings
    - Retrieve: Qdrant vector search with user-level filtering
    - Rerank: BGE reranker (FlagEmbedding)
    """

    def __init__(self, settings: Optional[Settings] = None) -> None:
        self.settings = settings or Settings.load()
        self._embeddings = self._build_embeddings()
        self._splitter = RecursiveCharacterTextSplitter(
            chunk_size=self.settings.chunk_size,
            chunk_overlap=self.settings.chunk_overlap,
            separators=["\n\n", "\n", ".", " ", ""],
        )
        self._qdrant = QdrantClient(
            url=self.settings.qdrant_url,
            api_key=self.settings.qdrant_api_key or None,
        )
        self._reranker = None
        self._ensure_collection()

    def _build_embeddings(self):
        provider = self.settings.embedding_provider
        if provider == "openai":
            kwargs: Dict[str, str] = {
                "model": self.settings.embedding_model,
                "api_key": self.settings.openai_api_key,
            }
            if self.settings.openai_base_url:
                kwargs["base_url"] = self.settings.openai_base_url
            return OpenAIEmbeddings(**kwargs)
        return HuggingFaceEmbeddings(model_name=self.settings.embedding_model)

    def _distance(self) -> models.Distance:
        dist = self.settings.qdrant_distance.lower()
        if dist == "dot":
            return models.Distance.DOT
        if dist == "euclid":
            return models.Distance.EUCLID
        return models.Distance.COSINE

    def _ensure_collection(self) -> None:
        name = self.settings.qdrant_collection
        try:
            self._qdrant.get_collection(name)
            return
        except Exception:
            pass
        self._qdrant.create_collection(
            collection_name=name,
            vectors_config=models.VectorParams(
                size=self.settings.qdrant_vector_size,
                distance=self._distance(),
            ),
        )

    def _get_reranker(self):
        if self._reranker is not None:
            return self._reranker
        if FlagReranker is None:
            raise RuntimeError("FlagEmbedding is required for BGE rerank.")
        self._reranker = FlagReranker(
            self.settings.bge_rerank_model,
            use_fp16=self.settings.bge_use_fp16,
        )
        return self._reranker

    def embed(self, text: str) -> List[float]:
        return list(self._embeddings.embed_query(text))

    def ingest(
        self,
        user_id: str,
        document_id: str,
        text: str,
        metadata: Optional[Dict[str, str]] = None,
    ) -> int:
        metadata = metadata or {}
        chunks = self._splitter.split_text(text)
        if not chunks:
            return 0

        vectors = self._embeddings.embed_documents(chunks)
        points: List[models.PointStruct] = []
        for idx, (chunk, vector) in enumerate(zip(chunks, vectors)):
            chunk_id = f"{document_id}-chunk-{idx}"
            payload = {
                "user_id": user_id,
                "doc_id": document_id,
                "chunk_id": chunk_id,
                "content": chunk,
                "metadata": metadata,
            }
            points.append(
                models.PointStruct(
                    id=str(uuid.uuid4()),
                    vector=vector,
                    payload=payload,
                )
            )

        self._qdrant.upsert(
            collection_name=self.settings.qdrant_collection,
            points=points,
            wait=True,
        )
        return len(points)

    def retrieve(self, user_id: str, query: str, top_k: int) -> List[RetrieveDoc]:
        vector = self._embeddings.embed_query(query)
        flt = models.Filter(
            must=[
                models.FieldCondition(
                    key="user_id",
                    match=models.MatchValue(value=user_id),
                )
            ]
        )

        hits = self._qdrant.search(
            collection_name=self.settings.qdrant_collection,
            query_vector=vector,
            query_filter=flt,
            limit=top_k,
            with_payload=True,
            with_vectors=False,
        )

        docs: List[RetrieveDoc] = []
        for hit in hits:
            payload = hit.payload or {}
            meta = payload.get("metadata", {})
            docs.append(
                RetrieveDoc(
                    doc_id=str(payload.get("doc_id", "")),
                    chunk_id=str(payload.get("chunk_id", "")),
                    content=str(payload.get("content", "")),
                    score=float(hit.score or 0.0),
                    metadata={str(k): str(v) for k, v in meta.items()},
                )
            )
        return docs

    def rerank(self, query: str, docs: List[RetrieveDoc], top_n: int) -> List[RetrieveDoc]:
        if not docs:
            return []
        reranker = self._get_reranker()
        pairs = [[query, d.content] for d in docs]
        scores = reranker.compute_score(pairs)
        if isinstance(scores, float):
            scores = [scores]

        rescored: List[Tuple[float, RetrieveDoc]] = []
        for score, doc in zip(scores, docs):
            # Keep a weighted blend to preserve retrieval signal.
            doc.score = float(doc.score * 0.6 + float(score) * 0.4)
            rescored.append((doc.score, doc))
        rescored.sort(key=lambda x: x[0], reverse=True)
        return [d for _, d in rescored[:top_n]]

    def kg_placeholder(self, query: str) -> str:
        # Placeholder for future KG lookup service.
        tokens = [t for t in re.split(r"[^0-9a-zA-Z\u4e00-\u9fa5]+", query.lower()) if t]
        entities = [t for t in tokens if len(t) >= 2][:5]
        if not entities:
            return ""
        return " -> ".join(entities)

