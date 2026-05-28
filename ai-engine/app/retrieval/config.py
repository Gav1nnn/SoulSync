from __future__ import annotations

import os
from dataclasses import dataclass
from pathlib import Path


@dataclass(frozen=True)
class RetrievalSettings:
    enabled: bool
    provider: str
    base_url: str
    model_name: str
    cache_dir: Path
    top_k: int
    keyword_top_k: int
    embedding_min_score: float
    timeout_seconds: float
    query_prefix: str
    document_prefix: str


def _env(name: str, fallback: str = "") -> str:
    return os.getenv(name, fallback).strip()


def _env_bool(name: str, fallback: bool) -> bool:
    raw_value = _env(name)
    if not raw_value:
        return fallback
    return raw_value.lower() in {"1", "true", "yes"}


def _env_int(name: str, fallback: int) -> int:
    raw_value = _env(name)
    if not raw_value:
        return fallback

    try:
        return int(raw_value)
    except ValueError:
        return fallback


def _env_float(name: str, fallback: float) -> float:
    raw_value = _env(name)
    if not raw_value:
        return fallback

    try:
        return float(raw_value)
    except ValueError:
        return fallback


def default_cache_dir() -> Path:
    return Path(__file__).resolve().parents[2] / ".cache" / "rag"


def load_retrieval_settings() -> RetrievalSettings:
    return RetrievalSettings(
        enabled=not _env_bool("RAG_EMBEDDING_DISABLED", False),
        provider=_env("RAG_EMBEDDING_PROVIDER", "ollama"),
        base_url=(_env("RAG_EMBEDDING_BASE_URL") or "http://127.0.0.1:11434").rstrip("/"),
        model_name=_env("RAG_EMBEDDING_MODEL", "qwen3-embedding:0.6b"),
        cache_dir=Path(_env("RAG_EMBEDDING_CACHE_DIR")) if _env("RAG_EMBEDDING_CACHE_DIR") else default_cache_dir(),
        top_k=max(_env_int("RAG_TOP_K", 3), 1),
        keyword_top_k=max(_env_int("RAG_KEYWORD_TOP_K", 6), 1),
        embedding_min_score=_env_float("RAG_EMBEDDING_MIN_SCORE", 0.15),
        timeout_seconds=_env_float("RAG_EMBEDDING_TIMEOUT_SECONDS", 60.0),
        query_prefix=_env("RAG_QUERY_PREFIX", "为这个问题检索相关的项目资料："),
        document_prefix=_env("RAG_DOCUMENT_PREFIX", "项目资料片段："),
    )
