from __future__ import annotations

import json
import os
from dataclasses import dataclass
from importlib import util
from pathlib import Path

from app.orchestration.frontend_codegen import next_generated_frontend_action
from app.schemas import (
    APICandidate,
    AgentObservation,
    AgentStepRequest,
    ProjectDocSnippet,
    WorkspaceCandidate,
    WorkspaceSummary,
)


ROOT = Path(__file__).resolve().parent
CASES_PATH = ROOT / "cases.json"
THRESHOLDS = {
    "task_completion": 1.0,
    "tool_correctness": 1.0,
    "plan_adherence": 0.75,
    "plan_quality": 0.75,
    "frontend_quality": 0.8,
    "contextual_relevancy": 0.75,
    "faithfulness": 1.0,
    "persona_memory_safety": 1.0,
}


@dataclass
class EvalResult:
    case_id: str
    passed: bool
    score: float
    metrics: dict[str, float]
    checks: list[str]
    failures: list[str]


def run() -> int:
    os.environ.setdefault("DEEPEVAL_TELEMETRY_OPT_OUT", "1")
    deepeval_available = util.find_spec("deepeval") is not None
    results = [evaluate_case(case) for case in load_cases()]
    report = {
        "deepeval_available": deepeval_available,
        "telemetry_opt_out": os.environ.get("DEEPEVAL_TELEMETRY_OPT_OUT") == "1",
        "passed": all(result.passed for result in results),
        "results": [result.__dict__ for result in results],
    }
    print(json.dumps(report, ensure_ascii=False, indent=2))
    return 0 if report["passed"] else 1


def load_cases() -> list[dict]:
    with CASES_PATH.open("r", encoding="utf-8") as file:
        data = json.load(file)
    if not isinstance(data, list):
        raise ValueError("eval cases must be a list")
    return data


def evaluate_case(case: dict) -> EvalResult:
    changed_files: list[str] = []
    generated: dict[str, str] = {}
    actions: list[dict[str, str]] = []
    checks: list[str] = []
    failures: list[str] = []
    request = build_request(case, changed_files)

    for step_index in range(1, 6):
        action = next_generated_frontend_action(request)
        if action is None:
            break
        if action.type != "write_file" or not action.path or action.content is None:
            failures.append(f"unexpected action at step {step_index}: {action.type}")
            break
        changed_files.append(action.path)
        generated[action.path] = action.content
        actions.append({"type": action.type, "path": action.path, "reason": action.reason})
        request = request.model_copy(
            update={
                "step_index": step_index + 1,
                "changed_files": list(changed_files),
                "previous_observation": AgentObservation(
                    status="ok",
                    message=f"Wrote {len(action.content)} bytes.",
                    path=action.path,
                ),
            }
        )

    expected_files = case.get("expected_files", [])
    for path in expected_files:
        if path in generated:
            checks.append(f"generated {path}")
        else:
            failures.append(f"missing generated file: {path}")

    combined_output = "\n".join(generated.values())
    for snippet in case.get("required_snippets", []):
        if snippet in combined_output:
            checks.append(f"found snippet: {snippet}")
        else:
            failures.append(f"missing snippet: {snippet}")

    metrics = evaluate_metrics(case, generated, actions)
    for metric, score in metrics.items():
        threshold = THRESHOLDS[metric]
        if score >= threshold:
            checks.append(f"{metric} {score:.2f} >= {threshold:.2f}")
        else:
            failures.append(f"{metric} {score:.2f} < {threshold:.2f}")

    score = sum(metrics.values()) / max(len(metrics), 1)
    return EvalResult(
        case_id=case["id"],
        passed=not failures,
        score=score,
        metrics=metrics,
        checks=checks,
        failures=failures,
    )


def evaluate_metrics(case: dict, generated: dict[str, str], actions: list[dict[str, str]]) -> dict[str, float]:
    return {
        "task_completion": task_completion_score(case, generated),
        "tool_correctness": tool_correctness_score(case, actions),
        "plan_adherence": plan_adherence_score(case, actions),
        "plan_quality": plan_quality_score(case),
        "frontend_quality": frontend_quality_score(generated),
        "contextual_relevancy": contextual_relevancy_score(case, generated),
        "faithfulness": faithfulness_score(case, generated),
        "persona_memory_safety": persona_memory_safety_score(case, generated),
    }


def task_completion_score(case: dict, generated: dict[str, str]) -> float:
    expected_files = case.get("expected_files", [])
    required_snippets = case.get("required_snippets", [])
    total = len(expected_files) + len(required_snippets)
    if total == 0:
        return 1.0
    combined_output = "\n".join(generated.values())
    hits = sum(1 for path in expected_files if path in generated)
    hits += sum(1 for snippet in required_snippets if snippet in combined_output)
    return hits / total


