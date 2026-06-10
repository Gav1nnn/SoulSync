from __future__ import annotations

import json
import os
from dataclasses import dataclass
from importlib import util
from pathlib import Path

from app.orchestration.frontend_codegen import next_generated_frontend_action
from app.schemas import APICandidate, AgentObservation, AgentStepRequest, WorkspaceCandidate, WorkspaceSummary


ROOT = Path(__file__).resolve().parent
CASES_PATH = ROOT / "cases.json"


@dataclass
class EvalResult:
    case_id: str
    passed: bool
    score: float
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

    score = len(checks) / max(len(checks) + len(failures), 1)
    return EvalResult(
        case_id=case["id"],
        passed=not failures,
        score=score,
        checks=checks,
        failures=failures,
    )


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
