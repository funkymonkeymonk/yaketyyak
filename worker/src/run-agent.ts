import { execFileSync } from "node:child_process";
import { mkdirSync, rmSync, writeFileSync } from "node:fs";
import { join } from "node:path";
import { Context } from "@temporalio/activity";
import {
  AuthStorage,
  createAgentSession,
  DefaultResourceLoader,
  getAgentDir,
  ModelRegistry,
  SessionManager,
} from "@earendil-works/pi-coding-agent";
import { DEFAULT_PI_MODEL, DEFAULT_PI_TOOLS, INCOMPATIBLE_MODELS, type PiConfig } from "./types.js";

// Default heartbeat interval: heartbeat every 30s so Temporal's 2-minute
// heartbeat timeout is never reached even during long-running tool calls.
const HEARTBEAT_INTERVAL_MS = 30_000;

// Default maximum wall-clock time for a single agent run (2 hours).
// Overridden per-yak via PiConfig.maxRunTimeSeconds.
const DEFAULT_MAX_RUN_TIME_SECONDS = 2 * 60 * 60;

export async function RunAgent(
  yakName: string,
  workspaceName: string,
  cfg: PiConfig,
  feedbackContext?: string,
): Promise<void> {
  const repoRoot = process.cwd();
  const workspacePath = join(repoRoot, ".workspaces", workspaceName);
  const modelId = cfg.model || DEFAULT_PI_MODEL;
  const tools = cfg.tools?.length ? cfg.tools : DEFAULT_PI_TOOLS;
  const maxRunMs = (cfg.maxRunTimeSeconds ?? DEFAULT_MAX_RUN_TIME_SECONDS) * 1000;

  if (INCOMPATIBLE_MODELS.has(modelId)) {
    throw new Error(
      `Model "${modelId}" does not support Pi tool-calling and cannot be used for agent runs. ` +
      `Choose a compatible model (e.g. claude-sonnet-4-6, claude-haiku-4-5-20251001, claude-opus-4-5-20251101) ` +
      `via --pi-model or the yak's agent_config.`,
    );
  }

  const contextFile = writeContextFile(yakName, workspacePath, feedbackContext);

  // Start an async background heartbeat so Temporal never sees a heartbeat gap
  // longer than HEARTBEAT_INTERVAL_MS, regardless of how long individual tool
  // calls take.  The ticker also enforces the maxRunTime wall-clock limit by
  // cancelling the AbortController when the deadline is exceeded.
  const abort = new AbortController();
  let lastTool = "starting";
  const startedAt = Date.now();

  const heartbeatTicker = (async () => {
    while (!abort.signal.aborted) {
      await new Promise<void>((resolve) => {
        const t = setTimeout(resolve, HEARTBEAT_INTERVAL_MS);
        abort.signal.addEventListener("abort", () => { clearTimeout(t); resolve(); }, { once: true });
      });
      if (abort.signal.aborted) break;

      const elapsed = Date.now() - startedAt;
      if (elapsed >= maxRunMs) {
        console.error(`[RunAgent] max run time exceeded (${Math.round(elapsed / 1000)}s), aborting`);
        abort.abort(new Error(`RunAgent exceeded max run time of ${cfg.maxRunTimeSeconds ?? DEFAULT_MAX_RUN_TIME_SECONDS}s`));
        break;
      }

      try {
        Context.current().heartbeat(`tool: ${lastTool} (${Math.round(elapsed / 1000)}s elapsed)`);
      } catch {
        // Activity may have been cancelled by Temporal — stop ticking.
        abort.abort();
        break;
      }
    }
  })();

  try {
    await runPi({
      workspacePath,
      modelId,
      tools,
      skills: cfg.skills ?? [],
      contextFile,
      abortSignal: abort.signal,
      onTool: (name) => { lastTool = name; },
    });

    if (abort.signal.aborted) {
      throw abort.signal.reason instanceof Error
        ? abort.signal.reason
        : new Error("RunAgent aborted");
    }
  } finally {
    abort.abort(); // stop the ticker
    await heartbeatTicker;
    try { rmSync(contextFile, { force: true }); } catch { /* ignore */ }
  }
}

