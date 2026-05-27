# Tutorial: From Zero to Yak Shaving

In this tutorial you will set up yaketyyak from scratch using devenv, start a Temporal workflow, mark a yak as ready on your personal fork, and watch it get implemented autonomously.

## Prerequisites

- [Nix](https://nixos.org/download) with [flakes](https://nixos.org/wiki/Flakes) enabled
- [direnv](https://direnv.net) — auto-loads the dev environment when you `cd` into the project
- A GitHub [personal access token](https://github.com/settings/tokens) with `repo` scope
- The [yx CLI](https://github.com/mattwynne/yaks) installed and connected to your repository
- A GitHub repository you own (or a fork of one) to use as your test target

## Step 1: Clone yaketyyak

You can use either the upstream repo or your own fork. If you plan to develop yaketyyak itself, use your fork:

```bash
git clone https://github.com/your-name/yaketyyak
cd yaketyyak

# Optional: add upstream to pull future changes
git remote add upstream https://github.com/funkymonkeymonk/yaketyyak
```

## Step 2: Activate the dev environment

If you have direnv installed, allow it:

```bash
direnv allow
```

This reads `.envrc`, which runs `devenv` and puts Go, `temporal`, `gopls`, and other tools in your PATH.

Without direnv, activate manually:

```bash
devenv shell
```

Either way, verify it worked:

```bash
go version        # should show Go from nixpkgs
temporal --version
```

## Step 3: Build the yyx CLI

```bash
devenv tasks run yyx:install
```

This builds the `yyx` binary and runs linting and tests. The resulting binary is your control tool for starting workflows and sending signals.

## Step 4: Start all processes

```bash
export GITHUB_TOKEN=ghp_xxx
devenv up
```

This starts both the Temporal dev server and one worker process via `process-compose`. Leave it running in a terminal.

You should see the worker log:
```
Worker started, polling task queue: yaketyyak-tasks
```

The Temporal Web UI is available at `http://localhost:8233`.

To run additional workers, open another terminal and scale:
```bash
devenv up --scale worker=3
```

## Step 5: Start the workflow

In a second terminal:

```bash
cd yaketyyak
direnv allow
yyx start \
    --repo your-name/your-repo \
    --repo-root /path/to/your/repo \
    --agent pi
```

Replace `your-name/your-repo` with your GitHub repository and the path to your local checkout.

Output:

```
✦ Started workflow: yaketyyak-yak-shaving
```

Now the workflow exists on Temporal and is waiting for signals.

## Step 6: Point yyx at your fork (if using one)

If you cloned your own fork of yaketyyak in Step 1, you are already pointing at your fork — the `yyx` binary you built is running from your code. To develop features or fix bugs, edit the Go source in `cmd/` or `temporal/`, rebuild, and test on your fork's repositories.

The key commands when using your fork:

```bash
# Rebuild after changes
go build -o yyx .

# Start a workflow against your personal project
yyx start --repo your-name/test-repo --repo-root ~/code/test-repo --agent opencode
```

You can have multiple yaketyyak forks in different directories, each with its own modifications.

## Step 7: Mark a yak as ready

In your target repository (the one you passed to `--repo`), create a yak with a `@g2g` tag:

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
yx tag add "Add unit tests for the payment module" @g2g
yx sync
```

## Step 8: Trigger a g2g scan

Back in the yaketyyak directory:

```bash
yyx g2g-scan
```

The workflow will:
1. Sync yaks
2. Find your `@g2g`-tagged yak
3. Claim it and remove the `@g2g` tag
4. Dispatch your agent to implement it
5. Create a PR
6. Watch CI on the PR

## Step 9: Check progress

```bash
yyx status
```

You'll see the current phase and which yak is being worked on.

## Step 10: Merge completes automatically

Once CI passes on the PR, the workflow merges it and marks the yak as done. Verify:

```bash
yx ls --all | grep "Add unit tests"
```

If successful, the yak should show state `done`.

## What you learned

- How to set up the yaketyyak dev environment with devenv
- How to build the `yyx` CLI with `devenv tasks run yyx:install`
- How to start all processes (Temporal dev server + worker) with `devenv up`
- How to scale workers with `devenv up --scale worker=N`
- How to start the BarberWorkflow pointing at your repository or fork
- How to mark a yak with `@g2g` as ready for autonomous implementation
- How to trigger a g2g scan and monitor progress
- How the workflow handles the full cycle from triage to merge

> For a deeper understanding of the architecture, see [Architecture](../explanation/architecture.md).
> For production deployment, see [Integrate with GitHub Actions](../how-to/integrate-github-actions.md).
> For all workflow parameters, see [Workflow Options](../reference/workflow-options.md).
