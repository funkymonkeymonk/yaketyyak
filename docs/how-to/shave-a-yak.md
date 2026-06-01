# How to Shave a Yak

Shave a single yak through an autonomous implement-validate-review loop before creating a PR.

## From the TUI

Navigate to the yak you want to shave and press **`v`**:

```
  j/k        navigate the yak tree
  v          shave the selected yak (shave loop)
```

The TUI shows a status message:
```
Shaving yak: Add unit tests for payment module...
```

## From the CLI

```bash
yy
```

The TUI scans the current directory for repos with `.yaks/` directories. Press `v` on the target yak. The workflow ID is `yy-shave-<yak-name-slug>` — check its status:

```bash
yy status --workflow-id yy-shave-add-unit-tests-for-payment-module
```

## What happens

The shave loop runs up to 3 iterations (retries):

1. **Implement** — The agent writes tests and implementation inside an isolated jj workspace
2. **Validate** — `go vet`, `go test`, `go build` run in the workspace
3. **Adversarial review** — A fresh, independent agent reviews the diff against the original yak brief

If validation or review fails, the next iteration gets the failure feedback and tries again. Only when both pass is a PR created:

4. **Create PR** — Pushes the branch and opens a PR via `gh pr create`
5. **Watch CI** — Polls GitHub checks on the PR
6. **Merge** — If CI passes, merges via `gh pr merge --squash --delete-branch`
7. **Mark done** — Runs `yx done` + `yx sync`

If all retries are exhausted, the yak is tagged `@needs-human` with the full failure log.

## When to use the shave loop vs the barber

| | Shave loop (`v`) | Barber (`s`) |
|---|---|---|
| Scope | One yak at a time | Continuous: all `@g2g` yaks |
| Quality gate | Validation + adversarial review before PR | CI-only gate after PR |
| Retries | Configurable max (default 3) | Unlimited (rework loop) |
| Workspace | Isolated jj workspace per yak | Agent creates PR directly |

Use the shave loop when you want a single yak verified end-to-end before opening a PR. Use the barber when you want continuous autonomous shaving of `@g2g`-tagged yaks.

> For the barber workflow, see [Start the Workflow](start-the-workflow.md).
> For why the adversarial review exists, see [Barber and Shave Workflows](../explanation/barber-and-shave-workflows.md).
> For the full workflow definition, see [ShaveWorkflow Reference](../reference/shave-workflow.md).
