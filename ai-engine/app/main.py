from fastapi import FastAPI, HTTPException

from app.orchestration import generate_reply
from app.schemas import GenerateRequest, GenerateResponse

app = FastAPI(title="SoulSync AI Engine")


@app.get("/health")
def health() -> dict[str, str]:
    return {"service": "ai-engine", "status": "ok"}


@app.post("/generate", response_model=GenerateResponse)
def generate(payload: GenerateRequest) -> GenerateResponse:
    user_message = payload.user_message.strip()
    if not user_message:
        raise HTTPException(status_code=400, detail="user_message must not be empty")

    return generate_reply(payload.model_copy(update={"user_message": user_message}))
