---
name: testing-zellij-layouts
description: Use when modifying zellij layout KDL files in this project, before presenting layout changes to the user or committing them
---

# Testing Zellij Layouts

## Overview

Zellij KDL layout files have a specific schema that's easy to get wrong. A layout that parses successfully may still have behavioral issues (like panes showing "waiting to run"). Always test layout changes before presenting them.

## When to Test

Any edit to `devenv/zellij/layout.kdl` or any other `.kdl` layout file in this project.

## How to Test

Parse-check the layout without affecting the running session:

```bash
zellij --layout path/to/layout.kdl 2>&1
```

⚠️ **Only run this test OUTSIDE a zellij session.** If you run `zellij --layout` from inside a zellij session, it creates a new tab instead of testing the layout.

Since this runs outside a real TTY, you'll either see:
- **Layout parse error** — output includes `Failed to parse Zellij configuration` with a line number and `split_direction should be either "horizontal" or "vertical"` or similar
- **TTY error (layout OK)** — output shows `could not enable raw mode: Os { code: 6, ... }` with no parse error. **This means the layout is valid.** The raw-mode error is from running without a terminal, not from your layout.

## Common Traps

| Symptom | Cause | Fix |
|---|---|---|
| `split_direction should be either "horizontal" or "vertical" found: stacked` | Stacked panes not supported in layouts | Use `stacked=true` property on pane, or use separate tabs |
| `Failed to parse Zellij configuration` | Invalid KDL syntax at the reported line | Check attribute names, closing braces, quoting |
| Pane shows "waiting to run" in zellij | `start_suspended true` on the pane | Remove `start_suspended true` from that pane |
| Pane with children and no `split_direction` | Layout may render incorrectly | Add `split_direction="vertical"` or `split_direction="horizontal"` |

## Zellij Layout KDL Reference

Important syntax rules for this project:

- `split_direction` only accepts `"horizontal"` or `"vertical"` — no `"stacked"`
- To stack panes (cycle between them in the same space), set `stacked=true` on the parent pane
- `start_suspended true` makes a pane show "waiting to run" — remove it to auto-start
- Commands must be in `$PATH` inside the devenv shell
- Plugin panes use `plugin location="..."` instead of `command="..."`
- `focus=true` sets the initial focused pane

## Verification Flow

```
Edit layout.kdl
  → Parse-check: `zellij --layout devenv/zellij/layout.kdl 2>&1` (outside session!)
  → Check for "Failed to parse"
  → If parse error: fix reported line, re-test
  → If only raw-mode error: layout is valid
  → Apply to running session: `zellij action override-layout devenv/zellij/layout.kdl --apply-only-to-active-tab`
  → Present to user
```

## Related

- This project's layout: `devenv/zellij/layout.kdl`
- Layout is launched by the `dev-session` script in `devenv.nix`
