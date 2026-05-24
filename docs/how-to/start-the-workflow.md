# How to Start the BarberWorkflow

Start the durable barber that waits for signals and processes yaks.

## Basic start

```bash
yyx start \
    --repo owner/repo \
    --repo-root /path/to/repo \
    --agent pi
```

This creates a Temporal workflow that runs until manually stopped. It accepts CI signals, g2g scans, and PR feedback.

## g2g-only mode

If you want the workflow to only process yaks explicitly tagged `@g2g`:

```bash
yyx start \
    --repo owner/repo \
    --repo-root /path/to/repo \
    --agent pi \
    --g2g-mode \
    --g2g-scan-interval 30
```

The `--g2g-scan-interval` controls how often (in minutes) the workflow scans for new `@g2g` yaks when idle. Default is 60.

## Choosing an agent

| Agent | Flag | Requires |
|-------|------|----------|
| Pi | `--agent pi` | `pi` CLI |
| Claude Code | `--agent claude-code` | `claude` CLI + `ANTHROPIC_API_KEY` |
| Codex | `--agent codex` | `codex` CLI + `OPENAI_API_KEY` |
| OpenCode | `--agent opencode` | `opencode` CLI |

## Multiple repos

Run separate workflows for each repo. Each uses a different work directory. The workflow ID is fixed (`yaketyyak-yak-shaving`), so only one instance runs at a time per repo.

> For the full list of workflow parameters, see [Workflow Options](../reference/workflow-options.md).
> To shave a single yak with the shave loop (pre-PR validation and review), see [Shave a Yak](shave-a-yak.md).
> To learn the architecture, see [Architecture](../explanation/architecture.md).
