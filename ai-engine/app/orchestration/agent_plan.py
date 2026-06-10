from __future__ import annotations

import json

from app.llm.client import LLMClientError, build_conversation_context, build_memory_context, build_knowledge_context
from app.llm.client import load_settings, request_chat_completion
from app.persona.prompt_builder import build_persona_instruction
from app.retrieval.retriever import retrieve_knowledge_result
from app.schemas import AgentPlanAction, AgentPlanRequest, AgentPlanResponse


def generate_agent_plan(request: AgentPlanRequest) -> AgentPlanResponse:
    context_used = ["persona", "workspace.summary"]
    used_memory_ids = [memory.id for memory in request.memories]
    if used_memory_ids:
        context_used.append("memory")
    if request.recent_messages:
        context_used.append("conversation")

    knowledge_hits = []
    retrieval_strategies: list[str] = []
    try:
        retrieval_result = retrieve_knowledge_result(request.goal)
        knowledge_hits = retrieval_result.hits
        retrieval_strategies = retrieval_result.strategies
    except Exception:
        context_used.append("knowledge_unavailable")

    if knowledge_hits:
        context_used.append("knowledge")
        context_used.extend(retrieval_strategies)
        used_knowledge_chunk_ids = [hit.chunk.chunk_id for hit in knowledge_hits]
    else:
        context_used.extend(retrieval_strategies)
        used_knowledge_chunk_ids = []

    try:
        response = generate_llm_agent_plan(request, knowledge_hits)
        context_used.append(response.planner)
        return response.model_copy(
            update={
                "context_used": context_used,
                "used_memory_ids": used_memory_ids,
                "used_knowledge_chunk_ids": used_knowledge_chunk_ids,
            }
        )
    except LLMClientError:
        response = build_fallback_agent_plan(request)
        context_used.append("mock_planner")
        return response.model_copy(
            update={
                "context_used": context_used,
                "used_memory_ids": used_memory_ids,
                "used_knowledge_chunk_ids": used_knowledge_chunk_ids,
            }
        )


def generate_llm_agent_plan(request: AgentPlanRequest, knowledge_hits) -> AgentPlanResponse:
    settings = load_settings()
    if not settings.enabled:
        raise LLMClientError("llm is disabled")

    messages = [
        {
            "role": "system",
            "content": (
                build_persona_instruction(request.persona, request.character_name)
                + "\n\n你现在是 SoulSync 的前端开发 Agent planner。"
                "只输出 JSON，不要输出 markdown。"
                "你只能制定短计划和建议读取文件，不能声称已经修改文件。"
                "Go 会执行文件和命令动作，Python 不直接操作文件或 shell。"
                "JSON 格式：{\"plan\": [\"...\"], \"files_to_read\": [\"path\"], "
                "\"initial_action\": {\"type\": \"read_file|list_dir|search_text|finish\", "
                "\"path\": \"...\", \"query\": \"...\", \"reason\": \"...\"}}。"
            ),
        },
        {
            "role": "system",
            "content": build_workspace_context(request),
        },
    ]
    if request.memories:
        messages.append({"role": "system", "content": build_memory_context(request.memories)})
    if request.recent_messages:
        messages.append({"role": "system", "content": build_conversation_context(request.recent_messages)})
    if knowledge_hits:
        messages.append({"role": "system", "content": build_knowledge_context(knowledge_hits)})
    messages.append({"role": "user", "content": request.goal})

    response_format = {"type": "json_object"} if settings.provider == "deepseek" else None
    content = request_chat_completion(
        settings,
        messages,
        response_format=response_format,
        max_tokens=900,
    )
    data = parse_json_object(content)
    data["planner"] = settings.provider
    return normalize_agent_plan_response(request, data)


