"""OpenAI 兼容 Chat Completions（Ollama / DeepSeek 等）。"""

from __future__ import annotations

import httpx

from app.config import llm_api_key, llm_base_url, llm_enabled, llm_model
from app.persona import PersonaProfile, build_system_prompt

CHAT_COMPLETIONS_PATH = "/chat/completions"
DEFAULT_TIMEOUT_SECONDS = 120.0


class LLMNotConfiguredError(RuntimeError):
    """未启用 LLM（未配置 Ollama 或 API Key）。"""


class LLMAPIError(RuntimeError):
    """上游 LLM 返回错误或响应格式异常。"""


# 兼容旧命名
DeepSeekNotConfiguredError = LLMNotConfiguredError
DeepSeekAPIError = LLMAPIError


def is_configured() -> bool:
    return llm_enabled()


def chat_completion(
    messages: list[dict[str, str]],
    *,
    model: str | None = None,
    timeout: float = DEFAULT_TIMEOUT_SECONDS,
) -> str:
    """调用 POST /chat/completions，返回 assistant 文本。"""
    if not llm_enabled():
        raise LLMNotConfiguredError("LLM is not configured (set LLM_PROVIDER=ollama or LLM_API_KEY)")

    api_key = llm_api_key()
    url = f"{llm_base_url()}{CHAT_COMPLETIONS_PATH}"
    payload = {
        "model": model or llm_model(),
        "messages": messages,
        "stream": False,
    }

    headers = {"Content-Type": "application/json"}
    if api_key:
        headers["Authorization"] = f"Bearer {api_key}"

    try:
        response = httpx.post(url, headers=headers, json=payload, timeout=timeout)
    except httpx.HTTPError as exc:
        raise LLMAPIError(f"llm request failed: {exc}") from exc

    if response.status_code >= 400:
        detail = response.text.strip() or response.reason_phrase
        raise LLMAPIError(f"llm returned status {response.status_code}: {detail}")

    try:
        data = response.json()
        return data["choices"][0]["message"]["content"].strip()
    except (KeyError, IndexError, TypeError) as exc:
        raise LLMAPIError("unexpected llm response shape") from exc


def generate_with_persona(
    persona: PersonaProfile,
    character_name: str,
    user_message: str,
) -> str:
    """用结构化 persona 构造 system prompt，再调用 LLM 生成回复。"""
    messages = [
        {
            "role": "system",
            "content": build_system_prompt(persona, character_name),
        },
        {"role": "user", "content": user_message},
    ]
    return chat_completion(messages)
