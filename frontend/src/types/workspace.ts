export type Workspace = {
  path: string;
  branch: string;
  dirty: boolean;
  updated_at: string;
};

export type WorkspaceResponse = {
  workspace: Workspace | null;
};
