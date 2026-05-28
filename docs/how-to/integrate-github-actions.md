> **This feature is not yet implemented.** This document describes planned behaviour.

# How to Integrate with GitHub Actions

Connect your CI pipeline to yaketyyak so every build completion triggers a yak shaving turn.

## Prerequisites

- A running Temporal dev server or Temporal Cloud namespace
- The yaketyyak worker running somewhere accessible to GitHub Actions

## Option 1: Signal from the same repo

Add this workflow to your repository:

```yaml
# .github/workflows/yaketyyak.yml
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
      - run: |
          ./yyx ci \
            --conclusion ${{ github.event.workflow_run.conclusion }} \
            --branch ${{ github.event.workflow_run.head_branch }} \
            --sha ${{ github.event.workflow_run.head_sha }}
```

## Option 2: Signal from a dedicated yaketyyak repo

If the yaketyyak worker and starter live in a separate infrastructure repo, clone it first:

```yaml
      - uses: actions/checkout@v4
        with:
          repository: your-org/yaketyyak
          path: yaketyyak
      - uses: actions/setup-go@v5
        with:
          go-version: stable
      - run: |
          cd yaketyyak
          go build -o yyx .
      - run: |
          cd yaketyyak
          ./yyx ci \
            --conclusion ${{ github.event.workflow_run.conclusion }} \
            --branch ${{ github.event.workflow_run.head_branch }} \
            --sha ${{ github.event.workflow_run.head_sha }}
```

## Option 3: Multiple CI workflows

Listen for any of your CI workflows by name:

```yaml
on:
  workflow_run:
    workflows: ["CI", "Lint", "Integration Tests"]
    types: [completed]
```

The workflow handles each signal independently.

## Troubleshooting

| Problem | Check |
|---------|-------|
| Signal not received | Temporal dev server running? Worker started? |
| Connection refused | Temporal target defaults to `localhost:7233` |
| Auth error | Set `TEMPORAL_NAMESPACE` and `TEMPORAL_API_KEY` for Cloud |

> For a walkthrough of the full CI-driven flow, see [CI-Driven Loop Tutorial](../tutorials/ci-driven-loop.md).
> For all workflow options, see [Workflow Options](../reference/workflow-options.md).
