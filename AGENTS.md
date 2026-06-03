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
devenv tasks run yy
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

`yy` — a Cobra CLI that starts/manages Temporal workflows for autonomous
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

1. `yy start` kicks off a Temporal `BarberWorkflow`
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

## Yak Templates

Standard templates for creating and refining yaks are in `.yaks/templates/`:

- `.yaks/templates/basic.md` — standard shave-ready yak with Problem / Acceptance Criteria / Files / Notes sections
- `.yaks/templates/with-agent-config.md` — same as basic, plus an `agent_config` field section with example `yx field` commands

Copy the appropriate template and fill it in when creating or refining a yak. Use `with-agent-config.md` when you need to control which model or tools the agent uses.

## TUI design

The TUI follows patterns distilled from [gh-dash](https://github.com/dlvhdr/gh-dash). All TUI
design decisions — layout, panes, resizing, component structure, key handling,
styling — must follow the authoritative reference at
[docs/explanation/tui-design.md](docs/explanation/tui-design.md).

## Code conventions

- Go 1.26+
- Temporal workflows in `temporal/`
- CLI commands in `cmd/` using Cobra
- Tests: none yet (`go test ./...` should pass)

## KDL files

Zellij layouts and other config use the KDL format. The dev environment provides
`kdlfmt` for formatting and validation:

```
kdlfmt check devenv/zellij/layout.kdl    # validate (exit code 0 if OK)
kdlfmt format devenv/zellij/layout.kdl   # auto-format in-place
```

Run `kdlfmt check` on any edited `.kdl` file before presenting the result.
