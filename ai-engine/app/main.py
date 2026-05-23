"""HTTP 入口：对外暴露 /health、/generate、/llm/deepseek/chat。"""

from dotenv import load_dotenv
from fastapi import FastAPI, HTTPException
import sys
import os

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

from core.config import _llm_enabled, _llm_provider, _llm_model, _llm_base_url
from core.orchestration import generate_reply
from core.schemas import (
    GenerateRequest,
    GenerateResponse,
)
from core.RAG import RAG

load_dotenv()

app = FastAPI(title="SoulSync AI Engine")
r = RAG()
r._run_rag()


@app.get("/health")
def health() -> dict[str, str]:
    """供 Go / 运维探活。"""
    return {
        "service": "ai-engine",
        "status": "ok",
        "llm_configured": str(_llm_enabled()).lower(),
        "llm_provider": _llm_provider() if _llm_enabled() else "none",
        "llm_model": _llm_model() if _llm_enabled() else "",
        "llm_base_url": _llm_base_url() if _llm_enabled() else "",
        # 兼容旧字段
        "deepseek_configured": str(_llm_enabled()).lower(),
    }


@app.post("/generate", response_model=GenerateResponse)
def generate(payload: GenerateRequest) -> GenerateResponse:
    """Go 内部调用的生成接口；校验后交给 orchestration。"""
    user_message = payload.user_message.strip()
    print("用户消息：", user_message)
    if not user_message:
        raise HTTPException(status_code=400, detail="user_message must not be empty")

    return generate_reply(payload.model_copy(update={"user_message": user_message}),r)
