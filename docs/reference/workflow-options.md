# Workflow Options Reference

Flags accepted by `yy shave`.

## --repo-root

Type: `string` (optional, default: current directory)

Absolute or relative path to the local checkout of the repository to work in. The workflow passes this to Pi as the working directory and uses it for jj workspace operations.

```bash
yy shave my-yak --repo-root /path/to/repo
```

## --pi-model

Type: `string` (optional, default: `"claude-sonnet-4-6"`)

LiteLLM model name to pass to Pi. If unset, Pi uses `claude-sonnet-4-6`.

```bash
yy shave my-yak --pi-model claude-haiku-4-5-20251001
```

### Known-good models

Only models that correctly implement the tool-calling protocol used by Pi will work.
Models **not** on this list may produce `malformed_model_output` errors.

| Model | Speed | Cost | Recommended for |
|-------|-------|------|-----------------|
| `claude-haiku-4-5-20251001` | fast | cheap | docs, config, simple fixes |
| `claude-sonnet-4-6` | medium | medium | **default** — general purpose |
| `claude-opus-4-5-20251101` | slow | expensive | complex refactors, architecture |

### Incompatible models

The following models are **known incompatible** with Pi tool-calling. The worker
will fast-fail with a clear error if one of these is configured:

- `moonshotai.kimi-k2.5`
- `moonshot.kimi-k2-thinking`

To extend this list, edit `INCOMPATIBLE_MODELS` in `worker/src/types.ts`.

## --pi-tools

Type: `[]string` (optional, default: `read,bash,edit,write`)

Comma-separated list of Pi tools to enable. The defaults are sufficient for most yaks.

```bash
yy shave my-yak --pi-tools read,bash,edit,write
```

## --pi-skill

Type: `[]string` (optional, repeatable)

Path(s) to Pi skill files to load via `--skill`. Can be specified multiple times.

```bash
yy shave my-yak \
    --pi-skill /path/to/skill-a.md \
    --pi-skill /path/to/skill-b.md
```

## Environment variables (worker)

These are read by the **worker process**, not by `yy shave` itself.

| Variable | Required | Description |
|----------|----------|-------------|
| `LITELLM_BASE_URL` | Yes | LiteLLM gateway URL (e.g. `http://localhost:4000`) |
| `LITELLM_API_KEY` | Yes | LiteLLM API key |
| `GITHUB_TOKEN` | Yes | GitHub personal access token with `repo` scope |

> For how to use these flags, see [Start the Workflow](../how-to/start-the-workflow.md).
> For the data types behind these options, see [Data Types](data-types.md).
