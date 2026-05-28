> **This feature is not yet implemented.** This document describes planned behaviour.

# How to Mark a Yak as Good to Go

The `@g2g` tag tells yaketyyak that a yak is ready for autonomous implementation.

## Create a yak

```bash
yx add "Add unit tests for the payment module"
```

## Add context

Every yak needs context before it can be implemented autonomously:

```bash
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
```

## Tag it @g2g

```bash
yx tag add "Add unit tests for the payment module" @g2g
yx sync
```

That's it. The next time the workflow scans for g2g yaks (either via `g2g_signal`, periodic scan, or CI trigger), it picks up this yak.

## Verify the tag

```bash
yx show "Add unit tests for the payment module"
```

Look for `Tags: [@g2g]` in the output.

## Remove the tag manually

If you change your mind:

```bash
yx tag remove "Add unit tests for the payment module" @g2g
yx sync
```

> For a tutorial walking through the full process, see [From Zero to Yak Shaving](../tutorials/from-zero-to-shaving.md).
> For the signals reference, see [Signals](../reference/signals.md).
