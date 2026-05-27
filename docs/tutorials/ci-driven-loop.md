# Tutorial: CI-Driven Yak Shaving Loop

In this tutorial you will learn how yaketyyak reacts to CI events by connecting it to a GitHub Actions pipeline and watching the autonomous loop in action.

## Prerequisites

- Completed the [From Zero to Yak Shaving](from-zero-to-shaving.md) tutorial
- A GitHub Actions CI pipeline already set up on your repository
- The yaketyyak workflow and worker running (from the previous tutorial)

## What you'll learn

By the end of this tutorial you will understand how CI completions trigger yak shaving turns, how the workflow handles PR feedback, and how the autonomous loop closes.

## Step 1: See what happens when no CI is connected

With your workflow running, check its status:

```bash
yyx status
```

You'll see the workflow is idle, waiting for signals. There are no CI signals arriving because nothing is connected yet.

## Step 2: Add the CI signal workflow

Create `.github/workflows/yaketyyak.yml` in your target repository:

```yaml
name: CI Signal

on:
  workflow_run:
    workflows: ["CI"]
    types: [completed]

jobs:
  signal-temporal:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: stable
      - run: go build -o yyx .
      - name: Signal yaketyyak workflow
        run: |
          ./yyx ci \
            --conclusion ${{ github.event.workflow_run.conclusion }} \
            --branch ${{ github.event.workflow_run.head_branch }} \
            --sha ${{ github.event.workflow_run.head_sha }}
```

Commit and push:

```bash
git add .github/workflows/yaketyyak.yml
git commit -m "ci: add yaketyyak temporal integration"
git push
```

## Step 3: Watch the loop start

Trigger a CI run on your repository (push a commit, open a PR, or manually run a workflow).

Within seconds, the yaketyyak workflow receives the `ci_signal`. Watch it react:

```bash
yyx status
```

The phase changes from `idle` to `triaging` as the workflow:
1. Receives the CI signal
2. Syncs yaks
3. Checks for `@g2g`-tagged yaks
4. Falls back to regular actionable yaks if none are tagged

## Step 4: Send PR feedback into the loop

When someone reviews the PR and leaves a comment, send it back into the workflow:

```bash
yyx pr-feedback \
    --pr-number 42 \
    --comment "Please fix the lint warning in payment.py" \
    --author reviewer
```

The workflow re-dispatches the agent to address the feedback and re-watches CI.

## Step 5: Observe the merge

Once CI passes on the PR, the workflow merges it and marks the yak as done. The status returns to `idle`, ready for the next signal.

```bash
yyx status
```

## What you learned

- How CI completion events connect to the Temporal workflow as signals
- How the workflow reacts to CI signals — triaging, claiming, implementing
- How PR feedback feeds back into the agent rework loop
- How the loop closes autonomously when CI passes and the PR merges

This is the core yak shaving loop: CI → agent → PR → CI → merge. Each CI completion starts a new turn, and the barber manages the entire lifecycle durably.

> For alternative ways to wire up CI (multi-workflow, separate repo), see [Integrate with GitHub Actions](../how-to/integrate-github-actions.md).
> For a full reference of all signals, see [Signals Reference](../reference/signals.md).
> To understand why this architecture uses Temporal, see [Why Temporal](../explanation/why-temporal.md).