def tool_correctness_score(case: dict, actions: list[dict[str, str]]) -> float:
    expected_files = case.get("expected_files", [])
    if len(actions) != len(expected_files):
        return 0.0
    for action, expected_path in zip(actions, expected_files):
        if action["type"] != "write_file" or action["path"] != expected_path:
            return 0.0
    return 1.0


def plan_adherence_score(case: dict, actions: list[dict[str, str]]) -> float:
    plan = case.get("plan", [])
    expected_order = ["类型", "API client", "页面"]
    if not plan or len(actions) < len(expected_order):
        return 0.0
    matches = 0
    for action, expected in zip(actions, expected_order):
        if expected.lower() in action["reason"].lower() or expected in action["reason"]:
            matches += 1
    return matches / len(expected_order)


def plan_quality_score(case: dict) -> float:
    plan = case.get("plan", [])
    expected_keywords = ["接口", "类型", "API client", "页面"]
    if not plan:
        return 0.0
    joined_plan = "\n".join(plan)
    hits = sum(1 for keyword in expected_keywords if keyword in joined_plan)
    return hits / len(expected_keywords)


def frontend_quality_score(generated: dict[str, str]) -> float:
    combined_output = "\n".join(generated.values())
    expected = [
        "<script setup",
        "ref(",
        "isLoading",
        "errorMessage",
        "No data yet.",
        "scoped",
        "Promise<",
        "response.ok",
    ]
    hits = sum(1 for snippet in expected if snippet in combined_output)
    return hits / len(expected)


def contextual_relevancy_score(case: dict, generated: dict[str, str]) -> float:
    api = case["api"]
    combined_output = "\n".join(generated.values())
    expected = [api["path"], api["method"], resource_label(api["path"])]
    hits = sum(1 for item in expected if item in combined_output)
    return hits / len(expected)


def faithfulness_score(case: dict, generated: dict[str, str]) -> float:
    api = case["api"]
    combined_output = "\n".join(generated.values())
    forbidden_paths = case.get("forbidden_api_paths", ["/api/users", "/api/orders"])
    other_paths = [path for path in forbidden_paths if path != api["path"]]
    if any(path in combined_output for path in other_paths):
        return 0.0
    return 1.0 if api["path"] in combined_output else 0.0


def persona_memory_safety_score(case: dict, generated: dict[str, str]) -> float:
    raw_output = "\n".join(generated.values())
    combined_output = raw_output.lower()
    forbidden_persona_leakage = ["berry", "学姐", "毒舌", "二次元"]
    if any(term in combined_output for term in forbidden_persona_leakage):
        return 0.0
    memory_requirements = case.get("memories", [])
    if memory_requirements and not all(snippet in raw_output for snippet in ["Loading...", "No data yet."]):
        return 0.0
    return 1.0


def resource_label(api_path: str) -> str:
    cleaned = api_path.strip("/")
    if not cleaned:
        return "Item"
    part = cleaned.split("/")[-1]
    if part.endswith("s"):
        part = part[:-1]
    return part[:1].upper() + part[1:]


def build_request(case: dict, changed_files: list[str]) -> AgentStepRequest:
    api = case["api"]
    return AgentStepRequest(
        goal=case["goal"],
        plan=["读取接口", "生成类型", "生成 API client", "生成页面"],
        workspace_summary=WorkspaceSummary(
            workspace_path=str(ROOT / "fixtures" / "frontend_gin_project"),
            root_name="frontend_gin_project",
            frontend_frameworks=["Vue", "Vite"],
            backend_frameworks=["Gin"],
            backend_route_candidates=[
                WorkspaceCandidate(path=api["handler_file"], kind="go.gin.routes", reason="fixture route")
            ],
            api_candidates=[
                APICandidate(
                    method=api["method"],
                    path=api["path"],
                    handler=api["handler"],
                    handler_file=api["handler_file"],
                    type_definitions=[
                        WorkspaceCandidate(path=api["handler_file"], kind="go.struct", reason="fixture response")
                    ],
                )
            ],
            frontend_entry_candidates=[
                WorkspaceCandidate(path=case["frontend_entry"], kind="frontend.page", reason="fixture page")
            ],
            api_client_candidates=[
                WorkspaceCandidate(path=case["api_client"], kind="frontend.api_client", reason="fixture client")
            ],
            project_doc_candidates=[
                WorkspaceCandidate(path=path, kind="project.docs", reason="fixture docs")
                for path in case.get("project_docs", [])
            ],
            project_doc_snippets=[
                ProjectDocSnippet(
                    path=path,
                    kind="project.docs",
                    content="Use existing Vue views, API clients, loading state, error state, and empty state.",
                )
                for path in case.get("project_docs", [])
            ],
            type_file_candidates=[
                WorkspaceCandidate(path=case["type_file"], kind="types", reason="fixture type")
            ],
            validation_commands=["cd frontend && npm run build"],
        ),
        step_index=1,
        previous_observation=AgentObservation(status="ok", message="fixture observation"),
        changed_files=changed_files,
    )


if __name__ == "__main__":
    raise SystemExit(run())
