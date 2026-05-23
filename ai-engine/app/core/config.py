"""LLM 配置：支持 Ollama 本地、DeepSeek 云端等 OpenAI 兼容端点。"""

import os

def _env(name: str, fallback: str = "") -> str:
    return os.getenv(name, fallback).strip()

def _llm_enabled() -> bool:
    if _env("LLM_DISABLED").lower() in ("1", "true", "yes"):
        return False
    return bool(_env("LLM_API_KEY") or _env("DEEPSEEK_API_KEY") or _env("LLM_BASE_URL"))

def _llm_provider() -> str:
    return _env("LLM_PROVIDER").lower()

def _llm_mode() -> str:
    return _env("ASK_MODE").lower()

def _llm_model() -> str:
    return _env("LLM_MODEL").lower()

def _llm_base_url() -> str:
    return _env("LLM_BASE_URL").lower()

