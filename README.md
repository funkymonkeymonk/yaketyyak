# yaketyyak

Autonomous yak shaving, end to end.

yaketyyak runs a durable Temporal workflow that claims a yak, dispatches a coding agent (Pi, Codex, Claude Code, or OpenCode) to implement it, opens a PR, handles review feedback, merges, and marks the yak done — all without human intervention. It survives crashes and reboots.

CI signals are one way the workflow is nudged forward: when a build breaks during shaving, the CI result resumes the workflow so the agent can fix it. A `@g2g` tag and PR review feedback do the same.

```
Yak claimed ──────────────────→ implement → PR → merge → done
                                    ↑
CI finished ──┐                     │ (resume after interruption)
Yak tagged @g2g ─┤─────────────────┘
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
