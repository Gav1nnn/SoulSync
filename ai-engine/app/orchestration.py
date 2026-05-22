from app.evaluation import build_generate_response
from app.persona import build_mock_reply
from app.schemas import GenerateRequest, GenerateResponse


def generate_reply(request: GenerateRequest) -> GenerateResponse:
    """Stage 1 orchestration: persona injection + mock generation."""
    user_message = request.user_message.strip()
    reply = build_mock_reply(
        request.persona,
        request.character_name,
        user_message,
    )
    return build_generate_response(
        reply=reply,
        character_name=request.character_name,
        persona=request.persona,
    )