def build_fallback_agent_plan(request: AgentPlanRequest) -> AgentPlanResponse:
    summary = request.workspace_summary
    files_to_read = unique_non_empty(
        [
            first_path(summary.backend_route_candidates),
            first_path(summary.api_client_candidates),
            first_path(summary.frontend_entry_candidates),
            first_path(summary.type_file_candidates),
        ]
    )
    plan = [
        f"确认目标：{request.goal}",
        "读取 workspace summary，先定位接口、API client、页面入口和类型文件。",
    ]
    if first_path(summary.backend_route_candidates):
        plan.append(f"读取后端路由候选：{first_path(summary.backend_route_candidates)}")
    if first_path(summary.api_client_candidates):
        plan.append(f"检查 API client 约定：{first_path(summary.api_client_candidates)}")
    if first_path(summary.frontend_entry_candidates):
        plan.append(f"参考前端页面入口：{first_path(summary.frontend_entry_candidates)}")
    plan.append("给出第一轮只读动作，等待 Go 执行。")

    if files_to_read:
        initial_action = AgentPlanAction(
            type="read_file",
            path=files_to_read[0],
            reason="先读取最相关的接口或前端入口候选。",
        )
    else:
        initial_action = AgentPlanAction(
            type="list_dir",
            path=".",
            reason="summary 中没有明确候选，先列出项目根目录。",
        )

    return AgentPlanResponse(
        plan=plan,
        files_to_read=files_to_read,
        initial_action=initial_action,
        planner="mock_planner",
    )


def build_workspace_context(request: AgentPlanRequest) -> str:
    summary = request.workspace_summary
    sections = [
        f"Workspace: {summary.root_name} ({summary.workspace_path})",
        f"Package managers: {', '.join(summary.package_managers) or 'none'}",
        f"Frontend frameworks: {', '.join(summary.frontend_frameworks) or 'none'}",
        f"Backend frameworks: {', '.join(summary.backend_frameworks) or 'none'}",
        "Backend route candidates:\n" + candidate_lines(summary.backend_route_candidates),
        "API candidates:\n" + api_candidate_lines(summary.api_candidates),
        "API client candidates:\n" + candidate_lines(summary.api_client_candidates),
        "Frontend entry candidates:\n" + candidate_lines(summary.frontend_entry_candidates),
        "Type file candidates:\n" + candidate_lines(summary.type_file_candidates),
        "Validation commands:\n" + "\n".join(f"- {command}" for command in summary.validation_commands[:8]),
    ]
    if request.project_context:
        sections.append("Project context:\n" + "\n".join(f"- {item}" for item in request.project_context[:12]))
    return "\n\n".join(sections)


def candidate_lines(candidates) -> str:
    if not candidates:
        return "- none"
    return "\n".join(f"- {candidate.path} ({candidate.kind}): {candidate.reason}" for candidate in candidates[:8])


def api_candidate_lines(candidates) -> str:
    if not candidates:
        return "- none"
    lines = []
    for candidate in candidates[:8]:
        type_paths = ", ".join(item.path for item in candidate.type_definitions[:4]) or "none"
        lines.append(
            f"- {candidate.method} {candidate.path} handler={candidate.handler} "
            f"file={candidate.handler_file} types={type_paths}"
        )
    return "\n".join(lines)


def parse_json_object(content: str) -> dict:
    raw = content.strip()
    if raw.startswith("```"):
        raw = raw.strip("`")
        raw = raw.removeprefix("json").strip()
    try:
        data = json.loads(raw)
    except json.JSONDecodeError as exc:
        raise LLMClientError("planner returned invalid json") from exc
    if not isinstance(data, dict):
        raise LLMClientError("planner returned non-object json")
    return data


def normalize_agent_plan_response(request: AgentPlanRequest, data: dict) -> AgentPlanResponse:
    plan = data.get("plan")
    files_to_read = data.get("files_to_read", [])
    initial_action = data.get("initial_action")
    if not isinstance(plan, list) or not all(isinstance(item, str) and item.strip() for item in plan):
        raise LLMClientError("planner response missing plan")
    if not isinstance(files_to_read, list):
        files_to_read = []
    files_to_read = [item for item in files_to_read if isinstance(item, str) and item.strip()]
    if not isinstance(initial_action, dict):
        fallback = build_fallback_agent_plan(request)
        initial_action = fallback.initial_action.model_dump()

    return AgentPlanResponse(
        plan=plan[:8],
        files_to_read=unique_non_empty(files_to_read[:8]),
        initial_action=AgentPlanAction.model_validate(initial_action),
        planner=data.get("planner") or "llm",
    )


def first_path(candidates) -> str:
    if not candidates:
        return ""
    return candidates[0].path


def unique_non_empty(values: list[str]) -> list[str]:
    seen = set()
    result = []
    for value in values:
        trimmed = value.strip()
        if not trimmed or trimmed in seen:
            continue
        seen.add(trimmed)
        result.append(trimmed)
    return result
