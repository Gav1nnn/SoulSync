from __future__ import annotations

from pathlib import Path

from app.retrieval.chunker import chunk_markdown_document
from app.retrieval.schemas import KnowledgeChunk


def default_docs_root() -> Path:
    return Path(__file__).resolve().parents[3] / "docs"


def load_docs_chunks(docs_root: Path | None = None) -> list[KnowledgeChunk]:
    root = docs_root or default_docs_root()
    chunks: list[KnowledgeChunk] = []

    if not root.exists():
        return chunks

    for path in sorted(root.rglob("*")):
        if not path.is_file() or path.suffix.lower() not in {".md", ".txt"}:
            continue

        raw_text = path.read_text(encoding="utf-8")
        relative_path = str(path.relative_to(root.parent))
        chunks.extend(chunk_markdown_document(relative_path, raw_text))

    return chunks
