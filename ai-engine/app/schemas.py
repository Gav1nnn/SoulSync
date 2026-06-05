from pydantic import BaseModel, Field

from app.persona.profile import PersonaProfile, default_berry_persona


class MemoryContext(BaseModel):
    id: str
    content: str
    type: str = "project_fact"


class MemoryCandidate(BaseModel):
    type: str
    content: str
    reason: str
    confidence: float = Field(ge=0.0, le=1.0)


class GenerateRequest(BaseModel):
    user_message: str = Field(min_length=1)
    character_id: str = "berry"
    character_name: str = "Berry"
    persona: PersonaProfile = Field(default_factory=default_berry_persona)
    memories: list[MemoryContext] = Field(default_factory=list)


class GenerateResponse(BaseModel):
    reply: str
    persona: str
    context_used: list[str]
    used_persona: bool = True
    used_memory_ids: list[str] = Field(default_factory=list)
    used_knowledge_chunk_ids: list[str] = Field(default_factory=list)
    memory_written: bool = False
    memory_candidates: list[MemoryCandidate] = Field(default_factory=list)
