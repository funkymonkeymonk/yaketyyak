package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/funkymonkeymonk/yaketyyak/temporal"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

func main() {
	c, err := temporal.NewClient()
	if err != nil {
		log.Fatalf("Failed to create Temporal client: %v", err)
	}
	defer c.Close()

	// Workflow-only worker — activities are handled by the TypeScript worker.
	// MaxConcurrentActivityExecutionSize=0 effectively disables activity polling.
	w := worker.New(c, temporal.TaskQueue, worker.Options{
		MaxConcurrentActivityExecutionSize: 1,
	})

	w.RegisterWorkflowWithOptions(temporal.YakWorkflow, workflow.RegisterOptions{
		Name: "YakWorkflow",
	})

	if err := w.Start(); err != nil {
		log.Fatalf("Failed to start workflow worker: %v", err)
	}

	log.Printf("Workflow worker started on task queue: %s", temporal.TaskQueue)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	w.Stop()
}
