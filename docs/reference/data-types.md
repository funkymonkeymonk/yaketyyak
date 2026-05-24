# Data Types Reference

Core data types used by the `BarberWorkflow`.

## CISignal

Carries CI pipeline completion data.

```python
@dataclass
class CISignal:
    conclusion: str    # "success" | "failure" | "cancelled"
    branch: str        # git branch
    sha: str           # commit SHA
    details: str       # optional JSON with run metadata
```

## PRFeedback

Carries a PR review comment for the rework loop.

```python
@dataclass
class PRFeedback:
    pr_number: int     # GitHub PR number
    comment: str       # review body
    author: str        # reviewer handle
```

## YakInfo

Tracks the currently active yak.

```python
@dataclass
class YakInfo:
    name: str          # yak name
    state: str         # "todo" | "wip" | "done"
    context: str       # yak context markdown
    tags: list[str]    # yak tags (e.g. ["@g2g"])
    pr_number: int | None
    pr_url: str | None
    g2g: bool          # was this g2g-triggered?
```

## WorkflowState

Full workflow state accessible via the `status()` query.

```python
@dataclass
class WorkflowState:
    phase: str                    # "idle" | "triaging" | "claiming" | "implementing" | "watching-ci" | "reviewing"
    current_yak: YakInfo | None
    pending_ci_signals: list[CISignal]
    pending_pr_feedback: list[PRFeedback]
    pending_g2g_scans: int
    completed_yaks: int
    failed_yaks: int
    repo: str
    repo_root: str
    g2g_mode: bool
```

> For signal payloads, see [Signals Reference](signals.md).
> For the full workflow definition, see `temporal/workflow.go`.
