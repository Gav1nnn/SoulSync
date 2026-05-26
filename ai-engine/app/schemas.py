from pydantic import BaseModel, Field

from app.persona.profile import PersonaProfile, default_berry_persona


class GenerateRequest(BaseModel):
    user_message: str = Field(min_length=1)
    character_id: str = "berry"
    character_name: str = "Berry"
    persona: PersonaProfile = Field(default_factory=default_berry_persona)


class GenerateResponse(BaseModel):
    reply: str
    persona: str
    context_used: list[str]
    used_persona: bool = True
    used_memory_ids: list[str] = Field(default_factory=list)
    used_knowledge_chunk_ids: list[str] = Field(default_factory=list)
    memory_written: bool = False
