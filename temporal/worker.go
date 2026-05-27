package temporal

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

func RunWorker() {
	c, err := client.Dial(client.Options{
		HostPort: "localhost:7233",
	})
	if err != nil {
		log.Fatalf("Failed to create Temporal client: %v", err)
	}
	defer c.Close()

	w := worker.New(c, TaskQueue, worker.Options{})

	w.RegisterWorkflowWithOptions(YakWorkflow, workflow.RegisterOptions{
		Name: "YakWorkflow",
	})

	w.RegisterActivity(YakClaim)
	w.RegisterActivity(YakRelease)
	w.RegisterActivity(YakMarkDone)
	w.RegisterActivity(WritePRToYak)
	w.RegisterActivity(InitWorkspace)
	w.RegisterActivity(CleanupWorkspace)
	w.RegisterActivity(RunAgent)
	w.RegisterActivity(CreateDraftPR)
	w.RegisterActivity(WatchPRMerged)

	if err := w.Start(); err != nil {
		log.Fatalf("Failed to start worker: %v", err)
	}

	log.Printf("Worker started, polling task queue: %s", TaskQueue)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	w.Stop()
}
