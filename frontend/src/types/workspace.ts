export type Workspace = {
  path: string;
  branch: string;
  dirty: boolean;
  updated_at: string;
};

export type WorkspaceResponse = {
  workspace: Workspace | null;
};

export type WorkspaceTreeItem = {
  path: string;
  type: "file" | "dir";
};

export type WorkspaceCandidate = {
  path: string;
  kind: string;
  reason: string;
};

export type WorkspaceSummary = {
  workspace_path: string;
  root_name: string;
  tree: WorkspaceTreeItem[];
  package_managers: string[];
  frontend_frameworks: string[];
  backend_frameworks: string[];
  backend_route_candidates: WorkspaceCandidate[];
  type_file_candidates: WorkspaceCandidate[];
  frontend_entry_candidates: WorkspaceCandidate[];
  api_client_candidates: WorkspaceCandidate[];
  validation_commands: string[];
  generated_at: string;
};

export type WorkspaceSummaryResponse = {
  summary: WorkspaceSummary;
};
