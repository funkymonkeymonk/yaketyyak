# Tutorial: From Zero to Yak Shaving

In this tutorial you will set up yaketyyak from scratch using devenv, start a Temporal worker, create a yak, and watch it get implemented autonomously by the Pi coding agent.

## Prerequisites

- [Nix](https://nixos.org/download) with [flakes](https://nixos.wiki/wiki/Flakes) enabled
- [direnv](https://direnv.net) — auto-loads the dev environment when you `cd` into the project
- A GitHub [personal access token](https://github.com/settings/tokens) with `repo` scope
- The [yx CLI](https://github.com/mattwynne/yaks) installed and connected to your repository
- `LITELLM_BASE_URL` and `LITELLM_API_KEY` for your LiteLLM gateway

## Step 1: Clone yaketyyak

```bash
git clone https://github.com/funkymonkeymonk/yaketyyak
cd yaketyyak
```

## Step 2: Activate the dev environment

If you have direnv installed, allow it:

```bash
direnv allow
```

This reads `.envrc`, which runs `devenv` and puts Go, `temporal`, and other tools in your PATH.

Without direnv, activate manually:

```bash
devenv shell
```

Verify it worked:

```bash
go version        # Go from nixpkgs
temporal --version
```

## Step 3: Build the yyx CLI

```bash
devenv tasks run yyx:install
```

This builds the `yyx` binary and installs it locally. Verify:

```bash
yyx --help
```

You should see the `shave` subcommand listed.

## Step 4: Start all processes

```bash
export GITHUB_TOKEN=ghp_xxx
export LITELLM_BASE_URL=http://your-litellm-host:4000
export LITELLM_API_KEY=your-key
devenv up
```

This starts both the Temporal dev server and one worker process. Leave it running in a terminal.

You should see the worker log:

```
Worker started, polling task queue: yaketyyak-tasks
```

The Temporal Web UI is available at `http://localhost:8233`.

## Step 5: Create a yak with context

In your target repository:

```bash
cd /path/to/your/repo
yx add "Add unit tests for the payment module"
cat <<'EOF' | yx context "Add unit tests for the payment module"
## Goal
Add unit test coverage for the payment module's core logic.

## Acceptance Criteria
- [ ] Tests for the validate method
- [ ] Tests for the process_refund method
- [ ] All existing tests still pass

## Files
- tests/test_payment.py
- src/payment.py
EOF
yx sync
```

## Step 6: Start the YakWorkflow

```bash
yyx shave "Add unit tests for the payment module" \
    --repo-url https://github.com/your-name/your-repo
```

Output:

```
Started YakWorkflow for "Add unit tests for the payment module"
  Workflow ID: yyx-yak-add-unit-tests-for-the-payment-module
  Run ID:      <temporal-run-id>
  Repo URL:    https://github.com/your-name/your-repo
```

The workflow is now running. It will:
1. Claim the yak with `yx start`
2. Clone the repo into an isolated workspace under `.workspaces/`
3. Dispatch Pi via LiteLLM to implement the yak
4. Open a draft PR
5. Wait for you to review and merge the PR

## Step 7: Monitor progress

Check the workflow status in the Temporal Web UI at `http://localhost:8233`, or query it directly:

```bash
temporal workflow query \
    --workflow-id yyx-yak-add-unit-tests-for-the-payment-module \
    --type yak_status
```

You'll see the current phase (`claiming`, `implementing`, `waiting-for-merge`, etc.).

## Step 8: Review and merge the PR

When the `implementing` phase completes, a draft PR is opened. Review the changes, mark it ready for review, and merge it. The workflow detects the merge and:

1. Marks the yak done with `yx done`
2. Syncs with `yx sync`
3. Cleans up the workspace

Verify:

```bash
cd /path/to/your/repo
yx ls --all | grep "payment module"
```

The yak should show state `done`.

## Step 9: Cancel a shave (optional)

If you want to abandon a running shave, send the `wont-do` signal:

```bash
temporal workflow signal \
    --workflow-id yyx-yak-add-unit-tests-for-the-payment-module \
    --name wont-do
```

The workflow releases the yak (returns it to `todo`) and stops.

## What you learned

- How to set up the yaketyyak dev environment with devenv
- How to build the `yyx` CLI with `devenv tasks run yyx:install`
- How to start all processes (Temporal dev server + worker) with `devenv up`
- How to create a yak with context and start a `YakWorkflow` with `yyx shave`
- How to monitor workflow progress via the Temporal Web UI or `yak_status` query
- How the workflow handles the full cycle from claim to merge
- How to cancel a running shave with the `wont-do` signal

> For a deeper understanding of the architecture, see [Architecture](../explanation/architecture.md).
> For all workflow flags, see [Workflow Options](../reference/workflow-options.md).
> For the full workflow definition, see [YakWorkflow Reference](../reference/yak-workflow.md).
