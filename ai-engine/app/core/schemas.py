"""Go ↔ Python 的 /generate 请求与响应契约。"""

from pydantic import BaseModel, Field

from core.persona import PersonaProfile, default_berry_persona


class GenerateRequest(BaseModel):
    """由 Go 传入：用户消息 + 角色标识 + 结构化 persona。"""

    user_message: str = Field(min_length=1)
    character_id: str = "berry"
    character_name: str = "Berry"
    # 未传时使用默认 Berry；直连 ai-engine 测试时同样生效
    persona: PersonaProfile = Field(default_factory=default_berry_persona)


class DeepSeekChatRequest(BaseModel):
    """直连 DeepSeek 的调试/扩展接口；messages 遵循 OpenAI Chat 格式。"""

    messages: list[dict[str, str]] = Field(min_length=1)
    model: str | None = None


class DeepSeekChatResponse(BaseModel):
    reply: str
    model: str
    provider: str = "ollama"


class GenerateResponse(BaseModel):
    """回复正文 + 供 Go 写入 GenerationTrace 的元数据。"""

    reply: str
    persona: str  # 角色展示名，如 "Berry"
    context_used: list[str]
    used_persona: bool = True
    # 以下字段为 Stage 3/4 预留，当前固定为空/false
    used_memory_ids: list[str] = Field(default_factory=list)
    used_knowledge_chunk_ids: list[str] = Field(default_factory=list)
    memory_written: bool = False
