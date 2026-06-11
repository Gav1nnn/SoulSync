import type { AgentTask, AgentTaskResponse, AgentTasksResponse } from "../types/agent";

type ErrorResponse = {
  error?: string;
};

export class AgentApiError extends Error {
  constructor(message: string) {
    super(message);
    this.name = "AgentApiError";
  }
}

export async function createAgentTask(goal: string): Promise<AgentTaskResponse> {
  let response: Response;

  try {
    response = await fetch("/api/agent/tasks", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ goal }),
    });
  } catch {
    throw new AgentApiError("请求没有发到后端。先检查 backend 是否已经启动。");
  }

  return decodeAgentTaskResponse(response, "任务创建失败，请先确认 workspace 已连接。");
}

export async function fetchAgentTask(taskId: string): Promise<AgentTaskResponse> {
  let response: Response;

  try {
    response = await fetch(`/api/agent/tasks/${encodeURIComponent(taskId)}`);
  } catch {
    throw new AgentApiError("请求没有发到后端。先检查 backend 是否已经启动。");
  }

  return decodeAgentTaskResponse(response, "任务读取失败，请稍后重试。");
}

export async function fetchAgentTasks(limit = 20): Promise<AgentTasksResponse> {
  let response: Response;

  try {
    response = await fetch(`/api/agent/tasks?limit=${encodeURIComponent(String(limit))}`);
  } catch {
    throw new AgentApiError("请求没有发到后端。先检查 backend 是否已经启动。");
  }

  const payload = (await response.json()) as Partial<AgentTasksResponse> & ErrorResponse;

  if (!response.ok) {
    throw new AgentApiError(payload.error || "任务列表读取失败，请稍后重试。");
  }

  if (!Array.isArray(payload.tasks) || payload.tasks.some((task) => !isAgentTask(task))) {
    throw new AgentApiError("任务列表响应结构不完整，请检查接口返回。");
  }

  return payload as AgentTasksResponse;
}

export async function retryAgentTask(taskId: string): Promise<AgentTaskResponse> {
  let response: Response;

  try {
    response = await fetch(`/api/agent/tasks/${encodeURIComponent(taskId)}/retry`, {
      method: "POST",
    });
  } catch {
    throw new AgentApiError("请求没有发到后端。先检查 backend 是否已经启动。");
  }

  return decodeAgentTaskResponse(response, "任务重试失败，请确认任务处于失败状态。");
}

async function decodeAgentTaskResponse(
  response: Response,
  fallbackMessage: string,
): Promise<AgentTaskResponse> {
  const payload = (await response.json()) as Partial<AgentTaskResponse> & ErrorResponse;

  if (!response.ok) {
    throw new AgentApiError(payload.error || fallbackMessage);
  }

  if (!isAgentTask(payload.task)) {
    throw new AgentApiError("任务响应结构不完整，请检查接口返回。");
  }

  return payload as AgentTaskResponse;
}

function isAgentTask(task: unknown): task is AgentTask {
  if (!task || typeof task !== "object") {
    return false;
  }

  const candidate = task as Partial<AgentTask>;
  return Boolean(
    candidate.id &&
      candidate.goal &&
      candidate.status &&
      typeof candidate.retry_count === "number" &&
      Array.isArray(candidate.plan) &&
      Array.isArray(candidate.files_to_read) &&
      Array.isArray(candidate.planner_context_used) &&
      Array.isArray(candidate.steps) &&
      Array.isArray(candidate.logs) &&
      Array.isArray(candidate.changed_files),
  );
}
