import { Worker } from "@temporalio/worker";
import {
  YakClaim,
  YakRelease,
  YakMarkDone,
  WritePRToYak,
  InitWorkspace,
  CleanupWorkspace,
  CreateDraftPR,
  WatchPRMerged,
} from "./activities.js";
import { RunAgent } from "./run-agent.js";
import { TASK_QUEUE } from "./types.js";

async function main() {
  const worker = await Worker.create({
    taskQueue: TASK_QUEUE,
    activities: {
      YakClaim,
      YakRelease,
      YakMarkDone,
      WritePRToYak,
      InitWorkspace,
      CleanupWorkspace,
      RunAgent,
      CreateDraftPR,
      WatchPRMerged,
    },
  });

  console.log(`Worker started, polling task queue: ${TASK_QUEUE}`);
  await worker.run();
}

main().catch((err) => {
  console.error("Worker failed:", err);
  process.exit(1);
});
