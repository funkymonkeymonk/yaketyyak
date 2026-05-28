# Data Types Reference

Core data types used by `YakWorkflow`.

## PiConfig

Configuration for invoking the Pi coding agent. The provider is always LiteLLM, configured via `LITELLM_BASE_URL` and `LITELLM_API_KEY` environment variables in the worker process.

TypeScript (worker):
```typescript
interface PiConfig {
  model?: string;   // LiteLLM model name; uses default if empty
  tools?: string[]; // e.g. ["read", "bash", "edit", "write"]
  skills?: string[]; // paths to skill files
}
```

Go (CLI):
```go
type PiConfig struct {
    Model  string   `json:"model,omitempty"`
    Tools  []string `json:"tools,omitempty"`
    Skills []string `json:"skills,omitempty"`
}
```

### Defaults

```typescript
const DEFAULT_PI_TOOLS = ["read", "bash", "edit", "write"];
const DEFAULT_PI_MODEL = "claude-sonnet-4-6";
```

`DEFAULT_PI_MODEL` is used when no `--pi-model` flag is provided. `DEFAULT_PI_TOOLS` are the tools enabled for every shave run unless overridden with `--pi-tools`.

## YakWorkflowState

The queryable state of a running `YakWorkflow`. Accessible via the `yak_status` Temporal query.

```typescript
interface YakWorkflowState {
  yakName: string;
  phase: string;
  workspace: string;
  prUrl: string;
  prNumber: number;
}
```

### Phase values

| Phase | Description |
|-------|-------------|
| `init` | Workflow just started |
| `claiming` | Running `yx start` to claim the yak |
| `init-workspace` | Cloning repo into isolated workspace |
| `implementing` | Pi agent is running |
| `creating-pr` | Pushing branch and opening draft PR |
| `waiting-for-merge` | Polling GitHub API for PR merge |
| `done` | PR merged, yak marked done |

## PRResult

Returned by the `CreateDraftPR` activity.

```typescript
interface PRResult {
  prUrl: string;
  prNumber: number;
}
```

> For signal definitions, see [Signals Reference](signals.md).
> For workflow flags that configure PiConfig, see [Workflow Options](workflow-options.md).
