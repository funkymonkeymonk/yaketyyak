# Activities Reference

All Temporal activities used by the `BarberWorkflow`.

## yx_sync

Runs `yx sync` to pull the latest yak state from shared git refs.

Retry: 3 attempts, 30s timeout

## yak_triage_g2g

Finds yaks in `todo` state tagged with `@g2g`. Parses `yx ls --format json` output.

Returns: list of `{name, tags, g2g}`

Retry: 2 attempts, 30s timeout

## yak_triage

Fallback triage when no `@g2g` yaks exist. Delegates to `yak-triage.sh` from the shave-yaks skill, or falls back to `yx ls --format json`.

Returns: list of `{name}`

Retry: 3 attempts, 30s timeout

## yak_claim

Safely claims a yak using `yak-claim.sh` from the shave-yaks skill. Falls back to direct `yx start` + `yx sync`.

Returns: `{name, state, claimed: bool}`

Retry: 3 attempts, 30s timeout

## yak_remove_g2g_tag

Removes the `@g2g` tag from a yak after successful claim. Prevents double-claiming.

Retry: none, 15s timeout

## dispatch_agent

Dispatches an AI coding agent to implement a yak.

| Parameter | Type | Description |
|-----------|------|-------------|
| `yak_name` | string | Name of the yak |
| `agent_type` | string | `pi`, `claude-code`, `codex`, or `opencode` |
| `repo_root` | string | Absolute path to repo |
| `g2g` | bool | Whether this was a g2g-triggered yak |

The activity generates a prompt from the yak's context and pipes it to the agent CLI. It expects a JSON line in the output with `pr_number`, `pr_url`, and `branch`.

Retry: 2 attempts, 2h timeout

## watch_pr_ci

Polls GitHub checks on a PR until all complete. Returns `"success"` or `"failure"`.

Poll interval: 30s

Retry: 1 attempt (no retry — relies on next signal), 4h timeout

## merge_pr

Merges a passing PR using `gh pr merge --squash --delete-branch`.

Retry: 3 attempts, 5m timeout

## yak_mark_done

Marks a yak as done with `yx done` and syncs.

Retry: none, 30s timeout

## yak_mark_refinement

Tags a yak as `@needs-human` when it lacks sufficient context for autonomous implementation.

Retry: none, 30s timeout

## _check_refinement

Checks if a yak has enough context to implement autonomously. Delegates to `yak-needs-refinement.sh` from the shave-yaks skill.

Returns: `None` if clear, or a string reason if needs refinement.

Retry: none, 30s timeout

> For signal definitions, see [Signals Reference](signals.md).
> For workflow parameters, see [Workflow Options](workflow-options.md).
> For shave loop activities, see [Shave Activities Reference](shave-activities.md).
