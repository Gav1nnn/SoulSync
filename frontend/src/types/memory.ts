export type MemoryStatus = "active" | "disabled";

export type Memory = {
  id: string;
  type: string;
  content: string;
  reason: string;
  confidence: number;
  status: MemoryStatus;
  source_trace_id: string;
  source_message_id: string;
  created_at: string;
  updated_at: string;
  last_used_at: string;
};

export type MemoriesResponse = {
  memories: Memory[];
};

export type MemoryResponse = {
  memory: Memory;
};
