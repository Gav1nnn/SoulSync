from __future__ import annotations

import json
import os
import socket
from dataclasses import dataclass
from urllib import error, request

from app.persona.examples import berry_few_shot_messages
from app.persona.profile import PersonaProfile
from app.persona.prompt_builder import build_persona_instruction


CHAT_COMPLETIONS_PATH = "/chat/completions"


class LLMClientError(RuntimeError):
    pass


@dataclass(frozen=True)
class LLMSettings:
    provider: str
    base_url: str
    model: str
    api_key: str
    enabled: bool
    timeout_seconds: float


def _env(name: str, fallback: str = "") -> str:
    return os.getenv(name, fallback).strip()


def _env_enabled(name: str) -> bool:
    return _env(name).lower() in {"1", "true", "yes"}


def _env_float(name: str, fallback: float) -> float:
    raw_value = _env(name)
    if not raw_value:
        return fallback

    try:
        return float(raw_value)
    except ValueError:
        return fallback


def current_provider() -> str:
    explicit = _env("LLM_PROVIDER").lower()
    if explicit:
        return explicit
    if _env("DEEPSEEK_API_KEY"):
        return "deepseek"
    return "ollama"


def llm_enabled() -> bool:
    return not _env_enabled("LLM_DISABLED")


def load_settings() -> LLMSettings:
    provider = current_provider()

    if provider == "deepseek":
        base_url = (_env("LLM_BASE_URL") or _env("DEEPSEEK_BASE_URL") or "https://api.deepseek.com").rstrip("/")
        model = _env("LLM_MODEL") or _env("DEEPSEEK_MODEL") or "deepseek-chat"
        api_key = _env("LLM_API_KEY") or _env("DEEPSEEK_API_KEY")
        timeout_seconds = _env_float("LLM_TIMEOUT_SECONDS", 60.0)
    else:
        base_url = (_env("LLM_BASE_URL") or "http://127.0.0.1:11434/v1").rstrip("/")
        model = _env("LLM_MODEL") or "qwen3.5:4b"
        api_key = _env("LLM_API_KEY") or "ollama"
        timeout_seconds = _env_float("LLM_TIMEOUT_SECONDS", 180.0)

    enabled = llm_enabled()
    if provider == "deepseek" and not api_key:
        enabled = False

    return LLMSettings(
        provider=provider,
        base_url=base_url,
        model=model,
        api_key=api_key,
        enabled=enabled,
        timeout_seconds=timeout_seconds,
    )


def generate_persona_reply(persona: PersonaProfile, character_name: str, user_message: str) -> tuple[str, str]:
    settings = load_settings()
    if not settings.enabled:
        raise LLMClientError("llm is disabled")

    payload = {
        "model": settings.model,
        "messages": [
            {
                "role": "system",
                "content": build_persona_instruction(persona, character_name),
            },
            *berry_few_shot_messages(),
            {
                "role": "user",
                "content": user_message,
            },
        ],
        "stream": False,
    }

    headers = {"Content-Type": "application/json"}
    if settings.api_key:
        headers["Authorization"] = f"Bearer {settings.api_key}"

    http_request = request.Request(
        url=settings.base_url + CHAT_COMPLETIONS_PATH,
        data=json.dumps(payload).encode("utf-8"),
        headers=headers,
        method="POST",
    )

    try:
        with request.urlopen(http_request, timeout=settings.timeout_seconds) as response:
            raw_body = response.read()
    except error.HTTPError as exc:
        body = exc.read().decode("utf-8", errors="ignore").strip()
        raise LLMClientError(f"llm returned status {exc.code}: {body}") from exc
    except error.URLError as exc:
        raise LLMClientError(f"llm request failed: {exc.reason}") from exc
    except (TimeoutError, socket.timeout) as exc:
        raise LLMClientError("llm request timed out") from exc
    except OSError as exc:
        raise LLMClientError(f"llm request failed: {exc}") from exc

    try:
        data = json.loads(raw_body.decode("utf-8"))
        reply = data["choices"][0]["message"]["content"].strip()
    except (KeyError, IndexError, TypeError, json.JSONDecodeError) as exc:
        raise LLMClientError("unexpected llm response shape") from exc

    return reply, settings.provider
