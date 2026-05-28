# Workflow Options Reference

Flags accepted by `yyx shave`.

## --repo-url

Type: `string` (required)

GitHub repository URL. Used for cloning and PR creation.

```bash
yyx shave my-yak --repo-url https://github.com/owner/repo
```

## --pi-model

Type: `string` (optional, default: `"claude-sonnet-4-6"`)

LiteLLM model name to pass to Pi. If unset, the default model (`claude-sonnet-4-6`) is used.

```bash
yyx shave my-yak --repo-url https://github.com/owner/repo --pi-model anthropic/claude-sonnet-4
```

## --pi-tools

Type: `[]string` (optional, default: `read,bash,edit,write`)

Comma-separated list of Pi tools to enable. The defaults are sufficient for most yaks.

```bash
yyx shave my-yak --repo-url https://github.com/owner/repo --pi-tools read,bash,edit,write
```

## --pi-skill

Type: `[]string` (optional, repeatable)

Path(s) to Pi skill files to load. Can be specified multiple times.

```bash
yyx shave my-yak --repo-url https://github.com/owner/repo \
    --pi-skill /path/to/skill-a.md \
    --pi-skill /path/to/skill-b.md
```

## Environment variables (worker)

These are read by the **worker process**, not by `yyx shave` itself.

| Variable | Required | Description |
|----------|----------|-------------|
| `LITELLM_BASE_URL` | Yes | LiteLLM gateway URL (e.g. `http://localhost:4000`) |
| `LITELLM_API_KEY` | Yes | LiteLLM API key |
| `GITHUB_TOKEN` | Yes | GitHub personal access token with `repo` scope |

> For how to use these flags, see [Start the Workflow](../how-to/start-the-workflow.md).
> For the data types behind these options, see [Data Types](data-types.md).
