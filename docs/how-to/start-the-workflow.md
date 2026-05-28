# How to Start a YakWorkflow

Start a durable `YakWorkflow` to shave a single yak autonomously.

## Basic usage

```bash
yyx shave <yak-name>
```

The yak name can be given as space-separated words or hyphenated:

```bash
yyx shave update documentation
yyx shave update-documentation-to-reflect-current-architecture-7yf2
```

Both forms start the same workflow. The deterministic workflow ID is derived from the yak name (e.g. `yyx-yak-update-documentation`).

## With options

```bash
yyx shave <yak-name> \
    --repo-root /path/to/repo \
    --pi-model anthropic/claude-sonnet-4 \
    --pi-tools read,bash,edit,write \
    --pi-skill /path/to/skill.md
```

## What happens

1. The workflow claims the yak via `yx start` and syncs
2. An isolated jj workspace is created under `.workspaces/`
3. Pi is dispatched via LiteLLM to implement the yak in the workspace
4. The workflow creates a draft PR with `gh pr create`
5. The workflow watches for the PR to be merged
6. On merge, the yak is marked done with `yx done` and synced

## Output

```
Started YakWorkflow for "update documentation"
  Workflow ID: yyx-yak-update-documentation
  Run ID:      <temporal-run-id>
  Repo root:   /path/to/repo
```

## Environment variables

| Variable | Required | Description |
|----------|----------|-------------|
| `LITELLM_BASE_URL` | Yes | LiteLLM gateway URL |
| `LITELLM_API_KEY` | Yes | LiteLLM API key |
| `GITHUB_TOKEN` | Yes | GitHub personal access token with `repo` scope |

These are read by the **worker process**, not by the `yyx shave` command itself.

> For the full list of flags, see [Workflow Options](../reference/workflow-options.md).
> For a walkthrough from scratch, see [From Zero to Yak Shaving](../tutorials/from-zero-to-shaving.md).
> For the workflow definition, see [YakWorkflow Reference](../reference/yak-workflow.md).
