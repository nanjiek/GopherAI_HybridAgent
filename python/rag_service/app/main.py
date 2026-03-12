from fastapi import FastAPI

from .engine import RAGEngine
from .schemas import (
    EmbedRequest,
    EmbedResponse,
    IngestRequest,
    IngestResponse,
    KGRequest,
    KGResponse,
    RerankRequest,
    RerankResponse,
    RetrieveRequest,
    RetrieveResponse,
)

app = FastAPI(title="GopherMind RAG Service", version="0.1.0")
engine = RAGEngine()


@app.get("/healthz")
def healthz() -> dict:
    return {"status": "ok"}


@app.post("/embed", response_model=EmbedResponse)
def embed(req: EmbedRequest) -> EmbedResponse:
    return EmbedResponse(vector=engine.embed(req.text))


@app.post("/retrieve", response_model=RetrieveResponse)
def retrieve(req: RetrieveRequest) -> RetrieveResponse:
    docs = engine.retrieve(req.user_id, req.query, req.top_k)
    return RetrieveResponse(documents=docs)


@app.post("/rerank", response_model=RerankResponse)
def rerank(req: RerankRequest) -> RerankResponse:
    docs = engine.rerank(req.query, req.docs, req.top_n)
    return RerankResponse(documents=docs)


@app.post("/kg/placeholder", response_model=KGResponse)
def kg_placeholder(req: KGRequest) -> KGResponse:
    return KGResponse(context=engine.kg_placeholder(req.query))


@app.post("/ingest", response_model=IngestResponse)
def ingest(req: IngestRequest) -> IngestResponse:
    chunks = engine.ingest(
        user_id=req.user_id,
        document_id=req.document_id,
        text=req.text,
        metadata=req.metadata,
    )
    return IngestResponse(chunks=chunks)
