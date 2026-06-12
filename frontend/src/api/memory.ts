import type { MemoriesResponse, MemoryResponse, MemoryStatus } from "../types/memory";

type ErrorResponse = {
  error?: string;
};

export class MemoryApiError extends Error {
  constructor(message: string) {
    super(message);
    this.name = "MemoryApiError";
  }
}

export async function fetchMemories(): Promise<MemoriesResponse> {
  let response: Response;

  try {
    response = await fetch("/api/memories");
  } catch {
    throw new MemoryApiError("请求没有发到后端。先检查 backend 是否已经启动。");
  }

  const payload = (await response.json()) as Partial<MemoriesResponse> & ErrorResponse;
  if (!response.ok) {
    throw new MemoryApiError(payload.error || "Memory 读取失败，请稍后重试。");
  }
  if (!Array.isArray(payload.memories)) {
    throw new MemoryApiError("Memory 响应结构不完整，请检查接口返回。");
  }

  return payload as MemoriesResponse;
}

export async function updateMemoryStatus(id: string, status: MemoryStatus): Promise<MemoryResponse> {
  let response: Response;

  try {
    response = await fetch(`/api/memories/${encodeURIComponent(id)}`, {
      method: "PATCH",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ status }),
    });
  } catch {
    throw new MemoryApiError("请求没有发到后端。先检查 backend 是否已经启动。");
  }

  const payload = (await response.json()) as Partial<MemoryResponse> & ErrorResponse;
  if (!response.ok) {
    throw new MemoryApiError(payload.error || "Memory 状态更新失败，请稍后重试。");
  }
  if (!payload.memory || !payload.memory.id || !payload.memory.status) {
    throw new MemoryApiError("Memory 更新响应结构不完整，请检查接口返回。");
  }

  return payload as MemoryResponse;
}
