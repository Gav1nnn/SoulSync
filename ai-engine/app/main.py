"""HTTP 入口：对外暴露 /health、/generate、/llm/deepseek/chat。"""

from dotenv import load_dotenv
from fastapi import FastAPI, HTTPException

from app.config import llm_base_url, llm_enabled, llm_model, llm_provider
from app.llm import (
    LLMAPIError,
    LLMNotConfiguredError,
    chat_completion,
    is_configured,
)
from app.orchestration import generate_reply
from app.schemas import (
    DeepSeekChatRequest,
    DeepSeekChatResponse,
    GenerateRequest,
    GenerateResponse,
)

load_dotenv()

app = FastAPI(title="SoulSync AI Engine")


@app.get("/health")
def health() -> dict[str, str]:
    """供 Go / 运维探活。"""
    return {
        "service": "ai-engine",
        "status": "ok",
        "llm_configured": str(llm_enabled()).lower(),
        "llm_provider": llm_provider() if llm_enabled() else "none",
        "llm_model": llm_model() if llm_enabled() else "",
        "llm_base_url": llm_base_url() if llm_enabled() else "",
        # 兼容旧字段
        "deepseek_configured": str(llm_enabled()).lower(),
    }


@app.post("/generate", response_model=GenerateResponse)
def generate(payload: GenerateRequest) -> GenerateResponse:
    """Go 内部调用的生成接口；校验后交给 orchestration。"""
    user_message = payload.user_message.strip()
    if not user_message:
        raise HTTPException(status_code=400, detail="user_message must not be empty")

    return generate_reply(payload.model_copy(update={"user_message": user_message}))


@app.post("/llm/deepseek/chat", response_model=DeepSeekChatResponse)
def llm_chat(payload: DeepSeekChatRequest) -> DeepSeekChatResponse:
    """直连 LLM Chat Completions（Ollama / DeepSeek 等），用于调试。"""
    if not is_configured():
        raise HTTPException(
            status_code=503,
            detail="LLM is not configured (set LLM_PROVIDER=ollama or LLM_API_KEY)",
        )

    try:
        reply = chat_completion(payload.messages, model=payload.model)
    except LLMNotConfiguredError as exc:
        raise HTTPException(status_code=503, detail=str(exc)) from exc
    except LLMAPIError as exc:
        raise HTTPException(status_code=502, detail=str(exc)) from exc

    return DeepSeekChatResponse(
        reply=reply,
        model=payload.model or llm_model(),
        provider=llm_provider(),
    )
