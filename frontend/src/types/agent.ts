import type { Workspace } from "./workspace";

export type AgentTaskStatus =
  | "queued"
  | "planning"
  | "running"
  | "verifying"
  | "completed"
  | "failed";

export type AgentTaskLog = {
  at: string;
  status: AgentTaskStatus;
  message: string;
};

export type AgentVerification = {
  status: string;
  command: string;
  output: string[];
};

export type AgentTask = {
  id: string;
  goal: string;
  status: AgentTaskStatus;
  workspace?: Workspace;
  plan: string[];
  logs: AgentTaskLog[];
  changed_files: string[];
  verification?: AgentVerification;
  error?: string;
  created_at: string;
  updated_at: string;
  completed_at?: string;
};

export type AgentTaskResponse = {
  task: AgentTask;
};
