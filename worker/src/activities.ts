import { execFileSync } from "node:child_process";
import { mkdirSync, rmSync } from "node:fs";
import { join } from "node:path";
import { Context } from "@temporalio/activity";
import { Octokit } from "@octokit/rest";
import type { PRResult, PRWatchResult, ReviewComment } from "./types.js";

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

export async function InitWorkspace(
  repoUrl: string,
  yakName: string,
): Promise<string> {
  const slug = sanitizeID(yakName.toLowerCase());
  const workspaceName = `shave-${slug}`;
  const branch = `shave/${slug}`;

  // Determine workspaces root relative to the repo being cloned.
  // We store clones alongside the main checkout in .workspaces/.
  const repoRoot = process.cwd();
  const workspacePath = join(repoRoot, ".workspaces", workspaceName);

  // Clean any stale workspace from a previous run.
  rmSync(workspacePath, { recursive: true, force: true });
  mkdirSync(join(repoRoot, ".workspaces"), { recursive: true });

  // Clone the repo into a fresh isolated directory.
  const cloneUrl = authenticatedUrl(repoUrl);
  run("git", ["clone", "--depth", "1", cloneUrl, workspacePath]);

  // Create and switch to the shave branch.
  run("git", ["checkout", "-b", branch], { cwd: workspacePath });

  // Configure git identity for commits inside the workspace.
  run("git", ["config", "user.email", "yaketyyak[bot]@users.noreply.github.com"], { cwd: workspacePath });
  run("git", ["config", "user.name", "yaketyyak[bot]"], { cwd: workspacePath });

  return workspaceName;
}

export async function CleanupWorkspace(workspaceName: string): Promise<void> {
  const repoRoot = process.cwd();
  const workspacePath = join(repoRoot, ".workspaces", workspaceName);
  rmSync(workspacePath, { recursive: true, force: true });
}

// --- PR lifecycle ---

export async function CreateDraftPR(
  repoUrl: string,
  workspaceName: string,
  yakName: string,
): Promise<PRResult> {
  const repoRoot = process.cwd();
  const workspacePath = join(repoRoot, ".workspaces", workspaceName);
  const slug = sanitizeID(yakName.toLowerCase());
  const branch = `shave/${slug}`;
  const { owner, repo } = parseRepoUrl(repoUrl);

  // Push the branch.
  const cloneUrl = authenticatedUrl(repoUrl);
  run("git", ["push", cloneUrl, `${branch}:${branch}`], { cwd: workspacePath });

  // Create the draft PR via Octokit.
  const octokit = getOctokit();
  const { data: pr } = await octokit.rest.pulls.create({
    owner,
    repo,
    title: yakName,
    head: branch,
    base: "main",
    draft: true,
    body: `Automated shave by [yaketyyak](https://github.com/funkymonkeymonk/yaketyyak)\n\nYak: \`${yakName}\``,
  });

  return { prUrl: pr.html_url, prNumber: pr.number };
}

export async function WatchPRState(
  prNumber: number,
  repoUrl: string,
  lastReviewId: number,
): Promise<PRWatchResult> {
  const { owner, repo } = parseRepoUrl(repoUrl);
  const octokit = getOctokit();
  const pollMs = 60_000;

  while (true) {
    Context.current().heartbeat(`watching PR #${prNumber}`);

    try {
      const { data: pr } = await octokit.rest.pulls.get({ owner, repo, pull_number: prNumber });
      if (pr.merged) return { outcome: "merged" };
      if (pr.state === "closed") return { outcome: "closed" };
    } catch {
      // transient error — retry next tick
    }

    try {
      const { data: reviews } = await octokit.rest.pulls.listReviews({
        owner,
        repo,
        pull_number: prNumber,
      });

      // Find the most recent CHANGES_REQUESTED review with id > lastReviewId.
      const feedbackReview = reviews
        .filter((r) => r.state === "CHANGES_REQUESTED" && r.id > lastReviewId)
        .sort((a, b) => b.id - a.id)[0];

      if (feedbackReview) {
        // Fetch per-line comments for this review.
        const comments: ReviewComment[] = [];
        try {
          const { data: rawComments } = await octokit.rest.pulls.listCommentsForReview({
            owner,
            repo,
            pull_number: prNumber,
            review_id: feedbackReview.id,
          });
          for (const c of rawComments) {
            comments.push({
              path: c.path,
              line: c.line ?? c.original_line ?? null,
              body: c.body,
            });
          }
        } catch {
          // Best-effort: continue without per-line comments
        }

        return {
          outcome: "feedback",
          reviewId: feedbackReview.id,
          reviewBody: feedbackReview.body ?? "",
          reviewComments: comments,
        };
      }
    } catch {
      // transient error — retry next tick
    }

    await sleep(pollMs);
  }
}

export async function RespondToReview(
  repoUrl: string,
  prNumber: number,
  _reviewId: number,
  message: string,
): Promise<void> {
  const { owner, repo } = parseRepoUrl(repoUrl);
  const octokit = getOctokit();
  await octokit.rest.issues.createComment({
    owner,
    repo,
    issue_number: prNumber,
    body: message,
  });
}

export async function PushFeedbackCommit(
  repoUrl: string,
  workspaceName: string,
  round: number,
): Promise<void> {
  const repoRoot = process.cwd();
  const workspacePath = join(repoRoot, ".workspaces", workspaceName);
  const cloneUrl = authenticatedUrl(repoUrl);

  // Stage all changes made by the agent.
  run("git", ["add", "-A"], { cwd: workspacePath });

  // Commit. Allow empty commits in case the agent made no file changes.
  tryRun("git", ["commit", "--allow-empty", "-m", `feedback round ${round}: address review comments`], { cwd: workspacePath });

  // Determine the current branch name and push.
  const branch = run("git", ["rev-parse", "--abbrev-ref", "HEAD"], { cwd: workspacePath });
  run("git", ["push", cloneUrl, `${branch}:${branch}`], { cwd: workspacePath });
}

// --- helpers ---

interface RunOpts { cwd?: string }

function run(cmd: string, args: string[], opts: RunOpts = {}): string {
  return execFileSync(cmd, args, {
    cwd: opts.cwd ?? process.cwd(),
    encoding: "utf8",
  }).trim();
}

function tryRun(cmd: string, args: string[], opts: RunOpts = {}): void {
  try { run(cmd, args, opts); } catch { /* best-effort */ }
}

function getOctokit(): Octokit {
  const token = process.env.GITHUB_TOKEN;
  if (!token) throw new Error("GITHUB_TOKEN is not set");
  return new Octokit({ auth: token });
}

/** Inject GITHUB_TOKEN into the clone URL for authenticated push/pull. */
function authenticatedUrl(repoUrl: string): string {
  const token = process.env.GITHUB_TOKEN;
  if (!token) throw new Error("GITHUB_TOKEN is not set");
  return repoUrl.replace("https://", `https://x-access-token:${token}@`);
}

function parseRepoUrl(repoUrl: string): { owner: string; repo: string } {
  // Accepts https://github.com/owner/repo or github.com/owner/repo
  const clean = repoUrl.replace(/^https?:\/\//, "").replace(/\.git$/, "");
  const parts = clean.split("/");
  const owner = parts[parts.length - 2];
  const repo = parts[parts.length - 1];
  if (!owner || !repo) throw new Error(`Cannot parse repo URL: ${repoUrl}`);
  return { owner, repo };
}

function sanitizeID(name: string): string {
  return name.replace(/[^a-z0-9\-_]/g, "-");
}

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}
