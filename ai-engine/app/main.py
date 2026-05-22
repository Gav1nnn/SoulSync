"""HTTP 入口：对外暴露 /health 与 /generate，不包含生成逻辑。"""

from fastapi import FastAPI, HTTPException

from app.orchestration import generate_reply
from app.schemas import GenerateRequest, GenerateResponse

app = FastAPI(title="SoulSync AI Engine")


@app.get("/health")
def health() -> dict[str, str]:
    """供 Go / 运维探活。"""
    return {"service": "ai-engine", "status": "ok"}


@app.post("/generate", response_model=GenerateResponse)
def generate(payload: GenerateRequest) -> GenerateResponse:
    """Go 内部调用的生成接口；校验后交给 orchestration。"""
    user_message = payload.user_message.strip()
    if not user_message:
        raise HTTPException(status_code=400, detail="user_message must not be empty")

    return generate_reply(payload.model_copy(update={"user_message": user_message}))
