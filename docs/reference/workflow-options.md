# Workflow Options Reference

Flags accepted by `yyx shave`.

## --repo-root

Type: `string` (optional, default: current directory)

Absolute or relative path to the local checkout of the repository to work in. The workflow passes this to Pi as the working directory and uses it for jj workspace operations.

```bash
yyx shave my-yak --repo-root /path/to/repo
```

## --pi-model

Type: `string` (optional, default: `""`)

LiteLLM model name to pass to Pi. If unset, Pi uses the gateway's configured default model.

```bash
yyx shave my-yak --pi-model anthropic/claude-sonnet-4
```

## --pi-tools

Type: `[]string` (optional, default: `read,bash,edit,write`)

Comma-separated list of Pi tools to enable. The defaults are sufficient for most yaks.

```bash
yyx shave my-yak --pi-tools read,bash,edit,write
```

## --pi-skill

Type: `[]string` (optional, repeatable)

Path(s) to Pi skill files to load via `--skill`. Can be specified multiple times.

```bash
yyx shave my-yak \
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
