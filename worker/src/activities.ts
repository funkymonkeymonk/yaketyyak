import { execFileSync, execSync } from "node:child_process";
import { mkdirSync, rmSync } from "node:fs";
import { join } from "node:path";
import { Context } from "@temporalio/activity";
import type { PRResult } from "./types.js";

// --- Yak lifecycle ---

export async function YakClaim(yakName: string): Promise<void> {
  run("yx", ["start", yakName]);
  tryRun("yx", ["sync"]);
}

export async function YakRelease(yakName: string, _reason: string): Promise<void> {
  tryRun("yx", ["state", yakName, "todo"]);
  tryRun("yx", ["sync"]);
}

export async function YakMarkDone(yakName: string): Promise<void> {
  run("yx", ["done", yakName]);
  tryRun("yx", ["sync"]);
}

export async function WritePRToYak(yakName: string, prUrl: string): Promise<void> {
  tryRun("yx", ["field", yakName, "pr", prUrl]);
}

// --- Workspace lifecycle ---

export async function InitWorkspace(repoRoot: string, yakName: string): Promise<string> {
  const slug = sanitizeID(yakName.toLowerCase());
  const workspaceName = `shave-${slug}`;
  const workspacePath = `.workspaces/${workspaceName}`;
  const fullPath = join(repoRoot, workspacePath);

  // Ensure parent exists and clear any stale directory from a previous run.
  mkdirSync(join(repoRoot, ".workspaces"), { recursive: true });
  rmSync(fullPath, { recursive: true, force: true });

  run("jj", ["git", "fetch"], { cwd: repoRoot });
  run("jj", ["workspace", "add", "--name", workspaceName, workspacePath], { cwd: repoRoot });

  return workspaceName;
}

export async function CleanupWorkspace(repoRoot: string, workspaceName: string): Promise<void> {
  tryRun("jj", ["workspace", "forget", workspaceName], { cwd: repoRoot });
}

// --- PR lifecycle ---

export async function CreateDraftPR(
  repoRoot: string,
  workspaceName: string,
  _yakName: string,
): Promise<PRResult> {
  const workspacePath = join(repoRoot, ".workspaces", workspaceName);

  run("jj", ["git", "push", "--allow-new"], { cwd: workspacePath });

  const repo = repoRemoteToPath(repoRoot);
  const createArgs = ["pr", "create", "--draft", "--fill", ...(repo ? ["--repo", repo] : [])];
  const output = run("gh", createArgs, { cwd: workspacePath });

  return parsePROutput(output);
}

export async function WatchPRMerged(prNumber: number, repoRoot: string): Promise<boolean> {
  const repo = repoRemoteToPath(repoRoot);
  const pollMs = 60_000;

  while (true) {
    Context.current().heartbeat(`watching PR #${prNumber}`);

    try {
      const args = ["pr", "view", String(prNumber), "--json", "state,merged", ...(repo ? ["--repo", repo] : [])];
      const out = run("gh", args, { cwd: repoRoot });
      const pr = JSON.parse(out) as { state: string; merged: boolean };
      if (pr.merged) return true;
    } catch {
      // transient gh error — retry next tick
    }

    await sleep(pollMs);
  }
}

// --- helpers ---

interface RunOpts {
  cwd?: string;
  env?: NodeJS.ProcessEnv;
}

function run(cmd: string, args: string[], opts: RunOpts = {}): string {
  return execFileSync(cmd, args, {
    cwd: opts.cwd,
    env: opts.env ?? process.env,
    encoding: "utf8",
  }).trim();
}

function tryRun(cmd: string, args: string[], opts: RunOpts = {}): void {
  try {
    run(cmd, args, opts);
  } catch {
    // best-effort
  }
}

function repoRemoteToPath(repoRoot: string): string {
  try {
    const url = execFileSync("git", ["remote", "get-url", "origin"], {
      cwd: repoRoot,
      encoding: "utf8",
    }).trim();
    return url
      .replace(/^.*:\/\//, "")
      .replace(/^git@/, "")
      .replace(":", "/")
      .replace(/\.git$/, "")
      .replace(/^\//, "");
  } catch {
    return "";
  }
}

function parsePROutput(output: string): PRResult {
  for (const line of output.split("\n")) {
    const trimmed = line.trim();
    if (trimmed.includes("/pull/")) {
      const parts = trimmed.split("/pull/");
      if (parts.length === 2) {
        const prNumber = parseInt(parts[1].trim(), 10);
        return { prUrl: trimmed, prNumber: isNaN(prNumber) ? 0 : prNumber };
      }
    }
  }
  throw new Error(`Could not parse PR URL from: ${output}`);
}

function sanitizeID(name: string): string {
  return name.replace(/[^a-z0-9\-_]/g, "-");
}

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}
