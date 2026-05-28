// Shared types that mirror temporal/types.go.
// The Go workflow and TypeScript worker must agree on these shapes.

export interface PiConfig {
  model?: string;
  tools?: string[];
  skills?: string[];
  /** Maximum wall-clock seconds for a single agent run. Defaults to 7200 (2h). */
  maxRunTimeSeconds?: number;
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

/**
 * Models known to be incompatible with Pi's tool-calling format.
 * Adding a model here causes RunAgent to fast-fail before wasting an agent run.
 *
 * Tested against the LiteLLM gateway as of 2026-05-27:
 *   moonshotai.kimi-k2.5          → malformed_model_output
 *   moonshot.kimi-k2-thinking     → malformed_model_output
 */
export const INCOMPATIBLE_MODELS = new Set([
  "moonshotai.kimi-k2.5",
  "moonshot.kimi-k2-thinking",
]);
