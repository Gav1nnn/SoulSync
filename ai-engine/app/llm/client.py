from __future__ import annotations

import json
import os
import re
import socket
from dataclasses import dataclass
from urllib import error, request

from app.persona.examples import berry_few_shot_messages
from app.persona.profile import PersonaProfile
from app.persona.prompt_builder import build_persona_instruction
from app.retrieval.schemas import RetrievalHit
from app.schemas import ConversationMessage, MemoryCandidate, MemoryContext


CHAT_COMPLETIONS_PATH = "/chat/completions"
USER_PROFILE_PATTERNS = (
    r"我叫\s*[\w\u4e00-\u9fff]+",
    r"我的名字是\s*[\w\u4e00-\u9fff]+",
    r"以后叫我\s*[\w\u4e00-\u9fff]+",
    r"叫我\s*[\w\u4e00-\u9fff]+",
    r"称呼我为?\s*[\w\u4e00-\u9fff]+",
    r"my name is\s+[\w-]+",
    r"call me\s+[\w-]+",
    r"i am\s+[\w-]+",
    r"i'm\s+[\w-]+",
)
USER_PROFILE_QUESTION_MARKERS = (
    "我叫什么",
    "我叫啥",
    "我的名字是什么",
    "名字是什么",
    "还记得我叫",
    "还记得我的名字",
    "怎么称呼我",
)


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
    thinking_disabled: bool


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
        model = _env("LLM_MODEL") or _env("DEEPSEEK_MODEL") or "deepseek-v4-flash"
        api_key = _env("LLM_API_KEY") or _env("DEEPSEEK_API_KEY")
        timeout_seconds = _env_float("LLM_TIMEOUT_SECONDS", 60.0)
        thinking_disabled = not _env_enabled("DEEPSEEK_THINKING_ENABLED")
    else:
        base_url = (_env("LLM_BASE_URL") or "http://127.0.0.1:11434/v1").rstrip("/")
        model = _env("LLM_MODEL") or "qwen3.5:4b"
        api_key = _env("LLM_API_KEY") or "ollama"
        timeout_seconds = _env_float("LLM_TIMEOUT_SECONDS", 180.0)
        thinking_disabled = False

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
        thinking_disabled=thinking_disabled,
    )


def generate_persona_reply(persona: PersonaProfile, character_name: str, user_message: str) -> tuple[str, str]:
    return generate_persona_reply_with_knowledge(
        persona,
        character_name,
        user_message,
        knowledge_hits=[],
    )


def build_knowledge_context(knowledge_hits: list[RetrievalHit]) -> str:
    lines = ["以下是与当前问题相关的项目资料，请优先基于这些资料回答，并在必要时明确说明资料来源："]

    for index, hit in enumerate(knowledge_hits, start=1):
        chunk = hit.chunk
        lines.append(
            f"[{index}] id={chunk.chunk_id} source={chunk.source_path} section={chunk.section}\n{chunk.content}"
        )

    return "\n\n".join(lines)


def build_memory_context(memories: list[MemoryContext]) -> str:
    lines = ["以下是长期项目记忆。请把它们当作当前项目和用户偏好的稳定上下文，优先遵守："]

    for index, memory in enumerate(memories, start=1):
        lines.append(f"[{index}] id={memory.id} type={memory.type}\n{memory.content}")

    return "\n\n".join(lines)


def build_conversation_context(recent_messages: list[ConversationMessage]) -> str:
    lines = ["以下是最近对话上下文。它只用于理解当前连续对话，不代表长期事实："]

    for message in recent_messages:
        role = "用户" if message.role == "user" else "Berry"
        lines.append(f"{role}: {message.content}")

    return "\n".join(lines)


def request_chat_completion(
    settings: LLMSettings,
    messages: list[dict[str, str]],
    *,
    response_format: dict[str, str] | None = None,
    max_tokens: int | None = None,
) -> str:
    payload: dict[str, object] = {
        "model": settings.model,
        "messages": messages,
        "stream": False,
    }
    if settings.provider == "deepseek" and settings.thinking_disabled:
        payload["thinking"] = {"type": "disabled"}
    if response_format:
        payload["response_format"] = response_format
    if max_tokens is not None:
        payload["max_tokens"] = max_tokens

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
        return data["choices"][0]["message"]["content"].strip()
    except (KeyError, IndexError, TypeError, json.JSONDecodeError) as exc:
        raise LLMClientError("unexpected llm response shape") from exc


