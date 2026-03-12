from typing import Dict, List

from pydantic import BaseModel, Field


class EmbedRequest(BaseModel):
    text: str = Field(..., min_length=1)


class EmbedResponse(BaseModel):
    vector: List[float]


class RetrieveRequest(BaseModel):
    user_id: str = Field(..., min_length=1)
    query: str = Field(..., min_length=1)
    top_k: int = Field(default=20, ge=1, le=100)


class RetrieveDoc(BaseModel):
    doc_id: str
    chunk_id: str
    content: str
    score: float
    metadata: Dict[str, str] = Field(default_factory=dict)


class RetrieveResponse(BaseModel):
    documents: List[RetrieveDoc]


class RerankRequest(BaseModel):
    query: str = Field(..., min_length=1)
    docs: List[RetrieveDoc]
    top_n: int = Field(default=5, ge=1, le=50)


class RerankResponse(BaseModel):
    documents: List[RetrieveDoc]


class KGRequest(BaseModel):
    query: str = Field(..., min_length=1)


class KGResponse(BaseModel):
    context: str


class IngestRequest(BaseModel):
    user_id: str = Field(..., min_length=1)
    document_id: str = Field(..., min_length=1)
    text: str = Field(..., min_length=1)
    metadata: Dict[str, str] = Field(default_factory=dict)


class IngestResponse(BaseModel):
    chunks: int
