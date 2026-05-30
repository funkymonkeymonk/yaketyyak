# yaketyyak — Agent Guide

## Dev environment

This project uses **devenv** (Nix-based). If `.direnv/` exists, the shell is
loaded automatically. Otherwise run `devenv shell`.

### Discovering tasks

Tasks are defined in `devenv.nix`. **Do not hardcode task names** — discover
them dynamically:

```
devenv tasks list
devenv tasks run <name>
```

To run all tasks in a namespace:

```
devenv tasks run yyx
devenv tasks run project
```

### MCP server

devenv provides an MCP server for AI assistants that support it:

```
devenv mcp                        # stdio mode (for Claude Code etc.)
devenv mcp --http 8080            # HTTP mode
```

Exposes `search_packages` (nixpkgs) and `search_options` (devenv config).

## Project architecture

`yyx` — a Cobra CLI that starts/manages Temporal workflows for autonomous
yak-shaving (CI → yak → PR lifecycle).

```
cmd/              CLI subcommands (root, start, worker, cisignal, ...)
main.go           Entry point
temporal/         Temporal workflow definitions
devenv/           Zellij layout and other dev config
docs/             Diataxis documentation (tutorials, how-to, reference, explanation)
.yaks/            Yak task state (managed by `yx` CLI)
```

### Key workflow

1. `yyx start` kicks off a Temporal `BarberWorkflow`
2. The workflow listens for signals: CI results, g2g tags, PR feedback
3. On trigger, it dispatches a coding agent (Pi, Claude Code, Codex, OpenCode)
4. The agent implements changes, opens a PR, and the workflow monitors CI

### LLM configuration

The workflow uses an LLM-backed agent configured via `--llm-base-url` and
`--llm-model` flags. The API key is read from environment variables.

## Version control

This project uses **jj** (Jujutsu). The `.jj/` directory contains the repo;
`.jj/repo/store/` backs it with git. Treat jj as the primary VCS interface.

## Task tracking

Yaks are managed by `yx` CLI. State is stored in `.yaks/`:

```
yx list           # show all yaks
yx add <name>     # create a yak
yx start <name>   # mark as wip
yx done <name>    # mark complete
```

## TUI design

The TUI follows patterns distilled from [gh-dash](https://github.com/dlvhdr/gh-dash). All TUI
design decisions — layout, panes, resizing, component structure, key handling,
styling — must follow the authoritative reference at
[docs/explanation/tui-design.md](docs/explanation/tui-design.md).

## Session Protocol

Every work session — human or agent — follows this two-phase ritual.

### Triage (session start, ~5 min)

1. `yx sync` — pull latest state from remote
2. `yx ls` — survey the full map
3. Set a hard stop time before starting (e.g. "done by 17:00")
4. Pick **at most 2 yaks** to work on; note them explicitly before touching any code
5. Check for `@needs-human` yaks and surface them to the human before diving in:
   ```
   yx ls | grep @needs-human
   ```

**WIP limit: 2 yaks maximum per session.** More than two means context-switching
overhead exceeds value delivered. If a yak blocks, note it and move to the other
one — do not open a third.

### Wrap (session end)

1. `yx sync` — push all state changes made during the session
2. Update context on any in-progress yak with a progress note:
   ```
   echo "Did X, Y. Blocked on Z." | yx context "<yak name>"
   ```
3. Leave a one-line handoff note in every wip yak's context for the next session
4. Prune noise if the map is cluttered:
   ```
   yx ls --all   # identify done/stale yaks
   yx prune      # remove them
   ```
5. `yx sync` — final push so the next session (human or agent) starts clean

## Code conventions

- Go 1.26+
- Temporal workflows in `temporal/`
- CLI commands in `cmd/` using Cobra
- Tests: none yet (`go test ./...` should pass)
