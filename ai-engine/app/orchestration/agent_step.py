from __future__ import annotations

import json

from app.llm.client import LLMClientError, load_settings, request_chat_completion
from app.persona.prompt_builder import build_persona_instruction
from app.schemas import AgentPlanAction, AgentStepRequest, AgentStepResponse


ALLOWED_ACTIONS = {"read_file", "list_dir", "search_text", "write_file", "run_command", "finish"}


def generate_agent_step(request: AgentStepRequest) -> AgentStepResponse:
    context_used = ["persona", "workspace.summary", "agent.observation"]
    if request.read_files:
        context_used.append("agent.read_files")
    if request.changed_files:
        context_used.append("agent.changed_files")

    try:
        response = generate_llm_agent_step(request)
        context_used.append(response.stepper)
        return response.model_copy(update={"context_used": context_used})
    except LLMClientError:
        response = build_fallback_agent_step(request)
        context_used.append("mock_stepper")
        return response.model_copy(update={"context_used": context_used})


def generate_llm_agent_step(request: AgentStepRequest) -> AgentStepResponse:
    settings = load_settings()
    if not settings.enabled:
        raise LLMClientError("llm is disabled")

    messages = [
        {
            "role": "system",
            "content": (
                build_persona_instruction(request.persona, request.character_name)
                + "\n\n你现在是 SoulSync 的 ReAct stepper。只输出 JSON，不要输出 markdown。"
                "Python 只决定下一步，Go 执行文件和命令动作。"
                "不要输出思维链。只给 action 和对上一轮 observation 的简短 summary。"
                "允许 action.type: read_file, list_dir, search_text, write_file, run_command, finish。"
                "write_file 必须给 path 和完整 content；run_command 只能使用 workspace summary 中列出的 validation command。"
                "JSON 格式：{\"summary\":\"...\", \"action\":{\"type\":\"...\", \"path\":\"...\", "
                "\"query\":\"...\", \"command\":\"...\", \"content\":\"...\", \"reason\":\"...\"}}。"
            ),
        },
        {"role": "system", "content": build_step_context(request)},
        {"role": "user", "content": request.goal},
    ]

    response_format = {"type": "json_object"} if settings.provider == "deepseek" else None
    content = request_chat_completion(
        settings,
        messages,
        response_format=response_format,
        max_tokens=1400,
    )
    data = parse_json_object(content)
    data["stepper"] = settings.provider
    return normalize_agent_step_response(request, data)


def build_fallback_agent_step(request: AgentStepRequest) -> AgentStepResponse:
    observation = request.previous_observation
    if observation is None:
        return AgentStepResponse(
            summary="缺少上一轮 observation，先结束任务。",
            action=AgentPlanAction(type="finish", reason="没有可继续执行的 observation。"),
            stepper="mock_stepper",
        )

    if observation.status != "ok":
        return AgentStepResponse(
            summary=f"上一轮动作失败：{observation.message}",
            action=AgentPlanAction(type="finish", reason="上一轮动作失败，停止继续改动。"),
            stepper="mock_stepper",
        )

    if request.step_index <= 2:
        next_path = next_unread_file(request)
        if next_path:
            return AgentStepResponse(
                summary=f"已读取 {observation.path or 'workspace'}，继续读取 planner 指定的下一个候选文件。",
                action=AgentPlanAction(type="read_file", path=next_path, reason="补齐项目上下文后再决定改动。"),
                stepper="mock_stepper",
            )

    return AgentStepResponse(
        summary=build_finish_summary(request),
        action=AgentPlanAction(type="finish", reason="当前阶段完成只读项目理解，等待后续接口识别和生成阶段。"),
        stepper="mock_stepper",
    )