async function runPi(opts: {
  workspacePath: string;
  modelId: string;
  tools: string[];
  skills: string[];
  contextFile: string;
  abortSignal: AbortSignal;
  onTool: (name: string) => void;
}): Promise<void> {
  const { workspacePath, modelId, tools, contextFile, abortSignal, onTool } = opts;

  const authStorage = AuthStorage.create();
  if (process.env.LITELLM_API_KEY) {
    authStorage.setRuntimeApiKey("litellm", process.env.LITELLM_API_KEY);
  }
  const modelRegistry = ModelRegistry.create(authStorage);

  const loader = new DefaultResourceLoader({
    cwd: workspacePath,
    agentDir: getAgentDir(),
    additionalExtensionPaths: ["npm:pi-provider-litellm"],
  });
  await loader.reload();

  // Create the session without a model — let the extension register its models first.
  const { session } = await createAgentSession({
    cwd: workspacePath,
    tools,
    authStorage,
    modelRegistry,
    resourceLoader: loader,
    sessionManager: SessionManager.inMemory(workspacePath),
  });

  // Now resolve the model — extension has had a chance to register litellm models.
  const available = await modelRegistry.getAvailable();
  const resolved = available.find(
    (m) => m.provider === "litellm" && (m.id === modelId || m.id.includes(modelId)),
  );

  if (!resolved) {
    const ids = available.filter((m) => m.provider === "litellm").map((m) => m.id);
    session.dispose();
    throw new Error(
      `LiteLLM model "${modelId}" not found. Available: ${ids.join(", ") || "(none — is LITELLM_BASE_URL set?)"}`,
    );
  }

  await session.setModel(resolved);

  session.subscribe((event) => {
    switch (event.type) {
      case "message_update":
        if (event.assistantMessageEvent.type === "text_delta") {
          process.stdout.write(event.assistantMessageEvent.delta);
        }
        break;
      case "tool_execution_start":
        console.log(`[pi] tool: ${event.toolName}`);
        onTool(event.toolName);
        break;
      case "tool_execution_end":
        if (event.isError) {
          console.error(`[pi] tool error: ${event.toolName}`);
        }
        break;
      case "agent_end":
        console.log("[pi] agent done");
        break;
    }
  });

  const prompt = [
    `@${contextFile}`,
    "Implement this yak. Follow the spec exactly. When done, commit your changes with git (git add -A && git commit).",
  ].join("\n");

  // Race the agent prompt against the abort signal so max-run-time is enforced
  // even if the Pi session itself hangs.
  await Promise.race([
    session.prompt(prompt),
    new Promise<never>((_, reject) => {
      if (abortSignal.aborted) {
        reject(abortSignal.reason);
      } else {
        abortSignal.addEventListener("abort", () => reject(abortSignal.reason), { once: true });
      }
    }),
  ]);
  session.dispose();
}

function writeContextFile(yakName: string, workspacePath: string, feedbackContext?: string): string {
  const out = execFileSync("yx", ["show", yakName, "--format", "json"], {
    encoding: "utf8",
  });

  const data = JSON.parse(out) as { context?: string; has_context?: boolean };
  if (!data.has_context || !data.context) {
    throw new Error(
      `Yak "${yakName}" has no context — add a spec with: yx context ${yakName}`,
    );
  }

  mkdirSync(workspacePath, { recursive: true });
  const contextFile = join(workspacePath, ".yak-context.md");

  let contents = data.context;
  if (feedbackContext) {
    contents += `\n\n---\n\n${feedbackContext}\n`;
  }

  writeFileSync(contextFile, contents, "utf8");
  return contextFile;
}
