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

export type AgentTaskResult = {
  summary: string;
  failure_file?: string;
  next_suggestions: string[];
};

export type AgentPlanAction = {
  type: string;
  path?: string;
  query?: string;
  command?: string;
  content?: string;
  reason: string;
};

export type AgentObservation = {
  status: string;
  message: string;
  path?: string;
  items?: string[];
  matches?: string[];
  content?: string;
  command?: string;
  output?: string[];
};

export type AgentTaskStep = {
  index: number;
  action: AgentPlanAction;
  observation: AgentObservation;
  summary: string;
  context_used: string[];
  stepper: string;
  started_at: string;
  finished_at: string;
  duration_ms: number;
};

export type AgentTask = {
  id: string;
  goal: string;
  status: AgentTaskStatus;
  retry_count: number;
  workspace?: Workspace;
  branch_name?: string;
  plan: string[];
  files_to_read: string[];
  initial_action?: AgentPlanAction;
  planner?: string;
  planner_context_used: string[];
  steps: AgentTaskStep[];
  logs: AgentTaskLog[];
  changed_files: string[];
  verification?: AgentVerification;
  result?: AgentTaskResult;
  error?: string;
  created_at: string;
  updated_at: string;
  completed_at?: string;
};

export type AgentTaskResponse = {
  task: AgentTask;
};

export type AgentTasksResponse = {
  tasks: AgentTask[];
};
