"""编排层：串联 persona 注入 → 生成（LLM 或 mock）→ 评估。"""

from app.config import llm_enabled, llm_provider_label
from app.evaluation import build_generate_response
from app.llm import LLMAPIError, generate_with_persona
from app.persona import build_mock_reply
from app.schemas import GenerateRequest, GenerateResponse


def generate_reply(request: GenerateRequest) -> GenerateResponse:
    """LLM 已配置时走 Ollama/DeepSeek 等；失败或未配置则 mock。"""
    user_message = request.user_message.strip()
    llm_provider: str | None = None

    if llm_enabled():
        try:
            reply = generate_with_persona(
                request.persona,
                request.character_name,
                user_message,
            )
            llm_provider = llm_provider_label()
        except LLMAPIError:
            reply = build_mock_reply(
                request.persona,
                request.character_name,
                user_message,
            )
            llm_provider = "mock_fallback"
    else:
        reply = build_mock_reply(
            request.persona,
            request.character_name,
            user_message,
        )

    return build_generate_response(
        reply=reply,
        character_name=request.character_name,
        persona=request.persona,
        llm_provider=llm_provider,
    )
