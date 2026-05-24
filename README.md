# yaketyyak

Temporal workflow for CI-triggered and g2g-triggered autonomous yak shaving.

When a CI build finishes — or when a yak is tagged `@g2g` ("good to go") — a durable Temporal workflow claims the yak, dispatches a coding agent (Pi, Codex, Claude Code, or OpenCode), watches the resulting PR through CI, handles review feedback, merges, and marks the yak done. It survives crashes and reboots.

```
CI finished ──┐
Yak tagged @g2g ─┤──→ Temporal Workflow → claim → implement → PR → merge → done
PR feedback ───┘
```

## Prerequisites

- [Nix](https://nixos.org/download) with [flakes](https://nixos.wiki/wiki/Flakes) enabled
- [direnv](https://direnv.net) (recommended) — auto-loads the dev environment on `cd`
- A GitHub [personal access token](https://github.com/settings/tokens) with `repo` scope
- The [yx CLI](https://github.com/mattwynne/yaks) installed and connected to your repository

## Quick start

```bash
git clone https://github.com/funkymonkeymonk/yaketyyak
cd yaketyyak
direnv allow                          # activates devenv with Go + Temporal CLI
devenv task project:setup             # build the yyx CLI
GITHUB_TOKEN=ghp_xxx devenv up        # start Temporal dev server + worker
yyx start --repo your-name/your-repo --repo-root .  # start workflow
```

> For a full step-by-step walkthrough, see the [tutorial](docs/tutorials/from-zero-to-shaving.md).

## Using with your own fork

To develop yaketyyak or test changes against your own fork:

```bash
git clone https://github.com/your-name/yaketyyak
cd yaketyyak
git remote add upstream https://github.com/funkymonkeymonk/yaketyyak
direnv allow
go build -o yyx .
```

The `yyx` binary you just built will point the worker at your fork. See the [Workflow Options](docs/reference/workflow-options.md) for `--repo` and `--repo-root` flags.

## Quick links

| For... | Go here |
|--------|---------|
| Learning by doing | [Tutorials](docs/tutorials/from-zero-to-shaving.md) |
| Accomplishing a task | [How-to guides](docs/how-to/start-the-workflow.md) |
| API details & options | [Reference](docs/reference/signals.md) |
| Background & design | [Explanation](docs/explanation/architecture.md) |
