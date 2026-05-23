"""LLM 配置：支持 Ollama 本地、DeepSeek 云端等 OpenAI 兼容端点。"""

import os


def _env(name: str, fallback: str = "") -> str:
    return os.getenv(name, fallback).strip()


def llm_provider() -> str:
    """ollama | deepseek | 或自定义 openai_compatible。"""
    explicit = _env("LLM_PROVIDER").lower()
    if explicit:
        return explicit
    if _env("DEEPSEEK_API_KEY"):
        return "deepseek"
    return "openai_compatible"


def llm_base_url() -> str:
    custom = _env("LLM_BASE_URL")
    if custom:
        return custom.rstrip("/")

    provider = llm_provider()
    if provider == "ollama":
        return "http://127.0.0.1:11434/v1"
    if _env("DEEPSEEK_BASE_URL"):
        return _env("DEEPSEEK_BASE_URL", "https://api.deepseek.com").rstrip("/")
    return "https://api.deepseek.com"


def llm_api_key() -> str:
    key = _env("LLM_API_KEY") or _env("DEEPSEEK_API_KEY")
    if key:
        return key
    # Ollama 不校验 key，但当前 HTTP 客户端仍会带 Bearer 头
    if llm_provider() == "ollama":
        return "ollama"
    return ""


def llm_model() -> str:
    model = _env("LLM_MODEL") or _env("DEEPSEEK_MODEL")
    if model:
        return model
    if llm_provider() == "ollama":
        return "qwen2.5:3b"
    return "deepseek-chat"


def llm_enabled() -> bool:
    if _env("LLM_DISABLED").lower() in ("1", "true", "yes"):
        return False
    if llm_provider() == "ollama":
        return True
    return bool(_env("LLM_API_KEY") or _env("DEEPSEEK_API_KEY") or _env("LLM_BASE_URL"))


def llm_provider_label() -> str:
    """写入 context_used 的标签。"""
    return llm_provider()


# 兼容旧代码引用
def deepseek_api_key() -> str:
    return llm_api_key()


def deepseek_base_url() -> str:
    return llm_base_url()


def deepseek_model() -> str:
    return llm_model()


def deepseek_enabled() -> bool:
    return llm_enabled()
