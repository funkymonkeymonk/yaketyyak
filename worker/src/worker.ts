import { Worker } from "@temporalio/worker";
import * as activities from "./activities.js";
import { RunAgent } from "./run-agent.js";
import { TASK_QUEUE } from "./types.js";
import { fileURLToPath } from "node:url";
import { dirname, join } from "node:path";
import * as http from "node:http";

const __dirname = dirname(fileURLToPath(import.meta.url));

async function main() {
  const port = parseInt(process.env.HEALTH_PORT ?? "8080", 10);
  let ready = false;

  const server = http.createServer((req, res) => {
    if (req.url === "/health" && req.method === "GET") {
      if (ready) {
        res.writeHead(200, { "Content-Type": "application/json" });
        res.end(JSON.stringify({ status: "ok", taskQueue: TASK_QUEUE }));
      } else {
        res.writeHead(503, { "Content-Type": "application/json" });
        res.end(JSON.stringify({ status: "error", reason: "worker not yet ready" }));
      }
    } else {
      res.writeHead(404);
      res.end();
    }
  });

  await new Promise<void>((resolve) => server.listen(port, resolve));
  console.log(`Health server listening on port ${port}`);

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

  ready = true;
  console.log(`Worker started, polling task queue: ${TASK_QUEUE}`);
  await worker.run();
}

main().catch((err) => {
  console.error("Worker failed:", err);
  process.exit(1);
});
