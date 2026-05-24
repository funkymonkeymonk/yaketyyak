# Why Temporal

Three approaches were prototyped before converging on Temporal. This explains the decision.

## Approaches considered

### 1. Pi extension

A Pi extension that polls GitHub CI status and calls `pi -p` to start a new agent turn.

**Strengths:** Lightweight, no infrastructure, fits Pi's philosophy.

**Limitations:** Pi only. No durability — crash loses the loop. No PR feedback handling. Polling-based (rate limits, delay).

### 2. Open Ralph Wiggum wrapper

A bash wrapper around [Open Ralph Wiggum](https://github.com/Th0rgal/open-ralph-wiggum) that loops until a completion signal.

**Strengths:** Cross-agent (Claude Code, Codex, OpenCode). Simple bash script.

**Limitations:** Terminal-session-scoped. No durability. No PR feedback integration. Manual status checking.

### 3. Temporal workflow (chosen)

A Temporal.io durable workflow that receives signals and orchestrates the full lifecycle.

**Strengths:** Durable execution, signal-driven, long-running, retry policies, visibility (Web UI), any agent via activity dispatch.

**Limitation:** Requires Temporal infrastructure (dev server or Cloud).

## Why Temporal won

| Dimension | Pi | ORW | Temporal |
|-----------|----|-----|----------|
| Survives crash | No | No | Yes (event history replay) |
| PR feedback loop | Manual | Manual | Signal-driven rework |
| Long-running (days+) | Terminal session | Terminal session | Designed for this |
| Multi-agent | Pi only | Claude Code, Codex, OpenCode | Any (activity dispatch) |
| Visibility | None | None | Web UI + CLI |
| Retries | None | None | Per-activity retry policy |
| Complexity | Low | Low | Moderate (Temporal infra) |

Temporal scales down (dev server on a laptop) and up (Temporal Cloud for teams). The infrastructure cost is justified by durability — losing a 6-hour agent loop to a crash is worse than running a dev server.

> For the architecture, see [Architecture](architecture.md).
> For a tutorial getting started, see [From Zero to Yak Shaving](../tutorials/from-zero-to-shaving.md).
