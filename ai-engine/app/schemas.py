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


class APICandidate(BaseModel):
    method: str
    path: str
    handler: str = ""
    handler_file: str = ""
    type_definitions: list[WorkspaceCandidate] = Field(default_factory=list)
    reason: str = ""


class WorkspaceSummary(BaseModel):
    workspace_path: str = ""
    root_name: str = ""
    tree: list[WorkspaceTreeItem] = Field(default_factory=list)
    package_managers: list[str] = Field(default_factory=list)
    frontend_frameworks: list[str] = Field(default_factory=list)
    backend_frameworks: list[str] = Field(default_factory=list)
    backend_route_candidates: list[WorkspaceCandidate] = Field(default_factory=list)
    api_candidates: list[APICandidate] = Field(default_factory=list)
    project_doc_candidates: list[WorkspaceCandidate] = Field(default_factory=list)
    type_file_candidates: list[WorkspaceCandidate] = Field(default_factory=list)
    frontend_entry_candidates: list[WorkspaceCandidate] = Field(default_factory=list)
    api_client_candidates: list[WorkspaceCandidate] = Field(default_factory=list)
    validation_commands: list[str] = Field(default_factory=list)


class AgentPlanAction(BaseModel):
    type: str
    path: str | None = None
    query: str | None = None
    command: str | None = None
    content: str | None = None
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


class AgentObservation(BaseModel):
    status: str
    message: str
    path: str | None = None
    items: list[str] = Field(default_factory=list)
    matches: list[str] = Field(default_factory=list)
    content: str | None = None
    command: str | None = None
    output: list[str] = Field(default_factory=list)


class AgentReadFile(BaseModel):
    path: str
    content: str


class AgentTaskStep(BaseModel):
    index: int
    action: AgentPlanAction
    observation: AgentObservation
    summary: str = ""
    context_used: list[str] = Field(default_factory=list)
    stepper: str = ""
    started_at: str | None = None
    finished_at: str | None = None
    duration_ms: int = 0


class AgentStepRequest(BaseModel):
    goal: str = Field(min_length=1)
    plan: list[str] = Field(default_factory=list)
    workspace_summary: WorkspaceSummary
    step_index: int = 1
    previous_observation: AgentObservation | None = None
    read_files: list[AgentReadFile] = Field(default_factory=list)
    changed_files: list[str] = Field(default_factory=list)
    recent_steps: list[AgentTaskStep] = Field(default_factory=list)
    character_name: str = "Berry"
    persona: PersonaProfile = Field(default_factory=default_berry_persona)
    project_context: list[str] = Field(default_factory=list)


class AgentStepResponse(BaseModel):
    action: AgentPlanAction
    summary: str = ""
    context_used: list[str] = Field(default_factory=list)
    stepper: str
