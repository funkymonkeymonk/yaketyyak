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
import { DEFAULT_PI_MODEL, DEFAULT_PI_TOOLS, type PiConfig } from "./types.js";

export async function RunAgent(
  yakName: string,
  repoRoot: string,
  workspaceName: string,
  cfg: PiConfig,
): Promise<void> {
  const workspacePath = join(repoRoot, ".workspaces", workspaceName);
  const modelId = cfg.model || DEFAULT_PI_MODEL;
  const tools = cfg.tools?.length ? cfg.tools : DEFAULT_PI_TOOLS;

  const contextFile = writeContextFile(yakName, workspacePath);

  try {
    await runPi({ workspacePath, modelId, tools, skills: cfg.skills ?? [], contextFile });
  } finally {
    try { rmSync(contextFile, { force: true }); } catch { /* ignore */ }
  }
}

async function runPi(opts: {
  workspacePath: string;
  modelId: string;
  tools: string[];
  skills: string[];
  contextFile: string;
}): Promise<void> {
  const { workspacePath, modelId, tools, contextFile } = opts;

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
        Context.current().heartbeat(`tool: ${event.toolName}`);
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
    "Implement this yak. Follow the spec exactly. When done, commit your changes with jj.",
  ].join("\n");

  await session.prompt(prompt);
  session.dispose();
}

function writeContextFile(yakName: string, workspacePath: string): string {
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
  writeFileSync(contextFile, data.context, "utf8");
  return contextFile;
}
