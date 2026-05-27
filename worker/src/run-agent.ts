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
  const model = cfg.model || DEFAULT_PI_MODEL;
  const tools = cfg.tools?.length ? cfg.tools : DEFAULT_PI_TOOLS;

  // Fetch yak context from yx and write to a temp file in the workspace.
  const contextFile = writeContextFile(yakName, workspacePath);

  try {
    await runPi({ workspacePath, model, tools, skills: cfg.skills ?? [], contextFile });
  } finally {
    try { rmSync(contextFile, { force: true }); } catch { /* ignore */ }
  }
}

async function runPi(opts: {
  workspacePath: string;
  model: string;
  tools: string[];
  skills: string[];
  contextFile: string;
}): Promise<void> {
  const { workspacePath, model, tools, contextFile } = opts;

  // Auth: Pi picks up LITELLM_BASE_URL + LITELLM_API_KEY from environment.
  // Use setRuntimeApiKey to inject so it doesn't touch auth.json on disk.
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

  // Resolve the LiteLLM model.
  const resolvedModel = modelRegistry.find("litellm", model);
  if (!resolvedModel) {
    throw new Error(`Model not found: litellm/${model}. Check LITELLM_BASE_URL is set and the model exists.`);
  }

  const { session } = await createAgentSession({
    cwd: workspacePath,
    model: resolvedModel,
    tools,
    authStorage,
    modelRegistry,
    resourceLoader: loader,
    sessionManager: SessionManager.inMemory(workspacePath),
  });

  // Stream events to logs and heartbeat on tool executions.
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