def build_step_context(request: AgentStepRequest) -> str:
    summary = request.workspace_summary
    sections = [
        f"Step index: {request.step_index}",
        f"Goal: {request.goal}",
        "Plan:\n" + "\n".join(f"- {item}" for item in request.plan[:8]),
        f"Workspace: {summary.root_name} ({summary.workspace_path})",
        "API candidates:\n" + api_candidate_lines(summary.api_candidates),
        "Validation commands:\n" + "\n".join(f"- {command}" for command in summary.validation_commands[:8]),
        "Previous observation:\n" + observation_context(request),
        "Read files:\n" + read_files_context(request),
        "Changed files:\n" + ("\n".join(f"- {path}" for path in request.changed_files) or "- none"),
    ]
    if request.project_context:
        sections.append("Project context:\n" + "\n".join(f"- {item}" for item in request.project_context[:12]))
    if request.recent_steps:
        sections.append("Recent steps:\n" + recent_steps_context(request))
    return "\n\n".join(sections)


def observation_context(request: AgentStepRequest) -> str:
    observation = request.previous_observation
    if observation is None:
        return "- none"
    lines = [f"- status: {observation.status}", f"- message: {observation.message}"]
    if observation.path:
        lines.append(f"- path: {observation.path}")
    if observation.items:
        lines.append("- items:\n" + "\n".join(f"  - {item}" for item in observation.items[:30]))
    if observation.matches:
        lines.append("- matches:\n" + "\n".join(f"  - {item}" for item in observation.matches[:30]))
    if observation.output:
        lines.append("- output:\n" + "\n".join(f"  - {item}" for item in observation.output[:30]))
    if observation.content:
        lines.append("- content excerpt:\n" + observation.content[:6000])
    return "\n".join(lines)


def read_files_context(request: AgentStepRequest) -> str:
    if not request.read_files:
        return "- none"
    sections = []
    for item in request.read_files[-4:]:
        sections.append(f"## {item.path}\n{item.content[:5000]}")
    return "\n\n".join(sections)


def recent_steps_context(request: AgentStepRequest) -> str:
    return "\n".join(
        f"- #{step.index} {step.action.type}: {step.observation.status} {step.observation.message}"
        for step in request.recent_steps[-4:]
    )


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


def next_unread_file(request: AgentStepRequest) -> str:
    read_paths = {file.path for file in request.read_files}
    candidates = []
    candidates.extend(candidate.path for candidate in request.workspace_summary.backend_route_candidates)
    candidates.extend(candidate.path for candidate in request.workspace_summary.api_client_candidates)
    candidates.extend(candidate.path for candidate in request.workspace_summary.frontend_entry_candidates)
    candidates.extend(candidate.path for candidate in request.workspace_summary.type_file_candidates)
    for path in candidates:
        if path and path not in read_paths:
            return path
    return ""


def build_finish_summary(request: AgentStepRequest) -> str:
    read_count = len(request.read_files)
    changed_count = len(request.changed_files)
    return f"已完成当前 ReAct 轮次：读取 {read_count} 个文件，改动 {changed_count} 个文件。"


def parse_json_object(content: str) -> dict:
    raw = content.strip()
    if raw.startswith("```"):
        raw = raw.strip("`")
        raw = raw.removeprefix("json").strip()
    try:
        data = json.loads(raw)
    except json.JSONDecodeError as exc:
        raise LLMClientError("stepper returned invalid json") from exc
    if not isinstance(data, dict):
        raise LLMClientError("stepper returned non-object json")
    return data


def normalize_agent_step_response(request: AgentStepRequest, data: dict) -> AgentStepResponse:
    action = data.get("action")
    if not isinstance(action, dict):
        fallback = build_fallback_agent_step(request)
        action = fallback.action.model_dump()

    normalized_action = AgentPlanAction.model_validate(action)
    if normalized_action.type not in ALLOWED_ACTIONS:
        normalized_action = AgentPlanAction(type="finish", reason="stepper returned unsupported action")

    summary = data.get("summary")
    if not isinstance(summary, str) or not summary.strip():
        summary = build_finish_summary(request)

    return AgentStepResponse(
        action=normalized_action,
        summary=summary.strip(),
        stepper=data.get("stepper") or "llm",
    )
