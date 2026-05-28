from __future__ import annotations

import math
import re
from collections import defaultdict
from functools import lru_cache

import numpy as np

from app.retrieval.config import RetrievalSettings, load_retrieval_settings
from app.retrieval.embedder import EmbeddingUnavailableError, get_encoder
from app.retrieval.index_store import load_cached_index, save_cached_index
from app.retrieval.loader import load_docs_chunks
from app.retrieval.schemas import KnowledgeChunk, RetrievalHit, RetrievalResult


TOKEN_PATTERN = re.compile(r"[A-Za-z0-9_./-]+|[\u4e00-\u9fff]{1,4}")
RRF_K = 60


def tokenize(text: str) -> list[str]:
    return [token.lower() for token in TOKEN_PATTERN.findall(text)]


def score_chunk(query_tokens: list[str], chunk: KnowledgeChunk) -> float:
    if not query_tokens:
        return 0.0

    content_tokens = tokenize(chunk.content)
    section_tokens = tokenize(chunk.section)
    if not content_tokens and not section_tokens:
        return 0.0

    content_set = set(content_tokens)
    section_set = set(section_tokens)

    score = 0.0
    for token in query_tokens:
        if token in section_set:
            score += 3.0
        if token in content_set:
            score += 1.0

    score /= math.sqrt(max(len(content_set), 1))
    return score


@lru_cache(maxsize=1)
def cached_chunks() -> tuple[KnowledgeChunk, ...]:
    return tuple(load_docs_chunks())


def chunk_text_for_embedding(chunk: KnowledgeChunk) -> str:
    return f"{chunk.section}\n{chunk.content}"


def keyword_hits_for_query(chunks: tuple[KnowledgeChunk, ...], query: str, top_k: int) -> list[RetrievalHit]:
    query_tokens = tokenize(query)
    hits: list[RetrievalHit] = []

    for chunk in chunks:
        score = score_chunk(query_tokens, chunk)
        if score <= 0:
            continue
        hits.append(RetrievalHit(chunk=chunk, score=score))

    hits.sort(key=lambda item: item.score, reverse=True)
    return hits[:top_k]


def load_or_build_embedding_index(
    chunks: tuple[KnowledgeChunk, ...],
    settings: RetrievalSettings,
) -> np.ndarray:
    cached_index = load_cached_index(list(chunks), settings)
    if cached_index is not None:
        return cached_index.embeddings

    encoder = get_encoder(settings)
    texts = [chunk_text_for_embedding(chunk) for chunk in chunks]
    embeddings = encoder.encode_documents(texts)
    if embeddings.shape[0] != len(chunks):
        raise EmbeddingUnavailableError("embedding index size does not match chunk count")

    save_cached_index(list(chunks), embeddings, settings)
    return embeddings


def embedding_hits_for_query(
    chunks: tuple[KnowledgeChunk, ...],
    query: str,
    settings: RetrievalSettings,
    top_k: int,
) -> list[RetrievalHit]:
    if not chunks:
        return []

    index = load_or_build_embedding_index(chunks, settings)
    encoder = get_encoder(settings)
    query_vector = encoder.encode_query(query)
    scores = np.sum(index * query_vector, axis=1, dtype=np.float32)
    if scores.ndim != 1:
        raise EmbeddingUnavailableError("query similarity output has unexpected shape")

    ranked_indexes = np.argsort(scores)[::-1]
    hits: list[RetrievalHit] = []
    for index_position in ranked_indexes[:top_k]:
        score = float(scores[index_position])
        if score < settings.embedding_min_score:
            continue
        hits.append(RetrievalHit(chunk=chunks[int(index_position)], score=score))

    return hits


def fuse_hits(
    embedding_hits: list[RetrievalHit],
    keyword_hits: list[RetrievalHit],
    top_k: int,
) -> list[RetrievalHit]:
    if not embedding_hits:
        return keyword_hits[:top_k]
    if not keyword_hits:
        return embedding_hits[:top_k]

    fused_scores: defaultdict[str, float] = defaultdict(float)
    best_hits: dict[str, RetrievalHit] = {}

    for rank, hit in enumerate(embedding_hits, start=1):
        fused_scores[hit.chunk.chunk_id] += 1.2 / (RRF_K + rank)
        best_hits[hit.chunk.chunk_id] = hit

    for rank, hit in enumerate(keyword_hits, start=1):
        fused_scores[hit.chunk.chunk_id] += 1.0 / (RRF_K + rank)
        existing = best_hits.get(hit.chunk.chunk_id)
        if existing is None or hit.score > existing.score:
            best_hits[hit.chunk.chunk_id] = hit

    ranked_ids = sorted(fused_scores, key=lambda chunk_id: fused_scores[chunk_id], reverse=True)
    return [
        RetrievalHit(chunk=best_hits[chunk_id].chunk, score=fused_scores[chunk_id])
        for chunk_id in ranked_ids[:top_k]
    ]


def retrieve_knowledge_result(query: str) -> RetrievalResult:
    settings = load_retrieval_settings()
    chunks = cached_chunks()
    keyword_hits = keyword_hits_for_query(chunks, query, settings.keyword_top_k)
    strategies: list[str] = []

    if keyword_hits:
        strategies.append("knowledge.keyword")

    if not settings.enabled:
        return RetrievalResult(hits=keyword_hits[: settings.top_k], strategies=strategies)

    try:
        embedding_hits = embedding_hits_for_query(chunks, query, settings, max(settings.top_k, settings.keyword_top_k))
        strategies.insert(0, "knowledge.embedding")
    except EmbeddingUnavailableError:
        strategies.append("knowledge.lexical_fallback")
        return RetrievalResult(hits=keyword_hits[: settings.top_k], strategies=strategies)

    fused_hits = fuse_hits(embedding_hits, keyword_hits, settings.top_k)
    return RetrievalResult(hits=fused_hits, strategies=strategies)
