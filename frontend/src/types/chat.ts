export type ChatRole = "user" | "assistant";

export type ChatMessage = {
  id: string;
  trace_id?: string;
  role: ChatRole;
  content: string;
  created_at?: string;
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

export type Trace = {
  trace_id: string;
  user_message: string;
  reply: string;
  context_used: string[];
  used_memory_ids: string[];
  used_knowledge_chunk_ids: string[];
  memory_written: boolean;
  memory_candidate_count: number;
  started_at: string;
  finished_at: string;
  duration_ms: number;
};

export type MessagesResponse = {
  messages: ChatMessage[];
};

export type TraceResponse = {
  trace: Trace;
};

export type ChatErrorCode =
  | "invalid_request"
  | "service_unavailable"
  | "network"
  | "unknown";
