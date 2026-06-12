from dotenv import load_dotenv
from fastapi import FastAPI, HTTPException

from app.llm.client import current_provider, llm_enabled
from app.orchestration.agent_plan import generate_agent_plan
from app.orchestration.agent_step import generate_agent_step
from app.orchestration.generate_reply import generate_reply
from app.schemas import AgentPlanRequest, AgentPlanResponse, AgentStepRequest, AgentStepResponse, GenerateRequest, GenerateResponse


load_dotenv()

app = FastAPI(title="SoulSync AI Engine")


@app.get("/healthz")
def healthz() -> dict[str, str]:
    return {
        "service": "ai-engine",
        "status": "ok",
        "llm_enabled": str(llm_enabled()).lower(),
        "llm_provider": current_provider() if llm_enabled() else "mock",
    }


@app.post("/generate", response_model=GenerateResponse)
def generate(payload: GenerateRequest) -> GenerateResponse:
    user_message = payload.user_message.strip()
    if not user_message:
        raise HTTPException(status_code=400, detail="user_message must not be empty")

    return generate_reply(payload.model_copy(update={"user_message": user_message}))


@app.post("/agent/plan", response_model=AgentPlanResponse)
def agent_plan(payload: AgentPlanRequest) -> AgentPlanResponse:
    goal = payload.goal.strip()
    if not goal:
        raise HTTPException(status_code=400, detail="goal must not be empty")

    return generate_agent_plan(payload.model_copy(update={"goal": goal}))


@app.post("/agent/step", response_model=AgentStepResponse)
def agent_step(payload: AgentStepRequest) -> AgentStepResponse:
    goal = payload.goal.strip()
    if not goal:
        raise HTTPException(status_code=400, detail="goal must not be empty")

    return generate_agent_step(payload.model_copy(update={"goal": goal}))
