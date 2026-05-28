import { Worker } from "@temporalio/worker";
import * as activities from "./activities.js";
import { RunAgent } from "./run-agent.js";
import { TASK_QUEUE } from "./types.js";
import { fileURLToPath } from "node:url";
import { dirname, join } from "node:path";

const __dirname = dirname(fileURLToPath(import.meta.url));

async function main() {
  const worker = await Worker.create({
    taskQueue: TASK_QUEUE,
    // Temporal runs workflow code in a deterministic v8 sandbox.
    // workflowsPath points at the compiled workflow module.
    workflowsPath: join(__dirname, "workflow.js"),
    activities: {
      ...activities,
      RunAgent,
    },
  });

  console.log(`Worker started, polling task queue: ${TASK_QUEUE}`);
  await worker.run();
}

main().catch((err) => {
  console.error("Worker failed:", err);
  process.exit(1);
});
