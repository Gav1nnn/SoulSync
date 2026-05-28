from __future__ import annotations

import json
from dataclasses import dataclass
from urllib import error, request

import numpy as np

from app.retrieval.config import RetrievalSettings


class EmbeddingUnavailableError(RuntimeError):
    pass


@dataclass
class OllamaEmbeddingEncoder:
    settings: RetrievalSettings

    def _build_input_texts(self, texts: list[str], prefix: str) -> list[str]:
        if not prefix:
            return texts
        return [f"{prefix}{text}" for text in texts]

    def _normalize(self, matrix: np.ndarray) -> np.ndarray:
        norms = np.linalg.norm(matrix, axis=1, keepdims=True)
        norms[norms == 0] = 1.0
        return (matrix / norms).astype(np.float32)

    def _embed_via_ollama(self, texts: list[str]) -> np.ndarray:
        payload = {
            "model": self.settings.model_name,
            "input": texts,
        }
        http_request = request.Request(
            url=self.settings.base_url + "/api/embed",
            data=json.dumps(payload).encode("utf-8"),
            headers={"Content-Type": "application/json"},
            method="POST",
        )

        try:
            with request.urlopen(http_request, timeout=self.settings.timeout_seconds) as response:
                raw_body = response.read()
        except error.HTTPError as exc:
            body = exc.read().decode("utf-8", errors="ignore").strip()
            raise EmbeddingUnavailableError(f"embedding service returned status {exc.code}: {body}") from exc
        except error.URLError as exc:
            raise EmbeddingUnavailableError(f"embedding service unavailable: {exc.reason}") from exc
        except TimeoutError as exc:
            raise EmbeddingUnavailableError("embedding request timed out") from exc
        except OSError as exc:
            raise EmbeddingUnavailableError(f"embedding request failed: {exc}") from exc

        try:
            data = json.loads(raw_body.decode("utf-8"))
        except json.JSONDecodeError as exc:
            raise EmbeddingUnavailableError("embedding service returned invalid json") from exc

        embeddings = data.get("embeddings")
        if embeddings is None:
            single_embedding = data.get("embedding")
            if single_embedding is not None:
                embeddings = [single_embedding]

        if not embeddings or not isinstance(embeddings, list):
            raise EmbeddingUnavailableError("embedding service returned an unexpected response shape")

        matrix = np.asarray(embeddings, dtype=np.float32)
        if matrix.ndim != 2:
            raise EmbeddingUnavailableError("embedding response has unexpected dimensions")

        return self._normalize(matrix)

    def encode(self, texts: list[str], prefix: str = "") -> np.ndarray:
        if not texts:
            return np.empty((0, 0), dtype=np.float32)

        prepared_texts = self._build_input_texts(texts, prefix)
        if self.settings.provider != "ollama":
            raise EmbeddingUnavailableError(f"unsupported embedding provider: {self.settings.provider}")
        return self._embed_via_ollama(prepared_texts)

    def encode_documents(self, texts: list[str]) -> np.ndarray:
        return self.encode(texts, prefix=self.settings.document_prefix)

    def encode_query(self, text: str) -> np.ndarray:
        matrix = self.encode([text], prefix=self.settings.query_prefix)
        if matrix.size == 0:
            raise EmbeddingUnavailableError("query embedding is empty")
        return matrix[0]


_ENCODER_CACHE: dict[str, OllamaEmbeddingEncoder] = {}


def get_encoder(settings: RetrievalSettings) -> OllamaEmbeddingEncoder:
    cache_key = f"{settings.provider}:{settings.base_url}:{settings.model_name}"
    encoder = _ENCODER_CACHE.get(cache_key)
    if encoder is None:
        encoder = OllamaEmbeddingEncoder(settings)
        _ENCODER_CACHE[cache_key] = encoder
    return encoder
