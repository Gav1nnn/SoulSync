from __future__ import annotations

from hashlib import sha1

from app.retrieval.schemas import KnowledgeChunk


def chunk_markdown_document(source_path: str, raw_text: str) -> list[KnowledgeChunk]:
    chunks: list[KnowledgeChunk] = []
    title = source_path.split("/")[-1]
    current_section = title
    current_lines: list[str] = []

    def flush_chunk() -> None:
        nonlocal current_lines
        content = "\n".join(line.strip() for line in current_lines if line.strip()).strip()
        if not content:
            current_lines = []
            return

        normalized_source = source_path.replace("\\", "/")
        digest = sha1(f"{normalized_source}:{current_section}:{content}".encode("utf-8")).hexdigest()[:12]
        chunks.append(
            KnowledgeChunk(
                chunk_id=f"chunk-{digest}",
                source_path=normalized_source,
                title=title,
                section=current_section,
                content=content,
            )
        )
        current_lines = []

    for line in raw_text.splitlines():
        stripped = line.strip()
        if stripped.startswith("#"):
            flush_chunk()
            current_section = stripped.lstrip("#").strip() or title
            continue

        if not stripped:
            flush_chunk()
            continue

        current_lines.append(stripped)

    flush_chunk()
    return chunks
