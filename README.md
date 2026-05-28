# yaketyyak

Autonomous yak shaving, end to end.

yaketyyak runs a durable Temporal workflow that claims a yak, dispatches the Pi coding agent via LiteLLM to implement it, opens a draft PR, then waits for the PR to be merged and marks the yak done — all without human intervention. It survives crashes and reboots.

```
Yak claimed ──────────────────→ implement → draft PR → merged → done
```

## Prerequisites

- [Nix](https://nixos.org/download) with [flakes](https://nixos.wiki/wiki/Flakes) enabled
- [direnv](https://direnv.net) (recommended) — auto-loads the dev environment on `cd`
- A GitHub [personal access token](https://github.com/settings/tokens) with `repo` scope
- The [yx CLI](https://github.com/mattwynne/yaks) installed and connected to your repository
- `LITELLM_BASE_URL` and `LITELLM_API_KEY` set in the worker environment

## Quick start

```bash
git clone https://github.com/funkymonkeymonk/yaketyyak
cd yaketyyak
direnv allow                          # activates devenv with Go + Temporal CLI
devenv tasks run yyx:install          # build the yyx CLI
GITHUB_TOKEN=ghp_xxx devenv up        # start Temporal dev server + worker
yyx shave <yak-name>                  # start a YakWorkflow for the named yak
```

> For a full step-by-step walkthrough, see the [tutorial](docs/tutorials/from-zero-to-shaving.md).

## How it works

1. `yyx shave <yak-name>` starts a `YakWorkflow` on the Temporal task queue
2. The worker claims the yak, creates an isolated jj workspace, and dispatches Pi via LiteLLM
3. Pi implements the yak and commits in the workspace
4. The workflow opens a draft PR and waits for it to be merged
5. On merge, the workflow marks the yak done and cleans up the workspace

## Quick links

| For... | Go here |
|--------|---------|
| Learning by doing | [Tutorials](docs/tutorials/from-zero-to-shaving.md) |
| Accomplishing a task | [How-to guides](docs/how-to/start-the-workflow.md) |
| API details & options | [Reference](docs/reference/yak-workflow.md) |
| Background & design | [Explanation](docs/explanation/architecture.md) |
