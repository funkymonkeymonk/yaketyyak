> **This feature is not yet implemented.** This document describes planned behaviour.

# How to Tag Yaks as Good to Go

Automatic `@g2g` scanning — where the workflow picks up yaks tagged `@g2g` and shaves them without explicit invocation — is not yet implemented. The `g2g_signal` and `yyx g2g-scan` command do not exist in the current codebase.

To shave a yak today, use `yyx shave` directly:

```bash
yyx shave <yak-name> --repo-url https://github.com/owner/repo
```

> For the current way to start a shave, see [Start the Workflow](start-the-workflow.md).
> For all workflow options, see [Workflow Options](../reference/workflow-options.md).
