from fastapi import FastAPI, HTTPException
from pydantic import BaseModel, Field


app = FastAPI(title="SoulSync AI Engine")


class ReplyRequest(BaseModel):
    message: str = Field(min_length=1)


class ReplyResponse(BaseModel):
    reply: str
    persona: str
    context_used: list[str]


@app.get("/healthz")
def healthz() -> dict[str, str]:
    return {"service": "ai-engine", "status": "ok"}


@app.post("/reply", response_model=ReplyResponse)
def reply(payload: ReplyRequest) -> ReplyResponse:
    message = payload.message.strip()
    if not message:
        raise HTTPException(status_code=400, detail="message must not be empty")

    return ReplyResponse(
        reply=(
            f"我是 Berry。你这句“{message}”我已经接住了。"
            "先别急着乱堆页面，我会先帮你把结构、状态和接口边界捋顺，再开始写前端。"
        ),
        persona="Berry",
        context_used=["persona"],
    )
