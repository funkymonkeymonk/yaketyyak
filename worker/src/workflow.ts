import {
  condition,
  defineQuery,
  defineSignal,
  proxyActivities,
  setHandler,
  upsertSearchAttributes,
} from "@temporalio/workflow";
import type * as allActivities from "./activities.js";
import type { PiConfig, PRResult, WorkflowConfig, YakWorkflowState } from "./types.js";

// Default max agent run time: 2 hours. The Temporal startToCloseTimeout is set
// to maxRunTimeSeconds + 5 minutes so Temporal only kills the activity if the
// in-process abort mechanism itself fails.
const DEFAULT_MAX_RUN_TIME_SECONDS = 2 * 60 * 60;

const act = proxyActivities<typeof allActivities>({
  startToCloseTimeout: "30 seconds",
  retry: { maximumAttempts: 3 },
});

const actNoRetry = proxyActivities<typeof allActivities>({
  startToCloseTimeout: "60 seconds",
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

export const wontDoSignal = defineSignal("wont-do");
export const yakStatusQuery = defineQuery<YakWorkflowState>("yak_status");

export async function YakWorkflow(
  yakName: string,
  cfg: WorkflowConfig,
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

  // Set initial search attributes so the workflow is findable by yak name.
  upsertSearchAttributes({ YakName: [yakName], Phase: [state.phase] });

  const setPhase = (phase: string) => {
    state.phase = phase;
    upsertSearchAttributes({ Phase: [phase] });
  };

  // Build a RunAgent proxy whose startToCloseTimeout is derived from the
  // configured maxRunTimeSeconds (+ 5 min buffer so Temporal only fires if the
  // in-process abort mechanism itself fails).
  const agentMaxSecs = cfg.pi.maxRunTimeSeconds ?? DEFAULT_MAX_RUN_TIME_SECONDS;
  const agentTimeoutSecs = agentMaxSecs + 5 * 60;
  const { RunAgent } = proxyActivities<{
    RunAgent(yakName: string, workspaceName: string, cfg: PiConfig): Promise<void>;
  }>({
    startToCloseTimeout: `${agentTimeoutSecs} seconds`,
    heartbeatTimeout: "2 minutes",
    retry: { maximumAttempts: 1 },
  });

  // 1. Claim the yak.
  setPhase("claiming");
  await act.YakClaim(yakName);

  try {
    // 2. Init workspace — fresh git clone on a new branch.
    setPhase("init-workspace");
    state.workspace = await actNoRetry.InitWorkspace(cfg.repoUrl, yakName);

    try {
      // 3. Run Pi agent.
      setPhase("implementing");
      await RunAgent(yakName, state.workspace, cfg.pi);

      if (wontDo) {
        return `yak ${yakName} marked won't-do during implementation`;
      }

      // 4. Create draft PR.
      setPhase("creating-pr");
      const pr: PRResult = await CreateDraftPR(cfg.repoUrl, state.workspace, yakName);
      state.prUrl = pr.prUrl;
      state.prNumber = pr.prNumber;
      upsertSearchAttributes({ PrUrl: [pr.prUrl] });

      act.WritePRToYak(yakName, pr.prUrl).catch(() => {});

      // 5. Wait for merge or won't-do.
      setPhase("waiting-for-merge");
      let merged = false;
      WatchPRMerged(pr.prNumber, cfg.repoUrl).then((m) => { merged = m; }).catch(() => {});
      await condition(() => wontDo || merged);

      if (wontDo) {
        await act.YakRelease(yakName, "marked won't-do");
        return `yak ${yakName} marked won't-do`;
      }

    } finally {
      actNoRetry.CleanupWorkspace(state.workspace).catch(() => {});
    }

  } catch (err) {
    if (state.phase !== "done") {
      await act.YakRelease(yakName, `workflow interrupted at phase ${state.phase}`).catch(() => {});
    }
    throw err;
  }

  // 6. Close the yak.
  setPhase("done");
  await act.YakMarkDone(yakName);

  return `yak ${yakName} done (PR ${state.prUrl})`;
}
