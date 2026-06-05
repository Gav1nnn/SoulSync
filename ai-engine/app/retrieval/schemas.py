from dataclasses import dataclass
from typing import Any


@dataclass(frozen=True)
class KnowledgeChunk:
    chunk_id: str
    source_path: str
    title: str
    section: str
    content: str


@dataclass(frozen=True)
class RetrievalHit:
    chunk: KnowledgeChunk
    score: float


@dataclass(frozen=True)
class RetrievalResult:
    hits: list[RetrievalHit]
    strategies: list[str]


@dataclass(frozen=True)
class KnowledgeIndex:
    chunk_ids: tuple[str, ...]
    embeddings: Any
