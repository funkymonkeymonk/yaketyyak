---
name: triage-wrap
description: Session discipline for agentic work — triage at start, wrap at end. Load at the beginning of every work session to enforce the two-phase protocol and WIP limit.
---

# Session Protocol

Every work session — human or agent — follows this two-phase ritual. No
exceptions. Unbounded sessions cause scope creep, lost context, and yak state
that drifts from reality.

## Triage (session start, ~5 min)

Run these steps **before touching any code or claiming any yak**:

```bash
yx sync                       # 1. Pull latest state from remote
yx ls                         # 2. Survey the full map
# 3. Decide on a hard stop time and note it (e.g. "done by 17:00")
yx ls | grep @needs-human     # 4. Surface blocked yaks to the human first
```

Then **pick at most 2 yaks** and write them down:

```
Today's yaks:
  1. <yak name>
  2. <yak name>   (optional second)
```

Do not start work until this list is written.

### WIP limit: 2 yaks maximum

More than two means context-switching overhead exceeds value delivered. If a
yak blocks, note the blocker in its context and move to the other one. Never
open a third yak mid-session.

## Wrap (session end)

Run these steps **before closing the session**:

```bash
# 1. Update context on every in-progress yak
echo "Did X, Y. Blocked on Z. Next: W." | yx context "<yak name>"

# 2. Sync state to remote
yx sync

# 3. Prune noise if map is cluttered
yx ls --all          # identify done/stale yaks
yx prune             # remove them

# 4. Final sync so next session starts clean
yx sync
```

The progress note in step 1 is the handoff for the next session (human or
agent). It must answer: what was done, what is blocked, and what to do next.

## Quick reference

| Phase | Key command | Purpose |
|-------|-------------|---------|
| Triage | `yx sync` | Pull remote state |
| Triage | `yx ls` | Survey the map |
| Triage | `yx ls \| grep @needs-human` | Surface blockers |
| Triage | pick ≤ 2 yaks | Enforce WIP limit |
| Wrap | `echo "..." \| yx context "<yak>"` | Leave handoff note |
| Wrap | `yx sync` | Push state |
| Wrap | `yx prune` | Remove noise |
