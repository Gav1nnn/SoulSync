from __future__ import annotations

from hashlib import sha1
from pathlib import Path

import numpy as np

from app.retrieval.config import RetrievalSettings
from app.retrieval.schemas import KnowledgeChunk, KnowledgeIndex


def index_fingerprint(chunks: list[KnowledgeChunk], settings: RetrievalSettings) -> str:
    digest = sha1()
    digest.update(settings.provider.encode("utf-8"))
    digest.update(settings.base_url.encode("utf-8"))
    digest.update(settings.model_name.encode("utf-8"))
    digest.update(settings.query_prefix.encode("utf-8"))
    digest.update(settings.document_prefix.encode("utf-8"))
    for chunk in chunks:
        digest.update(chunk.chunk_id.encode("utf-8"))
    return digest.hexdigest()[:16]


def index_path(chunks: list[KnowledgeChunk], settings: RetrievalSettings) -> Path:
    return settings.cache_dir / f"knowledge-index-{index_fingerprint(chunks, settings)}.npz"


def load_cached_index(chunks: list[KnowledgeChunk], settings: RetrievalSettings) -> KnowledgeIndex | None:
    path = index_path(chunks, settings)
    if not path.exists():
        return None

    with np.load(path, allow_pickle=False) as data:
        chunk_ids = tuple(str(item) for item in data["chunk_ids"].tolist())
        expected_ids = tuple(chunk.chunk_id for chunk in chunks)
        if chunk_ids != expected_ids:
            return None

        return KnowledgeIndex(
            chunk_ids=chunk_ids,
            embeddings=data["embeddings"].astype(np.float32),
        )


def save_cached_index(chunks: list[KnowledgeChunk], embeddings: np.ndarray, settings: RetrievalSettings) -> None:
    settings.cache_dir.mkdir(parents=True, exist_ok=True)
    path = index_path(chunks, settings)
    np.savez_compressed(
        path,
        chunk_ids=np.array([chunk.chunk_id for chunk in chunks]),
        embeddings=embeddings.astype(np.float32),
    )
