// Shared types that mirror temporal/types.go.
// The Go workflow and TypeScript worker must agree on these shapes.

export interface PiConfig {
  model?: string;
  tools?: string[];
  skills?: string[];
}

export interface WorkflowConfig {
  repoUrl: string;  // e.g. "https://github.com/funkymonkeymonk/yaketyyak"
  pi: PiConfig;
}

export interface PRResult {
  prUrl: string;
  prNumber: number;
}

export interface YakWorkflowState {
  yakName: string;
  phase: string;
  workspace: string;
  prUrl: string;
  prNumber: number;
}

export const DEFAULT_PI_TOOLS = ["read", "bash", "edit", "write"];
export const DEFAULT_PI_MODEL = "claude-sonnet-4-6";
export const TASK_QUEUE = "yaketyyak-tasks";
