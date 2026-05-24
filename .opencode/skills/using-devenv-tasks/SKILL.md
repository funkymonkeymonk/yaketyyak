---
name: using-devenv-tasks
description: Use when working in any project with a devenv.nix file that defines tasks, when building, testing, linting, cleaning, or running project commands that may have devenv task equivalents
---

# Using Devenv Tasks

## Overview

When a project defines tasks in `devenv.nix`, always discover and use them instead of raw toolchain commands. Tasks ensure correct Nix environment, respect dependency ordering, and are the project's canonical way to run operations.

## Decision Flow

```
Working on a project → devenv.nix exists? → `devenv tasks list` → use tasks
                                          → no dev file → native commands
```

## Pattern

```bash
# Before: raw toolchain (bad)
go build -o yyx . && go test ./... && go vet ./...

# After: devenv tasks (correct)
devenv tasks list                    # discover tasks
devenv tasks run yyx:build           # chains lint → test → build
```

Never hardcode task names — `devenv tasks list` shows the dependency tree dynamically. Tasks use `namespace:name` format; running a namespace runs all tasks within it.

## Dependency Awareness

Tasks declare `after` dependencies — `devenv tasks run yyx:build` automatically runs its listed `after` tasks (lint, test, etc.) first. Pick the highest-level task that covers what you need.

## Common Mistakes & Red Flags

| Symptom | Fix |
|---|---|
| Typing `go build` from muscle memory | Stop. `devenv tasks list` first. |
| Running lint, test, build separately | Use the build task (it chains dependencies) |
| Guessing or hardcoding task names | Always discover with `devenv tasks list` |
| "I already know the Go/Cargo/npm commands" | The task may have custom env/ordering |
| "It's faster to type the native command" | It's faster to get wrong results too |
| "This is trivial, the overhead isn't worth it" | Tasks enforce correctness; trivial != safe |

**Any of these: Stop. Run `devenv tasks list`. Use the tasks.**
