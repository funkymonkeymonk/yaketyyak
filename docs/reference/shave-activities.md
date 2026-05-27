# Shave Activities Reference

Activities used exclusively by the `ShaveWorkflow`. For shared activities like `YakClaim`, `WatchPRCI`, `MergePR`, see the [main Activities Reference](activities.md).

## shave_init_workspace

Creates an isolated jj workspace for the yak.

| Parameter | Type | Description |
|-----------|------|-------------|
| `repo_root` | string | Absolute path to local checkout |
| `yak_name` | string | Yak name (for workspace naming) |

Runs:
1. `jj git fetch` — pull latest from remote
2. `jj workspace add --name shave-<slug> .workspaces/shave-<slug>`

Returns: workspace name (e.g. `"shave-add-unit-tests-for-payment-module"`)

Retry: 2 attempts, 60s timeout

## shave_implement

Dispatches an AI agent to implement (or fix) the yak.

| Parameter | Type | Description |
|-----------|------|-------------|
| `yak_name` | string | Name of the yak |
| `agent_type` | string | Agent CLI to dispatch |
| `repo_root` | string | Absolute path to local checkout |
| `workspace_name` | string | jj workspace name |
| `last_failure` | string | Validation errors from previous iteration |
| `review_issues` | string | Review issues from previous iteration |
| `iteration` | int | Current iteration number (1-indexed) |

**First iteration prompt:** The agent receives the original yak context + acceptance criteria and is instructed to write failing tests first (TDD), implement, verify, and commit locally. No PR is created.

**Subsequent iterations prompt:** The agent receives the same yak context plus the validation failures or review issues from the previous attempt. Instructed to fix all issues and commit locally.

Retry: 1 attempt (no retry — failures feed into next iteration), 2h timeout

## shave_validate

Runs Go validation checks in the workspace.

| Parameter | Type | Description |
|-----------|------|-------------|
| `repo_root` | string | Absolute path to local checkout |
| `workspace_name` | string | jj workspace name |

Runs sequentially:
1. `go vet ./...`
2. `go test ./...`
3. `go build ./...`

Returns: `ShaveValidationResult{passed: bool, output: string}`

All three must pass for `passed: true`. The first failing command's output is included in the result.

Retry: 1 attempt, 15m timeout

## shave_adversarial_review

Dispatches a fresh, independent agent to critically review the implementation diff against the original yak brief.

| Parameter | Type | Description |
|-----------|------|-------------|
| `yak_name` | string | Name of the yak |
| `agent_type` | string | Agent CLI to dispatch |
| `repo_root` | string | Absolute path to local checkout |
| `workspace_name` | string | jj workspace name |

The reviewer receives:
1. The original yak context and acceptance criteria
2. The `jj diff` of all changes in the workspace

The reviewer does **not** receive the implementer's reasoning or approach — this prevents anchoring bias.

The reviewer is instructed to find:
- Bugs and logic errors
- Security vulnerabilities
- Missing edge cases and error handling
- Insufficient test coverage
- Code that does not match acceptance criteria

Returns: `ShaveReviewResult{passed: bool, issues: string, items: []string}`

Retry: 1 attempt, 1h timeout

## shave_create_pr

Pushes the workspace branch and creates a Pull Request.

| Parameter | Type | Description |
|-----------|------|-------------|
| `repo_root` | string | Absolute path to local checkout |
| `workspace_name` | string | jj workspace name |
| `repo` | string | GitHub repo in `owner/repo` format |

Runs:
1. `jj git push` — push the workspace branch to GitHub
2. `gh pr create --repo <repo> --fill` — create PR with auto-filled title/body

Returns: `AgentResult{pr_number: int, pr_url: string}`

Retry: 2 attempts, 5m timeout

## shave_cleanup

Removes the isolated jj workspace.

| Parameter | Type | Description |
|-----------|------|-------------|
| `repo_root` | string | Absolute path to local checkout |
| `workspace_name` | string | jj workspace name |

Runs: `jj workspace forget <workspace_name>` in the repo root.

Retry: 1 attempt, 30s timeout

## Data types

### ShaveValidationResult

```go
type ShaveValidationResult struct {
    Passed bool   // true if all checks passed
    Output string // command output on failure
}
```

### ShaveReviewResult

```go
type ShaveReviewResult struct {
    Passed bool     // true if reviewer found no issues
    Issues string   // human-readable summary of issues
    Items  []string // individual issue descriptions
}
```

> For the workflow that orchestrates these activities, see [ShaveWorkflow Reference](shave-workflow.md).
> For shared activities, see [Activities Reference](activities.md).
