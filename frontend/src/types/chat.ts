export type ChatRole = "user" | "assistant";

export type ChatMessage = {
  id: string;
  role: ChatRole;
  content: string;
};

export type ChatStatus = "idle" | "sending" | "success" | "error";

export type ChatResponse = {
  reply: string;
  trace_id: string;
  persona: string;
  context_used: string[];
  used_memory_ids: string[];
  memory_written: boolean;
  memory_candidate_count: number;
};

export type ChatErrorCode =
  | "invalid_request"
  | "service_unavailable"
  | "network"
  | "unknown";
