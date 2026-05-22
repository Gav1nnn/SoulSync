from app.schemas import GenerateResponse
from app.persona import PersonaProfile


def build_generate_response(
    *,
    reply: str,
    character_name: str,
    persona: PersonaProfile,
) -> GenerateResponse:
    context_used = ["persona"]
    if persona.expertise:
        context_used.append("persona.expertise")
    if persona.sample_lines:
        context_used.append("persona.sample_lines")

    return GenerateResponse(
        reply=reply,
        persona=character_name,
        context_used=context_used,
        used_persona=True,
        used_memory_ids=[],
        used_knowledge_chunk_ids=[],
        memory_written=False,
    )
