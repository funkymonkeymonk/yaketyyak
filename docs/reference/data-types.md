# Data Types Reference

Core data types used by `YakWorkflow`.

## PiConfig

Configuration for invoking the Pi coding agent. The provider is always LiteLLM, configured via `LITELLM_BASE_URL` and `LITELLM_API_KEY` environment variables in the worker process.

```go
type PiConfig struct {
    Model  string   // LiteLLM model name; uses gateway default if empty
    Tools  []string // e.g. ["read", "bash", "edit", "write"]
    Skills []string // paths to skill files loaded via --skill
}
```

### Defaults

```go
var DefaultPiTools = []string{"read", "bash", "edit", "write"}
const DefaultPiModel = "claude-sonnet-4-6"
```

`DefaultPiModel` is used when no `--pi-model` flag is provided. `DefaultPiTools` are the tools enabled for every shave run unless overridden with `--pi-tools`.

## YakWorkflowState

The queryable state of a running `YakWorkflow`. Accessible via a Temporal query.

```go
type YakWorkflowState struct {
    YakName   string `json:"yak_name"`
    Phase     string `json:"phase"`
    Workspace string `json:"workspace"`
    PRURL     string `json:"pr_url"`
    PRNumber  int    `json:"pr_number"`
}
```

### Phase values

| Phase | Description |
|-------|-------------|
| `claiming` | Running `yx start` to claim the yak |
| `init-workspace` | Creating the jj workspace |
| `implementing` | Pi agent is running |
| `creating-pr` | Pushing branch and opening draft PR |
| `watching-pr` | Polling for the PR to be merged |
| `done` | PR merged, yak marked done |
| `failed` | Unrecoverable error; yak released |

## PRResult

Returned by the `CreateDraftPR` activity.

```go
type PRResult struct {
    PRURL    string `json:"pr_url"`
    PRNumber int    `json:"pr_number"`
}
```

> For signal payloads, see [Signals Reference](signals.md).
> For workflow flags that configure PiConfig, see [Workflow Options](workflow-options.md).