def generate_persona_reply_with_knowledge(
    persona: PersonaProfile,
    character_name: str,
    user_message: str,
    knowledge_hits: list[RetrievalHit],
) -> tuple[str, str]:
    return generate_persona_reply_with_context(
        persona,
        character_name,
        user_message,
        knowledge_hits,
        memories=[],
        recent_messages=[],
    )


def generate_persona_reply_with_context(
    persona: PersonaProfile,
    character_name: str,
    user_message: str,
    knowledge_hits: list[RetrievalHit],
    memories: list[MemoryContext],
    recent_messages: list[ConversationMessage],
) -> tuple[str, str]:
    settings = load_settings()
    if not settings.enabled:
        raise LLMClientError("llm is disabled")

    messages: list[dict[str, str]] = [
        {
            "role": "system",
            "content": build_persona_instruction(persona, character_name),
        },
        *berry_few_shot_messages(),
    ]
    if memories:
        messages.append(
            {
                "role": "system",
                "content": build_memory_context(memories),
            }
        )
    if recent_messages:
        messages.append(
            {
                "role": "system",
                "content": build_conversation_context(recent_messages),
            }
        )
    if knowledge_hits:
        messages.append(
            {
                "role": "system",
                "content": build_knowledge_context(knowledge_hits),
            }
        )
    messages.append(
        {
            "role": "user",
            "content": user_message,
        }
    )

    return request_chat_completion(settings, messages), settings.provider


def extract_memory_candidates(user_message: str, assistant_reply: str) -> list[MemoryCandidate]:
    settings = load_settings()
    if not settings.enabled:
        raise LLMClientError("llm is disabled")

    messages = [
        {
            "role": "system",
            "content": (
                "你是 SoulSync 的长期记忆抽取器。"
                "只抽取对后续前端协作长期有用的信息，例如用户姓名/称呼、项目技术栈、接口约定、用户偏好、前端约定、调试结论。"
                "不要抽取寒暄、一次性任务、普通问题、已经明显过期的信息。"
                "如果用户明确说“我叫...”“我是...”“以后叫我...”，应抽取为 user_profile。"
                "不要把助手自己的名字、身份、人设或自我介绍抽取为用户画像。"
                "只返回 JSON，格式为 {\"memories\": [{\"type\": \"user_profile|project_fact|user_preference|frontend_convention|api_convention|debug_note\", "
                "\"content\": \"...\", \"reason\": \"...\", \"confidence\": 0.0}]}。没有可保存记忆时返回 {\"memories\": []}。"
            ),
        },
        {
            "role": "user",
            "content": f"用户消息：{user_message}\n\n助手回复：{assistant_reply}",
        },
    ]

    try:
        response_format = {"type": "json_object"} if settings.provider == "deepseek" else None
        content = request_chat_completion(
            settings,
            messages,
            response_format=response_format,
            max_tokens=800,
        )
        raw = content.strip()
        if raw.startswith("```"):
            raw = raw.strip("`")
            raw = raw.removeprefix("json").strip()
        data = json.loads(raw)
        memories = data.get("memories", [])
        if not isinstance(memories, list):
            raise LLMClientError("memory extractor returned an unexpected response shape")
        candidates = [MemoryCandidate.model_validate(item) for item in memories]
        return filter_memory_candidates(user_message, candidates)
    except (json.JSONDecodeError, TypeError, ValueError) as exc:
        raise LLMClientError("memory extractor returned invalid json") from exc


def filter_memory_candidates(user_message: str, candidates: list[MemoryCandidate]) -> list[MemoryCandidate]:
    filtered: list[MemoryCandidate] = []
    normalized_message = user_message.lower()

    for candidate in candidates:
        if candidate.type == "user_profile" and not user_message_declares_profile(normalized_message):
            continue
        if candidate.type == "user_profile" and re.search(r"\bberry\b|berry", candidate.content, flags=re.IGNORECASE):
            continue
        filtered.append(candidate)

    return filtered


def user_message_declares_profile(normalized_message: str) -> bool:
    if "?" in normalized_message or "？" in normalized_message:
        return False
    if any(marker in normalized_message for marker in USER_PROFILE_QUESTION_MARKERS):
        return False
    return any(re.search(pattern, normalized_message, flags=re.IGNORECASE) for pattern in USER_PROFILE_PATTERNS)
