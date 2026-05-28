import {
  condition,
  defineQuery,
  defineSignal,
  proxyActivities,
  setHandler,
} from "@temporalio/workflow";
import type * as allActivities from "./activities.js";
import type { PiConfig, PRResult, YakWorkflowState } from "./types.js";

// Separate proxy groups with different timeout/retry settings.
const act = proxyActivities<typeof allActivities>({
  startToCloseTimeout: "30 seconds",
  retry: { maximumAttempts: 3 },
});

const actNoRetry = proxyActivities<typeof allActivities>({
  startToCloseTimeout: "60 seconds",
  retry: { maximumAttempts: 1 },
});

// RunAgent has a long timeout and is registered separately in the worker.
// We type it loosely here to avoid importing from run-agent.ts (Node API).
const { RunAgent } = proxyActivities<{
  RunAgent(yakName: string, repoRoot: string, workspaceName: string, cfg: PiConfig): Promise<void>;
}>({
  startToCloseTimeout: "4 hours",
  heartbeatTimeout: "2 minutes",
  retry: { maximumAttempts: 1 },
});

const { CreateDraftPR } = proxyActivities<typeof allActivities>({
  startToCloseTimeout: "5 minutes",
  retry: { maximumAttempts: 1 },
});

const { WatchPRMerged } = proxyActivities<typeof allActivities>({
  startToCloseTimeout: "168 hours",
  heartbeatTimeout: "2 minutes",
  retry: { maximumAttempts: 1 },
});

// Signals & queries
export const wontDoSignal = defineSignal("wont-do");
export const yakStatusQuery = defineQuery<YakWorkflowState>("yak_status");

export async function YakWorkflow(
  yakName: string,
  repoRoot: string,
  cfg: PiConfig,
): Promise<string> {
  const state: YakWorkflowState = {
    yakName,
    phase: "init",
    workspace: "",
    prUrl: "",
    prNumber: 0,
  };

  let wontDo = false;
  setHandler(wontDoSignal, () => { wontDo = true; });
  setHandler(yakStatusQuery, () => ({ ...state }));

  // 1. Claim the yak.
  state.phase = "claiming";
  await act.YakClaim(yakName);

  try {
    // 2. Init workspace.
    state.phase = "init-workspace";
    state.workspace = await actNoRetry.InitWorkspace(repoRoot, yakName);

    try {
      // 3. Run Pi agent.
      state.phase = "implementing";
      await RunAgent(yakName, repoRoot, state.workspace, cfg);

      if (wontDo) {
        return `yak ${yakName} marked won't-do during implementation`;
      }

      // 4. Create draft PR.
      state.phase = "creating-pr";
      const pr: PRResult = await CreateDraftPR(repoRoot, state.workspace, yakName);
      state.prUrl = pr.prUrl;
      state.prNumber = pr.prNumber;

      act.WritePRToYak(yakName, pr.prUrl).catch(() => {});

      // 5. Wait for merge or won't-do.
      state.phase = "waiting-for-merge";
      let merged = false;
      WatchPRMerged(pr.prNumber, repoRoot).then((m) => { merged = m; }).catch(() => {});
      await condition(() => wontDo || merged);

      if (wontDo) {
        await act.YakRelease(yakName, "marked won't-do");
        return `yak ${yakName} marked won't-do`;
      }

    } finally {
      actNoRetry.CleanupWorkspace(repoRoot, state.workspace).catch(() => {});
    }

  } catch (err) {
    if (state.phase !== "done") {
      await act.YakRelease(yakName, `workflow interrupted at phase ${state.phase}`).catch(() => {});
    }
    throw err;
  }

  // 6. Close the yak.
  state.phase = "done";
  await act.YakMarkDone(yakName);

  return `yak ${yakName} done (PR ${state.prUrl})`;
}
