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

export type AgentPlanAction = {
  type: string;
  path?: string;
  query?: string;
  command?: string;
  reason: string;
};

export type AgentTask = {
  id: string;
  goal: string;
  status: AgentTaskStatus;
  workspace?: Workspace;
  branch_name?: string;
  plan: string[];
  files_to_read: string[];
  initial_action?: AgentPlanAction;
  planner?: string;
  planner_context_used: string[];
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
