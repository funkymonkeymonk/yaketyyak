# Data Types Reference

Core data types used by the `BarberWorkflow`.

## CISignal

Carries CI pipeline completion data.

```go
type CISignal struct {
    Conclusion string // "success" | "failure" | "cancelled"
    Branch     string // git branch
    SHA        string // commit SHA
    Details    string // optional JSON with run metadata
}
```

## PRFeedback

Carries a PR review comment for the rework loop.

```go
type PRFeedback struct {
    PRNumber int    // GitHub PR number
    Comment  string // review body
    Author   string // reviewer handle
}
```

## YakInfo

Tracks the currently active yak.

```go
type YakInfo struct {
    Name     string
    State    string   // "todo" | "wip" | "done"
    Context  string   // yak context markdown
    Tags     []string // yak tags (e.g. ["@g2g"])
    PRNumber int
    PRURL    string
    G2G      bool     // was this g2g-triggered?
}
```

## WorkflowState

Full workflow state accessible via the `status()` query.

```go
type WorkflowState struct {
    Phase             string   // "idle" | "triaging" | "claiming" | "implementing" | "watching-ci" | "reviewing"
    CurrentYak        *YakInfo
    PendingCISignals  []CISignal
    PendingPRFeedback []PRFeedback
    PendingG2GScans   int
    CompletedYaks     int
    FailedYaks        int
    Repo              string
    RepoRoot          string
    G2GMode           bool
}
```

> For signal payloads, see [Signals Reference](signals.md).
> For the full workflow definition, see `temporal/workflow.go`.
