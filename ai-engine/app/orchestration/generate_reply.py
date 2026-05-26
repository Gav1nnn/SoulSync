from app.llm.client import LLMClientError, generate_persona_reply
from app.persona.examples import berry_few_shot_messages
from app.persona.mock import build_mock_reply
from app.schemas import GenerateRequest, GenerateResponse


def generate_reply(request: GenerateRequest) -> GenerateResponse:
    context_used = ["persona"]
    if berry_few_shot_messages():
        context_used.append("persona.examples")

    try:
        reply, provider = generate_persona_reply(
            request.persona,
            request.character_name,
            request.user_message,
        )
        context_used.append(provider)
    except LLMClientError:
        reply = build_mock_reply(
            request.persona,
            request.character_name,
            request.user_message,
        )
        context_used.append("mock_fallback")

    return GenerateResponse(
        reply=reply,
        persona=request.character_name,
        context_used=context_used,
        used_persona=True,
    )
