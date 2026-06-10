import type { WorkspaceResponse } from "../types/workspace";

type ErrorResponse = {
  error?: string;
};

export class WorkspaceApiError extends Error {
  constructor(message: string) {
    super(message);
    this.name = "WorkspaceApiError";
  }
}

export async function fetchCurrentWorkspace(): Promise<WorkspaceResponse> {
  let response: Response;

  try {
    response = await fetch("/api/workspaces/current");
  } catch {
    throw new WorkspaceApiError("请求没有发到后端。先检查 backend 是否已经启动。");
  }

  return decodeWorkspaceResponse(response, "项目状态读取失败，请稍后重试。");
}

export async function connectWorkspace(path: string): Promise<WorkspaceResponse> {
  let response: Response;

  try {
    response = await fetch("/api/workspaces", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ path }),
    });
  } catch {
    throw new WorkspaceApiError("请求没有发到后端。先检查 backend 是否已经启动。");
  }

  return decodeWorkspaceResponse(response, "项目连接失败，请检查路径。");
}

async function decodeWorkspaceResponse(
  response: Response,
  fallbackMessage: string,
): Promise<WorkspaceResponse> {
  const payload = (await response.json()) as Partial<WorkspaceResponse> & ErrorResponse;

  if (!response.ok) {
    throw new WorkspaceApiError(payload.error || fallbackMessage);
  }

  if (
    payload.workspace !== null &&
    payload.workspace !== undefined &&
    (!payload.workspace.path ||
      !payload.workspace.branch ||
      typeof payload.workspace.dirty !== "boolean" ||
      !payload.workspace.updated_at)
  ) {
    throw new WorkspaceApiError("项目状态响应结构不完整，请检查接口返回。");
  }

  return {
    workspace: payload.workspace ?? null,
  };
}
