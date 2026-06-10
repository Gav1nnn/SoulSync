from pydantic import BaseModel, Field

from app.persona.profile import PersonaProfile, default_berry_persona


class MemoryContext(BaseModel):
    id: str
    content: str
    type: str = "project_fact"


class ConversationMessage(BaseModel):
    role: str
    content: str


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
    recent_messages: list[ConversationMessage] = Field(default_factory=list)


class GenerateResponse(BaseModel):
    reply: str
    persona: str
    context_used: list[str]
    used_persona: bool = True
    used_memory_ids: list[str] = Field(default_factory=list)
    used_knowledge_chunk_ids: list[str] = Field(default_factory=list)
    memory_written: bool = False
    memory_candidates: list[MemoryCandidate] = Field(default_factory=list)


class WorkspaceTreeItem(BaseModel):
    path: str
    type: str


class WorkspaceCandidate(BaseModel):
    path: str
    kind: str
    reason: str = ""


class WorkspaceSummary(BaseModel):
    workspace_path: str = ""
    root_name: str = ""
    tree: list[WorkspaceTreeItem] = Field(default_factory=list)
    package_managers: list[str] = Field(default_factory=list)
    frontend_frameworks: list[str] = Field(default_factory=list)
    backend_frameworks: list[str] = Field(default_factory=list)
    backend_route_candidates: list[WorkspaceCandidate] = Field(default_factory=list)
    type_file_candidates: list[WorkspaceCandidate] = Field(default_factory=list)
    frontend_entry_candidates: list[WorkspaceCandidate] = Field(default_factory=list)
    api_client_candidates: list[WorkspaceCandidate] = Field(default_factory=list)
    validation_commands: list[str] = Field(default_factory=list)


class AgentPlanAction(BaseModel):
    type: str
    path: str | None = None
    query: str | None = None
    command: str | None = None
    reason: str = ""


class AgentPlanRequest(BaseModel):
    goal: str = Field(min_length=1)
    workspace_summary: WorkspaceSummary
    character_name: str = "Berry"
    persona: PersonaProfile = Field(default_factory=default_berry_persona)
    memories: list[MemoryContext] = Field(default_factory=list)
    recent_messages: list[ConversationMessage] = Field(default_factory=list)
    project_context: list[str] = Field(default_factory=list)


class AgentPlanResponse(BaseModel):
    plan: list[str]
    files_to_read: list[str] = Field(default_factory=list)
    initial_action: AgentPlanAction
    context_used: list[str] = Field(default_factory=list)
    used_memory_ids: list[str] = Field(default_factory=list)
    used_knowledge_chunk_ids: list[str] = Field(default_factory=list)
    planner: str
