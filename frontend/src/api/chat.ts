import type { ChatErrorCode, ChatResponse } from "../types/chat";

type ErrorResponse = {
  error?: string;
};

export class ChatApiError extends Error {
  code: ChatErrorCode;

  constructor(message: string, code: ChatErrorCode) {
    super(message);
    this.name = "ChatApiError";
    this.code = code;
  }
}

export async function sendChatMessage(message: string): Promise<ChatResponse> {
  let response: Response;

  try {
    response = await fetch("/api/chat", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ message }),
    });
  } catch {
    throw new ChatApiError(
      "请求没有发到后端。先检查 backend 和 ai-engine 是否已经启动。",
      "network",
    );
  }

  const payload = (await response.json()) as Partial<ChatResponse> & ErrorResponse;

  if (!response.ok) {
    if (response.status === 400) {
      throw new ChatApiError(
        payload.error || "请求格式不对，请重新输入后再试。",
        "invalid_request",
      );
    }

    if (response.status === 502) {
      throw new ChatApiError(
        payload.error || "后端已经收到请求，但 ai-engine 当前不可用。",
        "service_unavailable",
      );
    }

    throw new ChatApiError(
      payload.error || "请求失败，请稍后重试。",
      "unknown",
    );
  }

  if (
    !payload.reply ||
    !payload.trace_id ||
    !payload.persona ||
    !Array.isArray(payload.context_used) ||
    !Array.isArray(payload.used_memory_ids) ||
    typeof payload.memory_written !== "boolean" ||
    typeof payload.memory_candidate_count !== "number"
  ) {
    throw new ChatApiError("后端响应结构不完整，请检查接口返回。", "unknown");
  }

  return payload as ChatResponse;
}
