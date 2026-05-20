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
};

export type ChatErrorCode =
  | "invalid_request"
  | "service_unavailable"
  | "network"
  | "unknown";
