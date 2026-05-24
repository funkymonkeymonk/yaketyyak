# Workflow Options Reference

Parameters accepted by `BarberWorkflow.run`.

## repo

Type: `string` (required)

GitHub repository in `"owner/repo"` format. Used for GitHub API calls in `watch_pr_ci` and `merge_pr`.

## repo_root

Type: `string` (required)

Absolute path to the local checkout of the repository. Passed to the agent as the working directory.

## agent_type

Type: `string` (optional, default: `"pi"`)

Which AI coding agent to dispatch. One of:

| Value | CLI Command | Environment Variable Required |
|-------|-------------|------------------------------|
| `pi` | `pi -p` | none |
| `claude-code` | `claude -p` | `ANTHROPIC_API_KEY` |
| `codex` | `codex exec` | `OPENAI_API_KEY` |
| `opencode` | `opencode -p` | none |

## g2g_mode

Type: `bool` (optional, default: `false`)

If `true`, the workflow ONLY processes yaks tagged `@g2g`. Regular triage (un-tagged actionable yaks) is skipped. Use this when you want tight control over what the autonomous loop touches.

## g2g_scan_interval_minutes

Type: `int` (optional, default: `60`)

How often (in minutes) the workflow scans for `@g2g`-tagged yaks when idle. A scan is also triggered whenever a signal arrives (`ci_signal`, `g2g_signal`, `pr_feedback`, `resume`).

> For how to start the workflow with these options, see [Start the Workflow](../how-to/start-the-workflow.md).
> For the data types behind these options, see [Data Types](data-types.md).
